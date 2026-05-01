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

func TestParseKVURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantAddr   string
		wantPass   string
		wantTLS    bool
		wantErrStr string
	}{
		{
			name:     "parses redis URL with password",
			raw:      "redis://:secret@kv:6379",
			wantAddr: "kv:6379",
			wantPass: "secret",
			wantTLS:  false,
		},
		{
			name:     "rediss scheme sets TLSEnabled",
			raw:      "rediss://:tlspass@kv:6380",
			wantAddr: "kv:6380",
			wantPass: "tlspass",
			wantTLS:  true,
		},
		{
			name:     "URL-encoded password is decoded",
			raw:      "redis://:p%40ss%21word@kv:6379",
			wantAddr: "kv:6379",
			wantPass: "p@ss!word",
			wantTLS:  false,
		},
		{
			name:     "URL with no password",
			raw:      "redis://kv:6379",
			wantAddr: "kv:6379",
			wantPass: "",
			wantTLS:  false,
		},
		{
			name:     "empty raw returns default addr",
			raw:      "",
			wantAddr: "localhost:6379",
			wantPass: "",
			wantTLS:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseKVURL(tc.raw)
			if tc.wantErrStr != "" {
				if err == nil || err.Error() != tc.wantErrStr {
					t.Fatalf("ParseKVURL(%q) error = %v, want %q", tc.raw, err, tc.wantErrStr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseKVURL(%q) unexpected error: %v", tc.raw, err)
			}
			if got.Addr != tc.wantAddr {
				t.Errorf("Addr = %q, want %q", got.Addr, tc.wantAddr)
			}
			if got.Password != tc.wantPass {
				t.Errorf("Password = %q, want %q", got.Password, tc.wantPass)
			}
			if got.TLSEnabled != tc.wantTLS {
				t.Errorf("TLSEnabled = %v, want %v", got.TLSEnabled, tc.wantTLS)
			}
		})
	}
}

func TestExtractKVComponents(t *testing.T) {
	tests := []struct {
		name              string
		env               map[string]string
		needed            map[string]struct{}
		wantPass          string
		wantPass_present  bool
	}{
		{
			name:             "extracts password from KV_URL",
			env:              map[string]string{"KV_URL": "redis://:secret@kv:6379"},
			needed:           set("KV_PASSWORD"),
			wantPass:         "secret",
			wantPass_present: true,
		},
		{
			name:             "URL-encoded password is decoded",
			env:              map[string]string{"KV_URL": "redis://:p%40ss%21word@kv:6379"},
			needed:           set("KV_PASSWORD"),
			wantPass:         "p@ss!word",
			wantPass_present: true,
		},
		{
			name: "explicit KV_PASSWORD takes precedence over KV_URL",
			env: map[string]string{
				"KV_URL":      "redis://:url-pass@kv:6379",
				"KV_PASSWORD": "explicit-pass",
			},
			needed:           set("KV_PASSWORD"),
			wantPass:         "explicit-pass",
			wantPass_present: true,
		},
		{
			name:             "not extracted when KV_PASSWORD not in needed",
			env:              map[string]string{"KV_URL": "redis://:secret@kv:6379"},
			needed:           set(),
			wantPass:         "",
			wantPass_present: false,
		},
		{
			name:             "not extracted when KV_URL has no password",
			env:              map[string]string{"KV_URL": "redis://kv:6379"},
			needed:           set("KV_PASSWORD"),
			wantPass:         "",
			wantPass_present: false,
		},
		{
			name:             "no-op when KV_URL absent",
			env:              map[string]string{"UNRELATED": "value"},
			needed:           set("KV_PASSWORD"),
			wantPass:         "",
			wantPass_present: false,
		},
		{
			name:             "no-op when needed is empty",
			env:              map[string]string{"KV_URL": "redis://:secret@kv:6379"},
			needed:           set(),
			wantPass:         "",
			wantPass_present: false,
		},
		{
			name:             "rediss scheme accepted",
			env:              map[string]string{"KV_URL": "rediss://:tlspass@kv:6380"},
			needed:           set("KV_PASSWORD"),
			wantPass:         "tlspass",
			wantPass_present: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ExtractKVComponents(tc.env, tc.needed)

			gotPass, passPresent := tc.env["KV_PASSWORD"]
			if passPresent != tc.wantPass_present {
				t.Errorf("KV_PASSWORD present: got %v, want %v", passPresent, tc.wantPass_present)
			}
			if passPresent && gotPass != tc.wantPass {
				t.Errorf("KV_PASSWORD: got %q, want %q", gotPass, tc.wantPass)
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
