package migrator

import (
	"database/sql"
	"errors"
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

// expectMigrationTx sets up Begin + Commit expectations for a single migration.
func expectMigrationTx(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
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
	expectMigrationTx(mock)
	expectRecord(mock, "20240101000000_create_users", 1)

	// Migration 2
	expectMigrationTx(mock)
	expectRecord(mock, "20240102000000_create_posts", 1)

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
	expectMigrationTx(mock)
	expectRecord(mock, "20240102000000_create_posts", 2)

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
	expectMigrationTx(mock)
	expectRemove(mock, "20240102000000_create_posts")

	expectMigrationTx(mock)
	expectRemove(mock, "20240101000000_create_users")

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
	expectMigrationTx(mock)
	expectRemove(mock, "20240103000000_create_tags")

	expectMigrationTx(mock)
	expectRemove(mock, "20240102000000_create_posts")

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
	expectMigrationTx(mock)
	expectRemove(mock, "20240102000000_create_posts")

	expectMigrationTx(mock)
	expectRemove(mock, "20240101000000_create_users")

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

	expectEnsureTable(mock)
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

	if m.Register("20240101000000_create_users", &noopMigration{}) != nil {
		t.Fatalf("unexpected error: %v", m.Register("20240101000000_create_users", &noopMigration{}))
	}

	// Drop all tables
	mock.ExpectExec("DROP ALL").WillReturnResult(sqlmock.NewResult(0, 0))

	// Then Up()
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTx(mock)
	expectRecord(mock, "20240101000000_create_users", 1)

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

	mock.ExpectExec("DROP ALL").WillReturnError(errors.New("permission denied"))

	err := m.Fresh()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "drop all tables") {
		t.Errorf("%q not in %q", "drop all tables", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
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
	expectMigrationTx(mock)
	expectRemove(mock, "20240101000000_create_users")

	// Up phase: EnsureTable + GetApplied (empty) + NextBatch + Execute + Record
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTx(mock)
	expectRecord(mock, "20240101000000_create_users", 1)

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
	expectMigrationTx(mock)
	expectRecord(mock, "20240101000000_create_users", 1)

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
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS custom_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// GetApplied from custom table
	mock.ExpectQuery("SELECT migration, batch, created_at FROM custom_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}))

	// NextBatchNumber
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(
		sqlmock.NewRows([]string{"max"}).AddRow(0),
	)

	expectMigrationTx(mock)

	// Record into custom table
	mock.ExpectExec("INSERT INTO custom_migrations").
		WithArgs("20240101000000_create_users", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

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
	expectMigrationTx(mock)
	expectRecord(mock, "20240101000000_first", 1)

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
	expectMigrationTx(mock)
	expectRemove(mock, "20240101000000_first")

	// Up phase: both pending, first succeeds, second fails
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)

	// First migration succeeds
	expectMigrationTx(mock)
	expectRecord(mock, "20240101000000_first", 1)

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
