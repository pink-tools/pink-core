package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

func fatal(msg string, err error) {
	fmt.Fprintf(os.Stderr, "fatal: %s: %v\n", msg, err)
	os.Exit(1)
}

// LoadEnv loads .env file from service data directory.
// Does not override existing env vars (safe for initial load).
func LoadEnv(name string) {
	envPath := filepath.Join(ServiceDir(name), ".env")
	godotenv.Load(envPath)
}

// ReloadEnv re-reads .env file, overriding existing env vars.
func ReloadEnv(name string) {
	envPath := filepath.Join(ServiceDir(name), ".env")
	godotenv.Overload(envPath)
}

// HomeDir returns the current user's home directory.
func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// PinkToolsDir returns ~/pink-tools/
func PinkToolsDir() string {
	return filepath.Join(HomeDir(), "pink-tools")
}

// ServiceDir returns ~/pink-tools/{name}/
// Creates the directory if it doesn't exist.
func ServiceDir(name string) string {
	dir := filepath.Join(PinkToolsDir(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fatal("create service dir "+dir, err)
	}
	return dir
}

// BinaryPath returns full path to a service binary: ~/pink-tools/{name}/{name}
// Appends .exe on Windows.
func BinaryPath(name string) string {
	bin := name
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	return filepath.Join(ServiceDir(name), bin)
}

// DataDir is an alias for ServiceDir (backwards compatibility)
func DataDir(name string) string {
	return ServiceDir(name)
}

// AppDataDir returns platform-standard persistent data directory.
// macOS: ~/Library/Application Support/pink-tools/{name}/
// Linux: ~/.config/pink-tools/{name}/
// Windows: %AppData%/pink-tools/{name}/
func AppDataDir(name string) string {
	base, _ := os.UserConfigDir()
	dir := filepath.Join(base, "pink-tools", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fatal("create app data dir "+dir, err)
	}
	return dir
}

// SaveEnv merges values into a service's .env file.
// Creates the file if it doesn't exist.
func SaveEnv(name string, values map[string]string) error {
	envPath := filepath.Join(ServiceDir(name), ".env")

	existing, err := godotenv.Read(envPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", envPath, err)
	}
	if existing == nil {
		existing = make(map[string]string)
	}

	for k, v := range values {
		existing[k] = v
	}

	if err := godotenv.Write(existing, envPath); err != nil {
		return fmt.Errorf("write %s: %w", envPath, err)
	}
	return nil
}
