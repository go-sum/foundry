package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	composeWithSecrets := `
services:
  app:
    image: app
secrets:
  DB_PASSWORD:
  APP_SECRET:
`
	composeNoSecrets := `
services:
  app:
    image: app
`
	composeWithOverlap := `
secrets:
  DB_PASSWORD:
  REDIS_PASSWORD:
`
	composeEmptySecret := `
secrets:
  EMPTY_SECRET:
`

	tests := []struct {
		name         string
		composeFiles []string // file names; content written to tmpdir
		composeData  []string // content parallel to composeFiles
		envContent   string
		wantErr      bool
		wantFiles    map[string]string // filename → expected content
	}{
		{
			name:         "happy path: secrets written with correct content",
			composeFiles: []string{"docker-compose.yml"},
			composeData:  []string{composeWithSecrets},
			envContent:   "DB_PASSWORD=supersecret\nAPP_SECRET=mytoken\n",
			wantFiles: map[string]string{
				"DB_PASSWORD": "supersecret",
				"APP_SECRET":  "mytoken",
			},
		},
		{
			name:         "missing value: secret not in .env returns error",
			composeFiles: []string{"docker-compose.yml"},
			composeData:  []string{composeWithSecrets},
			envContent:   "DB_PASSWORD=supersecret\n",
			wantErr:      true,
			wantFiles: map[string]string{
				"DB_PASSWORD": "supersecret",
			},
		},
		{
			name:         "empty value: SECRET= writes empty file",
			composeFiles: []string{"docker-compose.yml"},
			composeData:  []string{composeEmptySecret},
			envContent:   "EMPTY_SECRET=\n",
			wantFiles: map[string]string{
				"EMPTY_SECRET": "",
			},
		},
		{
			name:         "comments and blanks in .env are ignored",
			composeFiles: []string{"docker-compose.yml"},
			composeData:  []string{composeWithSecrets},
			envContent:   "# this is a comment\n\nDB_PASSWORD=pass1\n# another comment\nAPP_SECRET=token2\n",
			wantFiles: map[string]string{
				"DB_PASSWORD": "pass1",
				"APP_SECRET":  "token2",
			},
		},
		{
			name:         "duplicate secret names across compose files written once",
			composeFiles: []string{"docker-compose.data.yml", "docker-compose.yml"},
			composeData:  []string{composeWithOverlap, composeWithSecrets},
			envContent:   "DB_PASSWORD=shared\nAPP_SECRET=tok\nREDIS_PASSWORD=red\n",
			wantFiles: map[string]string{
				"DB_PASSWORD":    "shared",
				"APP_SECRET":     "tok",
				"REDIS_PASSWORD": "red",
			},
		},
		{
			name:         "compose file with no secrets key is skipped without error",
			composeFiles: []string{"docker-compose.yml"},
			composeData:  []string{composeNoSecrets},
			envContent:   "UNRELATED=value\n",
			wantFiles:    map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			outDir := filepath.Join(tmp, "secrets")

			// Write compose files.
			var composePaths []string
			for i, fname := range tc.composeFiles {
				dest := filepath.Join(tmp, fname)
				if err := os.WriteFile(dest, []byte(tc.composeData[i]), 0600); err != nil {
					t.Fatalf("write compose file: %v", err)
				}
				composePaths = append(composePaths, dest)
			}

			// Write .env file.
			envPath := filepath.Join(tmp, ".env")
			if err := os.WriteFile(envPath, []byte(tc.envContent), 0600); err != nil {
				t.Fatalf("write .env: %v", err)
			}

			err := run(composePaths, envPath, outDir)

			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify expected files exist with correct content and permissions.
			for name, wantContent := range tc.wantFiles {
				path := filepath.Join(outDir, name)
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Errorf("file %s not found: %v", name, readErr)
					continue
				}
				if string(data) != wantContent {
					t.Errorf("file %s: got %q, want %q", name, string(data), wantContent)
				}
				info, statErr := os.Stat(path)
				if statErr != nil {
					t.Errorf("stat %s: %v", name, statErr)
					continue
				}
				if perm := info.Mode().Perm(); perm != 0600 {
					t.Errorf("file %s permissions: got %04o, want 0600", name, perm)
				}
			}
		})
	}
}
