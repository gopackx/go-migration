package migrator

import (
	"errors"
	"sort"
	"testing"

	"pgregory.net/rapid"
	"strings"
)

// validMigrationNameGen generates names matching both legacy and new formats:
//   - Legacy: ^\d{14}_[a-z][a-z0-9_]*$
//   - New:    ^\d{4}_\d{2}_\d{2}_\d{6}_\d{4}_[a-z][a-z0-9_]*$
func validMigrationNameGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		desc := rapid.StringMatching(`[a-z][a-z0-9_]{0,20}`).Draw(t, "desc")
		useNewFormat := rapid.Bool().Draw(t, "useNewFormat")
		if useNewFormat {
			prefix := rapid.StringMatching(`\d{4}_\d{2}_\d{2}_\d{6}_\d{4}`).Draw(t, "newPrefix")
			return prefix + "_" + desc
		}
		timestamp := rapid.StringMatching(`\d{14}`).Draw(t, "timestamp")
		return timestamp + "_" + desc
	})
}

// uniqueValidNamesGen generates a slice of distinct valid migration names (1..20).
func uniqueValidNamesGen() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		count := rapid.IntRange(1, 20).Draw(t, "count")
		seen := make(map[string]bool)
		names := make([]string, 0, count)
		for len(names) < count {
			name := validMigrationNameGen().Draw(t, "name")
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		return names
	})
}

// invalidMigrationNameGen generates strings that do NOT match the valid pattern.
func invalidMigrationNameGen() *rapid.Generator[string] {
	return rapid.OneOf(
		// empty string
		rapid.Just(""),
		// no timestamp prefix — starts with a letter
		rapid.StringMatching(`[a-z][a-z0-9_]{0,10}`),
		// short timestamp (less than 14 digits)
		rapid.Custom(func(t *rapid.T) string {
			ts := rapid.StringMatching(`\d{1,13}`).Draw(t, "shortts")
			desc := rapid.StringMatching(`[a-z][a-z0-9_]{0,10}`).Draw(t, "desc")
			return ts + "_" + desc
		}),
		// timestamp only, no underscore or description
		rapid.StringMatching(`\d{14}`),
		// timestamp + underscore but empty description
		rapid.Custom(func(t *rapid.T) string {
			ts := rapid.StringMatching(`\d{14}`).Draw(t, "ts")
			return ts + "_"
		}),
		// description starts with uppercase
		rapid.Custom(func(t *rapid.T) string {
			ts := rapid.StringMatching(`\d{14}`).Draw(t, "ts")
			desc := rapid.StringMatching(`[A-Z][a-z0-9_]{0,10}`).Draw(t, "desc")
			return ts + "_" + desc
		}),
		// description starts with digit
		rapid.Custom(func(t *rapid.T) string {
			ts := rapid.StringMatching(`\d{14}`).Draw(t, "ts")
			desc := rapid.StringMatching(`[0-9][a-z0-9_]{0,10}`).Draw(t, "desc")
			return ts + "_" + desc
		}),
		// contains hyphen
		rapid.Custom(func(t *rapid.T) string {
			ts := rapid.StringMatching(`\d{14}`).Draw(t, "ts")
			return ts + "_create-table"
		}),
	)
}

// Feature: go-migration, Property 1: Registration preserves retrievability
// Validates: Requirements 1.1
func TestProperty1_RegistrationPreservesRetrievability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := validMigrationNameGen().Draw(t, "name")
		m := &stubMigration{}
		r := NewRegistry()

		err := r.Register(name, m)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		got, err := r.Get(name)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if m != got {
			t.Errorf("retrieved migration should be the same instance that was registered")
		}
	})
}

// Feature: go-migration, Property 2: Registry maintains timestamp order
// Validates: Requirements 1.2
func TestProperty2_RegistryMaintainsTimestampOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		names := uniqueValidNamesGen().Draw(t, "names")
		r := NewRegistry()

		for _, name := range names {
			err := r.Register(name, &stubMigration{})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}

		all := r.GetAll()
		if len(all) != len(names) {
			t.Errorf("len = %d, want %d", len(all), len(names))
		}

		// Verify sorted ascending by name
		for i := 1; i < len(all); i++ {
			if !(all[i-1].Name < all[i].Name) {
				t.Errorf("expected %q < %q at positions %d, %d", all[i-1].Name, all[i].Name, i-1, i)
			}
		}

		// Verify the result matches a standard sort of the input names
		sorted := make([]string, len(names))
		copy(sorted, names)
		sort.Strings(sorted)
		for i, rm := range all {
			if sorted[i] != rm.Name {
				t.Errorf("got %v, want %v", rm.Name, sorted[i])
			}
		}
	})
}

// Feature: go-migration, Property 3: Duplicate registration is rejected
// Validates: Requirements 1.4
func TestProperty3_DuplicateRegistrationIsRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := validMigrationNameGen().Draw(t, "name")
		r := NewRegistry()

		err1 := r.Register(name, &stubMigration{})
		if err1 != nil {
			t.Errorf("first registration should succeed")
		}

		err2 := r.Register(name, &stubMigration{})
		if err2 == nil {
			t.Errorf("expected error, got nil")
		}
		if !errors.Is(err2, ErrDuplicateMigration) {
			t.Errorf("error should wrap ErrDuplicateMigration")
		}
		if !strings.Contains(err2.Error(), name) {
			t.Errorf("%q not in %q", name, err2.Error())
		}

		// Registry count should still be 1
		if 1 != r.Count() {
			t.Errorf("got %v, want %v", r.Count(), 1)
		}
	})
}

// Feature: go-migration, Property 4: Invalid names are rejected
// Validates: Requirements 1.5
func TestProperty4_InvalidNamesAreRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := invalidMigrationNameGen().Draw(t, "invalidName")
		r := NewRegistry()

		err := r.Register(name, &stubMigration{})
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidMigrationName) {
			t.Errorf("error should wrap ErrInvalidMigrationName for name %q", name)
		}

		// Registry should remain empty
		if 0 != r.Count() {
			t.Errorf("got %v, want %v", r.Count(), 0)
		}
	})
}
