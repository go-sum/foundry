package discover

import (
	"os/exec"
	"strings"
	"testing"
)

func repoRootT(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestDiscoverPackages(t *testing.T) {
	pkgs, err := DiscoverPackages(repoRootT(t))
	if err != nil {
		t.Fatalf("DiscoverPackages: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("expected at least one package")
	}

	// Build a lookup for assertions.
	byName := make(map[string]Package, len(pkgs))
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	// Top-level packages must be present.
	for _, name := range []string{"web", "assets", "componentry", "config"} {
		p, ok := byName[name]
		if !ok {
			t.Errorf("expected package %q to be discovered", name)
			continue
		}
		if p.Nested {
			t.Errorf("package %q should not be nested", name)
		}
		if p.Parent != "" {
			t.Errorf("package %q should have empty Parent, got %q", name, p.Parent)
		}
		if p.MirrorRepo != name {
			t.Errorf("package %q MirrorRepo = %q, want %q", name, p.MirrorRepo, name)
		}
		if p.Prefix != "pkg/"+name {
			t.Errorf("package %q Prefix = %q, want %q", name, p.Prefix, "pkg/"+name)
		}
	}

	// Nested package: web/render.
	nested, ok := byName["web/render"]
	if !ok {
		t.Fatal("expected nested package web/render")
	}
	if !nested.Nested {
		t.Error("web/render should be Nested=true")
	}
	if nested.Parent != "web" {
		t.Errorf("web/render Parent = %q, want %q", nested.Parent, "web")
	}
	if nested.MirrorRepo != "web" {
		t.Errorf("web/render MirrorRepo = %q, want %q", nested.MirrorRepo, "web")
	}
	if nested.Prefix != "pkg/web/render" {
		t.Errorf("web/render Prefix = %q, want %q", nested.Prefix, "pkg/web/render")
	}
}

func TestDiscoverPackage(t *testing.T) {
	p, err := DiscoverPackage(repoRootT(t), "web")
	if err != nil {
		t.Fatalf("DiscoverPackage: %v", err)
	}
	if p.Name != "web" {
		t.Errorf("Name = %q, want %q", p.Name, "web")
	}
	if p.Nested {
		t.Error("web should not be nested")
	}
}

func TestDiscoverPackage_NotFound(t *testing.T) {
	_, err := DiscoverPackage(repoRootT(t), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
}

func TestResolveTopLevel(t *testing.T) {
	pkgs, err := ResolveTopLevel(repoRootT(t))
	if err != nil {
		t.Fatalf("ResolveTopLevel: %v", err)
	}
	for _, p := range pkgs {
		if p.Nested {
			t.Errorf("ResolveTopLevel returned nested package %q", p.Name)
		}
	}
}

func TestResolvePackages_All(t *testing.T) {
	pkgs, err := ResolvePackages(repoRootT(t), "all")
	if err != nil {
		t.Fatalf("ResolvePackages(all): %v", err)
	}
	for _, p := range pkgs {
		if p.Nested {
			t.Errorf("ResolvePackages(all) returned nested package %q", p.Name)
		}
	}
}

func TestResolvePackages_Named(t *testing.T) {
	pkgs, err := ResolvePackages(repoRootT(t), "web")
	if err != nil {
		t.Fatalf("ResolvePackages(web): %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "web" {
		t.Errorf("expected exactly [web], got %v", pkgs)
	}
}
