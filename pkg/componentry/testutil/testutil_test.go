package testutil

import (
	"os"
	"path/filepath"
	"testing"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func TestRenderNode(t *testing.T) {
	got := RenderNode(t, h.Div(g.Text("hello")))
	if got != "<div>hello</div>" {
		t.Fatalf("RenderNode() = %q, want %q", got, "<div>hello</div>")
	}
}

func TestExample(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join("testdata", "example.txtar"), []byte("-- case one --\n<div>saved</div>\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Run("case_one", func(t *testing.T) {
		if got, want := LoadGolden(t), "<div>saved</div>"; got != want {
			t.Fatalf("LoadGolden() = %q, want %q", got, want)
		}
	})
}

func TestWidget(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("TESTDATA_UPDATE", "1")

	t.Run("writes_section", func(t *testing.T) {
		AssertEqualHTML(t, "", "<div>updated</div>")
	})

	data, err := os.ReadFile(filepath.Join("testdata", "widget.txtar"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got, want := string(data), "-- writes section --\n<div>updated</div>\n"; got != want {
		t.Fatalf("archive = %q, want %q", got, want)
	}
}

func TestInternalHelpers(t *testing.T) {
	if got, want := archivePathFor(t), filepath.Join("testdata", "internalhelpers.txtar"); got != want {
		t.Fatalf("archivePathFor() = %q, want %q", got, want)
	}

	t.Run("section_name", func(t *testing.T) {
		if got, want := sectionNameFor(t), "section name"; got != want {
			t.Fatalf("sectionNameFor() = %q, want %q", got, want)
		}
	})

	if got, want := itoa(42), "42"; got != want {
		t.Fatalf("itoa() = %q, want %q", got, want)
	}
}
