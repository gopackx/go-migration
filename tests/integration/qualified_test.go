//go:build integration

package integration_test

import (
	"database/sql"
	"testing"

	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
)

// qualifiedSchema is the PostgreSQL schema / MySQL database that holds the
// tracking table in TestQualifiedTrackingTable.
const qualifiedSchema = "gomig_app"

// tableExistsIn reports whether table exists in the given PostgreSQL schema /
// MySQL database.
func tableExistsIn(t *testing.T, b backend, db *sql.DB, schemaName, table string) bool {
	t.Helper()
	var n int
	query := b.q("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?")
	if err := db.QueryRow(query, schemaName, table).Scan(&n); err != nil {
		t.Fatalf("tableExistsIn(%s.%s): %v", schemaName, table, err)
	}
	return n > 0
}

// TestQualifiedTrackingTable keeps the tracking table in another schema
// (WithTableName("gomig_app.migrations")) while the migrated tables live in
// the default one, through Up, Status, Rollback, Up and Fresh. Fresh only
// clears the default schema, so it must drop the qualified tracking table
// itself: afterwards every migration has to be re-applied in batch 1.
//
// SQLite is skipped: its "schemas" are attached databases, which do not
// survive the connection Fresh discards; the SQL is covered by unit tests.
func TestQualifiedTrackingTable(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		var create, drop string
		switch b.name {
		case "postgres":
			create = "CREATE SCHEMA IF NOT EXISTS " + qualifiedSchema
			drop = "DROP SCHEMA IF EXISTS " + qualifiedSchema + " CASCADE"
		case "mysql":
			create = "CREATE DATABASE IF NOT EXISTS " + qualifiedSchema
			drop = "DROP DATABASE IF EXISTS " + qualifiedSchema
		default:
			t.Skip("schema-qualified tracking table is tested on PostgreSQL and MySQL only")
		}
		mustExec(t, b, db, drop)
		mustExec(t, b, db, create)
		t.Cleanup(func() { _, _ = db.Exec(drop) })

		trackingName := qualifiedSchema + ".migrations"
		quotedTracking := schema.QuoteQualified(b.grammar, trackingName)
		m := b.newMigrator(db, migrator.WithTableName(trackingName))
		register(t, m, createUsers, createPosts)

		// Dry-run needs the tracking table: it must be found in its schema.
		tracker := migrator.NewTrackerWithGrammar(db, trackingName, b.grammar)
		exists, err := tracker.TableExists()
		must(t, err, "TableExists before Up")
		if exists {
			t.Fatal("TableExists = true before Up")
		}

		must(t, m.Up(), "Up")
		if !tableExistsIn(t, b, db, qualifiedSchema, "migrations") {
			t.Fatalf("tracking table not created in schema %s", qualifiedSchema)
		}
		if tableExists(t, b, db, "migrations") {
			t.Error("tracking table created in the default schema")
		}
		if tableExists(t, b, db, trackingName) {
			t.Errorf("table literally named %q created", trackingName)
		}
		if !tableExists(t, b, db, "users") || !tableExists(t, b, db, "posts") {
			t.Fatal("users/posts not created in the default schema")
		}
		assertTrackingTable(t, db, quotedTracking, trackRow{createUsers, 1}, trackRow{createPosts, 1})
		exists, err = tracker.TableExists()
		must(t, err, "TableExists after Up")
		if !exists {
			t.Error("TableExists = false after Up")
		}

		statuses, err := m.Status()
		must(t, err, "Status")
		if len(statuses) != 2 {
			t.Fatalf("Status returned %d entries, want 2", len(statuses))
		}
		for _, s := range statuses {
			if !s.Applied || s.Batch != 1 {
				t.Errorf("status %+v, want applied in batch 1", s)
			}
		}

		must(t, m.Rollback(1), "Rollback(1)")
		if tableExists(t, b, db, "posts") {
			t.Error("posts still exists after Rollback(1)")
		}
		assertTrackingTable(t, db, quotedTracking, trackRow{createUsers, 1})

		must(t, m.Up(), "Up again")
		assertTrackingTable(t, db, quotedTracking, trackRow{createUsers, 1}, trackRow{createPosts, 2})

		must(t, m.Fresh(), "Fresh")
		if !tableExists(t, b, db, "users") || !tableExists(t, b, db, "posts") {
			t.Fatal("users/posts not recreated by Fresh")
		}
		if !tableExistsIn(t, b, db, qualifiedSchema, "migrations") {
			t.Fatalf("tracking table missing from schema %s after Fresh", qualifiedSchema)
		}
		assertTrackingTable(t, db, quotedTracking, trackRow{createUsers, 1}, trackRow{createPosts, 1})
	})
}
