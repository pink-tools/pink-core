package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

// LoadEnv loads .env file from service data directory
func LoadEnv(name string) {
	envPath := filepath.Join(ServiceDir(name), ".env")
	godotenv.Load(envPath)
}

// BaseDir returns parent of user's home directory.
// macOS: /Users, Windows: C:\Users, Linux: /home
func BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Dir(home)
}

// PinkToolsDir returns the pink-tools directory: /Users/pink-tools/
func PinkToolsDir() string {
	return filepath.Join(BaseDir(), "pink-tools")
}

// ServiceDir returns directory for a service: /Users/pink-tools/{name}/
// Creates the directory if it doesn't exist.
func ServiceDir(name string) string {
	dir := filepath.Join(PinkToolsDir(), name)
	os.MkdirAll(dir, 0755)
	return dir
}

// BinaryPath returns full path to a service binary: /Users/pink-tools/{name}/{name}
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
