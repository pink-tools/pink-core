# pink-core

Core framework for pink-tools services.

**Repository:** https://github.com/pink-tools/pink-core

## What It Does

Provides unified CLI parsing, IPC-based graceful shutdown, and signal handling for all pink-tools services. Solves Windows graceful shutdown problem where SIGINT/SIGTERM don't work for detached processes.

## File Structure

```
pink-core/
├── core.go       # Run(), Config, CLI handling
├── ipc.go        # IPC listener and client (TCP-based)
├── env.go        # Environment loading, DataDir
├── go.mod
├── go.sum
├── README.md
└── ai-docs/
    └── CLAUDE.md
```

## Public API

```go
// Main entry point
func Run(cfg Config, main func(ctx context.Context) error)

// Configuration
type Config struct {
    Name     string
    Version  string
    Usage    string             // optional, auto-generated if empty
    Commands map[string]Command
}

type Command struct {
    Desc string
    Run  func(args []string) error
}

// Helpers
func DataDir(name string) string    // ~/pink-tools/{name}/
func LoadEnv(name string)           // load .env from DataDir
func SendStop(name string) error    // send IPC STOP command
func IsRunning(name string) bool    // check via IPC PING
```

## Run() Flow

For daemons (main != nil):
1. Initialize pink-otel logging
2. Parse CLI args (--version, --help, --health, custom commands)
3. If CLI handled → return
4. Start IPC listener on random port
5. Write port to `~/pink-tools/{name}/{name}.port`
6. Set up signal handling (SIGINT, SIGTERM)
7. Call main(ctx)
8. On signal or IPC STOP → cancel context
9. Cleanup IPC listener, remove port file

For CLI tools (main == nil):
1. Initialize pink-otel logging
2. Parse CLI args
3. Run matched command or show usage

## IPC Protocol

TCP on localhost, random port.

**Port file:** `~/pink-tools/{service}/{service}.port`

**Commands:**
- `PING\n` → `PONG\n` (health check)
- `STOP\n` → `OK\n` (graceful shutdown, cancels context)

**Timeout:** 5 seconds for connections and responses.

## Built-in CLI Flags

All services get these automatically:
- `--version`, `-V` — print `{name} v{version}`
- `--help`, `-h`, `help` — print usage
- `--health` — check if daemon running via IPC PING

## Data Directory

`~/pink-tools/{service}/` contains:
- `.env` — environment configuration (loaded by LoadEnv)
- `{service}.port` — IPC port file (runtime, deleted on exit)

## Dependencies

- `github.com/pink-tools/pink-otel` — structured logging
- `github.com/joho/godotenv` — .env file loading

## Usage Pattern

### Daemon Service

```go
func main() {
    core.LoadEnv("my-daemon")

    core.Run(core.Config{
        Name:    "my-daemon",
        Version: version,
        Commands: map[string]core.Command{
            "stop":   {Desc: "Stop daemon", Run: stopCmd},
            "status": {Desc: "Check status", Run: statusCmd},
        },
    }, func(ctx context.Context) error {
        // Stop existing instance
        if core.IsRunning("my-daemon") {
            core.SendStop("my-daemon")
        }

        // Start your daemon logic
        daemon := NewDaemon()
        daemon.Start()

        <-ctx.Done() // Wait for shutdown signal
        daemon.Stop()
        return nil
    })
}

func stopCmd(args []string) error {
    if !core.IsRunning("my-daemon") {
        fmt.Println("not running")
        return nil
    }
    return core.SendStop("my-daemon")
}

func statusCmd(args []string) error {
    if core.IsRunning("my-daemon") {
        fmt.Println("running")
    } else {
        fmt.Println("stopped")
    }
    return nil
}
```

### CLI Tool

```go
func main() {
    core.LoadEnv("my-tool")

    core.Run(core.Config{
        Name:    "my-tool",
        Version: version,
        Commands: map[string]core.Command{
            "process": {Desc: "Process files", Run: processCmd},
        },
    }, nil) // nil main = CLI only, no IPC listener
}
```

## Used By

- pink-agent — Telegram bot + PTY for Claude Code
- pink-transcriber — Speech-to-text daemon (whisper.cpp)
- pink-voice — Voice input daemon with hotkey
- pink-elevenlabs — TTS CLI tool

## pink-orchestrator Integration

pink-orchestrator uses IPC to gracefully stop services:
1. Reads `{service}.port` file
2. Sends `STOP\n` via TCP
3. Waits for process exit (5s timeout)
4. Falls back to force kill if timeout
