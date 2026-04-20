package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadState_missing(t *testing.T) {
	dir := t.TempDir()
	sf := LoadState(filepath.Join(dir, "nonexistent.json"))
	if sf == nil {
		t.Fatal("LoadState returned nil")
	}
	if len(sf.Hashes) != 0 {
		t.Errorf("expected empty hashes, got %v", sf.Hashes)
	}
}

func TestHasChanged_noState(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	sf := LoadState(filepath.Join(dir, "state.json"))
	changed, err := sf.HasChanged("css:out.css", []string{f})
	if err != nil {
		t.Fatalf("HasChanged: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when no prior state")
	}
}

func TestHasChanged_noChange(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(dir, "state.json")
	sf := LoadState(statePath)
	if err := sf.MarkBuilt("css:out.css", []string{f}); err != nil {
		t.Fatalf("MarkBuilt: %v", err)
	}

	// Reload to confirm persistence then check again.
	sf2 := LoadState(statePath)
	changed, err := sf2.HasChanged("css:out.css", []string{f})
	if err != nil {
		t.Fatalf("HasChanged: %v", err)
	}
	if changed {
		t.Error("expected changed=false after MarkBuilt with same content")
	}
}

func TestHasChanged_fileModified(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(dir, "state.json")
	sf := LoadState(statePath)
	if err := sf.MarkBuilt("css:out.css", []string{f}); err != nil {
		t.Fatalf("MarkBuilt: %v", err)
	}

	// Modify the file.
	if err := os.WriteFile(f, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := sf.HasChanged("css:out.css", []string{f})
	if err != nil {
		t.Fatalf("HasChanged: %v", err)
	}
	if !changed {
		t.Error("expected changed=true after file content changed")
	}
}

func TestMarkBuilt_atomic(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(dir, "state.json")
	sf := LoadState(statePath)
	if err := sf.MarkBuilt("key1", []string{f}); err != nil {
		t.Fatalf("MarkBuilt: %v", err)
	}

	// Confirm the state file exists and is readable by a fresh LoadState.
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file missing after MarkBuilt: %v", err)
	}
	sf2 := LoadState(statePath)
	if _, ok := sf2.Hashes["key1"]; !ok {
		t.Error("expected key1 in loaded state")
	}

	// Confirm the tmp file was cleaned up.
	if _, err := os.Stat(statePath + ".tmp"); err == nil {
		t.Error("expected tmp file to be removed after atomic rename")
	}
}
