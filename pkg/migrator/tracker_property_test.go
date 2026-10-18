package migrator

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"pgregory.net/rapid"
)

// Feature: go-migration, Property 12: EnsureTable is idempotent
// **Validates: Requirements 5.2**
// For any database state, calling EnsureTable() multiple times should not
// produce an error and should result in the tracking table existing.
func TestProperty12_EnsureTableIsIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "callCount")

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer db.Close()

		// Expect EnsureTable to be called n times, all succeeding.
		for i := 0; i < n; i++ {
			mock.ExpectExec("CREATE TABLE IF NOT EXISTS \"migrations\"").
				WillReturnResult(sqlmock.NewResult(0, 0))
		}

		tracker := NewTracker(db, "migrations")

		for i := 0; i < n; i++ {
			err := tracker.EnsureTable()
			if err != nil {
				t.Errorf("EnsureTable call %d of %d should not error", i+1, n)
			}
		}

		if mock.ExpectationsWereMet() != nil {
			t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
		}
	})
}

// Feature: go-migration, Property 14: Query by batch returns correct subset
// **Validates: Requirements 5.5**
// For any set of recorded migrations across batches, GetByBatch(n) should
// return exactly the migrations whose batch number equals n.
func TestProperty14_QueryByBatchReturnsCorrectSubset(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of migration records with various batch numbers.
		numRecords := rapid.IntRange(1, 30).Draw(t, "numRecords")
		maxBatch := rapid.IntRange(1, 5).Draw(t, "maxBatch")

		type record struct {
			name  string
			batch int
		}

		now := time.Now().Truncate(time.Second)
		records := make([]record, numRecords)
		for i := 0; i < numRecords; i++ {
			batch := rapid.IntRange(1, maxBatch).Draw(t, "batch")
			name := rapid.StringMatching(`^20[0-9]{12}_[a-z]{3,10}$`).Draw(t, "name")
			records[i] = record{name: name, batch: batch}
		}

		// Pick a target batch to query.
		targetBatch := rapid.IntRange(1, maxBatch).Draw(t, "targetBatch")

		// Compute expected subset: records whose batch == targetBatch.
		var expected []record
		for _, r := range records {
			if r.batch == targetBatch {
				expected = append(expected, r)
			}
		}

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer db.Close()

		// Mock the DB to return only the expected subset (simulating the WHERE clause).
		rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"})
		for _, r := range expected {
			rows.AddRow(r.name, r.batch, now)
		}

		mock.ExpectQuery("SELECT migration, batch, created_at FROM \"migrations\" WHERE batch = \\$1 ORDER BY migration ASC").
			WithArgs(targetBatch).
			WillReturnRows(rows)

		tracker := NewTracker(db, "migrations")
		result, err := tracker.GetByBatch(targetBatch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify count matches.
		if len(result) != len(expected) {
			t.Errorf("len = %d, want %d", len(result), len(expected))
		}

		// Verify every returned record has the correct batch number.
		for _, r := range result {
			if targetBatch != r.Batch {
				t.Errorf("returned record should have batch=%d", targetBatch)
			}
		}

		if mock.ExpectationsWereMet() != nil {
			t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
		}
	})
}
