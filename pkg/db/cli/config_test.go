package dbcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/db"
)

func boolPtr(b bool) *bool { return &b }

// --- expandEnvWithDefaults ---

func TestExpandEnvWithDefaults_PlainVar(t *testing.T) {
	t.Setenv("MY_VAR", "hello")
	got := expandEnvWithDefaults("${MY_VAR}")
	if got != "hello" {
		t.Fatalf("expandEnvWithDefaults(%q) = %q, want %q", "${MY_VAR}", got, "hello")
	}
}

func TestExpandEnvWithDefaults_DefaultSyntax_EnvSet(t *testing.T) {
	t.Setenv("MY_VAR", "from_env")
	got := expandEnvWithDefaults("${MY_VAR:-fallback}")
	if got != "from_env" {
		t.Fatalf("expandEnvWithDefaults with env set = %q, want %q", got, "from_env")
	}
}

func TestExpandEnvWithDefaults_DefaultSyntax_EnvUnset(t *testing.T) {
	t.Setenv("MY_UNSET_VAR", "")
	// Unset via LookupEnv — use a var name that is definitely not set.
	got := expandEnvWithDefaults("${FOUNDRY_TEST_NEVER_SET_XYZ:-my_default}")
	if got != "my_default" {
		t.Fatalf("expandEnvWithDefaults with unset var = %q, want %q", got, "my_default")
	}
}

func TestExpandEnvWithDefaults_DefaultSyntax_EnvEmpty(t *testing.T) {
	t.Setenv("MY_EMPTY_VAR", "")
	got := expandEnvWithDefaults("${MY_EMPTY_VAR:-used_default}")
	if got != "used_default" {
		t.Fatalf("expandEnvWithDefaults with empty var = %q, want %q", got, "used_default")
	}
}

func TestExpandEnvWithDefaults_MultipleVars(t *testing.T) {
	t.Setenv("PGUSER", "admin")
	t.Setenv("PGPASSWORD", "secret")
	input := "user=${PGUSER:-postgres} pass=${PGPASSWORD:-changeme}"
	got := expandEnvWithDefaults(input)
	want := "user=admin pass=secret"
	if got != want {
		t.Fatalf("expandEnvWithDefaults multiple vars = %q, want %q", got, want)
	}
}

func TestExpandEnvWithDefaults_MultipleVars_MixedSetAndDefault(t *testing.T) {
	t.Setenv("PGUSER", "myuser")
	// PGPASSWORD deliberately not set — use a unique name to avoid leaking real env.
	input := "user=${PGUSER:-postgres} pass=${FOUNDRY_TEST_PW_UNSET:-changeme}"
	got := expandEnvWithDefaults(input)
	want := "user=myuser pass=changeme"
	if got != want {
		t.Fatalf("expandEnvWithDefaults mixed = %q, want %q", got, want)
	}
}

func TestExpandEnvWithDefaults_CredentialVarNames(t *testing.T) {
	// Regression: schema.yaml uses PGUSER / PGPASSWORD, not POSTGRES_USER.
	// Verify that PGUSER expands correctly and is NOT confused with POSTGRES_USER.
	t.Setenv("PGUSER", "pguser_val")
	t.Setenv("POSTGRES_USER", "postgres_user_val")

	got := expandEnvWithDefaults("${PGUSER:-postgres}")
	if got != "pguser_val" {
		t.Fatalf("PGUSER expansion = %q, want %q", got, "pguser_val")
	}

	got2 := expandEnvWithDefaults("${POSTGRES_USER:-postgres}")
	if got2 != "postgres_user_val" {
		t.Fatalf("POSTGRES_USER expansion = %q, want %q", got2, "postgres_user_val")
	}
}

// --- shouldScaffold ---

func TestShouldScaffold_NilIsTrue(t *testing.T) {
	e := schemaEntry{}
	if !e.shouldScaffold() {
		t.Fatal("shouldScaffold() = false for nil Scaffold field, want true")
	}
}

func TestShouldScaffold_ExplicitFalse(t *testing.T) {
	e := schemaEntry{Scaffold: boolPtr(false)}
	if e.shouldScaffold() {
		t.Fatal("shouldScaffold() = true for Scaffold=false, want false")
	}
}

func TestShouldScaffold_ExplicitTrue(t *testing.T) {
	e := schemaEntry{Scaffold: boolPtr(true)}
	if !e.shouldScaffold() {
		t.Fatal("shouldScaffold() = false for Scaffold=true, want true")
	}
}

// --- resolveSQL ---

// writeTempSQL creates a temp file with the given SQL content and returns the path.
func writeTempSQL(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "schema-*.sql")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

// TestResolveSQL_FilesystemFirst verifies that when both a source file and a
// resolver entry exist, the file content is returned (filesystem-first).
func TestResolveSQL_FilesystemFirst(t *testing.T) {
	const fileSQL = "CREATE TABLE from_file (id int)"
	const resolverSQL = "CREATE TABLE from_resolver (id int)"

	path := writeTempSQL(t, fileSQL)
	name := strings.TrimSuffix(filepath.Base(path), ".sql")

	cfg := &dbConfig{
		resolver: db.SchemaResolver{name: resolverSQL},
	}
	entry := schemaEntry{Source: path}

	got, err := cfg.resolveSQL(entry)
	if err != nil {
		t.Fatalf("resolveSQL returned unexpected error: %v", err)
	}
	if got != fileSQL {
		t.Fatalf("resolveSQL = %q, want file content %q", got, fileSQL)
	}
}

// TestResolveSQL_ResolverFallback verifies that when the source file does not
// exist but the resolver has a matching key, the resolver content is returned.
func TestResolveSQL_ResolverFallback(t *testing.T) {
	const resolverSQL = "CREATE TABLE from_resolver (id int)"

	nonExistent := filepath.Join(t.TempDir(), "missing.sql")
	cfg := &dbConfig{
		resolver: db.SchemaResolver{"missing": resolverSQL},
	}
	entry := schemaEntry{Source: nonExistent}

	got, err := cfg.resolveSQL(entry)
	if err != nil {
		t.Fatalf("resolveSQL returned unexpected error: %v", err)
	}
	if got != resolverSQL {
		t.Fatalf("resolveSQL = %q, want resolver content %q", got, resolverSQL)
	}
}

// TestResolveSQL_NoResolverNoFile_Error verifies that when the source file does
// not exist and there is no resolver, an error is returned.
func TestResolveSQL_NoResolverNoFile_Error(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "missing.sql")
	cfg := &dbConfig{}
	entry := schemaEntry{Source: nonExistent}

	_, err := cfg.resolveSQL(entry)
	if err == nil {
		t.Fatal("resolveSQL returned nil error; want error for missing file with no resolver")
	}
}
