package publish

import (
	"maps"
	"path"
	"strings"
	"sync"
)

// Registry maps sprite names to their public file paths.
type Registry struct {
	mu          sync.RWMutex
	spriteFiles map[string]string
	resolvePath func(string) string
}

// NewRegistry creates a new Registry with the default path resolver.
func NewRegistry() *Registry {
	return &Registry{
		spriteFiles: make(map[string]string),
		resolvePath: PublicPath,
	}
}

// RegisterSprite registers a single sprite name to its relative path.
func (r *Registry) RegisterSprite(name, relPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.spriteFiles[name] = relPath
}

// RegisterSprites registers multiple sprite name→path pairs.
func (r *Registry) RegisterSprites(sprites map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	maps.Copy(r.spriteFiles, sprites)
}

// SetPathFunc replaces the path resolution function.
func (r *Registry) SetPathFunc(fn func(string) string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolvePath = fn
}

// SpritePath returns the resolved public path for the named sprite.
// Returns an empty string if the name is not registered.
func (r *Registry) SpritePath(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	relPath, ok := r.spriteFiles[name]
	if !ok {
		return ""
	}
	return r.resolvePath(relPath)
}

// PublicPath returns the /public/<rel> URL for a relative path.
func PublicPath(rel string) string {
	rel = strings.TrimPrefix(rel, "/")
	return path.Join("/public", rel)
}
