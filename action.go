package core

// Action is a named operation exposed by a service (install, settings, etc.)
type Action struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

// FormSpec describes what an action needs from the user
type FormSpec struct {
	Title   string  `json:"title"`
	Message string  `json:"message,omitempty"`
	Fields  []Field `json:"fields"`
}

// Field is a single input in a form
type Field struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`                 // text, password, number, confirm, select, file, hotkey, url, range, sound
	Label      string   `json:"label"`
	Hint       string   `json:"hint,omitempty"`
	Required   bool     `json:"required,omitempty"`
	Default    any      `json:"default,omitempty"`
	Current    any      `json:"current,omitempty"`
	Validate   string   `json:"validate,omitempty"`   // regex or keyword
	Options    []string `json:"options,omitempty"`     // for select/sound
	Extensions []string `json:"extensions,omitempty"` // for file
	Min        float64  `json:"min,omitempty"`
	Max        float64  `json:"max,omitempty"`
	Step       float64  `json:"step,omitempty"`
}

// ActionHandler defines how an action describes itself and executes
type ActionHandler struct {
	Describe func() FormSpec
	Execute  func(values map[string]any) error
}
