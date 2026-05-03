package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExpandEnv_SetVar_ReturnsValue(t *testing.T) {
	t.Setenv("FOO", "bar")
	if got := ExpandEnv("FOO", "fallback"); got != "bar" {
		t.Errorf("ExpandEnv = %q, want %q", got, "bar")
	}
}

func TestExpandEnv_UnsetVar_ReturnsDefault(t *testing.T) {
	const key = "FOUNDRY_CONFIG_TEST_UNSET_AB17Z"
	os.Unsetenv(key)
	t.Cleanup(func() { os.Unsetenv(key) })
	if got := ExpandEnv(key, "fallback"); got != "fallback" {
		t.Errorf("ExpandEnv = %q, want %q", got, "fallback")
	}
}

// An explicitly-set empty variable overrides the default — it does not fall back.
func TestExpandEnv_ExplicitEmptyOverridesDefault(t *testing.T) {
	t.Setenv("EXPLICIT_EMPTY", "")
	if got := ExpandEnv("EXPLICIT_EMPTY", "fallback"); got != "" {
		t.Errorf("ExpandEnv = %q, want empty string", got)
	}
}

func TestExpandEnv_TrimsWhitespace(t *testing.T) {
	t.Setenv("SPACED", "  value  ")
	if got := ExpandEnv("SPACED", ""); got != "value" {
		t.Errorf("ExpandEnv = %q, want %q", got, "value")
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

func TestExpandEnvBool_UnsetReturnsDefault(t *testing.T) {
	const key = "FOUNDRY_CONFIG_TEST_BOOL_UNSET_AB17Z"
	os.Unsetenv(key)
	t.Cleanup(func() { os.Unsetenv(key) })
	got, err := ExpandEnvBool(key, true)
	if err != nil || !got {
		t.Errorf("ExpandEnvBool = %v, %v; want true, nil", got, err)
	}
}

func TestExpandEnvBool_SetTrue(t *testing.T) {
	t.Setenv("BOOL_TRUE", "true")
	got, err := ExpandEnvBool("BOOL_TRUE", false)
	if err != nil || !got {
		t.Errorf("ExpandEnvBool = %v, %v; want true, nil", got, err)
	}
}

func TestExpandEnvBool_SetFalse(t *testing.T) {
	t.Setenv("BOOL_FALSE", "false")
	got, err := ExpandEnvBool("BOOL_FALSE", true)
	if err != nil || got {
		t.Errorf("ExpandEnvBool = %v, %v; want false, nil", got, err)
	}
}

func TestExpandEnvBool_InvalidReturnsError(t *testing.T) {
	t.Setenv("BOOL_INVALID", "notabool")
	_, err := ExpandEnvBool("BOOL_INVALID", false)
	if err == nil {
		t.Error("expected error for invalid bool, got nil")
	}
}

func TestExpandEnvCSV_UnsetReturnsNil(t *testing.T) {
	const key = "FOUNDRY_CONFIG_TEST_CSV_UNSET_AB17Z"
	os.Unsetenv(key)
	t.Cleanup(func() { os.Unsetenv(key) })
	if got := ExpandEnvCSV(key); got != nil {
		t.Errorf("ExpandEnvCSV = %v, want nil", got)
	}
}

func TestExpandEnvCSV_EmptyReturnsNil(t *testing.T) {
	t.Setenv("CSV_EMPTY", "")
	if got := ExpandEnvCSV("CSV_EMPTY"); got != nil {
		t.Errorf("ExpandEnvCSV = %v, want nil", got)
	}
}

func TestExpandEnvCSV_SplitsAndTrims(t *testing.T) {
	t.Setenv("CSV_VALUES", " a , b , c ")
	got := ExpandEnvCSV("CSV_VALUES")
	if !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("ExpandEnvCSV = %v, want [a b c]", got)
	}
}

func TestExpandEnvCSV_DiscardsBlankEntries(t *testing.T) {
	t.Setenv("CSV_SPARSE", "a,,b")
	got := ExpandEnvCSV("CSV_SPARSE")
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("ExpandEnvCSV = %v, want [a b]", got)
	}
}
