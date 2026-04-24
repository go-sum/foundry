package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildInvokesHugoAndRebuildsOutput verifies that build() with overwrite=true:
//   - invokes hugo with --source and --destination
//   - removes stale output before invoking hugo
//   - accepts the newly generated files from hugo
func TestBuildInvokesHugoAndRebuildsOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "public", "doc")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	// Plant a stale file that should be removed before the build runs.
	stalePath := filepath.Join(outputDir, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	capturePath := filepath.Join(tmpDir, "hugo.args")
	fakeHugoDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(fakeHugoDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeHugoPath := filepath.Join(fakeHugoDir, "hugo")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"src=''\n" +
		"dest=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  case \"$1\" in\n" +
		"    --source) src=\"$2\"; shift 2 ;;\n" +
		"    --destination) dest=\"$2\"; shift 2 ;;\n" +
		"    *) shift ;;\n" +
		"  esac\n" +
		"done\n" +
		"printf '%s\\n%s\\n' \"$src\" \"$dest\" > \"$TEST_CAPTURE\"\n" +
		"mkdir -p \"$dest/guide\"\n" +
		"printf '<h1>Docs</h1>' > \"$dest/index.html\"\n" +
		"printf '<h1>Guide</h1>' > \"$dest/guide/index.html\"\n" +
		"printf '<h1>Missing</h1>' > \"$dest/404.html\"\n"
	if err := os.WriteFile(fakeHugoPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake hugo: %v", err)
	}

	t.Setenv("PATH", fakeHugoDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_CAPTURE", capturePath)

	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	if err := build(sourceDir, outputDir, true); err != nil {
		t.Fatalf("build() error = %v", err)
	}

	// Stale file must be gone.
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale docs output should have been removed, Stat err = %v", err)
	}
	// Hugo-generated files must be present.
	if _, err := os.Stat(filepath.Join(outputDir, "index.html")); err != nil {
		t.Fatalf("generated index.html missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "guide", "index.html")); err != nil {
		t.Fatalf("generated guide/index.html missing: %v", err)
	}

	// Verify the args captured by the fake hugo script.
	argsRaw, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(argsRaw)), "\n")
	if len(args) != 2 {
		t.Fatalf("captured args = %q, want source and destination on separate lines", string(argsRaw))
	}
	if got := filepath.Clean(args[0]); got != filepath.Clean(sourceDir) {
		t.Fatalf("source = %q, want %q", got, filepath.Clean(sourceDir))
	}
	if got := filepath.Clean(args[1]); got != filepath.Clean(outputDir) {
		t.Fatalf("destination = %q, want %q", got, filepath.Clean(outputDir))
	}
}

// TestBuildForwardsCustomSource verifies that a non-default source directory
// is forwarded verbatim to the hugo --source flag.
func TestBuildForwardsCustomSource(t *testing.T) {
	tmpDir := t.TempDir()
	capturePath := filepath.Join(tmpDir, "hugo.args")
	fakeHugoDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(fakeHugoDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeHugoPath := filepath.Join(fakeHugoDir, "hugo")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"src=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  case \"$1\" in\n" +
		"    --source) src=\"$2\"; shift 2 ;;\n" +
		"    *) shift ;;\n" +
		"  esac\n" +
		"done\n" +
		"printf '%s\\n' \"$src\" > \"$TEST_CAPTURE\"\n"
	if err := os.WriteFile(fakeHugoPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake hugo: %v", err)
	}

	t.Setenv("PATH", fakeHugoDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_CAPTURE", capturePath)

	sourceDir := filepath.Join(tmpDir, "my-docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	outputDir := filepath.Join(tmpDir, "public", "doc")
	if err := build(sourceDir, outputDir, false); err != nil {
		t.Fatalf("build() error = %v", err)
	}

	gotRaw, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	if got := filepath.Clean(strings.TrimSpace(string(gotRaw))); got != filepath.Clean(sourceDir) {
		t.Fatalf("source = %q, want %q", got, filepath.Clean(sourceDir))
	}
}

// TestBuildReturnsErrorWhenHugoFails verifies that a non-zero hugo exit code
// causes build() to return a non-nil error.
func TestBuildReturnsErrorWhenHugoFails(t *testing.T) {
	tmpDir := t.TempDir()
	fakeHugoDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(fakeHugoDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeHugoPath := filepath.Join(fakeHugoDir, "hugo")
	if err := os.WriteFile(fakeHugoPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake hugo: %v", err)
	}

	t.Setenv("PATH", fakeHugoDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	if err := build(sourceDir, filepath.Join(tmpDir, "public", "doc"), false); err == nil {
		t.Fatal("build() error = nil, want non-nil when hugo exits 1")
	}
}

// TestBuildScaffoldsWhenSourceMissing verifies that build() auto-scaffolds the
// source directory from the embedded template when it does not exist, rather
// than failing with an opaque Hugo error.
func TestBuildScaffoldsWhenSourceMissing(t *testing.T) {
	tmpDir := t.TempDir()
	fakeHugoDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(fakeHugoDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeHugoDir, "hugo"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake hugo: %v", err)
	}
	t.Setenv("PATH", fakeHugoDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := build(sourceDir, filepath.Join(tmpDir, "public", "doc"), false); err != nil {
		t.Fatalf("build() error = %v, want nil (auto-scaffold)", err)
	}
	// Verify the source directory was scaffolded.
	if _, err := os.Stat(filepath.Join(sourceDir, "hugo.toml")); err != nil {
		t.Fatalf("hugo.toml missing after auto-scaffold: %v", err)
	}
}

// TestBuildPreservesStaleOutputWithoutOverwrite verifies that stale files in the
// destination are NOT removed when overwrite is false.
func TestBuildPreservesStaleOutputWithoutOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "public", "docs")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	stalePath := filepath.Join(outputDir, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	fakeHugoDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(fakeHugoDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeHugoDir, "hugo"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake hugo: %v", err)
	}
	t.Setenv("PATH", fakeHugoDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	if err := build(sourceDir, outputDir, false); err != nil {
		t.Fatalf("build() error = %v", err)
	}
	if _, err := os.Stat(stalePath); err != nil {
		t.Fatal("stale file should be preserved when overwrite=false")
	}
}

// TestBuildRejectsEmptyDestination verifies that build() returns an error for an empty destination.
func TestBuildRejectsEmptyDestination(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	err := build(sourceDir, "", false)
	if err == nil {
		t.Fatal("build() error = nil, want non-nil for empty destination")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("error = %q, want it to contain 'must not be empty'", err.Error())
	}
}

// TestBuildRejectsRootDestination verifies that build() returns an error when destination is /.
func TestBuildRejectsRootDestination(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	err := build(sourceDir, "/", false)
	if err == nil {
		t.Fatal("build() error = nil, want non-nil for root destination")
	}
	if !strings.Contains(err.Error(), "filesystem root") {
		t.Fatalf("error = %q, want it to contain 'filesystem root'", err.Error())
	}
}

// TestBuildRejectsSingleComponentDestination verifies that build() returns an error
// when the destination has fewer than 2 path components.
func TestBuildRejectsSingleComponentDestination(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	// Use an absolute path with only one real component beyond root.
	err := build(sourceDir, "/tmp", false)
	if err == nil {
		t.Fatal("build() error = nil, want non-nil for single-component destination")
	}
	if !strings.Contains(err.Error(), "at least 2 components") {
		t.Fatalf("error = %q, want it to contain 'at least 2 components'", err.Error())
	}
}

// TestBuildReturnsErrorWhenHugoNotFound verifies that build() returns a descriptive error
// when hugo is not available on PATH.
func TestBuildReturnsErrorWhenHugoNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	// Set PATH to an empty directory so hugo cannot be found.
	t.Setenv("PATH", tmpDir)
	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	err := build(sourceDir, filepath.Join(tmpDir, "public", "docs"), false)
	if err == nil {
		t.Fatal("build() error = nil, want non-nil when hugo not found")
	}
	if !strings.Contains(err.Error(), "hugo not found on PATH") {
		t.Fatalf("error = %q, want it to contain 'hugo not found on PATH'", err.Error())
	}
}

// TestBuildResolvesRelativeDestination verifies that a relative destination
// path is resolved to an absolute path before being passed to hugo.
func TestBuildResolvesRelativeDestination(t *testing.T) {
	tmpDir := t.TempDir()
	fakeHugoDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(fakeHugoDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	capturePath := filepath.Join(tmpDir, "hugo.args")
	fakeHugoPath := filepath.Join(fakeHugoDir, "hugo")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"dest=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  case \"$1\" in\n" +
		"    --destination) dest=\"$2\"; shift 2 ;;\n" +
		"    *) shift ;;\n" +
		"  esac\n" +
		"done\n" +
		"printf '%s\\n' \"$dest\" > \"$TEST_CAPTURE\"\n"
	if err := os.WriteFile(fakeHugoPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake hugo: %v", err)
	}

	t.Setenv("PATH", fakeHugoDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_CAPTURE", capturePath)

	sourceDir := filepath.Join(tmpDir, ".docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	if err := build(sourceDir, "public/doc", false); err != nil {
		t.Fatalf("build() error = %v", err)
	}

	gotRaw, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	got := filepath.Clean(strings.TrimSpace(string(gotRaw)))
	want := filepath.Join(cwd, "public", "doc")
	if got != want {
		t.Fatalf("destination = %q, want %q", got, want)
	}
}
