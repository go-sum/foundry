package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunGoOrderListReportsChangedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	input := `package sample

func beta() {}

const answer = 42

type thing struct{}
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var out bytes.Buffer
	if err := runGoOrder([]string{path}, false, true, &out); err != nil {
		t.Fatalf("runGoOrder: %v", err)
	}

	if got, want := out.String(), path+"\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	gotBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(gotBytes) != input {
		t.Fatalf("file was modified:\n%s", gotBytes)
	}
}

func TestRunGoOrderListSkipsUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	input := `package sample

type thing struct{}

const answer = 42

func alpha() {}
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var out bytes.Buffer
	if err := runGoOrder([]string{path}, false, true, &out); err != nil {
		t.Fatalf("runGoOrder: %v", err)
	}

	if got := out.String(); got != "" {
		t.Fatalf("got %q, want empty output", got)
	}
}

func TestRunGoOrderPreviewsFormattedSourceByDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	input := `package sample

func beta() {}

const answer = 42

type thing struct{}
`
	want := `package sample

type thing struct{}

const answer = 42

func beta() {}
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var out bytes.Buffer
	if err := runGoOrder([]string{path}, false, false, &out); err != nil {
		t.Fatalf("runGoOrder: %v", err)
	}

	if got := out.String(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunGoOrderRejectsWriteAndList(t *testing.T) {
	err := runGoOrder([]string{"sample.go"}, true, true, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
