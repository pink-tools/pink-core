# pink-core

Core framework for [pink-tools](https://github.com/pink-tools) services. Provides unified CLI, IPC-based graceful shutdown, signal handling, environment management, and interactive setup forms.

## Install

```bash
go get github.com/pink-tools/pink-core
```

## Usage

### Daemon Service

```go
package main

import (
    "context"
    "fmt"
    "github.com/pink-tools/pink-core"
)

var version = "dev"

func main() {
    core.LoadEnv("my-service")

    core.Run(core.Config{
        Name:    "my-service",
        Version: version,
        Commands: map[string]core.Command{
            "stop":   {Desc: "Stop daemon", Run: func(args []string) error { return core.SendStop("my-service") }},
            "status": {Desc: "Check status", Run: func(args []string) error {
                if core.IsRunning("my-service") { fmt.Println("running") } else { fmt.Println("stopped") }
                return nil
            }},
        },
    }, func(ctx context.Context) error {
        // ctx.Done() fires on: SIGINT, SIGTERM, or IPC STOP
        <-ctx.Done()
        return nil
    })
}
```

### CLI Tool

```go
core.Run(core.Config{
    Name:    "my-tool",
    Version: version,
    Commands: map[string]core.Command{
        "process": {Desc: "Process files", Run: processCmd},
    },
}, nil) // nil = CLI-only, no daemon
```

## Features

### Automatic CLI Flags

All services get these for free:
- `--version`, `-V` — show version
- `--help`, `-h` — show usage
- `--health` — check if daemon is running (via IPC)
- `--claude` — print embedded agent context (for AI integration)

### IPC Graceful Shutdown

Cross-platform shutdown via TCP, works on Windows where signals don't reach detached processes.

```
Startup:  daemon listens on random TCP port, writes to ~/pink-tools/{service}/{service}.port
Shutdown: client reads port, sends "STOP\n", daemon cancels context and exits
Custom:   IPCHandler func in Config for service-specific IPC commands
```

### Environment Management

```go
core.LoadEnv("my-service")                        // Load ~/pink-tools/my-service/.env
core.ReloadEnv("my-service")                      // Reload .env (overrides existing)
core.SaveEnv("my-service", map[string]string{...}) // Merge into .env file
```

### Interactive Setup (Action System)

Services can expose interactive forms for configuration:
- TTY mode: prompts user in terminal
- `--describe`: outputs JSON form spec (for GUI rendering)
- `--config <json>`: programmatic input
- Field types: text, password, number, confirm, select, file, hotkey, url, range, sound

### Service Control

```go
core.IsRunning("my-service")          // Check via IPC PING
core.SendStop("my-service")           // Graceful shutdown
core.SendCommand("my-service", "cmd") // Custom IPC command
```

### Path Helpers

```go
core.HomeDir()                // User home directory
core.PinkToolsDir()          // ~/pink-tools/
core.ServiceDir("my-service") // ~/pink-tools/my-service/ (creates if needed)
core.BinaryPath("my-service") // ~/pink-tools/my-service/my-service
core.AppDataDir("my-service") // Platform-standard config dir
```

## Data Directory

All service data stored in `~/pink-tools/{service}/`:
- `.env` — environment configuration
- `{service}.port` — IPC port (created at runtime)

## Used By

- [pink-agent](https://github.com/pink-tools/pink-agent) — Telegram bot for Claude Code
- [pink-orchestrator](https://github.com/pink-tools/pink-orchestrator) — Service manager
- [pink-transcriber](https://github.com/pink-tools/pink-transcriber) — Speech-to-text CLI
- [pink-voice](https://github.com/pink-tools/pink-voice) — Voice input daemon
- [pink-elevenlabs](https://github.com/pink-tools/pink-elevenlabs) — TTS CLI
- [pink-whisper](https://github.com/pink-tools/pink-whisper) — Whisper.cpp daemon
