package migrator

import (
	"fmt"
	"sort"
)

// autoRegistry stores migrations registered via init() functions.
type autoRegistry struct {
	migrations []registeredMigration
}

// defaultAutoRegistry is the package-level global auto-registry.
var defaultAutoRegistry = &autoRegistry{
	migrations: make([]registeredMigration, 0),
}

// AutoRegister registers a migration in the global auto-registry.
// Intended to be called from init() functions in migration files.
// Panics if the name is invalid or duplicate (fail-fast at startup).
func AutoRegister(name string, m Migration) {
	if !namePattern.MatchString(name) {
		panic(fmt.Sprintf("AutoRegister: migration name %q is invalid (expected YYYYMMDDHHMMSS_description or YYYY_MM_DD_HHMMSS_RRRR_description)", name))
	}
	for _, existing := range defaultAutoRegistry.migrations {
		if existing.Name == name {
			panic(fmt.Sprintf("AutoRegister: duplicate migration name %q", name))
		}
	}
	// Insert in sorted order using binary search.
	idx := sort.Search(len(defaultAutoRegistry.migrations), func(i int) bool {
		return defaultAutoRegistry.migrations[i].Name >= name
	})
	defaultAutoRegistry.migrations = append(defaultAutoRegistry.migrations, registeredMigration{})
	copy(defaultAutoRegistry.migrations[idx+1:], defaultAutoRegistry.migrations[idx:])
	defaultAutoRegistry.migrations[idx] = registeredMigration{Name: name, Migration: m}
}

// GetAutoRegistered returns all auto-registered migrations in timestamp order.
func GetAutoRegistered() []registeredMigration {
	result := make([]registeredMigration, len(defaultAutoRegistry.migrations))
	copy(result, defaultAutoRegistry.migrations)
	return result
}

// ResetAutoRegistry clears the global auto-registry (for testing only).
func ResetAutoRegistry() {
	defaultAutoRegistry.migrations = make([]registeredMigration, 0)
}
