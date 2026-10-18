package migrator

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/schema"
	"strings"
)

// --- Helpers ---

func newTestMigrator(t *testing.T) (*Migrator, *sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := New(db, WithGrammar(&mockGrammar{}))
	return m, db, mock
}

// expectEnsureTable sets up the sqlmock expectation for CREATE TABLE IF NOT EXISTS.
func expectEnsureTable(mock sqlmock.Sqlmock) {
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
}

// expectMockHasTable expects mockGrammar's HasTable query ("SELECT 1") used
// by Tracker.TableExists, returning a count of 1 (exists) or 0.
func expectMockHasTable(mock sqlmock.Sqlmock, exists bool) {
	count := 0
	if exists {
		count = 1
	}
	mock.ExpectQuery("^SELECT 1$").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
}

// expectGetApplied sets up a query expectation returning the given migration names/batches.
func expectGetApplied(mock sqlmock.Sqlmock, records []MigrationRecord) {
	rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"})
	for _, r := range records {
		rows.AddRow(r.Name, r.Batch, r.CreatedAt)
	}
	mock.ExpectQuery("SELECT migration, batch, created_at FROM").WillReturnRows(rows)
}

// expectMaxBatch sets up the COALESCE(MAX(batch), 0) query.
func expectMaxBatch(mock sqlmock.Sqlmock, batch int) {
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(
		sqlmock.NewRows([]string{"max"}).AddRow(batch),
	)
}

// expectRecord sets up an INSERT expectation for recording a migration.
func expectRecord(mock sqlmock.Sqlmock, name string, batch int) {
	mock.ExpectExec("INSERT INTO").WithArgs(name, batch).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// expectRemove sets up a DELETE expectation for removing a migration record.
func expectRemove(mock sqlmock.Sqlmock, name string) {
	mock.ExpectExec("DELETE FROM").WithArgs(name).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

// keepMockDriverConn pins one extra pooled connection for the duration of a
// test. Fresh discards the connection it dropped tables on (driver.ErrBadConn),
// which makes database/sql Close it; sqlmock unregisters its DSN once every
// connection is closed, after which no new connection can be opened. Holding
// a spare connection keeps the DSN registered. Call the returned func (defer)
// to release it.
func keepMockDriverConn(t *testing.T, db *sql.DB) func() {
	t.Helper()
	spare, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("pin spare connection: %v", err)
	}
	return func() { _ = spare.Close() }
}

// expectMigrationTxRecord sets up the expectations for one successful
// transactional "up" migration: the tracking INSERT runs inside the
// migration's transaction, before commit.
func expectMigrationTxRecord(mock sqlmock.Sqlmock, name string, batch int) {
	mock.ExpectBegin()
	expectRecord(mock, name, batch)
	mock.ExpectCommit()
}

// expectMigrationTxRemove sets up the expectations for one successful
// transactional "down" migration: the tracking DELETE runs inside the
// migration's transaction, before commit.
func expectMigrationTxRemove(mock sqlmock.Sqlmock, name string) {
	mock.ExpectBegin()
	expectRemove(mock, name)
	mock.ExpectCommit()
}

// --- Up Tests ---

func TestUp_PendingMigrations(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	if m.Register("20240102000000_create_posts", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_create_posts", &noopMigration{}))
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, nil) // no applied migrations
	expectMaxBatch(mock, 0)     // first batch

	// Migration 1
	expectMigrationTxRecord(mock, "20240101000000_create_users", 1)

	// Migration 2
	expectMigrationTxRecord(mock, "20240102000000_create_posts", 1)

	err := m.Up()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestUp_NoPending(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
	})

	err := m.Up()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestUp_PartiallyApplied(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	if m.Register("20240102000000_create_posts", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_create_posts", &noopMigration{}))
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
	})
	expectMaxBatch(mock, 1) // next batch = 2

	// Only migration 2 should run
	expectMigrationTxRecord(mock, "20240102000000_create_posts", 2)

	err := m.Up()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Rollback Tests ---

func TestRollback_ByBatch(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	if m.Register("20240102000000_create_posts", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_create_posts", &noopMigration{}))
	}

	expectEnsureTable(mock)

	// GetLastBatch: max batch = 1, then GetByBatch(1)
	expectMaxBatch(mock, 1)
	rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
		AddRow("20240101000000_create_users", 1, time.Now()).
		AddRow("20240102000000_create_posts", 1, time.Now())
	mock.ExpectQuery("SELECT migration, batch, created_at FROM .* WHERE batch").
		WithArgs(1).WillReturnRows(rows)

	// Rollback in reverse order: posts first, then users
	expectMigrationTxRemove(mock, "20240102000000_create_posts")

	expectMigrationTxRemove(mock, "20240101000000_create_users")

	err := m.Rollback(0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestRollback_BySteps(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	if m.Register("20240102000000_create_posts", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_create_posts", &noopMigration{}))
	}
	if m.Register("20240103000000_create_tags", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240103000000_create_tags", &noopMigration{}))
	}

	expectEnsureTable(mock)

	// GetLastNMigrations(2): GetApplied returns all 3, we take last 2 reversed
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
		{Name: "20240102000000_create_posts", Batch: 1, CreatedAt: time.Now()},
		{Name: "20240103000000_create_tags", Batch: 2, CreatedAt: time.Now()},
	})

	// Rollback: tags first, then posts (reverse order)
	expectMigrationTxRemove(mock, "20240103000000_create_tags")

	expectMigrationTxRemove(mock, "20240102000000_create_posts")

	err := m.Rollback(2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Reset Tests ---

func TestReset(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	if m.Register("20240102000000_create_posts", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_create_posts", &noopMigration{}))
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
		{Name: "20240102000000_create_posts", Batch: 1, CreatedAt: time.Now()},
	})

	// Reverse order: posts first, then users
	expectMigrationTxRemove(mock, "20240102000000_create_posts")

	expectMigrationTxRemove(mock, "20240101000000_create_users")

	err := m.Reset()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Status Tests ---

func TestStatus(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	now := time.Now()
	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	if m.Register("20240102000000_create_posts", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_create_posts", &noopMigration{}))
	}

	expectMockHasTable(mock, true)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: now},
	})

	statuses, err := m.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("len = %d, want %d", len(statuses), 2)
	}

	if "20240101000000_create_users" != statuses[0].Name {
		t.Errorf("got %v, want %v", statuses[0].Name, "20240101000000_create_users")
	}
	if !statuses[0].Applied {
		t.Errorf("expected true")
	}
	if 1 != statuses[0].Batch {
		t.Errorf("got %v, want %v", statuses[0].Batch, 1)
	}
	if statuses[0].AppliedAt == nil {
		t.Errorf("expected non-nil")
	}

	if "20240102000000_create_posts" != statuses[1].Name {
		t.Errorf("got %v, want %v", statuses[1].Name, "20240102000000_create_posts")
	}
	if statuses[1].Applied {
		t.Errorf("expected false")
	}
	if 0 != statuses[1].Batch {
		t.Errorf("got %v, want %v", statuses[1].Batch, 0)
	}
	if statuses[1].AppliedAt != nil {
		t.Errorf("expected nil, got %v", statuses[1].AppliedAt)
	}

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Fresh Tests ---

func TestFresh(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}

	// List tables, then drop each one with a separate statement
	mock.ExpectQuery("LIST TABLES").WillReturnRows(
		sqlmock.NewRows([]string{"name"}).AddRow("users").AddRow("migrations"),
	)
	mock.ExpectExec("^DROP TABLE users$").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("^DROP TABLE migrations$").WillReturnResult(sqlmock.NewResult(0, 0))

	// Then Up()
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTxRecord(mock, "20240101000000_create_users", 1)

	err := m.Fresh()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestFresh_DropFails(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	mock.ExpectQuery("LIST TABLES").WillReturnRows(
		sqlmock.NewRows([]string{"name"}).AddRow("users").AddRow("posts"),
	)
	mock.ExpectExec("DROP TABLE users").WillReturnError(errors.New("permission denied"))

	err := m.Fresh()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "drop all tables") {
		t.Errorf("%q not in %q", "drop all tables", err.Error())
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("%q not in %q", "permission denied", err.Error())
	}
	// "DROP TABLE posts" must not run after the first failure, and Up()
	// must not run either.
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestFresh_ListTablesFails(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	mock.ExpectQuery("LIST TABLES").WillReturnError(errors.New("no access"))

	err := m.Fresh()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "list tables") || !strings.Contains(err.Error(), "no access") {
		t.Errorf("unexpected error text: %q", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestFresh_DryRun_PrintsDropStatements(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	var buf bytes.Buffer
	m := New(db, WithGrammar(&mockGrammar{}), WithDryRun(&buf))
	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}

	// Only the read-only list query runs. Nothing is dropped, and since the
	// tables would be gone after a real Fresh, the tracking table is not
	// consulted: every migration is shown as pending. No DDL/DML executes.
	mock.ExpectQuery("LIST TABLES").WillReturnRows(
		sqlmock.NewRows([]string{"name"}).AddRow("users").AddRow("posts"),
	)

	if err := m.Fresh(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"-- Fresh: drop all tables", "DROP TABLE users;", "DROP TABLE posts;", "-- Migration: 20240101000000_create_users"} {
		if !strings.Contains(out, want) {
			t.Errorf("%q not in dry-run output %q", want, out)
		}
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// Fresh must work with a real dialect: MySQL lists base tables and drops each
// one between FOREIGN_KEY_CHECKS toggles, all as separate statements.
func TestFresh_MySQLGrammar(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	g, err := ResolveGrammar("mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer keepMockDriverConn(t, db)()
	m := New(db, WithGrammar(g))
	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(CONCAT(DATABASE(), ':', ?), 0)")).
		WithArgs("go-migration:migrations").
		WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'")).
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}).AddRow("users").AddRow("migrations"))
	mock.ExpectExec("^SET FOREIGN_KEY_CHECKS = 0$").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("DROP TABLE IF EXISTS `users`")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("DROP TABLE IF EXISTS `migrations`")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("^SET FOREIGN_KEY_CHECKS = 1$").WillReturnResult(sqlmock.NewResult(0, 0))

	// Up(): MySQL tracking table DDL and ? placeholders.
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS `migrations` .*INT AUTO_INCREMENT PRIMARY KEY").
		WillReturnResult(sqlmock.NewResult(0, 0))
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `migrations` (migration, batch) VALUES (?, ?)")).
		WithArgs("20240101000000_create_users", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':', ?))")).
		WithArgs("go-migration:migrations").
		WillReturnRows(sqlmock.NewRows([]string{"release"}).AddRow(1))

	if err := m.Fresh(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFresh_NoGrammar(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	m := New(db) // no grammar
	err = m.Fresh()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fresh requires a grammar") {
		t.Errorf("%q not in %q", "fresh requires a grammar", err.Error())
	}
}

// --- Refresh Tests ---

func TestRefresh(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}

	// Reset phase: EnsureTable + GetApplied (1 applied) + Down + Remove
	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
	})
	expectMigrationTxRemove(mock, "20240101000000_create_users")

	// Up phase: EnsureTable + GetApplied (empty) + NextBatch + Execute + Record
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTxRecord(mock, "20240101000000_create_users", 1)

	err := m.Refresh()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Hook Integration Tests ---

func TestUp_BeforeHookError_AbortsMigration(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	m.BeforeMigrate(func(name, direction string) error {
		return errors.New("hook failed")
	})

	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)

	err := m.Up()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "hook failed") {
		t.Errorf("%q not in %q", "hook failed", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestUp_AfterHookCalled(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	var hookCalled bool
	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}
	m.AfterMigrate(func(name, direction string, d time.Duration) error {
		hookCalled = true
		if "20240101000000_create_users" != name {
			t.Errorf("got %v, want %v", name, "20240101000000_create_users")
		}
		if "up" != direction {
			t.Errorf("got %v, want %v", direction, "up")
		}
		return nil
	})

	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTxRecord(mock, "20240101000000_create_users", 1)

	err := m.Up()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hookCalled {
		t.Errorf("expected true")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Option Tests ---

func TestWithTableName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	m := New(db, WithGrammar(&mockGrammar{}), WithTableName("custom_migrations"))
	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}

	// EnsureTable should use custom table name
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS \"custom_migrations\"").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// GetApplied from custom table
	mock.ExpectQuery("SELECT migration, batch, created_at FROM \"custom_migrations\"").
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}))

	// NextBatchNumber
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(
		sqlmock.NewRows([]string{"max"}).AddRow(0),
	)

	// Record into custom table, inside the migration transaction
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO \"custom_migrations\"").
		WithArgs("20240101000000_create_users", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = m.Up()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Rollback edge case: no applied migrations ---

func TestRollback_NoApplied(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	expectEnsureTable(mock)
	expectMaxBatch(mock, 0) // no batches

	err := m.Rollback(0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestReset_NoApplied(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	expectEnsureTable(mock)
	expectGetApplied(mock, nil)

	err := m.Reset()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Edge-case tests (Task 8.4) ---

// failingUpMigration succeeds on Down but fails on Up.
type failingUpMigration struct {
	downCalled bool
}

func (m *failingUpMigration) Up(s *schema.Builder) error   { return errors.New("up failed") }
func (m *failingUpMigration) Down(s *schema.Builder) error { m.downCalled = true; return nil }

// failingDownMigration succeeds on Up but fails on Down.
type failingDownMigration struct {
	upCalled bool
}

func (m *failingDownMigration) Up(s *schema.Builder) error   { m.upCalled = true; return nil }
func (m *failingDownMigration) Down(s *schema.Builder) error { return errors.New("down failed") }

// TestUp_FailureStopsSubsequent verifies that when a migration's Up() fails,
// subsequent migrations are not executed and only prior successes are recorded.
// Validates: Requirement 2.4
func TestUp_FailureStopsSubsequent(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	first := &noopMigration{}
	second := &failingUpMigration{}
	third := &noopMigration{}

	if m.Register("20240101000000_first", first) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_first", first))
	}
	if m.Register("20240102000000_second", second) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_second", second))
	}
	if m.Register("20240103000000_third", third) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240103000000_third", third))
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)

	// First migration succeeds
	expectMigrationTxRecord(mock, "20240101000000_first", 1)

	// Second migration fails: begin + rollback (no commit, no record)
	mock.ExpectBegin()
	mock.ExpectRollback()

	err := m.Up()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "20240102000000_second") {
		t.Errorf("%q not in %q", "20240102000000_second", err.Error())
	}

	// First ran, third never ran
	if !first.upCalled {
		t.Errorf("expected true")
	}
	if third.upCalled {
		t.Errorf("expected false")
	}

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// TestRollback_FailureStopsFurtherRollback verifies that when a migration's
// Down() fails during rollback, earlier migrations are not rolled back.
// Validates: Requirement 3.5
func TestRollback_FailureStopsFurtherRollback(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	first := &noopMigration{}
	second := &noopMigration{}
	third := &failingDownMigration{}

	if m.Register("20240101000000_first", first) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_first", first))
	}
	if m.Register("20240102000000_second", second) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_second", second))
	}
	if m.Register("20240103000000_third", third) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240103000000_third", third))
	}

	expectEnsureTable(mock)

	// Rollback by batch: all 3 in batch 1
	expectMaxBatch(mock, 1)
	rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
		AddRow("20240101000000_first", 1, time.Now()).
		AddRow("20240102000000_second", 1, time.Now()).
		AddRow("20240103000000_third", 1, time.Now())
	mock.ExpectQuery("SELECT migration, batch, created_at FROM .* WHERE batch").
		WithArgs(1).WillReturnRows(rows)

	// Reverse order: third rolls back first — but it fails
	mock.ExpectBegin()
	mock.ExpectRollback()

	err := m.Rollback(0)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "20240103000000_third") {
		t.Errorf("%q not in %q", "20240103000000_third", err.Error())
	}

	// Second and first should NOT have been rolled back
	if second.downCalled {
		t.Errorf("expected false")
	}
	if first.downCalled {
		t.Errorf("expected false")
	}

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// TestRefresh_UpPhaseFailure verifies that when the up phase of Refresh fails,
// the error message contains the name of the failing migration.
// Validates: Requirement 4.5
func TestRefresh_UpPhaseFailure(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if m.Register("20240101000000_first", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_first", &noopMigration{}))
	}
	if m.Register("20240102000000_second", &failingUpMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240102000000_second", &failingUpMigration{}))
	}

	// Reset phase: first migration is applied, roll it back
	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_first", Batch: 1, CreatedAt: time.Now()},
	})
	expectMigrationTxRemove(mock, "20240101000000_first")

	// Up phase: both pending, first succeeds, second fails
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)

	// First migration succeeds
	expectMigrationTxRecord(mock, "20240101000000_first", 1)

	// Second migration fails
	mock.ExpectBegin()
	mock.ExpectRollback()

	err := m.Refresh()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "refresh up phase") {
		t.Errorf("%q not in %q", "refresh up phase", err.Error())
	}
	if !strings.Contains(err.Error(), "20240102000000_second") {
		t.Errorf("%q not in %q", "20240102000000_second", err.Error())
	}

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}
