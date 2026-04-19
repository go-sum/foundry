// Package testutil provides golden-file test helpers for componentry tests.
package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	g "maragu.dev/gomponents"
)

// RenderNode renders a gomponents node to a string for assertion.
func RenderNode(t *testing.T, node g.Node) string {
	t.Helper()
	var buf bytes.Buffer
	if err := node.Render(&buf); err != nil {
		t.Fatalf("RenderNode: render failed: %v", err)
	}
	return buf.String()
}

// LoadGolden reads the section matching this subtest from the txtar archive at
// testdata/<stem>.txtar, where <stem> is derived from the top-level test
// function name (stripped of "Test" prefix, lowercased). The section name is
// the leaf subtest name with underscores replaced by spaces.
//
// When TESTDATA_UPDATE=1, returns "" so AssertEqualHTML can write the section.
func LoadGolden(t *testing.T) string {
	t.Helper()
	if os.Getenv("TESTDATA_UPDATE") == "1" {
		return ""
	}
	path := archivePathFor(t)
	section := sectionNameFor(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("LoadGolden: cannot read archive %s: %v", path, err)
	}
	ar := txtar.Parse(data)
	for _, f := range ar.Files {
		if f.Name == section {
			return strings.TrimRight(string(f.Data), "\n")
		}
	}
	t.Fatalf("LoadGolden: section %q not found in %s", section, path)
	return ""
}

// AssertEqualHTML compares want and got, printing a line diff on mismatch.
// When TESTDATA_UPDATE=1, writes got into the correct section of the txtar
// archive derived from t.Name(), creating the archive if it does not exist.
func AssertEqualHTML(t *testing.T, want, got string) {
	t.Helper()
	if os.Getenv("TESTDATA_UPDATE") == "1" {
		path := archivePathFor(t)
		section := sectionNameFor(t)
		var ar *txtar.Archive
		if data, err := os.ReadFile(path); err == nil {
			ar = txtar.Parse(data)
		} else {
			ar = &txtar.Archive{}
		}
		updated := false
		for i, f := range ar.Files {
			if f.Name == section {
				ar.Files[i].Data = []byte(got + "\n")
				updated = true
				break
			}
		}
		if !updated {
			ar.Files = append(ar.Files, txtar.File{Name: section, Data: []byte(got + "\n")})
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("AssertEqualHTML: mkdir: %v", err)
		}
		if err := os.WriteFile(path, txtar.Format(ar), 0o644); err != nil {
			t.Fatalf("AssertEqualHTML: write archive: %v", err)
		}
		return
	}
	if want == got {
		return
	}
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	var sb strings.Builder
	sb.WriteString("HTML mismatch:\n")
	maxLines := len(wantLines)
	if len(gotLines) > maxLines {
		maxLines = len(gotLines)
	}
	for i := range maxLines {
		var w, gr string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			gr = gotLines[i]
		}
		if w != gr {
			sb.WriteString("line ")
			sb.WriteString(itoa(i + 1))
			sb.WriteString(":\n  want: ")
			sb.WriteString(w)
			sb.WriteString("\n   got: ")
			sb.WriteString(gr)
			sb.WriteString("\n")
		}
	}
	t.Error(sb.String())
}

// archivePathFor derives "testdata/<stem>.txtar" from t.Name().
// e.g. "TestInput/text_type_default" → "testdata/input.txtar"
func archivePathFor(t *testing.T) string {
	t.Helper()
	top := strings.SplitN(t.Name(), "/", 2)[0]
	stem := strings.TrimPrefix(top, "Test")
	return filepath.Join("testdata", strings.ToLower(stem)+".txtar")
}

// sectionNameFor returns the leaf segment of t.Name() with underscores
// replaced by spaces, matching the test case "name" field convention.
// e.g. "TestInput/text_type_default" → "text type default"
func sectionNameFor(t *testing.T) string {
	t.Helper()
	parts := strings.SplitN(t.Name(), "/", 2)
	if len(parts) < 2 {
		t.Fatalf("sectionNameFor: t.Name() %q has no subtest segment", t.Name())
	}
	return strings.ReplaceAll(parts[1], "_", " ")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
