package migrator

import (
	"fmt"
	"regexp"
	"sort"
)

// namePattern validates migration names in both legacy and Laravel-style formats:
//   - Legacy:  "20240215120000_create_users_table"  (14-digit timestamp + description)
//   - New:     "2024_02_15_120405_4827_create_users_table"  (YYYY_MM_DD_HHMMSS_RRRR + description)
var namePattern = regexp.MustCompile(`^(\d{14}|\d{4}_\d{2}_\d{2}_\d{6}_\d{4})_[a-z][a-z0-9_]*$`)

// registeredMigration pairs a migration name with its implementation.
type registeredMigration struct {
	Name      string
	Migration Migration
}

// Registry stores migrations in timestamp-sorted order.
type Registry struct {
	migrations []registeredMigration
}

// NewRegistry creates an empty migration registry.
func NewRegistry() *Registry {
	return &Registry{
		migrations: make([]registeredMigration, 0),
	}
}

// Register adds a migration with the given name to the registry.
// Names must match YYYYMMDDHHMMSS_description or YYYY_MM_DD_HHMMSS_RRRR_description and be unique.
func (r *Registry) Register(name string, m Migration) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("migration name %q: %w", name, ErrInvalidMigrationName)
	}

	// Check for duplicates.
	for _, existing := range r.migrations {
		if existing.Name == name {
			return fmt.Errorf("migration name %q: %w", name, ErrDuplicateMigration)
		}
	}

	// Insert in sorted order using binary search.
	idx := sort.Search(len(r.migrations), func(i int) bool {
		return r.migrations[i].Name >= name
	})

	r.migrations = append(r.migrations, registeredMigration{})
	copy(r.migrations[idx+1:], r.migrations[idx:])
	r.migrations[idx] = registeredMigration{Name: name, Migration: m}

	return nil
}

// GetAll returns all registered migrations in timestamp-sorted order.
func (r *Registry) GetAll() []registeredMigration {
	result := make([]registeredMigration, len(r.migrations))
	copy(result, r.migrations)
	return result
}

// Get retrieves a migration by name. Returns ErrMigrationNotFound if not found.
func (r *Registry) Get(name string) (Migration, error) {
	idx := sort.Search(len(r.migrations), func(i int) bool {
		return r.migrations[i].Name >= name
	})

	if idx < len(r.migrations) && r.migrations[idx].Name == name {
		return r.migrations[idx].Migration, nil
	}

	return nil, fmt.Errorf("migration name %q: %w", name, ErrMigrationNotFound)
}

// Count returns the number of registered migrations.
func (r *Registry) Count() int {
	return len(r.migrations)
}
