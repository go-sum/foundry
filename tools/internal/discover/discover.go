package discover

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrPackageNotFound is returned when a named package cannot be found.
var ErrPackageNotFound = errors.New("package not found")

// Package represents a discovered Go module under pkg/.
type Package struct {
	Name       string // "web", "web/render", "assets"
	Dir        string // absolute path to the module directory
	Module     string // module path: "github.com/go-sum/web"
	Prefix     string // subtree prefix relative to repo root: "pkg/web"
	MirrorRepo string // GitHub repo name: "web"
	Nested     bool   // true for sub-modules like web/render
	Parent     string // parent package name ("web" for web/render), empty if top-level
}

// DiscoverPackages walks pkg/ for go.mod files and returns all discovered packages.
func DiscoverPackages(repoRoot string) ([]Package, error) {
	pkgDir := filepath.Join(repoRoot, "pkg")

	var pkgs []Package
	err := filepath.WalkDir(pkgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}

		dir := filepath.Dir(path)

		// Compute the path relative to pkg/.
		rel, err := filepath.Rel(pkgDir, dir)
		if err != nil {
			return fmt.Errorf("rel path for %s: %w", dir, err)
		}

		// Normalize to forward slashes for the name.
		name := filepath.ToSlash(rel)

		mod, err := parseModulePath(path)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		// Determine if this is a nested module (more than one path segment).
		segments := strings.Split(name, "/")
		nested := len(segments) > 1

		mirrorRepo := segments[0]
		parent := ""
		if nested {
			parent = segments[0]
		}

		pkgs = append(pkgs, Package{
			Name:       name,
			Dir:        dir,
			Module:     mod,
			Prefix:     filepath.ToSlash(filepath.Join("pkg", name)),
			MirrorRepo: mirrorRepo,
			Nested:     nested,
			Parent:     parent,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk pkg/: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found under %s/pkg/", repoRoot)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].Name < pkgs[j].Name
	})

	return pkgs, nil
}

// DiscoverPackage returns a single package by name, or ErrPackageNotFound.
func DiscoverPackage(repoRoot, name string) (Package, error) {
	pkgs, err := DiscoverPackages(repoRoot)
	if err != nil {
		return Package{}, err
	}

	for _, p := range pkgs {
		if p.Name == name {
			return p, nil
		}
	}

	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = p.Name
	}
	return Package{}, fmt.Errorf("%w: %q (available: %s)", ErrPackageNotFound, name, strings.Join(names, ", "))
}

// ResolveTopLevel returns all top-level (non-nested) packages.
func ResolveTopLevel(repoRoot string) ([]Package, error) {
	pkgs, err := DiscoverPackages(repoRoot)
	if err != nil {
		return nil, err
	}

	var top []Package
	for _, p := range pkgs {
		if !p.Nested {
			top = append(top, p)
		}
	}
	return top, nil
}

// ResolvePackages resolves "all" to all top-level packages, or a single name to one package.
func ResolvePackages(repoRoot, nameOrAll string) ([]Package, error) {
	if nameOrAll == "all" {
		return ResolveTopLevel(repoRoot)
	}
	p, err := DiscoverPackage(repoRoot, nameOrAll)
	if err != nil {
		return nil, err
	}
	return []Package{p}, nil
}

// parseModulePath reads the module directive from a go.mod file.
func parseModulePath(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no module directive found in %s", goModPath)
}
