package version

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper that writes content to a file in dir.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile(%q): %v", path, err)
	}
	return path
}

// readFile is a test helper that reads the content of a file.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile(%q): %v", path, err)
	}
	return string(data)
}

// ---- ReadGoModVersion -------------------------------------------------------

const goModBasic = `module example.com/app

go 1.21

require (
	example.com/foo v1.2.3
	example.com/bar v0.9.0
)
`

const goModWithReplace = `module example.com/app

go 1.21

require (
	example.com/foo v1.2.3
	example.com/bar v0.9.0
	example.com/local v0.0.0-dev => ../local
)

replace (
	example.com/other v1.0.0 => ../other
)
`

func TestReadGoModVersion(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name       string
		content    string
		modulePath string
		want       string
		wantErr    bool
	}{
		{
			name:       "reads known module version",
			content:    goModBasic,
			modulePath: "example.com/foo",
			want:       "v1.2.3",
		},
		{
			name:       "reads second module in require block",
			content:    goModBasic,
			modulePath: "example.com/bar",
			want:       "v0.9.0",
		},
		{
			name:       "returns error for missing module",
			content:    goModBasic,
			modulePath: "example.com/missing",
			wantErr:    true,
		},
		{
			name:       "skips replace lines inside require block",
			content:    goModWithReplace,
			modulePath: "example.com/foo",
			want:       "v1.2.3",
		},
		{
			name:       "missing module in file with replace directives",
			content:    goModWithReplace,
			modulePath: "example.com/ghost",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, dir, "go.mod", tt.content)
			got, err := ReadGoModVersion(path, tt.modulePath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ReadGoModVersion(%q) expected error, got nil", tt.modulePath)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadGoModVersion(%q) unexpected error: %v", tt.modulePath, err)
			}
			if got != tt.want {
				t.Errorf("ReadGoModVersion(%q) = %q, want %q", tt.modulePath, got, tt.want)
			}
		})
	}
}

func TestReadGoModVersion_MissingFile(t *testing.T) {
	_, err := ReadGoModVersion("/nonexistent/path/go.mod", "example.com/foo")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// ---- WriteGoModVersion ------------------------------------------------------

func TestWriteGoModVersion(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		modulePath  string
		newVersion  string
		wantErr     bool
		wantContain string // substring that must appear in the result
		wantAbsent  string // substring that must NOT appear in the result
	}{
		{
			name:        "updates existing module version",
			content:     goModBasic,
			modulePath:  "example.com/foo",
			newVersion:  "v1.3.0",
			wantContain: "example.com/foo v1.3.0",
			wantAbsent:  "example.com/foo v1.2.3",
		},
		{
			name:        "leaves other require lines untouched",
			content:     goModBasic,
			modulePath:  "example.com/foo",
			newVersion:  "v1.3.0",
			wantContain: "example.com/bar v0.9.0",
		},
		{
			name:       "returns error for missing module",
			content:    goModBasic,
			modulePath: "example.com/notexist",
			newVersion: "v9.9.9",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "go.mod", tt.content)

			err := WriteGoModVersion(path, tt.modulePath, tt.newVersion)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("WriteGoModVersion(%q) expected error, got nil", tt.modulePath)
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteGoModVersion(%q) unexpected error: %v", tt.modulePath, err)
			}

			result := readFile(t, path)
			if tt.wantContain != "" {
				if !containsString(result, tt.wantContain) {
					t.Errorf("result does not contain %q\ngot:\n%s", tt.wantContain, result)
				}
			}
			if tt.wantAbsent != "" {
				if containsString(result, tt.wantAbsent) {
					t.Errorf("result should not contain %q\ngot:\n%s", tt.wantAbsent, result)
				}
			}
		})
	}
}

func TestWriteGoModVersion_PreservesIndentation(t *testing.T) {
	content := "module example.com/app\n\ngo 1.21\n\nrequire (\n\texample.com/foo v1.2.3\n\texample.com/bar v0.9.0\n)\n"
	dir := t.TempDir()
	path := writeFile(t, dir, "go.mod", content)

	if err := WriteGoModVersion(path, "example.com/foo", "v1.5.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := readFile(t, path)
	want := "\texample.com/foo v1.5.0"
	if !containsString(result, want) {
		t.Errorf("result does not contain indented line %q\ngot:\n%s", want, result)
	}
}

func TestWriteGoModVersion_MissingFile(t *testing.T) {
	err := WriteGoModVersion("/nonexistent/path/go.mod", "example.com/foo", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// ---- CopyModStripReplace ----------------------------------------------------

func TestCopyModStripReplace(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantContain []string
		wantAbsent  []string
	}{
		{
			name: "strips single-line replace directive",
			src: `module example.com/app

go 1.21

require (
	example.com/foo v1.2.3
)

replace example.com/foo v1.2.3 => ../foo
`,
			wantContain: []string{
				"module example.com/app",
				"go 1.21",
				"example.com/foo v1.2.3",
			},
			wantAbsent: []string{
				"replace example.com/foo",
			},
		},
		{
			name: "strips multi-line replace block",
			src: `module example.com/app

go 1.21

require (
	example.com/foo v1.2.3
	example.com/bar v0.9.0
)

replace (
	example.com/foo v1.2.3 => ../foo
	example.com/bar v0.9.0 => ../bar
)
`,
			wantContain: []string{
				"module example.com/app",
				"example.com/foo v1.2.3",
				"example.com/bar v0.9.0",
			},
			wantAbsent: []string{
				"replace (",
				"=> ../foo",
				"=> ../bar",
			},
		},
		{
			name: "preserves content without replace directives",
			src: `module example.com/app

go 1.21

require (
	example.com/foo v1.2.3
)
`,
			wantContain: []string{
				"module example.com/app",
				"go 1.21",
				"example.com/foo v1.2.3",
			},
			wantAbsent: []string{"replace"},
		},
		{
			name: "strips mixed single-line and block replace directives",
			src: `module example.com/app

go 1.21

require (
	example.com/foo v1.2.3
	example.com/bar v0.9.0
	example.com/baz v3.0.0
)

replace example.com/foo v1.2.3 => ../foo

replace (
	example.com/bar v0.9.0 => ../bar
)
`,
			wantContain: []string{
				"module example.com/app",
				"example.com/foo v1.2.3",
				"example.com/bar v0.9.0",
				"example.com/baz v3.0.0",
			},
			wantAbsent: []string{
				"replace example.com/foo",
				"replace (",
				"=> ../bar",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := writeFile(t, dir, "go.mod", tt.src)
			dst := filepath.Join(dir, "go.mod.out")

			if err := CopyModStripReplace(src, dst); err != nil {
				t.Fatalf("CopyModStripReplace() unexpected error: %v", err)
			}

			// Verify the destination file was created.
			if _, err := os.Stat(dst); err != nil {
				t.Fatalf("dst file not created: %v", err)
			}

			result := readFile(t, dst)

			for _, want := range tt.wantContain {
				if !containsString(result, want) {
					t.Errorf("result does not contain %q\ngot:\n%s", want, result)
				}
			}
			for _, absent := range tt.wantAbsent {
				if containsString(result, absent) {
					t.Errorf("result should not contain %q\ngot:\n%s", absent, result)
				}
			}
		})
	}
}

func TestCopyModStripReplace_MissingSource(t *testing.T) {
	dir := t.TempDir()
	err := CopyModStripReplace("/nonexistent/path/go.mod", filepath.Join(dir, "out.mod"))
	if err == nil {
		t.Fatal("expected error for missing source file, got nil")
	}
}

func TestCopyModStripReplace_CreatesOutputFile(t *testing.T) {
	src := `module example.com/app

go 1.21
`
	dir := t.TempDir()
	srcPath := writeFile(t, dir, "go.mod", src)
	dstPath := filepath.Join(dir, "subdir", "go.mod.out")

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := CopyModStripReplace(srcPath, dstPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := readFile(t, dstPath)
	if !containsString(result, "module example.com/app") {
		t.Errorf("output missing module line, got:\n%s", result)
	}
}

// containsString is a helper to avoid importing strings in tests.
// It replicates strings.Contains for readability.
func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
