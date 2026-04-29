package config

import (
	"testing"
)

func TestExtractDSNComponents(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		needed     map[string]struct{}
		wantPGUser string
		wantPGPass string
		wantPGPass_present bool
	}{
		{
			name: "extracts user and password from DATABASE_URL",
			env: map[string]string{
				"DATABASE_URL": "postgres://app:secret@db:5432/foundry?sslmode=disable",
			},
			needed:             set("PGUSER", "PGPASSWORD"),
			wantPGUser:         "app",
			wantPGPass:         "secret",
			wantPGPass_present: true,
		},
		{
			name: "URL-encoded password is decoded",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:p%40ss%21word@db:5432/foundry",
			},
			needed:             set("PGUSER", "PGPASSWORD"),
			wantPGUser:         "user",
			wantPGPass:         "p@ss!word",
			wantPGPass_present: true,
		},
		{
			name: "explicit values in env take precedence over DATABASE_URL",
			env: map[string]string{
				"DATABASE_URL": "postgres://dsn-user:dsn-pass@db:5432/foundry",
				"PGUSER":       "override-user",
				"PGPASSWORD":   "override-pass",
			},
			needed:             set("PGUSER", "PGPASSWORD"),
			wantPGUser:         "override-user",
			wantPGPass:         "override-pass",
			wantPGPass_present: true,
		},
		{
			name: "PGUSER not extracted when not in needed",
			env: map[string]string{
				"DATABASE_URL": "postgres://app:secret@db:5432/foundry",
			},
			needed:             set("PGPASSWORD"),
			wantPGUser:         "",
			wantPGPass:         "secret",
			wantPGPass_present: true,
		},
		{
			name: "PGPASSWORD not extracted when DATABASE_URL has no password",
			env: map[string]string{
				"DATABASE_URL": "postgres://user@db:5432/foundry",
				"PGUSER":       "user",
			},
			needed:             set("PGUSER", "PGPASSWORD"),
			wantPGUser:         "user",
			wantPGPass:         "",
			wantPGPass_present: false,
		},
		{
			name: "no-op when DATABASE_URL is absent",
			env: map[string]string{
				"UNRELATED": "value",
			},
			needed:             set("PGUSER", "PGPASSWORD"),
			wantPGUser:         "",
			wantPGPass:         "",
			wantPGPass_present: false,
		},
		{
			name: "no-op when needed is empty",
			env: map[string]string{
				"DATABASE_URL": "postgres://app:secret@db:5432/foundry",
			},
			needed:             set(),
			wantPGUser:         "",
			wantPGPass:         "",
			wantPGPass_present: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ExtractDSNComponents(tc.env, tc.needed)

			if got := tc.env["PGUSER"]; got != tc.wantPGUser {
				t.Errorf("PGUSER: got %q, want %q", got, tc.wantPGUser)
			}
			gotPass, passPresent := tc.env["PGPASSWORD"]
			if passPresent != tc.wantPGPass_present {
				t.Errorf("PGPASSWORD present: got %v, want %v", passPresent, tc.wantPGPass_present)
			}
			if passPresent && gotPass != tc.wantPGPass {
				t.Errorf("PGPASSWORD: got %q, want %q", gotPass, tc.wantPGPass)
			}
		})
	}
}

func set(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}
