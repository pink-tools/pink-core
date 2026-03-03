package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// HandleActions registers --actions and per-action subcommands into cfg.Commands.
// Call before core.Run().
func HandleActions(cfg *Config, actions []Action, handlers map[string]ActionHandler) {
	if cfg.Commands == nil {
		cfg.Commands = make(map[string]Command)
	}

	cfg.Commands["--actions"] = Command{
		Desc: "List available actions (JSON)",
		Run: func(args []string) error {
			data, err := json.Marshal(actions)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}

	for _, action := range actions {
		handler, ok := handlers[action.Name]
		if !ok {
			continue
		}
		h := handler // capture
		cfg.Commands[action.Name] = Command{
			Desc: action.Desc,
			Run:  makeActionRunner(h),
		}
	}
}

func makeActionRunner(h ActionHandler) func([]string) error {
	return func(args []string) error {
		if len(args) > 0 {
			switch args[0] {
			case "--describe":
				spec := h.Describe()
				data, err := json.MarshalIndent(spec, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil

			case "--config":
				if len(args) < 2 {
					return fmt.Errorf("--config requires JSON argument")
				}
				var values map[string]any
				if err := json.Unmarshal([]byte(args[1]), &values); err != nil {
					return fmt.Errorf("invalid JSON: %w", err)
				}
				return h.Execute(values)

			case "--yes":
				spec := h.Describe()
				values := fillDefaults(spec.Fields)
				return h.Execute(values)
			}
		}

		// No args: interactive if TTY, error otherwise
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("no TTY — use --describe or --config")
		}

		spec := h.Describe()
		values, err := promptFields(spec)
		if err != nil {
			return err
		}
		return h.Execute(values)
	}
}

func fillDefaults(fields []Field) map[string]any {
	values := make(map[string]any)
	for _, f := range fields {
		switch {
		case f.Current != nil:
			values[f.Name] = f.Current
		case f.Default != nil:
			values[f.Name] = f.Default
		}
	}
	return values
}

func promptFields(spec FormSpec) (map[string]any, error) {
	fmt.Println()
	fmt.Println(spec.Title)
	fmt.Println(strings.Repeat("─", len(spec.Title)))
	if spec.Message != "" {
		fmt.Println(spec.Message)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	values := make(map[string]any)

	for _, f := range spec.Fields {
		val, err := promptField(reader, f)
		if err != nil {
			return nil, err
		}
		if val != nil {
			values[f.Name] = val
		}
	}

	return values, nil
}

func promptField(reader *bufio.Reader, f Field) (any, error) {
	switch f.Type {
	case "confirm":
		return promptConfirm(reader, f)
	case "password":
		return promptPassword(f)
	case "select", "sound":
		return promptSelect(reader, f)
	case "number", "range":
		return promptNumber(reader, f)
	default:
		// text, url, file, hotkey — all readline
		return promptText(reader, f)
	}
}

func promptText(reader *bufio.Reader, f Field) (any, error) {
	label := f.Label
	if f.Required {
		label += " (required)"
	}
	fmt.Printf("%s:\n", label)
	if f.Hint != "" {
		fmt.Printf("  %s\n", f.Hint)
	}

	def := defaultString(f)
	if def != "" {
		fmt.Printf("> [%s] ", def)
	} else {
		fmt.Print("> ")
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)

	if line == "" {
		if def != "" {
			line = def
		} else if f.Required {
			return nil, fmt.Errorf("%s is required", f.Name)
		} else {
			return nil, nil
		}
	}

	if f.Validate != "" {
		re, err := regexp.Compile(f.Validate)
		if err != nil {
			return nil, fmt.Errorf("invalid validation pattern for %s: %w", f.Name, err)
		}
		if !re.MatchString(line) {
			return nil, fmt.Errorf("invalid value for %s (must match %s)", f.Name, f.Validate)
		}
	}

	fmt.Println()
	return line, nil
}

func promptPassword(f Field) (any, error) {
	label := f.Label
	if f.Required {
		label += " (required)"
	}
	fmt.Printf("%s:\n", label)
	if f.Hint != "" {
		fmt.Printf("  %s\n", f.Hint)
	}
	fmt.Print("> ")

	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		return nil, err
	}

	line := strings.TrimSpace(string(pwd))
	if line == "" && f.Required {
		return nil, fmt.Errorf("%s is required", f.Name)
	}
	if line == "" {
		return nil, nil
	}

	fmt.Println()
	return line, nil
}

func promptNumber(reader *bufio.Reader, f Field) (any, error) {
	label := f.Label
	if f.Required {
		label += " (required)"
	}
	if f.Min != 0 || f.Max != 0 {
		label += fmt.Sprintf(" [%.4g–%.4g]", f.Min, f.Max)
	}
	fmt.Printf("%s:\n", label)
	if f.Hint != "" {
		fmt.Printf("  %s\n", f.Hint)
	}

	def := defaultString(f)
	if def != "" {
		fmt.Printf("> [%s] ", def)
	} else {
		fmt.Print("> ")
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)

	if line == "" {
		if def != "" {
			line = def
		} else if f.Required {
			return nil, fmt.Errorf("%s is required", f.Name)
		} else {
			return nil, nil
		}
	}

	num, err := strconv.ParseFloat(line, 64)
	if err != nil {
		return nil, fmt.Errorf("%s: not a number", f.Name)
	}

	if f.Min != 0 || f.Max != 0 {
		if num < f.Min || num > f.Max {
			return nil, fmt.Errorf("%s: must be between %.4g and %.4g", f.Name, f.Min, f.Max)
		}
	}

	fmt.Println()
	return num, nil
}

func promptConfirm(reader *bufio.Reader, f Field) (any, error) {
	def := "n"
	if d, ok := f.Default.(bool); ok && d {
		def = "y"
	}
	if d, ok := f.Current.(bool); ok {
		if d {
			def = "y"
		} else {
			def = "n"
		}
	}

	fmt.Printf("%s [y/n] (%s): ", f.Label, def)

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(strings.ToLower(line))

	if line == "" {
		line = def
	}

	fmt.Println()
	return line == "y" || line == "yes", nil
}

func promptSelect(reader *bufio.Reader, f Field) (any, error) {
	label := f.Label
	if f.Required {
		label += " (required)"
	}
	fmt.Printf("%s:\n", label)
	if f.Hint != "" {
		fmt.Printf("  %s\n", f.Hint)
	}

	options := f.Options
	isSound := f.Type == "sound"
	for i, opt := range options {
		marker := "  "
		if currentString(f) == opt {
			marker = "* "
		}
		fmt.Printf("%s%d) %s\n", marker, i+1, opt)
	}
	if isSound {
		fmt.Printf("  %d) Custom path...\n", len(options)+1)
	}

	def := defaultString(f)
	if def != "" {
		fmt.Printf("> [%s] ", def)
	} else {
		fmt.Print("> ")
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)

	if line == "" {
		if def != "" {
			return def, nil
		}
		if f.Required {
			return nil, fmt.Errorf("%s is required", f.Name)
		}
		return nil, nil
	}

	idx, err := strconv.Atoi(line)
	if err != nil {
		// Allow typing the option value directly
		fmt.Println()
		return line, nil
	}

	if isSound && idx == len(options)+1 {
		fmt.Print("Path: ")
		path, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		fmt.Println()
		return strings.TrimSpace(path), nil
	}

	if idx < 1 || idx > len(options) {
		return nil, fmt.Errorf("%s: invalid choice %d", f.Name, idx)
	}

	fmt.Println()
	return options[idx-1], nil
}

func defaultString(f Field) string {
	if f.Current != nil {
		return fmt.Sprintf("%v", f.Current)
	}
	if f.Default != nil {
		return fmt.Sprintf("%v", f.Default)
	}
	return ""
}

func currentString(f Field) string {
	if f.Current != nil {
		return fmt.Sprintf("%v", f.Current)
	}
	return ""
}
