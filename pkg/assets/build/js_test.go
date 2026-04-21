package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sum/assets/config"
)

func TestRemoveStaleJS(t *testing.T) {
	dir := t.TempDir()

	managed := filepath.Join(dir, "app.js")
	stale := filepath.Join(dir, "old.js")

	if err := os.WriteFile(managed, []byte("// managed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("// stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		JS: config.JSConfig{
			Bundles: []config.JSBundle{
				{Entry: "src/app.js", Target: managed},
			},
		},
	}

	var out strings.Builder
	if err := RemoveStaleJS(cfg, &out); err != nil {
		t.Fatalf("RemoveStaleJS: %v", err)
	}

	if _, err := os.Stat(managed); err != nil {
		t.Errorf("managed file should still exist: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale file should have been removed")
	}
	if !strings.Contains(out.String(), "old.js") {
		t.Errorf("output should mention removed file, got: %s", out.String())
	}
}

// TestBuildJSIfChanged_SkipUnchangedBundle verifies that a second call with
// unchanged source files skips the bundle and prints the skip message.
func TestBuildJSIfChanged_SkipUnchangedBundle(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()
	stateDir := t.TempDir()

	entry := filepath.Join(srcDir, "app.js")
	target := filepath.Join(outDir, "app.bundle.js")
	statePath := filepath.Join(stateDir, "state.json")

	if err := os.WriteFile(entry, []byte("const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		JS: config.JSConfig{
			Bundles: []config.JSBundle{
				{Entry: entry, Target: target},
			},
		},
	}

	// First call — should build.
	state := LoadState(statePath)
	var out1 strings.Builder
	if err := BuildJSIfChanged(cfg, false, state, &out1); err != nil {
		t.Fatalf("first BuildJSIfChanged: %v", err)
	}

	// Second call with the same state — no source changes.
	state2 := LoadState(statePath)
	var out2 strings.Builder
	if err := BuildJSIfChanged(cfg, false, state2, &out2); err != nil {
		t.Fatalf("second BuildJSIfChanged: %v", err)
	}

	if !strings.Contains(out2.String(), "no changes, skipping") {
		t.Errorf("expected skip message on second call, got: %q", out2.String())
	}
}

// TestBuildJSIfChanged_RebuildOnSourceChange verifies that modifying a source
// file causes the bundle to be rebuilt on the next call.
func TestBuildJSIfChanged_RebuildOnSourceChange(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()
	stateDir := t.TempDir()

	entry := filepath.Join(srcDir, "app.js")
	target := filepath.Join(outDir, "app.bundle.js")
	statePath := filepath.Join(stateDir, "state.json")

	if err := os.WriteFile(entry, []byte("const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		JS: config.JSConfig{
			Bundles: []config.JSBundle{
				{Entry: entry, Target: target},
			},
		},
	}

	// First call — build and record state.
	state := LoadState(statePath)
	if err := BuildJSIfChanged(cfg, false, state, &strings.Builder{}); err != nil {
		t.Fatalf("first BuildJSIfChanged: %v", err)
	}

	// Modify the source file.
	if err := os.WriteFile(entry, []byte("const x = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second call — should rebuild.
	state2 := LoadState(statePath)
	var out2 strings.Builder
	if err := BuildJSIfChanged(cfg, false, state2, &out2); err != nil {
		t.Fatalf("second BuildJSIfChanged: %v", err)
	}

	if !strings.Contains(out2.String(), "bundled") {
		t.Errorf("expected rebuild on source change, got: %q", out2.String())
	}
	if strings.Contains(out2.String(), "no changes, skipping") {
		t.Errorf("should not skip when source changed, got: %q", out2.String())
	}
}

// TestBuildJSIfChanged_SkipsDownloads verifies that BuildJSIfChanged does not
// attempt any network downloads, even when downloads are configured with URLs
// that would fail.
func TestBuildJSIfChanged_SkipsDownloads(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()
	stateDir := t.TempDir()

	entry := filepath.Join(srcDir, "app.js")
	target := filepath.Join(outDir, "app.bundle.js")
	statePath := filepath.Join(stateDir, "state.json")
	bogusTarget := filepath.Join(outDir, "bogus.js")

	if err := os.WriteFile(entry, []byte("const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		JS: config.JSConfig{
			Downloads: []config.JSDownload{
				{Name: "bogus", URL: "http://127.0.0.1:0/bogus.js", Target: bogusTarget},
			},
			Bundles: []config.JSBundle{
				{Entry: entry, Target: target},
			},
		},
	}

	state := LoadState(statePath)
	var out strings.Builder
	if err := BuildJSIfChanged(cfg, false, state, &out); err != nil {
		t.Fatalf("BuildJSIfChanged should not error on bogus download URL: %v", err)
	}
}

// TestBuildJSIfChanged_MinifyChangeDetection verifies that the minify entries
// participate in change detection: second call skips, third call after source
// modification rebuilds.
func TestBuildJSIfChanged_MinifyChangeDetection(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()
	stateDir := t.TempDir()

	source := filepath.Join(srcDir, "inline.js")
	target := filepath.Join(outDir, "inline.min.js")
	statePath := filepath.Join(stateDir, "state.json")

	if err := os.WriteFile(source, []byte("const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		JS: config.JSConfig{
			Minify: []config.JSMinify{
				{Source: source, Target: target},
			},
		},
	}

	// First call — should minify.
	state := LoadState(statePath)
	var out1 strings.Builder
	if err := BuildJSIfChanged(cfg, false, state, &out1); err != nil {
		t.Fatalf("first BuildJSIfChanged: %v", err)
	}
	if !strings.Contains(out1.String(), "minified") {
		t.Errorf("expected minified message on first call, got: %q", out1.String())
	}

	// Second call — no changes, should skip.
	state2 := LoadState(statePath)
	var out2 strings.Builder
	if err := BuildJSIfChanged(cfg, false, state2, &out2); err != nil {
		t.Fatalf("second BuildJSIfChanged: %v", err)
	}
	if !strings.Contains(out2.String(), "no changes, skipping") {
		t.Errorf("expected skip message on second call, got: %q", out2.String())
	}

	// Modify the source file.
	if err := os.WriteFile(source, []byte("const x = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Third call — source changed, should rebuild.
	state3 := LoadState(statePath)
	var out3 strings.Builder
	if err := BuildJSIfChanged(cfg, false, state3, &out3); err != nil {
		t.Fatalf("third BuildJSIfChanged: %v", err)
	}
	if !strings.Contains(out3.String(), "minified") {
		t.Errorf("expected minified message on third call after source change, got: %q", out3.String())
	}
	if strings.Contains(out3.String(), "no changes, skipping") {
		t.Errorf("should not skip when source changed, got: %q", out3.String())
	}
}
