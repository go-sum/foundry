package version

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadDotVersion reads a single KEY=VALUE entry from the .versions file.
func ReadDotVersion(repoRoot, key string) (string, error) {
	path := filepath.Join(repoRoot, ".versions")
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("key %s not found in %s", key, path)
}

// WriteDotVersion updates a single KEY=VALUE entry in the .versions file,
// preserving all other entries.
func WriteDotVersion(repoRoot, key, value string) error {
	path := filepath.Join(repoRoot, ".versions")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		k, _, ok := strings.Cut(trimmed, "=")
		if ok && strings.TrimSpace(k) == key {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("key %s not found in %s", key, path)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}
