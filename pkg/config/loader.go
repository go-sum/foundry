package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// LoadParams configures the generic config loading pipeline.
// T is the application config struct type.
type LoadParams[T any] struct {
	// Base returns the base (production) config from environment variables and secrets.
	Base func() (T, error)

	// Rules builds validation registrars from the fully-initialized config.
	// Called after Base. May be nil.
	Rules func(T) []func(*validator.Validate)
}

// Load runs the full config pipeline: base → validate.
func Load[T any](p LoadParams[T]) (*T, error) {
	cfg, err := p.Base()
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
