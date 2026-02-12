package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pink-tools/pink-core/log"
)

// Config for Run()
type Config struct {
	Name       string
	Version    string
	Usage      string             // optional, auto-generated if empty
	Commands   map[string]Command // subcommands
	IPCHandler func(cmd string) string // custom IPC commands handler
}

// Command is a CLI subcommand
type Command struct {
	Desc string
	Run  func(args []string) error
}

// Run is the main entry point for all pink-tools services
//
// Automatically handles:
//   - CLI: --version, --help, --health, custom commands
//   - pink-otel initialization
//   - IPC listener for graceful shutdown (if main != nil)
//   - Signal handling (SIGINT, SIGTERM)
//   - Context cancellation on shutdown
func Run(cfg Config, main func(ctx context.Context) error) {
	log.Init(cfg.Name, cfg.Version)

	// Handle CLI
	if len(os.Args) > 1 {
		handled := handleCLI(cfg)
		if handled {
			return
		}
	}

	// No main function = CLI-only tool
	if main == nil {
		if len(os.Args) == 1 {
			printUsage(cfg)
		}
		return
	}

	// Singleton check - exit if another instance is running
	if IsRunning(cfg.Name) {
		fmt.Fprintf(os.Stderr, "%s is already running\n", cfg.Name)
		os.Exit(1)
	}

	// Daemon mode
	ctx, cancel := context.WithCancel(context.Background())

	// Start IPC listener for graceful shutdown
	ipcCleanup, err := startIPCListener(cfg.Name, cancel, cfg.IPCHandler)
	if err != nil {
		log.Error(ctx, "failed to start IPC listener", log.Attr{"error", err.Error()})
		os.Exit(1)
	}
	defer ipcCleanup()

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info(ctx, "received shutdown signal")
		cancel()
		<-sigCh
		log.Error(ctx, "forced shutdown")
		os.Exit(1)
	}()

	// Run main
	log.Info(ctx, "started "+cfg.Version)
	if err := main(ctx); err != nil {
		log.Error(ctx, "main exited with error", log.Attr{"error", err.Error()})
		os.Exit(1)
	}
	log.Info(ctx, "shutdown complete")
}

func handleCLI(cfg Config) bool {
	arg := os.Args[1]

	switch arg {
	case "--version", "-V":
		fmt.Printf("%s v%s\n", cfg.Name, cfg.Version)
		return true

	case "--help", "-h", "help":
		printUsage(cfg)
		return true

	case "--health":
		if IsRunning(cfg.Name) {
			fmt.Println("OK")
		} else {
			fmt.Println("NOT RUNNING")
			os.Exit(1)
		}
		return true
	}

	// Check custom commands
	if cmd, ok := cfg.Commands[arg]; ok {
		if err := cmd.Run(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return true
	}

	return false
}

func printUsage(cfg Config) {
	if cfg.Usage != "" {
		fmt.Println(cfg.Usage)
		return
	}

	// Auto-generate usage
	fmt.Printf("Usage: %s [command]\n\n", cfg.Name)
	fmt.Println("Commands:")
	fmt.Println("  --version, -V    Show version")
	fmt.Println("  --help, -h       Show this help")
	fmt.Println("  --health         Check if running")

	if len(cfg.Commands) > 0 {
		fmt.Println()
		for name, cmd := range cfg.Commands {
			if cmd.Desc != "" {
				fmt.Printf("  %-16s %s\n", name, cmd.Desc)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
	}
}
