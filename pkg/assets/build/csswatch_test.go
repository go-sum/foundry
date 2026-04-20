package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCSSWatchPaths(t *testing.T) {
	dir := t.TempDir()
	cssContent := `@import "./css/theme.css";
@source "../**/*.go";
@source "./components/**/*.html";
`
	inputPath := filepath.Join(dir, "tailwind.css")
	if err := os.WriteFile(inputPath, []byte(cssContent), 0o644); err != nil {
		t.Fatal(err)
	}

	paths, err := CSSWatchPaths(inputPath)
	if err != nil {
		t.Fatalf("CSSWatchPaths: %v", err)
	}

	pathSet := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		pathSet[p] = struct{}{}
	}

	// Input file must always be included.
	if _, ok := pathSet[inputPath]; !ok {
		t.Errorf("input file %s not in result", inputPath)
	}

	// @import "./css/theme.css" resolves relative to the CSS file's directory.
	wantImport := filepath.Join(dir, "css/theme.css")
	if _, ok := pathSet[wantImport]; !ok {
		t.Errorf("expected @import path %s in result, got %v", wantImport, paths)
	}

	// @source "../**/*.go" resolves relative to the CSS file's directory.
	wantSource := filepath.Join(dir, "../**/*.go")
	wantSourceClean := filepath.Clean(wantSource)
	if _, ok := pathSet[wantSourceClean]; !ok {
		t.Errorf("expected @source pattern %s in result, got %v", wantSourceClean, paths)
	}

	// @source "./components/**/*.html" resolves correctly.
	wantComp := filepath.Join(dir, "components/**/*.html")
	if _, ok := pathSet[wantComp]; !ok {
		t.Errorf("expected @source pattern %s in result, got %v", wantComp, paths)
	}
}

func TestCSSWatchPaths_skipsURLImports(t *testing.T) {
	dir := t.TempDir()
	cssContent := `@import "https://fonts.googleapis.com/css2?family=Inter";
@import "http://example.com/styles.css";
@import url("https://cdn.example.com/reset.css");
@source "../**/*.go";
`
	inputPath := filepath.Join(dir, "tailwind.css")
	if err := os.WriteFile(inputPath, []byte(cssContent), 0o644); err != nil {
		t.Fatal(err)
	}

	paths, err := CSSWatchPaths(inputPath)
	if err != nil {
		t.Fatalf("CSSWatchPaths: %v", err)
	}

	for _, p := range paths {
		if p == "https://fonts.googleapis.com/css2?family=Inter" {
			t.Error("https URL import should be excluded")
		}
		if p == "http://example.com/styles.css" {
			t.Error("http URL import should be excluded")
		}
	}
}

func TestCSSWatchPaths_includesInputFile(t *testing.T) {
	dir := t.TempDir()
	// CSS file with no directives at all.
	inputPath := filepath.Join(dir, "empty.css")
	if err := os.WriteFile(inputPath, []byte("/* empty */\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths, err := CSSWatchPaths(inputPath)
	if err != nil {
		t.Fatalf("CSSWatchPaths: %v", err)
	}

	found := false
	for _, p := range paths {
		if p == inputPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("input file not included in result: %v", paths)
	}
}
