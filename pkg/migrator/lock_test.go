package migrator

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
)

// lockingGrammar is a mockGrammar whose dialect uses a migration lock.
type lockingGrammar struct{ mockGrammar }

func (g *lockingGrammar) CompileTryLock(key string) string { return "SELECT try_lock($1)" }
func (g *lockingGrammar) CompileUnlock(key string) string  { return "SELECT unlock($1)" }

const testLockKey = "go-migration:migrations"

func expectTryLock(mock sqlmock.Sqlmock, result any) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT try_lock($1)")).
		WithArgs(testLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(result))
}

func expectUnlock(mock sqlmock.Sqlmock, result any) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT unlock($1)")).
		WithArgs(testLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"unlocked"}).AddRow(result))
}

// recordingLogger collects Error messages.
type recordingLogger struct{ errors []string }

func (l *recordingLogger) Info(string, ...any) {}
func (l *recordingLogger) Error(msg string, args ...any) {
	l.errors = append(l.errors, msg)
}

func newLockingMigrator(t *testing.T, opts ...Option) (*Migrator, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := newMockDB(t)
	m := New(db, append([]Option{WithGrammar(&lockingGrammar{})}, opts...)...)
	return m, db, mock
}

func TestLock_AcquiredOnFirstTry(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()

	expectTryLock(mock, true)
	expectUnlock(mock, true)

	release, err := m.Lock(context.Background())
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if got := db.Stats().InUse; got != 1 {
		t.Errorf("connections in use while locked = %d, want 1 (pinned lock connection)", got)
	}
	if err := release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if got := db.Stats().InUse; got != 0 {
		t.Errorf("connections in use after release = %d, want 0", got)
	}
	// Releasing twice is a no-op.
	if err := release(); err != nil {
		t.Errorf("second release: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLock_MySQLIntegerResult(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	m := New(db, WithGrammar(grammars.NewMySQLGrammar()))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(CONCAT(DATABASE(), ':', ?), 0)")).
		WithArgs(testLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':', ?))")).
		WithArgs(testLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"RELEASE_LOCK"}).AddRow(int64(1)))

	release, err := m.Lock(context.Background())
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if err := release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLock_AcquiredAfterRetry(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()

	expectTryLock(mock, false)
	expectTryLock(mock, int64(0))
	expectTryLock(mock, true)
	expectUnlock(mock, true)

	start := time.Now()
	release, err := m.Lock(context.Background())
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	// Two waits: 100ms then 200ms.
	if d := time.Since(start); d < 250*time.Millisecond {
		t.Errorf("Lock returned after %v, expected it to back off between attempts", d)
	}
	if err := release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLock_NullResultIsNotAcquired(t *testing.T) {
	m, db, mock := newLockingMigrator(t, WithLockTimeout(0))
	defer db.Close()

	expectTryLock(mock, nil) // MySQL GET_LOCK returns NULL on error

	_, err := m.Lock(context.Background())
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("err = %v, want ErrLockTimeout", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLock_TimeoutSingleAttempt(t *testing.T) {
	m, db, mock := newLockingMigrator(t, WithLockTimeout(0))
	defer db.Close()

	expectTryLock(mock, false)

	_, err := m.Lock(context.Background())
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("err = %v, want ErrLockTimeout", err)
	}
	if !strings.Contains(err.Error(), "could not acquire migration lock within 0s") {
		t.Errorf("unexpected error text: %q", err.Error())
	}
	if got := db.Stats().InUse; got != 0 {
		t.Errorf("connections in use after timeout = %d, want 0", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLock_TimeoutAfterPolling(t *testing.T) {
	m, db, mock := newLockingMigrator(t, WithLockTimeout(250*time.Millisecond))
	defer db.Close()

	// More "not acquired" answers than the poller can use in 250ms.
	for i := 0; i < 20; i++ {
		expectTryLock(mock, false)
	}

	start := time.Now()
	err := m.Up()
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("Up err = %v, want ErrLockTimeout", err)
	}
	if !strings.Contains(err.Error(), "within 250ms") {
		t.Errorf("unexpected error text: %q", err.Error())
	}
	if d := time.Since(start); d < 250*time.Millisecond || d > 5*time.Second {
		t.Errorf("Up returned after %v, want about 250ms", d)
	}
}

func TestLock_ContextCancelled(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()

	for i := 0; i < 20; i++ {
		expectTryLock(mock, false)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_, err := m.Lock(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestLock_UnlockFailureDiscardsConnection(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	expectTryLock(mock, true)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT unlock($1)")).
		WithArgs(testLockKey).
		WillReturnError(errors.New("connection reset"))

	release, err := m.Lock(context.Background())
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if got := db.Stats().OpenConnections; got != 2 {
		t.Fatalf("OpenConnections = %d, want 2 (spare + lock)", got)
	}
	err = release()
	if err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("release err = %v, want unlock failure", err)
	}
	// Only the spare remains: the lock connection was closed, not pooled,
	// so the server drops the lock with the session.
	if got := db.Stats().OpenConnections; got != 1 {
		t.Errorf("OpenConnections after failed unlock = %d, want 1", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLock_UnlockReturnsFalseDiscardsConnection(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	expectTryLock(mock, true)
	expectUnlock(mock, false)

	release, err := m.Lock(context.Background())
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if err := release(); err == nil {
		t.Fatal("release err = nil, want error for a lock that was not held")
	}
	if got := db.Stats().OpenConnections; got != 1 {
		t.Errorf("OpenConnections = %d, want 1", got)
	}
}

func TestLock_TryLockErrorDiscardsConnection(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT try_lock($1)")).WillReturnError(errors.New("boom"))

	_, err := m.Lock(context.Background())
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v, want try-lock failure", err)
	}
	if errors.Is(err, ErrLockTimeout) {
		t.Errorf("a query error is not a timeout: %v", err)
	}
	if got := db.Stats().OpenConnections; got != 1 {
		t.Errorf("OpenConnections = %d, want 1", got)
	}
}

func TestLock_SQLiteGrammarRunsNoQueries(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	db.SetMaxOpenConns(1) // usual SQLite configuration must not be rejected
	m := New(db, WithGrammar(grammars.NewSQLiteGrammar()))

	release, err := m.Lock(context.Background())
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if got := db.Stats().InUse; got != 0 {
		t.Errorf("connections in use = %d, want 0 (no pinned connection)", got)
	}
	if err := release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected queries: %v", err)
	}
}

func TestLock_WithoutLockRunsNoQueries(t *testing.T) {
	m, db, mock := newLockingMigrator(t, WithoutLock())
	defer db.Close()
	db.SetMaxOpenConns(1) // not rejected when locking is disabled

	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	// Only the usual Up queries, no lock.
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTxRecord(mock, "20240101000000_create_users", 1)

	if err := m.Up(); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLock_MaxOpenConnsOneIsRejected(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()
	db.SetMaxOpenConns(1)

	err := m.Up()
	if err == nil {
		t.Fatal("expected error for MaxOpenConns=1")
	}
	for _, want := range []string{"MaxOpenConns", "WithoutLock", "--no-lock"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected queries: %v", err)
	}
}

func TestUp_AcquiresLockBeforeTrackingTableDDL(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()

	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	// sqlmock matches in order: the lock comes before CREATE TABLE IF NOT
	// EXISTS and before the pending list is read, and is released last.
	expectTryLock(mock, true)
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTxRecord(mock, "20240101000000_create_users", 1)
	expectUnlock(mock, true)

	if err := m.Up(); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUp_ReleasesLockOnFailure(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()

	if err := m.Register("20240101000000_create_users", &failingMigration{}); err != nil {
		t.Fatal(err)
	}
	expectTryLock(mock, true)
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	mock.ExpectBegin()
	mock.ExpectRollback()
	expectUnlock(mock, true)

	if err := m.Up(); err == nil {
		t.Fatal("expected migration error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUp_UnlockFailureIsLoggedNotReturned(t *testing.T) {
	l := &recordingLogger{}
	m, db, mock := newLockingMigrator(t, WithLogger(l))
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	expectTryLock(mock, true)
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT unlock($1)")).WillReturnError(errors.New("gone"))

	if err := m.Up(); err != nil {
		t.Fatalf("Up: %v (an unlock failure must not fail a successful Up)", err)
	}
	if len(l.errors) != 1 {
		t.Errorf("logged errors = %v, want one", l.errors)
	}
}

func TestRefresh_HoldsOneLockAcrossBothPhases(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()

	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	expectTryLock(mock, true)
	// Reset phase.
	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()}})
	expectMigrationTxRemove(mock, "20240101000000_create_users")
	// Up phase.
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectMigrationTxRecord(mock, "20240101000000_create_users", 1)
	expectUnlock(mock, true)

	if err := m.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFresh_AcquiresLockBeforeDropping(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	expectTryLock(mock, true)
	mock.ExpectQuery("LIST TABLES").WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("users"))
	mock.ExpectExec("DROP TABLE users").WillReturnResult(sqlmock.NewResult(0, 0))
	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectUnlock(mock, true)

	if err := m.Fresh(); err != nil {
		t.Fatalf("Fresh: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestStatus_TakesNoLockAndIsReadOnly(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()

	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	if err := m.Register("20240102000000_create_posts", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	// Tracking table missing: only the existence check runs, no DDL.
	expectMockHasTable(mock, false)

	st, err := m.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(st) != 2 {
		t.Fatalf("len = %d, want 2", len(st))
	}
	for _, s := range st {
		if s.Applied || s.Batch != 0 || s.AppliedAt != nil {
			t.Errorf("%s reported applied: %+v", s.Name, s)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestStatus_DryRunTableMissingReportsPending(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	m := New(db, WithGrammar(grammars.NewPostgresGrammar()), WithDryRun(&strings.Builder{}))
	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	expectTrackingTableExists(mock, false)

	st, err := m.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(st) != 1 || st[0].Applied {
		t.Errorf("Status = %+v, want one pending migration", st)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestStatus_TableExistsCheckError(t *testing.T) {
	m, db, mock := newLockingMigrator(t)
	defer db.Close()
	mock.ExpectQuery("^SELECT 1$").WillReturnError(errors.New("permission denied"))

	if _, err := m.Status(); !errors.Is(err, ErrTrackingTable) {
		t.Errorf("err = %v, want ErrTrackingTable", err)
	}
}
