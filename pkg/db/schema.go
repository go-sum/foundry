package db

import "sort"

// SchemaProvider declares a package's desired-state DDL.
type SchemaProvider interface {
	Name() string
	SQL() string
	Priority() int
}

// Registry collects SchemaProviders and composes them into a single SQL string.
type Registry struct {
	providers []SchemaProvider
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds one or more providers to the registry.
func (r *Registry) Register(providers ...SchemaProvider) {
	r.providers = append(r.providers, providers...)
}

// Providers returns a priority-sorted copy of all registered providers.
func (r *Registry) Providers() []SchemaProvider {
	sorted := make([]SchemaProvider, len(r.providers))
	copy(sorted, r.providers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})
	return sorted
}

// Compose returns all provider SQL concatenated in priority order, separated by double newlines.
func (r *Registry) Compose() string {
	providers := r.Providers()
	if len(providers) == 0 {
		return ""
	}

	result := providers[0].SQL()
	for _, p := range providers[1:] {
		result += "\n\n" + p.SQL()
	}
	return result
}
