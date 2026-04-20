package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// StateFile holds a persisted map of build-key → input hash used for
// change detection. It is JSON on disk: {"hashes":{"css:/path":  "abc…"}}.
type StateFile struct {
	path   string
	Hashes map[string]string `json:"hashes"`
}

// LoadState loads the state file at path. If the file is missing or cannot be
// parsed it returns an empty StateFile so that the next HasChanged call
// triggers a full rebuild.
func LoadState(path string) *StateFile {
	sf := &StateFile{path: path, Hashes: make(map[string]string)}
	raw, err := os.ReadFile(path)
	if err != nil {
		return sf
	}
	if err := json.Unmarshal(raw, sf); err != nil {
		return sf
	}
	if sf.Hashes == nil {
		sf.Hashes = make(map[string]string)
	}
	return sf
}

// HasChanged reports whether the files covered by key have changed since the
// last MarkBuilt call. It returns true when the key is absent, any file is
// unreadable, or the SHA-256 of the sorted concatenated file contents differs
// from the stored hash.
func (s *StateFile) HasChanged(key string, files []string) (bool, error) {
	stored, ok := s.Hashes[key]
	if !ok {
		return true, nil
	}
	hash, err := hashFiles(files)
	if err != nil {
		return true, err
	}
	return hash != stored, nil
}

// MarkBuilt computes the hash for files, stores it under key, and persists the
// state file atomically (write to path+".tmp" then os.Rename).
func (s *StateFile) MarkBuilt(key string, files []string) error {
	hash, err := hashFiles(files)
	if err != nil {
		return fmt.Errorf("hash files: %w", err)
	}
	s.Hashes[key] = hash

	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.MkdirAll(dirOf(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write tmp state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

// ExpandGlobs expands each pattern (which may contain ** globs) using
// doublestar.FilepathGlob and returns a sorted, deduplicated list of absolute
// file paths. Patterns that are already absolute are expanded as-is; relative
// patterns are resolved relative to baseDir.
func ExpandGlobs(baseDir string, patterns []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, pattern := range patterns {
		if !isAbsolutePattern(pattern) {
			pattern = joinPath(baseDir, pattern)
		}
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", pattern, err)
		}
		for _, m := range matches {
			seen[m] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	sort.Strings(result)
	return result, nil
}

// hashFiles computes a single SHA-256 over the sorted concatenated contents of
// all files. The sort is applied inside the function so callers need not sort.
func hashFiles(files []string) (string, error) {
	sorted := make([]string, len(files))
	copy(sorted, files)
	sort.Strings(sorted)

	h := sha256.New()
	for _, f := range sorted {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", f, err)
		}
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func isAbsolutePattern(pattern string) bool {
	return strings.HasPrefix(pattern, "/")
}

func joinPath(base, rel string) string {
	return base + "/" + rel
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
