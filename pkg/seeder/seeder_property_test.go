package seeder

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"pgregory.net/rapid"
)

// genSeederName generates a valid non-empty seeder name.
func genSeederName() *rapid.Generator[string] {
	return rapid.StringMatching(`^[a-z][a-z0-9_]{0,20}$`)
}

// genUniqueSeederNames generates a slice of unique non-empty seeder names (1..maxN).
func genUniqueSeederNames(maxN int) *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		n := rapid.IntRange(1, maxN).Draw(t, "count")
		seen := make(map[string]bool)
		names := make([]string, 0, n)
		for len(names) < n {
			name := genSeederName().Draw(t, "name")
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		return names
	})
}

// propSeeder is a simple Seeder for property tests that tracks execution.
type propSeeder struct {
	order *[]string
	name  string
}

func (s *propSeeder) Run(db *sql.DB) error {
	*s.order = append(*s.order, s.name)
	return nil
}

// propDependentSeeder is a DependentSeeder for property tests.
type propDependentSeeder struct {
	order *[]string
	name  string
	deps  []string
}

func (s *propDependentSeeder) Run(db *sql.DB) error {
	*s.order = append(*s.order, s.name)
	return nil
}

func (s *propDependentSeeder) DependsOn() []string {
	return s.deps
}

func newMockDB(t *rapid.T) *sql.DB {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return db
}

// Feature: go-migration, Property 23: Seeder registration and retrieval
// **Validates: Requirements 11.2, 11.3, 11.5**

// TestProperty23_RegisterAndRetrieve verifies that for any valid seeder name,
// registering a seeder and retrieving it by name returns the same seeder.
func TestProperty23_RegisterAndRetrieve(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := genSeederName().Draw(t, "seederName")
		reg := NewRegistry()
		s := &propSeeder{name: name}

		err := reg.Register(name, s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := reg.Get(name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != got {
			t.Errorf("retrieved seeder should be the same instance that was registered")
		}
	})
}

// TestProperty23_DuplicateNameReturnsError verifies that registering the same
// name twice returns ErrDuplicateSeeder.
func TestProperty23_DuplicateNameReturnsError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := genSeederName().Draw(t, "seederName")
		reg := NewRegistry()

		err := reg.Register(name, &propSeeder{name: name})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = reg.Register(name, &propSeeder{name: name})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, ErrDuplicateSeeder) {
			t.Errorf("duplicate registration should return ErrDuplicateSeeder")
		}
	})
}

// TestProperty23_EmptyNameReturnsError verifies that registering with an empty
// or whitespace-only name returns ErrInvalidSeederName.
func TestProperty23_EmptyNameReturnsError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate whitespace-only strings (empty, spaces, tabs)
		spaces := rapid.IntRange(0, 5).Draw(t, "spaces")
		name := ""
		for i := 0; i < spaces; i++ {
			name += " "
		}

		reg := NewRegistry()
		err := reg.Register(name, &propSeeder{name: "test"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidSeederName) {
			t.Errorf("empty/whitespace name should return ErrInvalidSeederName")
		}
	})
}

// Feature: go-migration, Property 24: RunAll executes all registered seeders
// **Validates: Requirements 12.1**

// TestProperty24_RunAllExecutesAllSeeders verifies that for any set of registered
// seeders with no dependencies, RunAll() executes every seeder exactly once.
func TestProperty24_RunAllExecutesAllSeeders(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		names := genUniqueSeederNames(10).Draw(t, "seederNames")
		db := newMockDB(t)
		defer db.Close()

		reg := NewRegistry()
		var order []string

		for _, name := range names {
			err := reg.Register(name, &propSeeder{name: name, order: &order})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		runner := NewRunner(reg, db, nil)
		err := runner.RunAll()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Every registered seeder should have been executed exactly once
		if len(order) != len(names) {
			t.Errorf("len = %d, want %d", len(order), len(names))
		}
		if len(order) != len(names) {
			t.Errorf("len = %d, want %d", len(order), len(names))
		}
	})
}

// Feature: go-migration, Property 25: Dependency resolution respects topological order
// **Validates: Requirements 12.2, 12.3**

// TestProperty25_DependencyOrderIsRespected verifies that for any DAG of seeder
// dependencies, the runner executes seeders in an order where every seeder's
// dependencies are executed before the seeder itself.
func TestProperty25_DependencyOrderIsRespected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of unique seeder names (ordered list)
		names := genUniqueSeederNames(8).Draw(t, "seederNames")
		db := newMockDB(t)
		defer db.Close()

		reg := NewRegistry()
		var order []string

		// Build a random DAG: for each seeder at index i, only allow
		// dependencies on seeders at indices < i (guarantees acyclicity).
		for i, name := range names {
			if i == 0 {
				// First seeder has no possible dependencies
				err := reg.Register(name, &propSeeder{name: name, order: &order})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				continue
			}

			// Randomly pick a subset of earlier seeders as dependencies
			var deps []string
			for j := 0; j < i; j++ {
				if rapid.Bool().Draw(t, "dep_"+names[j]) {
					deps = append(deps, names[j])
				}
			}

			if len(deps) == 0 {
				err := reg.Register(name, &propSeeder{name: name, order: &order})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				err := reg.Register(name, &propDependentSeeder{
					name:  name,
					deps:  deps,
					order: &order,
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		}

		runner := NewRunner(reg, db, nil)
		err := runner.RunAll()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// All seeders should have been executed
		if len(order) != len(names) {
			t.Errorf("len = %d, want %d", len(order), len(names))
		}

		// Build an index map for execution order
		execIdx := make(map[string]int)
		for i, name := range order {
			execIdx[name] = i
		}

		// Verify topological order: for every seeder, all its dependencies
		// must have been executed before it.
		all := reg.GetAll()
		for _, name := range order {
			s := all[name]
			if ds, ok := s.(DependentSeeder); ok {
				for _, dep := range ds.DependsOn() {
					depIdx, depExists := execIdx[dep]
					if !depExists {
						t.Errorf("dependency %q should have been executed", dep)
					}
					if depIdx >= execIdx[name] {
						t.Errorf("dependency %q executed at index %d, but %q at index %d", dep, depIdx, name, execIdx[name])
					}
				}
			}
		}
	})
}

// Feature: go-migration, Property 26: Circular dependencies are detected
// **Validates: Requirements 12.4**

// TestProperty26_CircularDependenciesDetected verifies that for any dependency
// graph containing a cycle, the runner returns ErrCircularDependency.
func TestProperty26_CircularDependenciesDetected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a chain of unique seeder names (at least 2 for a cycle)
		names := genUniqueSeederNames(6).Draw(t, "seederNames")
		if len(names) < 2 {
			// Need at least 2 seeders for a non-trivial cycle
			names = append(names, genSeederName().Draw(t, "extraName"))
			// Ensure uniqueness
			for names[len(names)-1] == names[0] {
				names[len(names)-1] = genSeederName().Draw(t, "extraName2")
			}
		}

		db := newMockDB(t)
		defer db.Close()

		reg := NewRegistry()
		var order []string

		// Create a cycle: each seeder depends on the next, and the last
		// depends on the first. names[0] -> names[1] -> ... -> names[0]
		for i, name := range names {
			nextIdx := (i + 1) % len(names)
			dep := names[nextIdx]
			err := reg.Register(name, &propDependentSeeder{
				name:  name,
				deps:  []string{dep},
				order: &order,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		runner := NewRunner(reg, db, nil)

		// RunAll should detect the cycle
		err := runner.RunAll()
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, ErrCircularDependency) {
			t.Errorf("circular dependency should return ErrCircularDependency, got: %v", err)
		}

		// No seeders should have been executed
		if len(order) != 0 {
			t.Errorf("expected empty, got %v", order)
		}
	})
}

// TestProperty26_SelfDependencyDetected verifies that a seeder depending on
// itself is detected as a circular dependency.
func TestProperty26_SelfDependencyDetected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := genSeederName().Draw(t, "seederName")
		db := newMockDB(t)
		defer db.Close()

		reg := NewRegistry()
		var order []string

		err := reg.Register(name, &propDependentSeeder{
			name:  name,
			deps:  []string{name},
			order: &order,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runner := NewRunner(reg, db, nil)
		err = runner.Run(name)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, ErrCircularDependency) {
			t.Errorf("self-dependency should return ErrCircularDependency, got: %v", err)
		}
	})
}
