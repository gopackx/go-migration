package migrator

import (
	"errors"
	"sort"
	"testing"

	"pgregory.net/rapid"
	"strings"
)

// Feature: auto-discovery-migrations, Property 4: AutoDiscover loads all migrations
// For any set of migrations registered via AutoRegister, calling AutoDiscover() on a fresh
// Migrator SHALL result in the internal registry containing exactly the same set of
// migrations in the same order.
// **Validates: Requirements 2.1, 2.2**
func TestAutoDiscoverProperty4_LoadsAllMigrations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		defer ResetAutoRegistry()

		names := uniqueValidNamesGen().Draw(t, "names")

		// Auto-register all migrations
		for _, name := range names {
			AutoRegister(name, &stubMigration{})
		}

		// Create a fresh Migrator with nil db (no DB needed for registry-only test)
		m := New(nil)

		// Call AutoDiscover
		err := m.AutoDiscover()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify internal registry has the same migrations in the same order
		registryMigrations := m.registry.GetAll()
		if len(registryMigrations) != len(names) {
			t.Errorf("len = %d, want %d", len(registryMigrations), len(names))
		}

		// Expected order: sorted lexicographically (same as auto-registry order)
		sorted := make([]string, len(names))
		copy(sorted, names)
		sort.Strings(sorted)

		for i, rm := range registryMigrations {
			if sorted[i] != rm.Name {
				t.Errorf("migration at position %d should be %q, got %q", i, sorted[i], rm.Name)
			}
		}
	})
}

// Feature: auto-discovery-migrations, Property 5: AutoDiscover errors on conflict with manual registration
// For any valid migration name that is both auto-registered and manually registered via Register(),
// calling AutoDiscover() SHALL return an error wrapping ErrDuplicateMigration.
// **Validates: Requirements 2.3, 4.3**
func TestAutoDiscoverProperty5_ErrorsOnConflictWithManualRegistration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		defer ResetAutoRegistry()

		name := validMigrationNameGen().Draw(t, "name")

		// Register manually first
		m := New(nil)
		err := m.Register(name, &stubMigration{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Also auto-register the same name
		AutoRegister(name, &stubMigration{})

		// AutoDiscover should return an error wrapping ErrDuplicateMigration
		err = m.AutoDiscover()
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !errors.Is(err, ErrDuplicateMigration) {
			t.Errorf("error should wrap ErrDuplicateMigration, got: %v", err)
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("%q not in %q", name, err.Error())
		}
	})
}

// Feature: auto-discovery-migrations, Property 6: Combined auto and manual registration preserves order
// For any set of auto-registered migrations and any set of manually registered migrations
// (with no name overlap), after calling both AutoDiscover() and Register(), the Migrator's
// internal registry SHALL contain all migrations from both sources in correct timestamp-sorted order.
// **Validates: Requirements 4.1, 4.2**
func TestAutoDiscoverProperty6_CombinedAutoAndManualRegistrationPreservesOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		defer ResetAutoRegistry()

		// Generate a larger set of unique valid names, then split into two disjoint sets.
		// Use a combined pool of at least 2 names so both sets are non-empty.
		poolSize := rapid.IntRange(2, 20).Draw(t, "poolSize")
		seen := make(map[string]bool)
		pool := make([]string, 0, poolSize)
		for len(pool) < poolSize {
			name := validMigrationNameGen().Draw(t, "name")
			if !seen[name] {
				seen[name] = true
				pool = append(pool, name)
			}
		}

		// Split: at least 1 in each set
		splitIdx := rapid.IntRange(1, poolSize-1).Draw(t, "splitIdx")
		autoNames := pool[:splitIdx]
		manualNames := pool[splitIdx:]

		// Auto-register the first set
		for _, name := range autoNames {
			AutoRegister(name, &stubMigration{})
		}

		// Create a fresh Migrator
		m := New(nil)

		// Manually register the second set
		for _, name := range manualNames {
			err := m.Register(name, &stubMigration{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		// Call AutoDiscover to load auto-registered migrations
		err := m.AutoDiscover()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify internal registry contains ALL migrations from both sets
		registryMigrations := m.registry.GetAll()
		if len(registryMigrations) != len(pool) {
			t.Errorf("len = %d, want %d", len(registryMigrations), len(pool))
		}

		// Verify correct timestamp-sorted order
		sorted := make([]string, len(pool))
		copy(sorted, pool)
		sort.Strings(sorted)

		for i, rm := range registryMigrations {
			if sorted[i] != rm.Name {
				t.Errorf("migration at position %d should be %q, got %q", i, sorted[i], rm.Name)
			}
		}
	})
}

// Feature: auto-discovery-migrations, Property 7: WithAutoDiscover option loads migrations
// For any set of migrations registered via AutoRegister, creating a Migrator with
// WithAutoDiscover() option SHALL result in the internal registry containing all
// auto-registered migrations.
// **Validates: Requirements 6.2**
func TestAutoDiscoverProperty7_WithAutoDiscoverOptionLoadsMigrations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		defer ResetAutoRegistry()

		names := uniqueValidNamesGen().Draw(t, "names")

		// Auto-register all migrations
		for _, name := range names {
			AutoRegister(name, &stubMigration{})
		}

		// Create Migrator with WithAutoDiscover() option
		m := New(nil, WithAutoDiscover())

		// Verify internal registry contains all auto-registered migrations
		registryMigrations := m.registry.GetAll()
		if len(registryMigrations) != len(names) {
			t.Errorf("len = %d, want %d", len(registryMigrations), len(names))
		}

		// Expected order: sorted lexicographically
		sorted := make([]string, len(names))
		copy(sorted, names)
		sort.Strings(sorted)

		for i, rm := range registryMigrations {
			if sorted[i] != rm.Name {
				t.Errorf("migration at position %d should be %q, got %q", i, sorted[i], rm.Name)
			}
		}
	})
}
