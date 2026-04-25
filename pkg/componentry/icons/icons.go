package icons

import "sync"

// Key names the semantic icons used by pkg/components.
// Applications map these keys to whichever sprite and symbol they build.
type Key string

const (
	ChevronDown  Key = "chevron-down"
	ChevronLeft  Key = "chevron-left"
	ChevronRight Key = "chevron-right"
	ChevronsUp   Key = "chevrons-up"
	Close        Key = "close"
	ThemeLight   Key = "theme-light"
	ThemeDark    Key = "theme-dark"
	ThemeSystem  Key = "theme-system"
)

// Ref points a semantic icon key at a concrete sprite symbol.
type Ref struct {
	Sprite string
	ID     string
}

// Registry resolves semantic component icon keys to concrete sprite symbols.
type Registry struct {
	mu      sync.RWMutex
	symbols map[Key]Ref
}

// NewRegistry returns an empty semantic icon registry.
func NewRegistry() *Registry {
	return &Registry{
		symbols: make(map[Key]Ref),
	}
}

// Register associates a semantic component icon key with a concrete sprite symbol.
func (r *Registry) Register(key Key, ref Ref) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.symbols[key] = ref
}

// RegisterSet adds or replaces multiple semantic icon registrations.
func (r *Registry) RegisterSet(symbols map[Key]Ref) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, ref := range symbols {
		r.symbols[key] = ref
	}
}

// Resolve returns the concrete sprite symbol for a semantic icon key.
func (r *Registry) Resolve(key Key) (Ref, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ref, ok := r.symbols[key]
	return ref, ok
}

