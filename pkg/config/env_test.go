package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandEnv_SetVar_ReturnsValue(t *testing.T) {
	t.Setenv("FOO", "bar")
	if got := ExpandEnv("FOO", "fallback"); got != "bar" {
		t.Errorf("ExpandEnv = %q, want %q", got, "bar")
	}
}

func TestExpandEnv_UnsetVar_ReturnsDefault(t *testing.T) {
	t.Setenv("NOT_SET", "")
	if got := ExpandEnv("NOT_SET", "fallback"); got != "fallback" {
		t.Errorf("ExpandEnv = %q, want %q", got, "fallback")
	}
}

func TestExpandEnv_EmptyDefault_ReturnsEmpty(t *testing.T) {
	t.Setenv("EMPTY_DEFAULT", "")
	if got := ExpandEnv("EMPTY_DEFAULT", ""); got != "" {
		t.Errorf("ExpandEnv = %q, want empty", got)
	}
}

func withSecretsDir(t *testing.T, dir string) {
	t.Helper()
	prior := secretsDir
	secretsDir = dir
	t.Cleanup(func() { secretsDir = prior })
}

func TestExpandSecret_PrefersFile(t *testing.T) {
	dir := t.TempDir()
	withSecretsDir(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "MY_SECRET"), []byte("from-file"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MY_SECRET", "from-env")

	if got := ExpandSecret("MY_SECRET"); got != "from-file" {
		t.Errorf("ExpandSecret = %q, want %q", got, "from-file")
	}
}

func TestExpandSecret_FallsBackToEnv(t *testing.T) {
	withSecretsDir(t, t.TempDir())
	t.Setenv("ENV_ONLY_SECRET", "from-env")

	if got := ExpandSecret("ENV_ONLY_SECRET"); got != "from-env" {
		t.Errorf("ExpandSecret = %q, want %q", got, "from-env")
	}
}

func TestExpandSecret_TrimsTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	withSecretsDir(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "WITH_NEWLINE"), []byte("value\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := ExpandSecret("WITH_NEWLINE"); got != "value" {
		t.Errorf("ExpandSecret = %q, want %q", got, "value")
	}
}

func TestExpandSecret_BothAbsent_ReturnsEmpty(t *testing.T) {
	withSecretsDir(t, t.TempDir())
	t.Setenv("NOT_ANYWHERE", "")

	if got := ExpandSecret("NOT_ANYWHERE"); got != "" {
		t.Errorf("ExpandSecret = %q, want empty", got)
	}
}
