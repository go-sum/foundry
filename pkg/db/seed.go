package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Environment identifies a deployment tier for seed data filtering.
type Environment string

const (
	EnvDev  Environment = "dev"
	EnvTest Environment = "test"
	EnvProd Environment = "prod"
)

// Seeder provides seed data for one or more environments.
type Seeder interface {
	Name() string
	Environments() []Environment
	Seed(ctx context.Context, tx pgx.Tx) error
}

// SeedRegistry collects Seeders and runs them filtered by environment.
type SeedRegistry struct {
	seeders []Seeder
}

// NewSeedRegistry returns an empty SeedRegistry.
func NewSeedRegistry() *SeedRegistry {
	return &SeedRegistry{}
}

// Register adds one or more seeders to the registry.
func (r *SeedRegistry) Register(seeders ...Seeder) {
	r.seeders = append(r.seeders, seeders...)
}

// Run executes all seeders that include env in their Environments list.
// Each seeder runs in its own transaction; failures roll back only that seeder.
// All errors are collected and returned as a combined error.
func (r *SeedRegistry) Run(ctx context.Context, pool *pgxpool.Pool, env Environment) error {
	var errs []error

	for _, s := range r.seeders {
		if !matchesEnv(s.Environments(), env) {
			continue
		}

		if err := runSeeder(ctx, pool, s); err != nil {
			errs = append(errs, fmt.Errorf("seeder %q: %w", s.Name(), err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	combined := errs[0]
	for _, e := range errs[1:] {
		combined = fmt.Errorf("%w; %w", combined, e)
	}
	return combined
}

func matchesEnv(envs []Environment, target Environment) bool {
	for _, e := range envs {
		if e == target {
			return true
		}
	}
	return false
}

func runSeeder(ctx context.Context, pool *pgxpool.Pool, s Seeder) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := s.Seed(ctx, tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
