//go:build integration

package integration_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/seeder"
)

// Names containing every quote character the dialects use: a double quote
// (the PostgreSQL/SQLite identifier delimiter), a backtick (the MySQL
// delimiter) and a single quote (the string-literal delimiter used by the
// HasTable/HasColumn catalog queries), plus a space.
const (
	oddTable    = "odd\"tab`le's"
	oddColumn   = "we\"ird col`'s"
	oddColumn2  = "an\"other`'s"
	oddTracking = "track\"ing`'s"
	createOdd   = "20240201000001_create_odd_table"
	alterOdd    = "20240201000002_alter_odd_table"
)

func createOddMigration() migrator.Migration {
	return funcMigration{
		up: func(s *schema.Builder) error {
			if has, err := s.HasTable(oddTable); err != nil || has {
				return fmt.Errorf("HasTable before create = %v, %v; want false", has, err)
			}
			if err := s.Create(oddTable, func(t *schema.Blueprint) {
				t.ID()
				t.String(oddColumn, 50)
				t.Index(oddColumn)
			}); err != nil {
				return err
			}
			if has, err := s.HasTable(oddTable); err != nil || !has {
				return fmt.Errorf("HasTable after create = %v, %v; want true", has, err)
			}
			if has, err := s.HasColumn(oddTable, oddColumn); err != nil || !has {
				return fmt.Errorf("HasColumn after create = %v, %v; want true", has, err)
			}
			return nil
		},
		down: func(s *schema.Builder) error { return s.DropIfExists(oddTable) },
	}
}

func alterOddMigration() migrator.Migration {
	return funcMigration{
		up: func(s *schema.Builder) error {
			if has, err := s.HasColumn(oddTable, oddColumn2); err != nil || has {
				return fmt.Errorf("HasColumn before alter = %v, %v; want false", has, err)
			}
			return s.Alter(oddTable, func(t *schema.Blueprint) {
				t.String(oddColumn2, 20).Nullable()
			})
		},
		down: func(s *schema.Builder) error {
			return s.Alter(oddTable, func(t *schema.Blueprint) { t.DropColumn(oddColumn2) })
		},
	}
}

// quotedCount counts the rows of table, quoting its name for the backend.
func quotedCount(t *testing.T, b backend, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + b.grammar.QuoteIdentifier(table)).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestQuotedIdentifiers migrates, seeds, truncates and rolls back a table
// whose table, column, index and tracking-table names contain quote
// characters, on every backend.
func TestQuotedIdentifiers(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db, migrator.WithTableName(oddTracking))
		must(t, m.Register(createOdd, createOddMigration()), "register create")
		must(t, m.Register(alterOdd, alterOddMigration()), "register alter")

		must(t, m.Up(), "Up")
		if !tableExists(t, b, db, oddTable) {
			t.Fatalf("table %q not created", oddTable)
		}
		assertColumns(t, b, db, oddTable, "id", oddColumn, oddColumn2)
		if !indexExists(t, b, db, oddTable, "idx_"+oddTable+"_"+oddColumn) {
			t.Errorf("index on %q missing", oddColumn)
		}
		if !tableExists(t, b, db, oddTracking) {
			t.Fatalf("tracking table %q not created", oddTracking)
		}
		if tableExists(t, b, db, "migrations") {
			t.Error("default tracking table created although WithTableName was used")
		}
		assertTrackingTable(t, db, b.grammar.QuoteIdentifier(oddTracking),
			trackRow{createOdd, 1}, trackRow{alterOdd, 1})

		statuses, err := m.Status()
		must(t, err, "Status")
		for _, s := range statuses {
			if !s.Applied || s.Batch != 1 {
				t.Errorf("status %+v, want applied in batch 1", s)
			}
		}

		// Seeder batch insert and truncate quote the names too.
		must(t, seeder.CreateManyWithDialect(db, oddTable, []map[string]any{
			{oddColumn: "a", oddColumn2: "x"},
			{oddColumn: "b'c\"d`e", oddColumn2: nil},
		}, 10, b.dialect), "CreateMany")
		if n := quotedCount(t, b, db, oddTable); n != 2 {
			t.Errorf("rows after CreateMany = %d, want 2", n)
		}
		var got string
		query := b.q("SELECT " + b.grammar.QuoteIdentifier(oddColumn) + " FROM " + b.grammar.QuoteIdentifier(oddTable) +
			" WHERE " + b.grammar.QuoteIdentifier(oddColumn2) + " IS NULL")
		must(t, db.QueryRow(query).Scan(&got), "select inserted value")
		if got != "b'c\"d`e" {
			t.Errorf("value = %q", got)
		}
		runner := seeder.NewRunner(seeder.NewRegistry(), db, nil, seeder.WithDialect(b.dialect))
		must(t, runner.Truncate(oddTable), "Truncate")
		if n := quotedCount(t, b, db, oddTable); n != 0 {
			t.Errorf("rows after Truncate = %d, want 0", n)
		}

		// Roll back the alter, then everything.
		must(t, m.Rollback(1), "Rollback(1)")
		assertColumns(t, b, db, oddTable, "id", oddColumn)
		assertTrackingTable(t, db, b.grammar.QuoteIdentifier(oddTracking), trackRow{createOdd, 1})

		must(t, m.Reset(), "Reset")
		if tableExists(t, b, db, oddTable) {
			t.Error("table still exists after Reset")
		}
		assertTrackingTable(t, db, b.grammar.QuoteIdentifier(oddTracking))

		// Fresh drops and recreates everything, including the odd tracking
		// table, and re-applies both migrations in one batch.
		must(t, m.Up(), "Up again")
		must(t, m.Fresh(), "Fresh")
		assertColumns(t, b, db, oddTable, "id", oddColumn, oddColumn2)
		assertTrackingTable(t, db, b.grammar.QuoteIdentifier(oddTracking),
			trackRow{createOdd, 1}, trackRow{alterOdd, 1})

		// The tracker's own table check uses the escaped string literal.
		exists, err := migrator.NewTrackerWithGrammar(db, oddTracking, b.grammar).TableExists()
		must(t, err, "TableExists")
		if !exists {
			t.Error("Tracker.TableExists = false for the odd tracking table")
		}
	})
}
