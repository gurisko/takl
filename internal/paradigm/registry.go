package paradigm

import (
	"context"
	"fmt"
)

// Global registry for paradigms
var registry = make(map[string]func() Paradigm)

// Register adds a paradigm constructor to the global registry
// This should be called from init() functions in paradigm packages
func Register(id string, ctor func() Paradigm) {
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("paradigm already registered: %s", id))
	}
	registry[id] = ctor
}

// List returns all registered paradigm IDs
func List() []string {
	var ids []string
	for id := range registry {
		ids = append(ids, id)
	}
	return ids
}

// Create creates a new instance of the specified paradigm
func Create(id string) (Paradigm, error) {
	ctor, exists := registry[id]
	if !exists {
		return nil, fmt.Errorf("unknown paradigm: %s", id)
	}
	return ctor(), nil
}

// Resolver interface for paradigm resolution
type Resolver interface {
	Resolve(paradigmID string) (Paradigm, error)
}

// DefaultResolver resolves paradigms using the global registry
type DefaultResolver struct {
	Deps Deps
}

// NewDefaultResolver creates a resolver with the given dependencies
func NewDefaultResolver(deps Deps) *DefaultResolver {
	return &DefaultResolver{Deps: deps}
}

// Resolve creates and initializes a paradigm instance
func (r *DefaultResolver) Resolve(paradigmID string) (Paradigm, error) {
	p, err := Create(paradigmID)
	if err != nil {
		return nil, err
	}

	if err := p.Init(context.Background(), r.Deps, map[string]any{}); err != nil {
		return nil, fmt.Errorf("failed to initialize paradigm %s: %w", paradigmID, err)
	}

	return p, nil
}

// BuiltinIDs returns the list of built-in paradigm IDs
// This will be generated or updated as paradigms are added
func BuiltinIDs() []string {
	return []string{"scrum", "kanban", "support"}
}
