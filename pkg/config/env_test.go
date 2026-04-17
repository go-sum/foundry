package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentEnv_Unset_ReturnsProduction(t *testing.T) {
	t.Setenv("APP_ENV", "")
	if got := CurrentEnv(); got != Production {
		t.Errorf("CurrentEnv() = %q, want %q", got, Production)
	}
}

func TestCurrentEnv_SetToDevelopment_ReturnsDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	if got := CurrentEnv(); got != Development {
		t.Errorf("CurrentEnv() = %q, want %q", got, Development)
	}
}

func TestCurrentEnv_SetToTesting_ReturnsTesting(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	if got := CurrentEnv(); got != Testing {
		t.Errorf("CurrentEnv() = %q, want %q", got, Testing)
	}
}

func TestCurrentEnv_SetToProduction_ReturnsProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if got := CurrentEnv(); got != Production {
		t.Errorf("CurrentEnv() = %q, want %q", got, Production)
	}
}

func TestCurrentEnv_SetToUnknown_ReturnsProduction(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	if got := CurrentEnv(); got != Production {
		t.Errorf("CurrentEnv() = %q, want %q (fail-safe)", got, Production)
	}
}

func TestExpandEnv_PlainString_Unchanged(t *testing.T) {
	if got := ExpandEnv("no vars here"); got != "no vars here" {
		t.Errorf("ExpandEnv = %q, want unchanged", got)
	}
}

func TestExpandEnv_SimpleVar_Replaced(t *testing.T) {
	t.Setenv("FOO", "bar")
	if got := ExpandEnv("prefix ${FOO} suffix"); got != "prefix bar suffix" {
		t.Errorf("ExpandEnv = %q, want %q", got, "prefix bar suffix")
	}
}

func TestExpandEnv_VarWithDefault_UsesDefaultWhenUnset(t *testing.T) {
	t.Setenv("NOT_SET", "")
	if got := ExpandEnv("x=${NOT_SET:-fallback}"); got != "x=fallback" {
		t.Errorf("ExpandEnv = %q, want %q", got, "x=fallback")
	}
}

func TestExpandEnv_VarWithDefault_UsesEnvWhenSet(t *testing.T) {
	t.Setenv("IS_SET", "actual")
	if got := ExpandEnv("x=${IS_SET:-fallback}"); got != "x=actual" {
		t.Errorf("ExpandEnv = %q, want %q", got, "x=actual")
	}
}

func TestExpandEnv_MissingVarNoDefault_ReplacedWithEmpty(t *testing.T) {
	t.Setenv("NEVER_SET", "")
	if got := ExpandEnv("x=${NEVER_SET}y"); got != "x=y" {
		t.Errorf("ExpandEnv = %q, want %q", got, "x=y")
	}
}

func TestExpandEnv_MultipleVars_AllReplaced(t *testing.T) {
	t.Setenv("A", "1")
	t.Setenv("B", "2")
	if got := ExpandEnv("${A}-${B}"); got != "1-2" {
		t.Errorf("ExpandEnv = %q, want %q", got, "1-2")
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
