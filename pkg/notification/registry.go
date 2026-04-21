package notification

import (
	"fmt"
	"sync"
)

// Factory constructs a Sender from a provider-specific config map.
type Factory func(cfg map[string]string) (Sender, error)

// Registry is a thread-safe provider registry. There is no global instance;
// callers construct and own their Registry.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register adds a named factory. Panics on duplicate — this is an assembly-time invariant.
func (r *Registry) Register(name string, f Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[name]; exists {
		panic(fmt.Sprintf("notification: provider %q already registered", name))
	}
	r.factories[name] = f
}

// New constructs a Sender from the named provider and config.
// Returns ErrProviderUnknown when name has not been registered.
func (r *Registry) New(name string, cfg map[string]string) (Sender, error) {
	r.mu.RLock()
	f, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrProviderUnknown, name)
	}
	return f(cfg)
}
