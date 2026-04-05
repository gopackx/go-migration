package seeder

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"pgregory.net/rapid"
	"strings"
)

// Feature: library-improvements, Property 10: Tag-based seeder filtering
// **Validates: Requirements 6.1, 6.2**

// basicPropSeeder is a plain Seeder (no tags) that tracks whether Run was called.
type basicPropSeeder struct {
	name   string
	called *map[string]bool
}

func (s *basicPropSeeder) Run(db *sql.DB) error {
	(*s.called)[s.name] = true
	return nil
}

// taggedPropSeeder implements TaggedSeeder and tracks whether Run was called.
type taggedPropSeeder struct {
	name   string
	tags   []string
	called *map[string]bool
}

func (s *taggedPropSeeder) Run(db *sql.DB) error {
	(*s.called)[s.name] = true
	return nil
}

func (s *taggedPropSeeder) Tags() []string {
	return s.tags
}

// genTag generates a short tag string.
func genTag() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z]{2,10}`)
}

// genUniqueTags generates a slice of unique tag strings.
func genUniqueTags(maxN int) *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		n := rapid.IntRange(1, maxN).Draw(t, "tagCount")
		seen := make(map[string]bool)
		tags := make([]string, 0, n)
		for len(tags) < n {
			tag := genTag().Draw(t, "tag")
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
		return tags
	})
}

func TestProperty10_TagBasedSeederFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a pool of unique tags to draw from.
		allTags := genUniqueTags(8).Draw(t, "allTags")

		// Pick a target tag from the pool.
		targetIdx := rapid.IntRange(0, len(allTags)-1).Draw(t, "targetIdx")
		targetTag := allTags[targetIdx]

		// Generate unique seeder names.
		seederCount := rapid.IntRange(1, 12).Draw(t, "seederCount")
		seederNames := make([]string, 0, seederCount)
		seenNames := make(map[string]bool)
		for len(seederNames) < seederCount {
			name := rapid.StringMatching(`[a-z][a-z0-9_]{1,15}`).Draw(t, "seederName")
			if !seenNames[name] {
				seenNames[name] = true
				seederNames = append(seederNames, name)
			}
		}

		// Track which seeders were called.
		called := make(map[string]bool)

		// Build the registry with a mix of basic and tagged seeders.
		reg := NewRegistry()

		// Track which seeders we expect to be called (tagged with targetTag).
		expectedCalled := make(map[string]bool)

		for _, name := range seederNames {
			isTagged := rapid.Bool().Draw(t, "isTagged_"+name)

			if isTagged {
				// Pick a random subset of tags from the pool for this seeder.
				seederTagCount := rapid.IntRange(1, len(allTags)).Draw(t, "tagCount_"+name)
				seederTagIdxs := make(map[int]bool)
				for len(seederTagIdxs) < seederTagCount {
					idx := rapid.IntRange(0, len(allTags)-1).Draw(t, "tagIdx_"+name)
					seederTagIdxs[idx] = true
				}
				seederTags := make([]string, 0, seederTagCount)
				for idx := range seederTagIdxs {
					seederTags = append(seederTags, allTags[idx])
				}

				// Check if this seeder has the target tag.
				hasTarget := false
				for _, st := range seederTags {
					if st == targetTag {
						hasTarget = true
						break
					}
				}
				if hasTarget {
					expectedCalled[name] = true
				}

				err := reg.Register(name, &taggedPropSeeder{
					name:   name,
					tags:   seederTags,
					called: &called,
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				// Basic seeder — no tags, should never be called by RunByTag.
				err := reg.Register(name, &basicPropSeeder{
					name:   name,
					called: &called,
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		}

		// Create runner with a mock DB (seeders don't use it).
		db, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer db.Close()

		runner := NewRunner(reg, db, nil)

		// Execute RunByTag.
		err = runner.RunByTag(targetTag)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify: exactly the seeders with matching tags were called.
		for _, name := range seederNames {
			if expectedCalled[name] {
				if !called[name] {
					t.Errorf("seeder %q has target tag %q and should have been called", name, targetTag)
				}
			} else {
				if called[name] {
					t.Errorf("expected false")
				}
			}
		}

		// Verify the count matches exactly.
		if len(expectedCalled) != len(called) {
			t.Errorf("number of called seeders should match number of seeders with target tag")
		}
	})
}

// Feature: library-improvements, Property 11: Rollbackable seeder dispatch
// **Validates: Requirements 7.1, 7.2**

// rollbackablePropSeeder implements RollbackableSeeder and tracks calls.
type rollbackablePropSeeder struct {
	name           string
	runCalled      *map[string]int
	rollbackCalled *map[string]int
}

func (s *rollbackablePropSeeder) Run(db *sql.DB) error {
	(*s.runCalled)[s.name]++
	return nil
}

func (s *rollbackablePropSeeder) Rollback(db *sql.DB) error {
	(*s.rollbackCalled)[s.name]++
	return nil
}

// nonRollbackablePropSeeder is a plain Seeder that does NOT implement RollbackableSeeder.
type nonRollbackablePropSeeder struct {
	name      string
	runCalled *map[string]int
}

func (s *nonRollbackablePropSeeder) Run(db *sql.DB) error {
	(*s.runCalled)[s.name]++
	return nil
}

func TestProperty11_RollbackableSeederDispatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a unique seeder name.
		name := rapid.StringMatching(`[A-Za-z][A-Za-z0-9_]{1,20}`).Draw(t, "seederName")

		runCalled := make(map[string]int)
		rollbackCalled := make(map[string]int)

		reg := NewRegistry()
		err := reg.Register(name, &rollbackablePropSeeder{
			name:           name,
			runCalled:      &runCalled,
			rollbackCalled: &rollbackCalled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runner := NewRunner(reg, nil, nil)

		// Call Rollback on the rollbackable seeder.
		err = runner.Rollback(name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify Rollback was called exactly once.
		if 1 != rollbackCalled[name] {
			t.Errorf("Rollback method should be invoked exactly once for seeder %q", name)
		}

		// Verify Run was NOT called.
		if 0 != runCalled[name] {
			t.Errorf("Run method should not be invoked when calling Rollback for seeder %q", name)
		}
	})
}

// Feature: library-improvements, Property 12: Non-rollbackable seeder error
// **Validates: Requirements 7.3**

func TestProperty12_NonRollbackableSeederError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a unique seeder name.
		name := rapid.StringMatching(`[A-Za-z][A-Za-z0-9_]{1,20}`).Draw(t, "seederName")

		runCalled := make(map[string]int)

		reg := NewRegistry()
		err := reg.Register(name, &nonRollbackablePropSeeder{
			name:      name,
			runCalled: &runCalled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runner := NewRunner(reg, nil, nil)

		// Call Rollback on a non-rollbackable seeder.
		err = runner.Rollback(name)

		// Verify a non-nil error is returned.
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "does not support rollback") {
			t.Errorf("%q not in %q", "does not support rollback", err.Error())
		}

		// Verify Run was NOT called.
		if 0 != runCalled[name] {
			t.Errorf("Run method should not be invoked when calling Rollback for seeder %q", name)
		}
	})
}
