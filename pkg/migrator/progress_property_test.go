package migrator

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"pgregory.net/rapid"
)

// Feature: library-improvements, Property 7: Progress events have correct indices and totals
// **Validates: Requirements 4.1, 4.2**

func TestPropertyProgress_UpEventsHaveCorrectIndicesAndTotals(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 10).Draw(t, "migrationCount")
		baseTS := rapid.Int64Range(20200101000000, 20291231235959).Draw(t, "baseTS")

		// Generate unique sorted migration names.
		names := make([]string, count)
		for i := 0; i < count; i++ {
			names[i] = fmt.Sprintf("%014d_migration_%d", baseTS+int64(i)*100, i)
		}

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer db.Close()

		// Collect progress events.
		var events []ProgressEvent
		m := New(db, WithGrammar(&mockGrammar{}), WithProgress(func(e ProgressEvent) {
			events = append(events, e)
		}))

		for _, name := range names {
			if m.Register(name, &noopMigration{}) != nil {
				t.Fatalf("unexpected error: %v", m.Register(name, &noopMigration{}))
			}
		}

		// Set up mock expectations for Up(): no applied migrations.
		expectEnsureTable(mock)
		expectGetApplied(mock, nil)
		expectMaxBatch(mock, 0)
		for _, name := range names {
			expectMigrationTx(mock)
			expectRecord(mock, name, 1)
		}

		err = m.Up()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.ExpectationsWereMet() != nil {
			t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
		}

		// Verify progress events.
		if len(events) != count {
			t.Fatalf("Up should produce exactly N progress events")
		}
		for i, ev := range events {
			if i != ev.Index {
				t.Errorf("event[%d] Index", i)
			}
			if count != ev.Total {
				t.Errorf("event[%d] Total", i)
			}
			if names[i] != ev.Name {
				t.Errorf("event[%d] Name", i)
			}
			if "up" != ev.Direction {
				t.Errorf("event[%d] Direction", i)
			}
		}
	})
}

func TestPropertyProgress_ResetEventsHaveCorrectIndicesAndTotals(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 10).Draw(t, "migrationCount")
		baseTS := rapid.Int64Range(20200101000000, 20291231235959).Draw(t, "baseTS")

		names := make([]string, count)
		for i := 0; i < count; i++ {
			names[i] = fmt.Sprintf("%014d_migration_%d", baseTS+int64(i)*100, i)
		}

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer db.Close()

		var events []ProgressEvent
		m := New(db, WithGrammar(&mockGrammar{}), WithProgress(func(e ProgressEvent) {
			events = append(events, e)
		}))

		for _, name := range names {
			if m.Register(name, &noopMigration{}) != nil {
				t.Fatalf("unexpected error: %v", m.Register(name, &noopMigration{}))
			}
		}

		// Build applied records (all migrations applied, ascending order).
		applied := make([]MigrationRecord, count)
		for i, name := range names {
			applied[i] = MigrationRecord{Name: name, Batch: 1}
		}

		// Set up mock expectations for Reset().
		expectEnsureTable(mock)
		expectGetApplied(mock, applied)

		// Reset reverses applied, so Down runs in reverse order.
		for i := count - 1; i >= 0; i-- {
			expectMigrationTx(mock)
			expectRemove(mock, names[i])
		}

		err = m.Reset()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.ExpectationsWereMet() != nil {
			t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
		}

		// Verify progress events.
		if len(events) != count {
			t.Fatalf("Reset should produce exactly N progress events")
		}
		for i, ev := range events {
			if i != ev.Index {
				t.Errorf("event[%d] Index", i)
			}
			if count != ev.Total {
				t.Errorf("event[%d] Total", i)
			}
			// Reset reverses the applied list, so event names are in reverse order.
			if names[count-1-i] != ev.Name {
				t.Errorf("event[%d] Name", i)
			}
			if "down" != ev.Direction {
				t.Errorf("event[%d] Direction", i)
			}
		}
	})
}
