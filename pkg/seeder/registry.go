package seeder

import (
	"fmt"
	"strings"
)

// Registry stores registered seeders by name.
type Registry struct {
	seeders map[string]Seeder
}

// NewRegistry creates an empty seeder registry.
func NewRegistry() *Registry {
	return &Registry{
		seeders: make(map[string]Seeder),
	}
}

// Register adds a seeder with the given name to the registry.
// Names must be non-empty and unique.
func (r *Registry) Register(name string, s Seeder) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("seeder name %q: %w", name, ErrInvalidSeederName)
	}

	if _, exists := r.seeders[name]; exists {
		return fmt.Errorf("seeder name %q: %w", name, ErrDuplicateSeeder)
	}

	r.seeders[name] = s
	return nil
}

// Get retrieves a seeder by name. Returns ErrSeederNotFound if not found.
func (r *Registry) Get(name string) (Seeder, error) {
	s, ok := r.seeders[name]
	if !ok {
		return nil, fmt.Errorf("seeder name %q: %w", name, ErrSeederNotFound)
	}
	return s, nil
}

// GetAll returns a copy of all registered seeders keyed by name.
func (r *Registry) GetAll() map[string]Seeder {
	result := make(map[string]Seeder, len(r.seeders))
	for k, v := range r.seeders {
		result[k] = v
	}
	return result
}
