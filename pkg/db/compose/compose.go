package compose

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-sum/db"
)

// PlanDBConfig holds connection parameters for the ephemeral plan database used for schema diffing.
type PlanDBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func (c PlanDBConfig) dsn() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, sslMode)
}

// Config controls the compose operation.
type Config struct {
	Registry      *db.Registry
	MigrationsDir string
	PlanDB        PlanDBConfig
	DiffOnly      bool
}

// Generate compares the desired schema (from cfg.Registry) against the plan database,
// generates Up/Down SQL via pgschema diff, and writes a goose migration file to MigrationsDir.
// Returns the file path of the created migration, or "" when DiffOnly is true.
func Generate(ctx context.Context, cfg Config, name string) (string, error) {
	desiredSQL := cfg.Registry.Compose()
	if desiredSQL == "" {
		return "", fmt.Errorf("compose: registry is empty — no schema providers registered")
	}

	if _, err := exec.LookPath("pgschema"); err != nil {
		return "", fmt.Errorf("compose: pgschema binary not found in PATH — install it to use schema diffing (https://github.com/pgschema/pgschema)")
	}

	desiredFile, err := writeTempSQL(desiredSQL)
	if err != nil {
		return "", err
	}
	defer os.Remove(desiredFile) //nolint:errcheck

	upSQL, err := runPGSchemaDiff(ctx, cfg.PlanDB, desiredFile)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(upSQL) == "" {
		return "", nil
	}

	if cfg.DiffOnly {
		fmt.Print(upSQL)
		return "", nil
	}

	downSQL := GenerateDown(upSQL)

	seq, err := nextSequenceNumber(cfg.MigrationsDir)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%05d_%s.sql", seq, sanitizeName(name))
	filePath := filepath.Join(cfg.MigrationsDir, fileName)

	if err := os.MkdirAll(cfg.MigrationsDir, 0o755); err != nil {
		return "", fmt.Errorf("compose: mkdir: %w", err)
	}

	content := fmt.Sprintf("-- +goose Up\n%s\n-- +goose Down\n%s", upSQL, downSQL)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("compose: write migration: %w", err)
	}

	return filePath, nil
}

func writeTempSQL(sql string) (_ string, err error) {
	f, err := os.CreateTemp("", "foundry-schema-*.sql")
	if err != nil {
		return "", fmt.Errorf("compose: create temp file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	if _, err := f.WriteString(sql); err != nil {
		return "", fmt.Errorf("compose: write temp file: %w", err)
	}

	return f.Name(), nil
}

func runPGSchemaDiff(ctx context.Context, planDB PlanDBConfig, desiredFile string) (string, error) {
	args := []string{
		"diff",
		"--source", planDB.dsn(),
		"--target", desiredFile,
	}

	out, err := exec.CommandContext(ctx, "pgschema", args...).Output()
	if err != nil {
		return "", fmt.Errorf("compose: pgschema diff: %w", err)
	}

	return string(out), nil
}

func nextSequenceNumber(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("compose: read migrations dir: %w", err)
	}

	var nums []int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(e.Name(), "%d", &n); err == nil {
			nums = append(nums, n)
		}
	}

	if len(nums) == 0 {
		return 1, nil
	}

	sort.Ints(nums)
	return nums[len(nums)-1] + 1, nil
}

func sanitizeName(name string) string {
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_")
	return strings.ToLower(replacer.Replace(name))
}
