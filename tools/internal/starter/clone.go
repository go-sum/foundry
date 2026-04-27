package starter

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneOptions configures a RunClone call.
type CloneOptions struct {
	Source string // foundry repository root (auto-detected if empty)
	Target string // destination directory path
	Module string // new Go module path, e.g. "github.com/myorg/myapp"
}

// RunClone performs the full clone operation and writes a summary to w.
func RunClone(opts CloneOptions, w io.Writer) error {
	source := opts.Source
	if source == "" {
		var err error
		source, err = findSourceRoot()
		if err != nil {
			return err
		}
	}

	target := filepath.Clean(opts.Target)

	if entries, err := os.ReadDir(target); err == nil && len(entries) > 0 {
		return fmt.Errorf("clone: target directory %q already exists and is non-empty; remove it first", target)
	}

	manifestPath := filepath.Join(source, "tools", "starter", "manifest.yaml")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	filesCopied, err := copyFiles(source, target, manifest)
	if err != nil {
		return err
	}

	filesRenamed, err := applyRenames(target, manifest.Rename)
	if err != nil {
		return err
	}

	filesStripped, err := stripMonorepoBlocks(target)
	if err != nil {
		return err
	}

	filesRewritten, err := rewriteModule(target, manifest.ModuleRewrite.From, opts.Module)
	if err != nil {
		return err
	}

	airTransformed, err := transformAirToml(target)
	if err != nil {
		return err
	}

	taskfileTransformed, err := transformRootTaskfile(target)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Clone complete.\n")
	fmt.Fprintf(w, "  Files copied:    %d\n", filesCopied)
	fmt.Fprintf(w, "  Files renamed:   %d\n", filesRenamed)
	fmt.Fprintf(w, "  Blocks stripped: %d\n", filesStripped)
	fmt.Fprintf(w, "  Files rewritten: %d\n", filesRewritten)
	fmt.Fprintf(w, "  Air transformed: %d\n", airTransformed)
	fmt.Fprintf(w, "  Task transformed: %d\n", taskfileTransformed)
	fmt.Fprintf(w, "\nNext steps:\n")
	fmt.Fprintf(w, "  cd %s\n", target)
	fmt.Fprintf(w, "  go mod tidy\n")
	fmt.Fprintf(w, "  task db:migrate\n")
	fmt.Fprintf(w, "  task dev\n")
	return nil
}

// copyFiles enumerates source files via git ls-files and copies those not
// excluded by the manifest to target. Returns the number of files copied.
func copyFiles(source, target string, manifest Manifest) (int, error) {
	files, err := gitFiles(source)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, rel := range files {
		if IsExcluded(manifest, rel) {
			continue
		}
		if err := copyFile(filepath.Join(source, rel), filepath.Join(target, rel)); err != nil {
			return count, fmt.Errorf("copy %s: %w", rel, err)
		}
		count++
	}
	return count, nil
}

// gitFiles returns the list of file paths tracked by git (relative to dir),
// including untracked files not covered by .gitignore.
func gitFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "ls-files",
		"--cached",
		"--others",
		"--exclude-standard",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files in %s: %w", dir, err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		rel := filepath.ToSlash(line)
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			continue
		}
		files = append(files, rel)
	}
	return files, nil
}

// copyFile copies src to dst, creating all necessary parent directories.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close() //nolint:errcheck

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close() //nolint:errcheck

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// applyRenames renames files in the target directory per the rename rules.
func applyRenames(target string, rules []RenameRule) (int, error) {
	count := 0
	for _, rule := range rules {
		src := filepath.Join(target, filepath.FromSlash(rule.From))
		dst := filepath.Join(target, filepath.FromSlash(rule.To))
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			return count, fmt.Errorf("rename %s -> %s: %w", rule.From, rule.To, err)
		}
		count++
	}
	return count, nil
}

// stripMonorepoBlocks removes content between <!-- monorepo-only-start --> and
// <!-- monorepo-only-end --> markers from all Markdown files in target.
func stripMonorepoBlocks(target string) (int, error) {
	const startMarker = "<!-- monorepo-only-start -->"
	const endMarker = "<!-- monorepo-only-end -->"

	count := 0
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		if !strings.Contains(content, startMarker) {
			return nil
		}

		result := content
		for {
			start := strings.Index(result, startMarker)
			if start == -1 {
				break
			}
			end := strings.Index(result[start:], endMarker)
			if end == -1 {
				break
			}
			end += start + len(endMarker)
			if end < len(result) && result[end] == '\n' {
				end++
			}
			result = result[:start] + result[end:]
		}
		if result == content {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(result), info.Mode()); err != nil {
			return fmt.Errorf("strip %s: %w", path, err)
		}
		count++
		return nil
	})
	return count, err
}

// rewriteModuleContent rewrites fromModule to toModule in content,
// preserving fromModule+"/pkg/" paths (external monorepo dependencies).
// It handles both quoted imports (Go source files) and unquoted paths (go.mod).
func rewriteModuleContent(content, fromModule, toModule string) string {
	pkgPrefix := fromModule + "/pkg/"
	placeholder := "\x00FOUNDRY_PKG\x00"
	// Protect pkg/ paths before any replacement.
	content = strings.ReplaceAll(content, pkgPrefix, placeholder)
	// Rewrite module declaration line (go.mod).
	content = strings.ReplaceAll(content, "module "+fromModule+"\n", "module "+toModule+"\n")
	// Rewrite quoted imports in .go files.
	content = strings.ReplaceAll(content, `"`+fromModule+`/`, `"`+toModule+`/`)
	// Rewrite unquoted paths in go.mod require/replace blocks.
	content = strings.ReplaceAll(content, "\t"+fromModule+"/", "\t"+toModule+"/")
	content = strings.ReplaceAll(content, " "+fromModule+"/", " "+toModule+"/")
	// Restore protected pkg/ paths.
	content = strings.ReplaceAll(content, placeholder, pkgPrefix)
	return content
}

// rewriteModule replaces fromModule with toModule in go.mod and all *.go files,
// preserving fromModule+"/pkg/" paths (external monorepo dependencies).
func rewriteModule(target, fromModule, toModule string) (int, error) {
	count := 0

	goModPath := filepath.Join(target, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		data, err := os.ReadFile(goModPath)
		if err != nil {
			return count, fmt.Errorf("rewrite go.mod: %w", err)
		}
		original := string(data)
		updated := rewriteModuleContent(original, fromModule, toModule)
		if updated != original {
			info, err := os.Stat(goModPath)
			if err != nil {
				return count, fmt.Errorf("rewrite go.mod: %w", err)
			}
			if err := os.WriteFile(goModPath, []byte(updated), info.Mode()); err != nil {
				return count, fmt.Errorf("rewrite go.mod: %w", err)
			}
			count++
		}
	}

	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("rewrite %s: %w", path, err)
		}
		original := string(data)
		updated := rewriteModuleContent(original, fromModule, toModule)
		if updated == original {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("rewrite %s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(updated), info.Mode()); err != nil {
			return fmt.Errorf("rewrite %s: %w", path, err)
		}
		count++
		return nil
	})
	return count, err
}

// rewriteFileStrings performs string replacements in a file in-place.
// Returns true if any replacement was made.
func rewriteFileStrings(path string, replacements map[string]string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	content := string(data)
	original := content
	for old, newVal := range replacements {
		content = strings.ReplaceAll(content, old, newVal)
	}

	if content == original {
		return false, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
		return false, err
	}
	return true, nil
}

// transformRootTaskfile rewrites Taskfile.yml for use in a derived app:
// replaces `go run ../pkg/assets/cli` and `go run ../pkg/docs/cli` invocations
// with pre-built binary calls, matching how the tools image provides them.
func transformRootTaskfile(target string) (int, error) {
	path := filepath.Join(target, "Taskfile.yml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil
	}

	modified, err := rewriteFileStrings(path, map[string]string{
		"go run ../pkg/assets/cli ": "assets ",
		"go run ../pkg/docs/cli ":   "docs ",
	})
	if err != nil {
		return 0, fmt.Errorf("transform Taskfile.yml: %w", err)
	}
	if modified {
		return 1, nil
	}
	return 0, nil
}

// transformAirToml rewrites docker/app/.air.toml for use in a derived app:
// replaces `go run ../pkg/assets/cli` with the `assets` binary and removes
// the monorepo-specific `../pkg/componentry` entry from include_dir.
func transformAirToml(target string) (int, error) {
	path := filepath.Join(target, "docker", "app", ".air.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	content := string(data)
	original := content

	content = strings.ReplaceAll(content, "go run ../pkg/assets/cli ", "assets ")

	lines := strings.Split(content, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, `"../pkg/`) {
			continue
		}
		kept = append(kept, line)
	}
	content = strings.Join(kept, "\n")

	if content == original {
		return 0, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
		return 0, fmt.Errorf("transform .air.toml: %w", err)
	}
	return 1, nil
}

// findSourceRoot locates the foundry repository root, checking in order:
// FOUNDRY_ROOT env var, walking up from the executable, then the cwd.
func findSourceRoot() (string, error) {
	if v := os.Getenv("FOUNDRY_ROOT"); v != "" {
		if isFoundryRoot(v) {
			return v, nil
		}
		return "", fmt.Errorf("FOUNDRY_ROOT=%q does not look like a foundry root (missing go.work or tools/starter/manifest.yaml)", v)
	}

	exe, err := os.Executable()
	if err == nil {
		if root, ok := walkUpForRoot(filepath.Dir(exe)); ok {
			return root, nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("findSourceRoot: getwd: %w", err)
	}
	if isFoundryRoot(cwd) {
		return cwd, nil
	}
	if root, ok := walkUpForRoot(cwd); ok {
		return root, nil
	}

	return "", fmt.Errorf("findSourceRoot: cannot locate foundry root; set FOUNDRY_ROOT or run from within the foundry repository")
}

func walkUpForRoot(dir string) (string, bool) {
	for {
		if isFoundryRoot(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func isFoundryRoot(dir string) bool {
	goWork := filepath.Join(dir, "go.work")
	manifest := filepath.Join(dir, "tools", "starter", "manifest.yaml")
	return fileExists(goWork) && fileExists(manifest)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
