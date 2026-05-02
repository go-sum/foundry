package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// EnvOverlay pairs an environment name with a function that mutates the config for that environment.
type EnvOverlay[T any] struct {
	Name  string
	Apply func(*T)
}

// LoadParams configures the generic config loading pipeline.
// T is the application config struct type.
type LoadParams[T any] struct {
	// Base returns the base (production) config from environment variables and secrets.
	Base func() (T, error)

	// Env is the resolved environment name (e.g. from ExpandEnv).
	Env string

	// Overlays pairs environment names with mutation functions. Unknown environments pass through unchanged.
	Overlays []EnvOverlay[T]

	// SetEnv stores the resolved environment name on the config struct. May be nil.
	SetEnv func(*T, string)

	// Rules builds validation registrars from the fully-initialized config.
	// Called after overlay and SetEnv. Ignored by LoadEnv. May be nil.
	Rules func(T) []func(*validator.Validate)
}

// LoadEnv runs the config pipeline without validation: base → overlay → set env.
func LoadEnv[T any](p LoadParams[T]) (T, error) {
	cfg, err := p.Base()
	if err != nil {
		var zero T
		return zero, err
	}
	for _, o := range p.Overlays {
		if o.Name == p.Env {
			o.Apply(&cfg)
			break
		}
	}
	if p.SetEnv != nil {
		p.SetEnv(&cfg, p.Env)
	}
	return cfg, nil
}

// Load runs the full config pipeline: base → overlay → set env → validate.
func Load[T any](p LoadParams[T]) (*T, error) {
	cfg, err := LoadEnv(p)
	if err != nil {
		return nil, err
	}
	var registrars []func(*validator.Validate)
	if p.Rules != nil {
		registrars = p.Rules(cfg)
	}
	if err := Validate(cfg, registrars...); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}
