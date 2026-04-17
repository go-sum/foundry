package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Env string

const (
	Production  Env = "production"
	Development Env = "development"
	Testing     Env = "testing"
)

var currentEnvMap = map[Env]Env{
	Development: Development,
	Testing:     Testing,
	Production:  Production,
}

func CurrentEnv() Env {
	if env, ok := currentEnvMap[Env(os.Getenv("APP_ENV"))]; ok {
		return env
	}
	return Production
}

var envPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)(?::-([^}]*))?\}`)

func ExpandEnv(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := envPattern.FindStringSubmatch(match)
		if v := os.Getenv(parts[1]); v != "" {
			return v
		}
		return parts[2]
	})
}

var secretsDir = "/run/secrets"

func ExpandSecret(name string) string {
	if b, err := os.ReadFile(filepath.Join(secretsDir, name)); err == nil {
		return strings.TrimRight(string(b), "\n\r\t ")
	}
	return os.Getenv(name)
}
