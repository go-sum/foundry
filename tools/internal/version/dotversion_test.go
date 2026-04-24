package version

import (
	"os"
	"path/filepath"
	"testing"
)

// writeDotVersionFile writes a .versions file in dir with the given content
// and returns the dir path (repoRoot for ReadDotVersion/WriteDotVersion).
func writeDotVersionFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".versions")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeDotVersionFile: %v", err)
	}
	return dir
}

// ---- ReadDotVersion ---------------------------------------------------------

func TestReadDotVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "reads existing key",
			content: "FOO=v1.0.0\n",
			key:     "FOO",
			want:    "v1.0.0",
		},
		{
			name:    "reads key when multiple keys present",
			content: "FOO=v1.0.0\nBAR=v2.3.4\nBAZ=v0.0.1\n",
			key:     "BAR",
			want:    "v2.3.4",
		},
		{
			name:    "reads last key when multiple keys present",
			content: "FOO=v1.0.0\nBAR=v2.3.4\nBAZ=v0.0.1\n",
			key:     "BAZ",
			want:    "v0.0.1",
		},
		{
			name:    "returns error for missing key",
			content: "FOO=v1.0.0\nBAR=v2.3.4\n",
			key:     "MISSING",
			wantErr: true,
		},
		{
			name:    "returns error for empty file",
			content: "",
			key:     "FOO",
			wantErr: true,
		},
		{
			name:    "skips comment lines",
			content: "# this is a comment\nFOO=v1.0.0\n",
			key:     "FOO",
			want:    "v1.0.0",
		},
		{
			name:    "skips blank lines",
			content: "\n\nFOO=v1.0.0\n\n",
			key:     "FOO",
			want:    "v1.0.0",
		},
		{
			name:    "trims whitespace from value",
			content: "FOO= v1.0.0 \n",
			key:     "FOO",
			want:    "v1.0.0",
		},
		{
			name:    "handles key with no trailing newline",
			content: "FOO=v1.0.0",
			key:     "FOO",
			want:    "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeDotVersionFile(t, tt.content)
			got, err := ReadDotVersion(dir, tt.key)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ReadDotVersion(%q) expected error, got nil", tt.key)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadDotVersion(%q) unexpected error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("ReadDotVersion(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestReadDotVersion_MissingFile(t *testing.T) {
	_, err := ReadDotVersion("/nonexistent/dir", "FOO")
	if err == nil {
		t.Fatal("expected error for missing .versions file, got nil")
	}
}

// ---- WriteDotVersion --------------------------------------------------------

func TestWriteDotVersion(t *testing.T) {
	tests := []struct {
		name        string
		initial     string
		key         string
		value       string
		wantErr     bool
		wantContain string
		wantAbsent  string
	}{
		{
			name:        "updates existing key value",
			initial:     "FOO=v1.0.0\n",
			key:         "FOO",
			value:       "v2.0.0",
			wantContain: "FOO=v2.0.0",
			wantAbsent:  "FOO=v1.0.0",
		},
		{
			name:        "leaves other keys untouched",
			initial:     "FOO=v1.0.0\nBAR=v2.3.4\n",
			key:         "FOO",
			value:       "v9.9.9",
			wantContain: "BAR=v2.3.4",
		},
		{
			name:        "updates middle key, preserves surrounding keys",
			initial:     "AAA=v1.0.0\nBBB=v2.0.0\nCCC=v3.0.0\n",
			key:         "BBB",
			value:       "v99.0.0",
			wantContain: "BBB=v99.0.0",
		},
		{
			name:    "returns error for missing key",
			initial: "FOO=v1.0.0\n",
			key:     "NOTEXIST",
			value:   "v1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeDotVersionFile(t, tt.initial)

			err := WriteDotVersion(dir, tt.key, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("WriteDotVersion(%q) expected error, got nil", tt.key)
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteDotVersion(%q) unexpected error: %v", tt.key, err)
			}

			result := readFile(t, filepath.Join(dir, ".versions"))
			if tt.wantContain != "" && !containsString(result, tt.wantContain) {
				t.Errorf("result does not contain %q\ngot:\n%s", tt.wantContain, result)
			}
			if tt.wantAbsent != "" && containsString(result, tt.wantAbsent) {
				t.Errorf("result should not contain %q\ngot:\n%s", tt.wantAbsent, result)
			}
		})
	}
}

func TestWriteDotVersion_MissingFile(t *testing.T) {
	err := WriteDotVersion("/nonexistent/dir", "FOO", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for missing .versions file, got nil")
	}
}

// ---- Roundtrip --------------------------------------------------------------

func TestDotVersionRoundtrip(t *testing.T) {
	initial := "ALPHA=v1.0.0\nBETA=v2.0.0\nGAMMA=v3.0.0\n"
	dir := writeDotVersionFile(t, initial)

	// Update BETA.
	if err := WriteDotVersion(dir, "BETA", "v9.8.7"); err != nil {
		t.Fatalf("WriteDotVersion: %v", err)
	}

	// Read it back.
	got, err := ReadDotVersion(dir, "BETA")
	if err != nil {
		t.Fatalf("ReadDotVersion after write: %v", err)
	}
	if got != "v9.8.7" {
		t.Errorf("roundtrip: got %q, want %q", got, "v9.8.7")
	}

	// Verify other keys are unaffected.
	alpha, err := ReadDotVersion(dir, "ALPHA")
	if err != nil {
		t.Fatalf("ReadDotVersion(ALPHA): %v", err)
	}
	if alpha != "v1.0.0" {
		t.Errorf("ALPHA after update: got %q, want %q", alpha, "v1.0.0")
	}

	gamma, err := ReadDotVersion(dir, "GAMMA")
	if err != nil {
		t.Fatalf("ReadDotVersion(GAMMA): %v", err)
	}
	if gamma != "v3.0.0" {
		t.Errorf("GAMMA after update: got %q, want %q", gamma, "v3.0.0")
	}
}
