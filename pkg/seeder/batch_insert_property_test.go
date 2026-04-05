package seeder

import (
	"fmt"
	"math"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"pgregory.net/rapid"
	"strings"
)

// Feature: library-improvements, Property 8: Batch insert chunking
// **Validates: Requirements 5.1**

func TestPropertyBatchInsert_ChunkingProducesCorrectNumberOfStatements(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		recordCount := rapid.IntRange(1, 50).Draw(t, "recordCount")
		chunkSize := rapid.IntRange(1, 20).Draw(t, "chunkSize")

		// Generate records with consistent keys.
		records := make([]map[string]any, recordCount)
		for i := 0; i < recordCount; i++ {
			records[i] = map[string]any{
				"id":    i + 1,
				"name":  "item",
				"value": i * 10,
			}
		}

		expectedChunks := int(math.Ceil(float64(recordCount) / float64(chunkSize)))

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer db.Close()

		// Expect exactly ceil(R/C) Exec calls, each an INSERT statement.
		// Track the expected rows per chunk to verify total equals R.
		totalRows := 0
		for i := 0; i < expectedChunks; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if end > recordCount {
				end = recordCount
			}
			rowsInChunk := end - start
			totalRows += rowsInChunk

			mock.ExpectExec("INSERT INTO").
				WillReturnResult(sqlmock.NewResult(0, int64(rowsInChunk)))
		}

		err = CreateMany(db, "test_table", records, chunkSize)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify all expectations were met (correct number of INSERT statements).
		if mock.ExpectationsWereMet() != nil {
			t.Errorf("expected %d INSERT statements for %d records with chunk size %d", expectedChunks, recordCount, chunkSize)
		}

		// Verify total rows across all statements equals R.
		if recordCount != totalRows {
			t.Errorf("total rows across all chunks should equal record count")
		}
	})
}

// Feature: library-improvements, Property 9: Record key consistency validation
// **Validates: Requirements 5.6, 5.7**

func TestPropertyBatchInsert_RecordKeyConsistencyValidation(t *testing.T) {
	// Sub-test 1: Mismatched keys → error returned, no DB operations.
	t.Run("mismatched_keys_returns_error_no_db_ops", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a base set of at least 2 column keys.
			baseKeyCount := rapid.IntRange(2, 6).Draw(t, "baseKeyCount")
			baseKeys := make([]string, baseKeyCount)
			seen := make(map[string]bool)
			for i := range baseKeys {
				for {
					k := rapid.StringMatching(`[a-z][a-z0-9_]{1,8}`).Draw(t, fmt.Sprintf("baseKey%d", i))
					if !seen[k] {
						seen[k] = true
						baseKeys[i] = k
						break
					}
				}
			}

			// Generate at least 2 records total (need at least one consistent + one mismatched).
			recordCount := rapid.IntRange(2, 10).Draw(t, "recordCount")

			// Pick which record index (1..recordCount-1) will have mismatched keys.
			mismatchIdx := rapid.IntRange(1, recordCount-1).Draw(t, "mismatchIdx")

			// Build records: all use baseKeys except the mismatch record.
			records := make([]map[string]any, recordCount)
			for i := range records {
				records[i] = make(map[string]any)
				for _, k := range baseKeys {
					records[i][k] = "val"
				}
			}

			// Mutate the mismatch record: either add an extra key or remove one.
			addExtra := rapid.Bool().Draw(t, "addExtra")
			if addExtra {
				// Add a key that doesn't exist in baseKeys.
				var extraKey string
				for {
					extraKey = rapid.StringMatching(`[a-z][a-z0-9_]{1,8}`).Draw(t, "extraKey")
					if !seen[extraKey] {
						break
					}
				}
				records[mismatchIdx][extraKey] = "extra"
			} else {
				// Remove a random key from the mismatch record.
				removeIdx := rapid.IntRange(0, len(baseKeys)-1).Draw(t, "removeIdx")
				delete(records[mismatchIdx], baseKeys[removeIdx])
			}

			// Set up sqlmock with NO expectations — no DB operations should occur.
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer db.Close()

			chunkSize := rapid.IntRange(1, 20).Draw(t, "chunkSize")
			err = CreateMany(db, "test_table", records, chunkSize)

			// Must return an error.
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "mismatched keys") {
				t.Errorf("%q not in %q", "mismatched keys", err.Error())
			}

			// No DB expectations were set, so if any DB call happened, this fails.
			if mock.ExpectationsWereMet() != nil {
				t.Errorf("no database operations should have occurred")
			}
		})
	})

	// Sub-test 2: Consistent keys → no key-mismatch error.
	t.Run("consistent_keys_no_key_mismatch_error", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate column keys.
			keyCount := rapid.IntRange(1, 6).Draw(t, "keyCount")
			keys := make([]string, keyCount)
			seen := make(map[string]bool)
			for i := range keys {
				for {
					k := rapid.StringMatching(`[a-z][a-z0-9_]{1,8}`).Draw(t, fmt.Sprintf("key%d", i))
					if !seen[k] {
						seen[k] = true
						keys[i] = k
						break
					}
				}
			}

			recordCount := rapid.IntRange(1, 20).Draw(t, "recordCount")
			chunkSize := rapid.IntRange(1, 20).Draw(t, "chunkSize")

			// Build records with identical key sets.
			records := make([]map[string]any, recordCount)
			for i := range records {
				records[i] = make(map[string]any)
				for _, k := range keys {
					records[i][k] = fmt.Sprintf("val_%d", i)
				}
			}

			// Set up sqlmock expecting INSERT statements (one per chunk).
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer db.Close()

			expectedChunks := int(math.Ceil(float64(recordCount) / float64(chunkSize)))
			for i := 0; i < expectedChunks; i++ {
				start := i * chunkSize
				end := start + chunkSize
				if end > recordCount {
					end = recordCount
				}
				rowsInChunk := end - start
				mock.ExpectExec("INSERT INTO").
					WillReturnResult(sqlmock.NewResult(0, int64(rowsInChunk)))
			}

			err = CreateMany(db, "test_table", records, chunkSize)

			// Should NOT return a key-mismatch error.
			if err != nil {
				if strings.Contains(err.Error(), "mismatched keys") {
					t.Errorf("%q should not be in %q", "mismatched keys", err.Error())
				}
			}

			if mock.ExpectationsWereMet() != nil {
				t.Errorf("all expected INSERT statements should have been executed")
			}
		})
	})
}
