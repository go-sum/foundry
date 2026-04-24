package main

import (
	"testing"
)

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
