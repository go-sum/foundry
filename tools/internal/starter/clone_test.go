package starter

import (
	"os"
	"strings"
	"testing"
)

func TestIsExcluded(t *testing.T) {
	manifest := Manifest{
		Exclude: []string{
			".decisions/",
			".plans/",
			"go.work",
			"go.work.sum",
			"pkg/",
			"tools/",
		},
	}

	tests := []struct {
		path string
		want bool
	}{
		// Always excluded
		{".git", true},
		{".git/config", true},
		// Explicit file exclusions
		{"go.work", true},
		{"go.work.sum", true},
		// Directory exclusions (trailing slash)
		{".decisions/DESIGN_GUIDE.md", true},
		{".plans/myplan.md", true},
		{"pkg/web/go.mod", true},
		{"tools/go.mod", true},
		// Included paths
		{"starter/go.mod", false},
		{"starter/internal/app/app.go", false},
		{"README.md", false},
		{"Taskfile.yml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsExcluded(manifest, tt.path)
			if got != tt.want {
				t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestStripMonorepoBlocks(t *testing.T) {
	dir := t.TempDir()

	content := "# README\n\nSome intro.\n\n<!-- monorepo-only-start -->\nThis section is only for the monorepo.\n<!-- monorepo-only-end -->\nAfter the block.\n"
	want := "# README\n\nSome intro.\n\nAfter the block.\n"

	writeTestFile(t, dir+"/README.md", content)

	count, err := stripMonorepoBlocks(dir)
	if err != nil {
		t.Fatalf("stripMonorepoBlocks: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	got := readTestFile(t, dir+"/README.md")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestStripMonorepoBlocks_NoMarkers(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir+"/README.md", "# Hello\n\nNo markers here.\n")

	count, err := stripMonorepoBlocks(dir)
	if err != nil {
		t.Fatalf("stripMonorepoBlocks: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestRewriteModule(t *testing.T) {
	dir := t.TempDir()

	gomod := "module github.com/go-sum/foundry\n\ngo 1.26.0\n\nrequire (\n\tgithub.com/go-sum/foundry/tools v0.0.0\n)\n"
	gofile := "package main\n\nimport (\n\t\"github.com/go-sum/foundry/internal/app\"\n)\n"

	writeTestFile(t, dir+"/go.mod", gomod)
	writeTestFile(t, dir+"/main.go", gofile)

	count, err := rewriteModule(dir, "github.com/go-sum/foundry", "github.com/myorg/myapp")
	if err != nil {
		t.Fatalf("rewriteModule: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	gotMod := readTestFile(t, dir+"/go.mod")
	if !strings.Contains(gotMod, "module github.com/myorg/myapp") {
		t.Errorf("go.mod missing new module directive:\n%s", gotMod)
	}
	if strings.Contains(gotMod, "github.com/go-sum/foundry") {
		t.Errorf("go.mod still contains old module path:\n%s", gotMod)
	}

	gotGo := readTestFile(t, dir+"/main.go")
	if !strings.Contains(gotGo, `"github.com/myorg/myapp/internal/app"`) {
		t.Errorf("main.go import not rewritten:\n%s", gotGo)
	}
}

func TestTransformAirToml(t *testing.T) {
	dir := t.TempDir()
	airDir := dir + "/docker/app"
	if err := os.MkdirAll(airDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	input := `root = "."
tmp_dir = "tmp"

[build]
  pre_cmd = ["go run ../pkg/assets/cli build css --incremental", "go run ../pkg/assets/cli sprites --incremental", "go run ../pkg/assets/cli build js --incremental"]
  cmd = "go build -o ./tmp/server ./cmd/server"
  include_dir = ["../pkg/componentry"]
  include_ext = ["go", "html", "css"]
`
	want := `root = "."
tmp_dir = "tmp"

[build]
  pre_cmd = ["assets build css --incremental", "assets sprites --incremental", "assets build js --incremental"]
  cmd = "go build -o ./tmp/server ./cmd/server"
  include_ext = ["go", "html", "css"]
`
	writeTestFile(t, airDir+"/.air.toml", input)

	count, err := transformAirToml(dir)
	if err != nil {
		t.Fatalf("transformAirToml: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	got := readTestFile(t, airDir+"/.air.toml")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestTransformRootTaskfile(t *testing.T) {
	dir := t.TempDir()

	input := `tasks:
  build:assets:
    cmd: '{{.RUN_WITH_TOOLS}} go run ../pkg/assets/cli build all --config .assets.yaml --minify'

  build:docs:
    cmd: '{{.RUN_WITH_TOOLS}} go run ../pkg/docs/cli build'
`
	want := `tasks:
  build:assets:
    cmd: '{{.RUN_WITH_TOOLS}} assets build all --config .assets.yaml --minify'

  build:docs:
    cmd: '{{.RUN_WITH_TOOLS}} docs build'
`
	writeTestFile(t, dir+"/Taskfile.yml", input)

	count, err := transformRootTaskfile(dir)
	if err != nil {
		t.Fatalf("transformRootTaskfile: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	got := readTestFile(t, dir+"/Taskfile.yml")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestTransformRootTaskfile_Missing(t *testing.T) {
	count, err := transformRootTaskfile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestTransformAirToml_Missing(t *testing.T) {
	count, err := transformAirToml(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile %s: %v", path, err)
	}
	return string(data)
}
