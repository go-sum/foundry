package version

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadGoModVersion reads the version of modulePath from go.mod's require block.
func ReadGoModVersion(goModPath, modulePath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	inRequire := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if !inRequire {
			continue
		}

		// Skip replace lines.
		if strings.Contains(line, "=>") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[0] == modulePath {
			return parts[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("module %s not found in %s", modulePath, goModPath)
}

// WriteGoModVersion updates the version of modulePath in go.mod's require block.
func WriteGoModVersion(goModPath, modulePath, newVersion string) error {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "=>") {
			continue
		}
		parts := strings.Fields(trimmed)
		if len(parts) >= 2 && parts[0] == modulePath {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + modulePath + " " + newVersion
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("module %s not found in %s", modulePath, goModPath)
	}

	return os.WriteFile(goModPath, []byte(strings.Join(lines, "\n")), 0644)
}

// CopyModStripReplace copies src go.mod to dst, removing all replace directives.
func CopyModStripReplace(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	var out []string
	inReplace := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "replace (" {
			inReplace = true
			continue
		}
		if inReplace {
			if trimmed == ")" {
				inReplace = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "replace ") {
			continue
		}

		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(dst, []byte(strings.Join(out, "\n")+"\n"), 0644)
}
