package publish

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_withFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log('hello')"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := New(dir, "/public")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := m.Path("app.js")
	if !strings.HasPrefix(got, "/public/app.js?v=") {
		t.Errorf("Path = %q, want prefix /public/app.js?v=", got)
	}
	hash := strings.TrimPrefix(got, "/public/app.js?v=")
	if len(hash) != 8 {
		t.Errorf("hash length = %d, want 8", len(hash))
	}
}

func TestNew_missingDir(t *testing.T) {
	m, err := New("/nonexistent/dir", "/public")
	if err != nil {
		t.Fatalf("New with missing dir returned error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manifest")
	}
}

func TestManifest_Path_miss(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir, "/public")
	if err != nil {
		t.Fatal(err)
	}
	got := m.Path("unknown.js")
	if got != "/public/unknown.js" {
		t.Errorf("Path miss = %q, want %q", got, "/public/unknown.js")
	}
}

func TestMust_panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Must with error should panic")
		}
	}()
	Must(nil, errors.New("x"))
}

func TestMust_noError(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir, "/public")
	if err != nil {
		t.Fatal(err)
	}
	got := Must(m, nil)
	if got != m {
		t.Error("Must should return the manifest unchanged")
	}
}

func TestInit_Default(t *testing.T) {
	t.Cleanup(func() { Default = nil })
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init(dir, "/public"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if Default == nil {
		t.Fatal("Default should be set after Init")
	}
	got := Path("style.css")
	if !strings.HasPrefix(got, "/public/style.css?v=") {
		t.Errorf("Path = %q, want prefix /public/style.css?v=", got)
	}
}
