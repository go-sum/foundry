package build

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// CSSWatchPaths parses @source and @import directives from a Tailwind v4 CSS
// input file and returns the set of glob patterns (resolved to absolute paths)
// that should be monitored for changes. The input file itself is always
// included. Returned patterns may contain ** globs.
func CSSWatchPaths(inputPath string) ([]string, error) {
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, err
	}
	base := filepath.Dir(absInput)

	f, err := os.Open(absInput)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	seen := make(map[string]struct{})
	seen[absInput] = struct{}{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if p, ok := parseSourceDirective(line); ok {
			resolved := resolvePattern(base, p)
			seen[resolved] = struct{}{}
			continue
		}

		if p, ok := parseImportDirective(line); ok {
			resolved := resolvePattern(base, p)
			seen[resolved] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	return result, nil
}

// parseSourceDirective extracts the quoted path from a @source "..." line.
// Returns ("", false) if the line is not a @source directive.
func parseSourceDirective(line string) (string, bool) {
	if !strings.HasPrefix(line, "@source") {
		return "", false
	}
	rest := strings.TrimPrefix(line, "@source")
	rest = strings.TrimSpace(rest)
	rest = strings.TrimSuffix(rest, ";")
	p := extractQuoted(rest)
	if p == "" {
		return "", false
	}
	return p, true
}

// parseImportDirective extracts the quoted path from a @import "..." line.
// Returns ("", false) for URL imports (http://, https://, url(...)) and for
// lines that are not @import directives.
func parseImportDirective(line string) (string, bool) {
	if !strings.HasPrefix(line, "@import") {
		return "", false
	}
	rest := strings.TrimPrefix(line, "@import")
	rest = strings.TrimSpace(rest)
	rest = strings.TrimSuffix(rest, ";")

	// skip url() imports
	if strings.HasPrefix(rest, "url(") {
		return "", false
	}

	p := extractQuoted(rest)
	if p == "" {
		return "", false
	}

	// skip remote URLs embedded in quotes
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return "", false
	}
	return p, true
}

// extractQuoted returns the content between the first pair of " or ' characters.
func extractQuoted(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return ""
	}
	q := s[0]
	if q != '"' && q != '\'' {
		return ""
	}
	end := strings.IndexByte(s[1:], q)
	if end < 0 {
		return ""
	}
	return s[1 : end+1]
}

// resolvePattern resolves a (possibly glob) path relative to base into an
// absolute pattern. If p is already absolute it is returned unchanged.
func resolvePattern(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
