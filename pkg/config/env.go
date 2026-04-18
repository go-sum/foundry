package config

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandEnv(name, defaultVal string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return defaultVal
}

var secretsDir = "/run/secrets"

func ExpandSecret(name string) string {
	if b, err := os.ReadFile(filepath.Join(secretsDir, name)); err == nil {
		return strings.TrimRight(string(b), "\n\r\t ")
	}
	return os.Getenv(name)
}
