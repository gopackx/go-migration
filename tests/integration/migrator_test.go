//go:build integration

package integration_test

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gopackx/go-migration/pkg/database"
	"github.com/gopackx/go-migration/pkg/database/drivers"
	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
)

func register(t *testing.T, m *migrator.Migrator, names ...string) {
	t.Helper()
	for _, name := range names {
		var mig migrator.Migration
		switch name {
		case createUsers:
			mig = createUsersMigration()
		case createPosts:
			mig = createPostsMigration()
		case addBio:
			mig = addBioMigration()
		case alterUsers:
			mig = alterUsersMigration()
		default:
			t.Fatalf("register: unknown migration %q", name)
		}
		must(t, m.Register(name, mig), "register "+name)
	}
}

// TestUpStatusRollbackReset covers Up (multiple batches), Status,
// Rollback(0), Rollback(n) and Reset, plus the indexes, unique constraint and
// foreign key created by schema.Builder.
func TestUpStatusRollbackReset(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db)
		register(t, m, createUsers, createPosts)

		must(t, m.Up(), "first Up")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})
		if !tableExists(t, b, db, "users") || !tableExists(t, b, db, "posts") {
			t.Fatal("users/posts not created")
		}
		assertColumns(t, b, db, "users", "id", "name", "email")
		assertColumns(t, b, db, "posts", "id", "user_id", "title")
		if !indexExists(t, b, db, "users", "idx_users_name") {
			t.Error("index idx_users_name missing")
		}

		// Unique index on email is enforced.
		mustExec(t, b, db, "INSERT INTO users (name, email) VALUES (?, ?)", "Ann", "ann@example.com")
		if _, err := db.Exec(b.q("INSERT INTO users (name, email) VALUES (?, ?)"), "Ann 2", "ann@example.com"); err == nil {
			t.Error("duplicate email accepted: unique index not enforced")
		}
		// Foreign key is enforced, and ON DELETE CASCADE works.
		if _, err := db.Exec(b.q("INSERT INTO posts (user_id, title) VALUES (?, ?)"), 99999, "orphan"); err == nil {
			t.Error("orphan post accepted: foreign key not enforced")
		}
		var userID int64
		must(t, db.QueryRow(b.q("SELECT id FROM users WHERE email = ?"), "ann@example.com").Scan(&userID), "select user id")
		mustExec(t, b, db, "INSERT INTO posts (user_id, title) VALUES (?, ?)", userID, "hello")

		// Up with nothing pending is a no-op.
		must(t, m.Up(), "no-op Up")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})

		// A newly registered migration runs in batch 2.
		register(t, m, addBio)
		must(t, m.Up(), "second Up")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1}, trackRow{addBio, 2})
		assertColumns(t, b, db, "users", "id", "name", "email", "bio")

		st, err := m.Status()
		must(t, err, "Status")
		wantStatus := []struct {
			name  string
			batch int
		}{{createUsers, 1}, {createPosts, 1}, {addBio, 2}}
		if len(st) != len(wantStatus) {
			t.Fatalf("Status returned %d entries, want %d", len(st), len(wantStatus))
		}
		for i, w := range wantStatus {
			if st[i].Name != w.name || !st[i].Applied || st[i].Batch != w.batch || st[i].AppliedAt == nil {
				t.Errorf("Status[%d] = %+v, want applied %s in batch %d", i, st[i], w.name, w.batch)
			}
			// Generous bound: the server time zone may differ from UTC.
			if st[i].AppliedAt != nil {
				if d := time.Since(*st[i].AppliedAt); d > 24*time.Hour || d < -24*time.Hour {
					t.Errorf("Status[%d].AppliedAt = %v is not a recent time", i, st[i].AppliedAt)
				}
			}
		}

		// Rollback(0) rolls back the last batch only.
		must(t, m.Rollback(0), "Rollback(0)")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})
		assertColumns(t, b, db, "users", "id", "name", "email")
		if n := rowCount(t, db, "users"); n != 1 {
			t.Errorf("users rows after dropping bio = %d, want 1", n)
		}

		// Re-apply, then roll back the last two migrations individually.
		must(t, m.Up(), "re-Up")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1}, trackRow{addBio, 2})
		must(t, m.Rollback(2), "Rollback(2)")
		assertTracking(t, db, trackRow{createUsers, 1})
		if tableExists(t, b, db, "posts") {
			t.Error("posts still exists after Rollback(2)")
		}
		assertColumns(t, b, db, "users", "id", "name", "email")

		st, err = m.Status()
		must(t, err, "Status after Rollback(2)")
		if !st[0].Applied || st[1].Applied || st[2].Applied || st[1].AppliedAt != nil || st[2].Batch != 0 {
			t.Errorf("Status after Rollback(2) = %+v", st)
		}

		// Reset rolls back everything; the tracking table itself remains.
		must(t, m.Reset(), "Reset")
		assertTracking(t, db)
		if tableExists(t, b, db, "users") {
			t.Error("users still exists after Reset")
		}
		if !tableExists(t, b, db, "migrations") {
			t.Error("tracking table dropped by Reset")
		}
	})
}

// TestRefresh checks that Refresh = Reset + Up: data is gone and every
// migration ends up in batch 1.
func TestRefresh(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db)
		register(t, m, createUsers, createPosts)
		must(t, m.Up(), "Up batch 1")
		register(t, m, addBio)
		must(t, m.Up(), "Up batch 2")
		mustExec(t, b, db, "INSERT INTO users (name, email, bio) VALUES (?, ?, ?)", "Ann", "ann@example.com", "hi")

		must(t, m.Refresh(), "Refresh")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1}, trackRow{addBio, 1})
		if n := rowCount(t, db, "users"); n != 0 {
			t.Errorf("users rows after Refresh = %d, want 0", n)
		}
		assertColumns(t, b, db, "users", "id", "name", "email", "bio")
	})
}

// TestFresh checks that Fresh drops every table (including ones no migration
// knows about and ones holding foreign keys), reruns all migrations in batch
// 1, and leaves foreign key enforcement switched on afterwards.
func TestFresh(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db)
		register(t, m, createUsers)
		must(t, m.Up(), "Up batch 1")
		register(t, m, createPosts)
		must(t, m.Up(), "Up batch 2")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 2})

		mustExec(t, b, db, "INSERT INTO users (name, email) VALUES (?, ?)", "Ann", "ann@example.com")
		var userID int64
		must(t, db.QueryRow("SELECT id FROM users").Scan(&userID), "select user id")
		mustExec(t, b, db, "INSERT INTO posts (user_id, title) VALUES (?, ?)", userID, "hello")
		// Unmanaged tables, one of them referencing users.
		mustExec(t, b, db, "CREATE TABLE legacy_data (id INTEGER)")
		mustExec(t, b, db, fmt.Sprintf("CREATE TABLE legacy_refs (user_id %s, FOREIGN KEY (user_id) REFERENCES users (id))", b.fkColumnType))
		mustExec(t, b, db, "INSERT INTO legacy_refs (user_id) VALUES (?)", userID)

		must(t, m.Fresh(), "Fresh")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})
		for _, gone := range []string{"legacy_data", "legacy_refs"} {
			if tableExists(t, b, db, gone) {
				t.Errorf("%s survived Fresh", gone)
			}
		}
		if n := rowCount(t, db, "users"); n != 0 {
			t.Errorf("users rows after Fresh = %d, want 0", n)
		}
		if n := rowCount(t, db, "posts"); n != 0 {
			t.Errorf("posts rows after Fresh = %d, want 0", n)
		}
		if !indexExists(t, b, db, "users", "idx_users_name") {
			t.Error("idx_users_name missing after Fresh")
		}

		// Fresh switches FK checks off on its connection; that session state
		// must not leak into pooled connections.
		for i := 0; i < 3; i++ {
			if _, err := db.Exec(b.q("INSERT INTO posts (user_id, title) VALUES (?, ?)"), 99999, "orphan"); err == nil {
				t.Fatalf("orphan post accepted after Fresh (attempt %d): foreign key checks still disabled", i+1)
			}
		}

		// Fresh is repeatable.
		must(t, m.Fresh(), "second Fresh")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})
	})
}

// TestHooks checks hook order for Up and Rollback, that a failing before
// hook aborts the run, and that a failing after hook is ignored.
func TestHooks(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		var events []string
		m := b.newMigrator(db)
		register(t, m, createUsers, createPosts)
		m.BeforeMigrate(func(name, dir string) error {
			events = append(events, "before:"+dir+":"+name)
			return nil
		})
		m.AfterMigrate(func(name, dir string, d time.Duration) error {
			if d < 0 {
				t.Errorf("negative duration for %s", name)
			}
			events = append(events, "after:"+dir+":"+name)
			return nil
		})

		must(t, m.Up(), "Up")
		want := []string{
			"before:up:" + createUsers, "after:up:" + createUsers,
			"before:up:" + createPosts, "after:up:" + createPosts,
		}
		if strings.Join(events, "|") != strings.Join(want, "|") {
			t.Errorf("Up hook events = %v, want %v", events, want)
		}

		events = nil
		must(t, m.Rollback(0), "Rollback")
		want = []string{
			"before:down:" + createPosts, "after:down:" + createPosts,
			"before:down:" + createUsers, "after:down:" + createUsers,
		}
		if strings.Join(events, "|") != strings.Join(want, "|") {
			t.Errorf("Rollback hook events = %v, want %v", events, want)
		}
		assertTracking(t, db)

		// A failing before hook aborts before the migration runs; a failing
		// after hook does not fail the run.
		errHook := errors.New("hook says no")
		m2 := b.newMigrator(db)
		register(t, m2, createUsers, createPosts)
		m2.BeforeMigrate(func(name, dir string) error {
			if name == createPosts {
				return errHook
			}
			return nil
		})
		m2.AfterMigrate(func(string, string, time.Duration) error { return errBoom })
		err := m2.Up()
		if !errors.Is(err, errHook) {
			t.Fatalf("Up with failing before hook: err = %v, want %v", err, errHook)
		}
		assertTracking(t, db, trackRow{createUsers, 1})
		if !tableExists(t, b, db, "users") {
			t.Error("users missing: after-hook error must not undo the migration")
		}
		if tableExists(t, b, db, "posts") {
			t.Error("posts created although its before hook failed")
		}
	})
}

// TestFailedMigrationIsAtomic checks that a failing transactional migration
// leaves neither its data changes nor a tracking row behind, and that the
// error identifies the migration and the failing SQL.
func TestFailedMigrationIsAtomic(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db)
		register(t, m, createUsers)
		must(t, m.Up(), "Up")

		badSQL := "INSERT INTO no_such_table (x) VALUES (1)"
		must(t, m.Register(failingDML, funcMigration{
			up: func(s *schema.Builder) error {
				if err := s.RawExec("INSERT INTO users (name, email) VALUES ('Tx', 'tx@example.com')"); err != nil {
					return err
				}
				return s.RawExec(badSQL)
			},
			down: noopDown,
		}), "register failing DML")

		err := m.Up()
		if err == nil {
			t.Fatal("Up succeeded although the migration failed")
		}
		var migErr *migrator.MigrationError
		if !errors.As(err, &migErr) {
			t.Fatalf("error %v (%T) is not a *migrator.MigrationError", err, err)
		}
		if migErr.MigrationName != failingDML {
			t.Errorf("MigrationError.MigrationName = %q, want %q", migErr.MigrationName, failingDML)
		}
		if migErr.SQL != badSQL {
			t.Errorf("MigrationError.SQL = %q, want %q", migErr.SQL, badSQL)
		}
		if n := rowCount(t, db, "users"); n != 0 {
			t.Errorf("users rows after failed migration = %d, want 0 (insert not rolled back)", n)
		}
		assertTracking(t, db, trackRow{createUsers, 1})

		// DDL followed by an error: rolled back where DDL is transactional.
		m2 := b.newMigrator(db)
		register(t, m2, createUsers)
		must(t, m2.Register(failingDDL, funcMigration{
			up: func(s *schema.Builder) error {
				if err := s.Create("half_done", func(t *schema.Blueprint) {
					t.ID()
					t.String("x", 10)
				}); err != nil {
					return err
				}
				return errBoom
			},
			down: noopDown,
		}), "register failing DDL")
		err = m2.Up()
		if !errors.Is(err, errBoom) {
			t.Fatalf("Up: err = %v, want %v", err, errBoom)
		}
		assertTracking(t, db, trackRow{createUsers, 1})
		exists := tableExists(t, b, db, "half_done")
		if b.transactionalDDL && exists {
			t.Error("half_done table survived the rolled-back migration")
		}
		if !b.transactionalDDL {
			t.Logf("%s: DDL auto-commits; half_done exists after failure = %v (expected true)", b.name, exists)
		}
	})
}

// TestDisableTransaction checks migrations that opt out of the transaction.
// On PostgreSQL the migration uses CREATE INDEX CONCURRENTLY, which fails
// inside a transaction block.
func TestDisableTransaction(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db)
		register(t, m, createUsers)
		must(t, m.Up(), "Up users")

		if b.name == "postgres" {
			// Same SQL without DisableTransaction must fail on PostgreSQL.
			mt := b.newMigrator(db)
			register(t, mt, createUsers)
			must(t, mt.Register(txIndex, funcMigration{
				up:   func(s *schema.Builder) error { return s.RawExec(b.nonTxUp) },
				down: noopDown,
			}), "register tx index")
			if err := mt.Up(); err == nil {
				t.Fatal("CREATE INDEX CONCURRENTLY succeeded inside a transaction")
			}
			assertTracking(t, db, trackRow{createUsers, 1})
		}

		must(t, m.Register(nonTxIndex, nonTxMigration{funcMigration{
			up:   func(s *schema.Builder) error { return s.RawExec(b.nonTxUp) },
			down: func(s *schema.Builder) error { return s.RawExec(b.nonTxDown) },
		}}), "register non-tx migration")
		must(t, m.Up(), "Up non-tx")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{nonTxIndex, 2})
		if !indexExists(t, b, db, "users", "idx_users_email_name") {
			t.Error("idx_users_email_name not created")
		}

		must(t, m.Rollback(0), "Rollback non-tx")
		assertTracking(t, db, trackRow{createUsers, 1})
		if indexExists(t, b, db, "users", "idx_users_email_name") {
			t.Error("idx_users_email_name not dropped")
		}

		// Without a transaction, work done before the failure persists, and
		// no tracking row is written.
		mf := b.newMigrator(db)
		register(t, mf, createUsers)
		must(t, mf.Register(nonTxFailure, nonTxMigration{funcMigration{
			up: func(s *schema.Builder) error {
				if err := s.RawExec("INSERT INTO users (name, email) VALUES ('Kept', 'kept@example.com')"); err != nil {
					return err
				}
				return errBoom
			},
			down: noopDown,
		}}), "register failing non-tx migration")
		if err := mf.Up(); !errors.Is(err, errBoom) {
			t.Fatalf("Up failing non-tx: err = %v, want %v", err, errBoom)
		}
		assertTracking(t, db, trackRow{createUsers, 1})
		if n := rowCount(t, db, "users"); n != 1 {
			t.Errorf("users rows = %d, want 1 (non-transactional insert must persist)", n)
		}
	})
}

// TestAlterColumns covers Alter add/rename/drop column and HasTable/HasColumn
// used from inside a migration.
func TestAlterColumns(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db)
		register(t, m, createUsers, alterUsers)
		must(t, m.Up(), "Up")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{alterUsers, 1})
		assertColumns(t, b, db, "users", "id", "name", "email", "handle")

		mustExec(t, b, db, "INSERT INTO users (name, email, handle) VALUES (?, ?, ?)", "Ann", "ann@example.com", "@ann")
		var handle string
		must(t, db.QueryRow(b.q("SELECT handle FROM users WHERE email = ?"), "ann@example.com").Scan(&handle), "select handle")
		if handle != "@ann" {
			t.Errorf("handle = %q, want %q", handle, "@ann")
		}

		must(t, m.Rollback(0), "Rollback")
		// Rollback(0) undoes the whole batch (both migrations).
		assertTracking(t, db)
		if tableExists(t, b, db, "users") {
			t.Error("users still exists after rolling back batch 1")
		}

		// Apply in two batches so only the ALTER is rolled back.
		m2 := b.newMigrator(db)
		register(t, m2, createUsers)
		must(t, m2.Up(), "Up users")
		mustExec(t, b, db, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@example.com")
		register(t, m2, alterUsers)
		must(t, m2.Up(), "Up alter")
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{alterUsers, 2})
		must(t, m2.Rollback(0), "Rollback alter")
		assertTracking(t, db, trackRow{createUsers, 1})
		assertColumns(t, b, db, "users", "id", "name", "email")
		if hasColumn(t, b, db, "users", "handle") {
			t.Error("handle column survived Down")
		}
		var name string
		must(t, db.QueryRow(b.q("SELECT name FROM users WHERE email = ?"), "bob@example.com").Scan(&name), "select name")
		if name != "Bob" {
			t.Errorf("name = %q after dropping a column, want Bob", name)
		}
	})
}

// TestDryRun checks that dry-run requires the tracking table, prints SQL and
// never changes the database (Up, Rollback and Fresh).
func TestDryRun(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		var buf bytes.Buffer
		dry := b.newMigrator(db, migrator.WithDryRun(&buf))
		register(t, dry, createUsers, createPosts)

		err := dry.Up()
		if !errors.Is(err, migrator.ErrTrackingTable) {
			t.Fatalf("dry-run Up without tracking table: err = %v, want ErrTrackingTable", err)
		}
		if tableExists(t, b, db, "migrations") {
			t.Fatal("dry-run created the tracking table")
		}

		// migrate:install equivalent.
		must(t, migrator.NewTrackerWithGrammar(db, "migrations", b.grammar).EnsureTable(), "EnsureTable")

		buf.Reset()
		must(t, dry.Up(), "dry-run Up")
		out := buf.String()
		for _, want := range []string{"-- Migration: " + createUsers, "-- Migration: " + createPosts, "CREATE TABLE"} {
			if !strings.Contains(out, want) {
				t.Errorf("dry-run output missing %q:\n%s", want, out)
			}
		}
		if tableExists(t, b, db, "users") || tableExists(t, b, db, "posts") {
			t.Error("dry-run Up created tables")
		}
		assertTracking(t, db)

		st, err := dry.Status()
		must(t, err, "dry-run Status")
		for _, s := range st {
			if s.Applied {
				t.Errorf("dry-run Status reports %s applied", s.Name)
			}
		}

		// Apply for real, then dry-run Rollback and Fresh must not touch it.
		real := b.newMigrator(db)
		register(t, real, createUsers, createPosts)
		must(t, real.Up(), "real Up")

		buf.Reset()
		must(t, dry.Rollback(0), "dry-run Rollback")
		if !strings.Contains(buf.String(), "DROP TABLE") {
			t.Errorf("dry-run Rollback output missing DROP TABLE:\n%s", buf.String())
		}
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})
		if !tableExists(t, b, db, "users") || !tableExists(t, b, db, "posts") {
			t.Error("dry-run Rollback dropped tables")
		}

		buf.Reset()
		must(t, dry.Fresh(), "dry-run Fresh")
		if !strings.Contains(buf.String(), "-- Fresh: drop all tables") || !strings.Contains(buf.String(), "CREATE TABLE") {
			t.Errorf("dry-run Fresh output unexpected:\n%s", buf.String())
		}
		assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})
		if !tableExists(t, b, db, "users") || !tableExists(t, b, db, "posts") {
			t.Error("dry-run Fresh dropped tables")
		}
	})
}

// TestCustomTrackingTable checks WithTableName.
func TestCustomTrackingTable(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db, migrator.WithTableName("schema_migrations"))
		register(t, m, createUsers)
		must(t, m.Up(), "Up")
		assertTrackingTable(t, db, "schema_migrations", trackRow{createUsers, 1})
		if tableExists(t, b, db, "migrations") {
			t.Error("default tracking table created although WithTableName was used")
		}
		must(t, m.Fresh(), "Fresh")
		assertTrackingTable(t, db, "schema_migrations", trackRow{createUsers, 1})
	})
}

// TestSQLiteDriverOpen opens SQLite through pkg/database/drivers (driver name
// "sqlite", provided by modernc.org/sqlite) and runs a migration on it.
func TestSQLiteDriverOpen(t *testing.T) {
	d := drivers.NewSQLiteDriver()
	db, err := d.Open(database.ConnectionConfig{
		Driver:   "sqlite",
		Database: filepath.Join(t.TempDir(), "driver.db"),
		Options:  map[string]string{"_pragma": "foreign_keys(1)"},
	})
	must(t, err, "open")
	t.Cleanup(func() { _ = db.Close() })
	must(t, db.Ping(), "ping")

	var fk int
	must(t, db.QueryRow("PRAGMA foreign_keys").Scan(&fk), "PRAGMA foreign_keys")
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1 (Options not passed through)", fk)
	}

	m := migrator.New(db, migrator.WithGrammar(grammars.NewSQLiteGrammar()))
	register(t, m, createUsers, createPosts)
	must(t, m.Up(), "Up")
	assertTracking(t, db, trackRow{createUsers, 1}, trackRow{createPosts, 1})
}
