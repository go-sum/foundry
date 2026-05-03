package config

import (
	"errors"
	"fmt"
)

// Closer collects named cleanup functions and runs them in LIFO order on Close.
type Closer struct {
	closers []namedCloser
}

type namedCloser struct {
	name string
	fn   func() error
}

// Add registers a cleanup function to be called on Close.
// Functions are called in reverse registration order (LIFO).
func (c *Closer) Add(name string, fn func() error) {
	c.closers = append(c.closers, namedCloser{name: name, fn: fn})
}

// Close calls all registered cleanup functions in reverse order.
// All functions are called even if earlier ones fail.
// Returns a joined error containing all individual failures.
func (c *Closer) Close() error {
	var errs []error
	for i := len(c.closers) - 1; i >= 0; i-- {
		nc := c.closers[i]
		if err := nc.fn(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", nc.name, err))
		}
	}
	return errors.Join(errs...)
}
