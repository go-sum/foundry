package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-sum/db"
	pgplan "github.com/pgplex/pgschema/cmd/plan"
)

// PlanDBConfig holds connection parameters for schema diffing.
// Database is the target (current-state) database pgschema diffs FROM.
// ScratchDatabase is where pgschema loads the desired-state SQL for inspection;
// if empty it defaults to Database+"_plan".
type PlanDBConfig struct {
	Host            string `yaml:"host"`
	Port            string `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	ScratchDatabase string `yaml:"scratch_database"`
	SSLMode         string `yaml:"ssl_mode"`
}

func (c PlanDBConfig) sslMode() string {
	if c.SSLMode == "" {
		return "disable"
	}
	return c.SSLMode
}

func (c PlanDBConfig) scratchDatabase() string {
	if c.ScratchDatabase != "" {
		return c.ScratchDatabase
	}
	return c.Database + "_plan"
}

// Config controls the compose operation.
type Config struct {
	Registry      *db.Registry
	MigrationsDir string
	PlanDB        PlanDBConfig
	DiffOnly      bool
	BaseSQL       string // if set and no migrations exist, written as 00001_initial_schema.sql
}

// Result holds the paths of migration files created by Generate.
type Result struct {
	InitialSchema string // bootstrap migration path, empty if not created
	Migration     string // compose-generated migration path, or diff SQL in DiffOnly mode
}

// Generate compares the desired schema (from cfg.Registry) against the plan database,
// generates Up/Down SQL via pgschema plan, and writes a goose migration file to MigrationsDir.
// If BaseSQL is set and no migrations exist, a bootstrap 00001_initial_schema.sql is created first.
func Generate(ctx context.Context, cfg Config, name string) (Result, error) {
	desiredSQL := cfg.Registry.Compose()
	if desiredSQL == "" {
		return Result{}, fmt.Errorf("compose: registry is empty — no schema providers registered")
	}

	desiredFile, err := writeTempSQL(desiredSQL)
	if err != nil {
		return Result{}, err
	}
	defer os.Remove(desiredFile) //nolint:errcheck

	upSQL, err := runPGSchemaDiff(cfg.PlanDB, desiredFile)
	if err != nil {
		return Result{}, err
	}

	if strings.TrimSpace(upSQL) == "" {
		return Result{}, nil
	}

	if cfg.DiffOnly {
		return Result{Migration: upSQL}, nil
	}

	if err := os.MkdirAll(cfg.MigrationsDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("compose: mkdir: %w", err)
	}

	var result Result

	if cfg.BaseSQL != "" {
		bootstrapPath, err := writeInitialSchema(cfg.MigrationsDir, cfg.BaseSQL)
		if err != nil {
			return Result{}, err
		}
		result.InitialSchema = bootstrapPath
	}

	downSQL := GenerateDown(upSQL)

	seq, err := nextSequenceNumber(cfg.MigrationsDir)
	if err != nil {
		return Result{}, err
	}

	fileName := fmt.Sprintf("%05d_%s.sql", seq, sanitizeName(name))
	filePath := filepath.Join(cfg.MigrationsDir, fileName)

	content := fmt.Sprintf("-- +goose Up\n%s\n-- +goose Down\n%s", AnnotateStatements(upSQL), AnnotateStatements(downSQL))
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return Result{}, fmt.Errorf("compose: write migration: %w", err)
	}

	result.Migration = filePath
	return result, nil
}

func writeInitialSchema(dir, baseSQL string) (string, error) {
	seq, err := nextSequenceNumber(dir)
	if err != nil {
		return "", err
	}
	if seq != 1 {
		return "", nil
	}

	content := fmt.Sprintf("-- +goose Up\n%s\n-- +goose Down\n%s",
		AnnotateStatements(baseSQL), GenerateDown(baseSQL))
	filePath := filepath.Join(dir, "00001_initial_schema.sql")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("compose: write initial schema: %w", err)
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

func runPGSchemaDiff(planDB PlanDBConfig, desiredFile string) (string, error) {
	port := 5432
	if planDB.Port != "" {
		var err error
		port, err = strconv.Atoi(planDB.Port)
		if err != nil {
			return "", fmt.Errorf("compose: invalid port %q: %w", planDB.Port, err)
		}
	}

	cfg := &pgplan.PlanConfig{
		Host:            planDB.Host,
		Port:            port,
		DB:              planDB.Database,
		User:            planDB.User,
		Password:        planDB.Password,
		SSLMode:         planDB.sslMode(),
		File:            desiredFile,
		Schema:          "public",
		ApplicationName: "foundry-db",
		// Always use an external scratch DB to avoid embedded postgres (fails as root).
		PlanDBHost:     planDB.Host,
		PlanDBPort:     port,
		PlanDBDatabase: planDB.scratchDatabase(),
		PlanDBUser:     planDB.User,
		PlanDBPassword: planDB.Password,
		PlanDBSSLMode:  planDB.sslMode(),
	}

	provider, err := pgplan.CreateDesiredStateProvider(cfg)
	if err != nil {
		return "", fmt.Errorf("compose: pgschema desired state: %w", err)
	}
	defer provider.Stop() //nolint:errcheck

	migrationPlan, err := pgplan.GeneratePlan(cfg, provider)
	if err != nil {
		return "", fmt.Errorf("compose: pgschema plan: %w", err)
	}

	if !migrationPlan.HasAnyChanges() {
		return "", nil
	}

	return migrationPlan.ToSQL("raw"), nil
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

// AnnotateStatements wraps SQL statements that contain $$ dollar-quoted blocks
// with goose StatementBegin/StatementEnd annotations. Goose's default parser
// splits on semicolons, which incorrectly breaks PL/pgSQL function bodies that
// contain internal semicolons (e.g. END; inside $$ ... $$).
func AnnotateStatements(sql string) string {
	if sql == "" {
		return sql
	}
	lines := strings.Split(sql, "\n")
	var result []string
	var currentStmt []string
	inDollarQuote := false

	flushStmt := func() {
		if len(currentStmt) == 0 {
			return
		}
		stmtText := strings.Join(currentStmt, "\n")
		if strings.Contains(stmtText, "$$") {
			result = append(result, "-- +goose StatementBegin")
			result = append(result, currentStmt...)
			result = append(result, "-- +goose StatementEnd")
		} else {
			result = append(result, currentStmt...)
		}
		currentStmt = nil
	}

	for _, line := range lines {
		if strings.Count(line, "$$")%2 == 1 {
			inDollarQuote = !inDollarQuote
		}

		trimmed := strings.TrimSpace(line)

		// Blank lines between statements pass through directly.
		if trimmed == "" && !inDollarQuote && len(currentStmt) == 0 {
			result = append(result, line)
			continue
		}

		currentStmt = append(currentStmt, line)

		if !inDollarQuote && strings.HasSuffix(trimmed, ";") {
			flushStmt()
		}
	}
	flushStmt()

	return strings.Join(result, "\n")
}

func sanitizeName(name string) string {
	// Strip a leading numeric sequence prefix (e.g. "00004_") to prevent double-numbering
	// when callers pass a name that already includes the sequence (e.g. "00004_add_users").
	if i := strings.IndexByte(name, '_'); i > 0 && strings.TrimLeft(name[:i], "0123456789") == "" {
		name = name[i+1:]
	}
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_")
	return strings.ToLower(replacer.Replace(name))
}
