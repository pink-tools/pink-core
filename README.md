# pink-core

Core framework for [pink-tools](https://github.com/pink-tools) services. Provides unified CLI, IPC-based graceful shutdown, and signal handling.

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
    "github.com/pink-tools/pink-core"
)

var version = "dev"

func main() {
    core.LoadEnv("my-service")

    core.Run(core.Config{
        Name:    "my-service",
        Version: version,
        Commands: map[string]core.Command{
            "stop":   {Desc: "Stop daemon", Run: stopCmd},
            "status": {Desc: "Check status", Run: statusCmd},
        },
    }, func(ctx context.Context) error {
        // Main daemon logic
        // ctx.Done() fires on: SIGINT, SIGTERM, or IPC STOP
        <-ctx.Done()
        return nil
    })
}

func stopCmd(args []string) error {
    return core.SendStop("my-service")
}

func statusCmd(args []string) error {
    if core.IsRunning("my-service") {
        fmt.Println("running")
    } else {
        fmt.Println("stopped")
    }
    return nil
}
```

### CLI Tool

```go
package main

import (
    "github.com/pink-tools/pink-core"
)

var version = "dev"

func main() {
    core.LoadEnv("my-tool")

    core.Run(core.Config{
        Name:    "my-tool",
        Version: version,
        Commands: map[string]core.Command{
            "process": {Desc: "Process files", Run: processCmd},
        },
    }, nil) // nil = CLI-only, no daemon
}

func processCmd(args []string) error {
    // ...
    return nil
}
```

## Features

### Automatic CLI Handling

Built-in flags for all services:
- `--version`, `-V` — show version
- `--help`, `-h` — show usage
- `--health` — check if daemon is running (via IPC)

### IPC Graceful Shutdown

Cross-platform shutdown mechanism, especially important for Windows where SIGINT/SIGTERM don't work for detached processes.

```
Startup:
1. Daemon listens on random TCP port (localhost)
2. Writes port to ~/pink-tools/{service}/{service}.port
3. Waits for "STOP" command or OS signal

Shutdown:
1. Client reads port file
2. Sends "STOP\n" via TCP
3. Daemon cancels context, runs cleanup, exits
```

### Signal Handling

Unix: SIGINT, SIGTERM → context cancellation → graceful shutdown

### Environment Loading

```go
core.LoadEnv("my-service")
// Loads ~/pink-tools/my-service/.env
```

## API

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

## Data Directory

All service data stored in `~/pink-tools/{service}/`:
- `.env` — environment configuration
- `{service}.port` — IPC port (created at runtime)

## Used By

- [pink-agent](https://github.com/pink-tools/pink-agent) — Telegram bot for Claude Code
- [pink-transcriber](https://github.com/pink-tools/pink-transcriber) — Speech-to-text daemon
- [pink-voice](https://github.com/pink-tools/pink-voice) — Voice input with hotkey
- [pink-elevenlabs](https://github.com/pink-tools/pink-elevenlabs) — TTS CLI tool
