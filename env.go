package core

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv loads .env file from service data directory
func LoadEnv(name string) {
	envPath := filepath.Join(DataDir(name), ".env")
	godotenv.Load(envPath)
}

// DataDir returns the data directory for a service: ~/pink-tools/{name}/
func DataDir(name string) string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "pink-tools", name)
	os.MkdirAll(dir, 0755)
	return dir
}
