package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ExpandEnv returns the value of the named environment variable, trimmed of
// leading and trailing whitespace. If the variable is not set (absent from the
// environment), defaultVal is returned. A variable that is explicitly set to an
// empty string returns "" — it does not fall back to defaultVal.
func ExpandEnv(name, defaultVal string) string {
	if v, ok := os.LookupEnv(name); ok {
		return strings.TrimSpace(v)
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

// ExpandEnvBool parses the named environment variable as a boolean. If the
// variable is absent, defaultVal is returned. An unparseable value returns an
// error.
func ExpandEnvBool(name string, defaultVal bool) (bool, error) {
	raw := ExpandEnv(name, strconv.FormatBool(defaultVal))
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("parse bool %q: %w", raw, err)
	}
	return v, nil
}

// ExpandEnvCSV splits the named environment variable on commas, trims each
// element, and discards empty elements. Returns nil if the variable is absent
// or empty.
func ExpandEnvCSV(name string) []string {
	raw := ExpandEnv(name, "")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
