package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCreatesExpectedStructure(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, ".docs")

	if err := scaffold(target); err != nil {
		t.Fatalf("scaffold() error = %v", err)
	}

	expectedFiles := []string{
		"hugo.toml",
		"go.mod",
		".gitignore",
		filepath.Join("content", "_index.md"),
		filepath.Join("layouts", "_default", "baseof.html"),
		filepath.Join("layouts", "_default", "list.html"),
		filepath.Join("layouts", "_default", "single.html"),
		filepath.Join("layouts", "partials", "sidebar.html"),
		filepath.Join("layouts", "404.html"),
		filepath.Join("assets", "css", "docs.css"),
		filepath.Join("assets", "js", "theme.js"),
	}
	for _, rel := range expectedFiles {
		if _, err := os.Stat(filepath.Join(target, rel)); err != nil {
			t.Errorf("expected file missing: %s", rel)
		}
	}
}

func TestScaffoldStripsTmplSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, ".docs")

	if err := scaffold(target); err != nil {
		t.Fatalf("scaffold() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "go.mod")); err != nil {
		t.Fatal("go.mod not found after scaffold")
	}
	if _, err := os.Stat(filepath.Join(target, "go.mod.tmpl")); !os.IsNotExist(err) {
		t.Fatal("go.mod.tmpl should not exist after scaffold (suffix must be stripped)")
	}
}

func TestScaffoldFileContents(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, ".docs")

	if err := scaffold(target); err != nil {
		t.Fatalf("scaffold() error = %v", err)
	}

	gomod, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if !strings.Contains(string(gomod), "module docs") {
		t.Fatalf("go.mod = %q, want it to contain 'module docs'", string(gomod))
	}

	gitignore, err := os.ReadFile(filepath.Join(target, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), ".hugo_build.lock") {
		t.Fatalf(".gitignore = %q, want it to contain '.hugo_build.lock'", string(gitignore))
	}
	if !strings.Contains(string(gitignore), "resources/") {
		t.Fatalf(".gitignore = %q, want it to contain 'resources/'", string(gitignore))
	}
}

func TestInitDocsFailsIfTargetExists(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Change to tmpDir so initDocs() targets .docs relative to it.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err = initDocs()
	if err == nil {
		t.Fatal("initDocs() error = nil, want non-nil when .docs already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want it to contain 'already exists'", err.Error())
	}
}
