package migrator

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
)

// --- Tracker dialects ---

func TestTracker_MySQLGrammar_UsesQuestionMarkPlaceholders(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tr := NewTrackerWithGrammar(db, "migrations", grammars.NewMySQLGrammar())

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS migrations \(\s*id\s+INT AUTO_INCREMENT PRIMARY KEY`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT migration, batch, created_at FROM migrations WHERE batch = ? ORDER BY migration ASC")).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO migrations (migration, batch) VALUES (?, ?)")).
		WithArgs("20240101000000_create_users", 3).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM migrations WHERE migration = ?")).
		WithArgs("20240101000000_create_users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := tr.EnsureTable(); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}
	if _, err := tr.GetByBatch(3); err != nil {
		t.Fatalf("GetByBatch: %v", err)
	}
	if err := tr.Record("20240101000000_create_users", 3); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := tr.Remove("20240101000000_create_users"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestTracker_SQLiteGrammar_TrackingTableDDL(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tr := NewTrackerWithGrammar(db, "migrations", grammars.NewSQLiteGrammar())
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS migrations \(\s*id\s+INTEGER PRIMARY KEY AUTOINCREMENT`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO migrations (migration, batch) VALUES (?, ?)")).
		WithArgs("x", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := tr.EnsureTable(); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}
	if err := tr.Record("x", 1); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestNewTracker_DefaultsToPostgres(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tr := NewTracker(db, "migrations")
	mock.ExpectExec(`SERIAL PRIMARY KEY`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO migrations (migration, batch) VALUES ($1, $2)")).
		WithArgs("x", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := tr.EnsureTable(); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}
	if err := tr.Record("x", 1); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// created_at must scan whether the driver returns time.Time, []byte (MySQL
// without parseTime=true) or string (SQLite text).
func TestTracker_GetApplied_ScansTextTimestamps(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tr := NewTrackerWithGrammar(db, "migrations", grammars.NewMySQLGrammar())
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	mock.ExpectQuery("SELECT migration, batch, created_at FROM migrations").
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
			AddRow("a", 1, []byte("2024-01-02 03:04:05")).
			AddRow("b", 1, "2024-01-02T03:04:05Z").
			AddRow("c", 1, now))

	recs, err := tr.GetApplied()
	if err != nil {
		t.Fatalf("GetApplied: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("len = %d, want 3", len(recs))
	}
	for _, r := range recs {
		if !r.CreatedAt.Equal(now) {
			t.Errorf("%s: CreatedAt = %v, want %v", r.Name, r.CreatedAt, now)
		}
	}
}

// --- Tracker errors keep the driver cause ---

func TestTracker_ErrorsWrapDriverCause(t *testing.T) {
	cause := errors.New("pq: permission denied for schema public")

	tests := []struct {
		name   string
		expect func(sqlmock.Sqlmock)
		call   func(*Tracker) error
	}{
		{"EnsureTable", func(m sqlmock.Sqlmock) { m.ExpectExec("CREATE TABLE").WillReturnError(cause) },
			func(tr *Tracker) error { return tr.EnsureTable() }},
		{"GetApplied", func(m sqlmock.Sqlmock) { m.ExpectQuery("SELECT").WillReturnError(cause) },
			func(tr *Tracker) error { _, err := tr.GetApplied(); return err }},
		{"GetByBatch", func(m sqlmock.Sqlmock) { m.ExpectQuery("SELECT").WillReturnError(cause) },
			func(tr *Tracker) error { _, err := tr.GetByBatch(1); return err }},
		{"GetLastBatchNumber", func(m sqlmock.Sqlmock) { m.ExpectQuery("SELECT").WillReturnError(cause) },
			func(tr *Tracker) error { _, err := tr.GetLastBatchNumber(); return err }},
		{"Record", func(m sqlmock.Sqlmock) { m.ExpectExec("INSERT").WillReturnError(cause) },
			func(tr *Tracker) error { return tr.Record("x", 1) }},
		{"Remove", func(m sqlmock.Sqlmock) { m.ExpectExec("DELETE").WillReturnError(cause) },
			func(tr *Tracker) error { return tr.Remove("x") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()
			tt.expect(mock)

			err := tt.call(NewTracker(db, "migrations"))
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, ErrTrackingTable) {
				t.Errorf("errors.Is(err, ErrTrackingTable) = false: %v", err)
			}
			if !errors.Is(err, cause) {
				t.Errorf("errors.Is(err, cause) = false: %v", err)
			}
			if !strings.Contains(err.Error(), "permission denied for schema public") {
				t.Errorf("cause text missing from %q", err.Error())
			}
		})
	}
}

func TestTracker_ScanErrorWrapsCause(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"migration", "batch", "created_at"}).AddRow("a", "not-a-number", time.Now()))

	_, err := NewTracker(db, "migrations").GetApplied()
	if err == nil || !errors.Is(err, ErrTrackingTable) {
		t.Fatalf("got %v, want ErrTrackingTable", err)
	}
	if !strings.Contains(err.Error(), "not-a-number") {
		t.Errorf("scan cause missing from %q", err.Error())
	}
}

// --- Option order: grammar reaches the tracker regardless of order ---

func TestNew_GrammarReachesTracker_AnyOptionOrder(t *testing.T) {
	orders := map[string][]Option{
		"table_then_grammar": {WithTableName("schema_migrations"), WithGrammar(grammars.NewMySQLGrammar())},
		"grammar_then_table": {WithGrammar(grammars.NewMySQLGrammar()), WithTableName("schema_migrations")},
	}
	for name, opts := range orders {
		t.Run(name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			m := New(db, opts...)
			mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations .*INT AUTO_INCREMENT PRIMARY KEY`).
				WillReturnResult(sqlmock.NewResult(0, 0))
			if err := m.tracker.EnsureTable(); err != nil {
				t.Fatalf("EnsureTable: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
			if m.runner.grammar == nil {
				t.Errorf("runner grammar not set")
			}
		})
	}
}

func TestNew_LoggerBeforeGrammar_ReachesRunner(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	l := &testLogger{}
	m := New(db, WithLogger(l), WithGrammar(&mockGrammar{}))
	if m.runner.logger != l {
		t.Errorf("logger not propagated to runner")
	}
	if m.runner.grammar == nil {
		t.Errorf("grammar not propagated to runner")
	}
}

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Error(string, ...any) {}

// --- Reset runs before hooks ---

func TestReset_RunsBeforeHooks(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	if err := m.Register("20240102000000_create_posts", &noopMigration{}); err != nil {
		t.Fatal(err)
	}

	var calls []string
	m.BeforeMigrate(func(name, direction string) error {
		calls = append(calls, name+":"+direction)
		return nil
	})

	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
		{Name: "20240102000000_create_posts", Batch: 1, CreatedAt: time.Now()},
	})
	expectMigrationTxRemove(mock, "20240102000000_create_posts")
	expectMigrationTxRemove(mock, "20240101000000_create_users")

	if err := m.Reset(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"20240102000000_create_posts:down", "20240101000000_create_users:down"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Errorf("before hook calls = %v, want %v", calls, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestReset_BeforeHookError_Aborts(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	mig := &noopMigration{}
	if err := m.Register("20240101000000_create_users", mig); err != nil {
		t.Fatal(err)
	}
	m.BeforeMigrate(func(name, direction string) error { return errors.New("hook failed") })

	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
	})

	err := m.Reset()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), `before hook for "20240101000000_create_users"`) ||
		!strings.Contains(err.Error(), "hook failed") {
		t.Errorf("unexpected error text: %q", err.Error())
	}
	if mig.downCalled {
		t.Errorf("Down must not run when a before hook fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Tracking writes share the migration transaction ---

func TestUp_RecordFailure_RollsBackMigrationTx(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO migrations").
		WithArgs("20240101000000_create_users", 1).
		WillReturnError(errors.New("duplicate key"))
	mock.ExpectRollback()

	err := m.Up()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrTrackingTable) || !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRollback_RemoveFailure_RollsBackMigrationTx(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
	})
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM migrations").
		WithArgs("20240101000000_create_users").
		WillReturnError(errors.New("lock timeout"))
	mock.ExpectRollback()

	err := m.Rollback(1)
	if err == nil || !errors.Is(err, ErrTrackingTable) {
		t.Fatalf("got %v, want ErrTrackingTable", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUp_NoTransactionMigration_RecordsOnDBAfterSuccess(t *testing.T) {
	m, db, mock := newTestMigrator(t)
	defer db.Close()

	mig := &noTxMigration{}
	if err := m.Register("20240101000000_create_index_concurrently", mig); err != nil {
		t.Fatal(err)
	}

	expectEnsureTable(mock)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	// No Begin/Commit: the record is written directly on the DB.
	expectRecord(mock, "20240101000000_create_index_concurrently", 1)

	if err := m.Up(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mig.upCalled {
		t.Errorf("Up was not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRunner_ExecuteWithHook_RunsInsideTx(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	r := NewRunner(db, &mockGrammar{}, nil)

	mock.ExpectBegin()
	mock.ExpectExec("HOOK").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	var gotExecutor schema.Executor
	err := r.ExecuteWithHook(&noopMigration{}, "up", "20240101000000_x", func(ex schema.Executor) error {
		gotExecutor = ex
		_, err := ex.Exec("HOOK")
		return err
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotExecutor == nil {
		t.Fatalf("hook was not called")
	}
	if gotExecutor == schema.Executor(db) {
		t.Errorf("hook received *sql.DB, want the migration's *sql.Tx")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRunner_ExecuteWithHook_NotCalledOnMigrationError(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	r := NewRunner(db, &mockGrammar{}, nil)
	mock.ExpectBegin()
	mock.ExpectRollback()

	called := false
	err := r.ExecuteWithHook(&failingMigration{}, "up", "20240101000000_x", func(schema.Executor) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if called {
		t.Errorf("hook must not run when the migration fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRunner_CommitFailure_KeepsCause(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("connection reset"))

	err := NewRunner(db, &mockGrammar{}, nil).Execute(&noopMigration{}, "up")
	if !errors.Is(err, ErrTransactionFailed) {
		t.Errorf("errors.Is(err, ErrTransactionFailed) = false: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Errorf("commit cause missing: %v", err)
	}
}

// --- Dry-run: HasTable inside a migration must not panic ---

type hasTableMigration struct{ exists bool }

func (m *hasTableMigration) Up(s *schema.Builder) error {
	ok, err := s.HasTable("users")
	if err != nil {
		return err
	}
	m.exists = ok
	if !ok {
		return s.Create("users", func(bp *schema.Blueprint) { bp.ID() })
	}
	return nil
}
func (m *hasTableMigration) Down(s *schema.Builder) error { return s.DropIfExists("users") }

func TestUp_DryRun_HasTableDoesNotPanic(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	var buf bytes.Buffer
	m := New(db, WithGrammar(grammars.NewPostgresGrammar()), WithDryRun(&buf))
	mig := &hasTableMigration{}
	if err := m.Register("20240101000000_create_users", mig); err != nil {
		t.Fatal(err)
	}

	expectPGLock(mock)
	expectTrackingTableExists(mock, true)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectPGUnlock(mock)

	if err := m.Up(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mig.exists {
		t.Errorf("dry-run HasTable should report false")
	}
	out := buf.String()
	if !strings.Contains(out, "information_schema.tables") || !strings.Contains(out, `CREATE TABLE "users"`) {
		t.Errorf("unexpected dry-run output: %q", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Dry-run never executes DDL/DML ---

// expectTrackingTableExists expects the Postgres HasTable query for the
// "migrations" tracking table and returns the given result.
func expectTrackingTableExists(mock sqlmock.Sqlmock, exists bool) {
	count := 0
	if exists {
		count = 1
	}
	mock.ExpectQuery(regexp.QuoteMeta(grammars.NewPostgresGrammar().CompileHasTable("migrations"))).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
}

// expectPGLock expects the PostgreSQL migration lock for the default
// "migrations" tracking table to be acquired on the first attempt.
func expectPGLock(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs("go-migration:migrations").
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
}

// expectPGUnlock expects the PostgreSQL migration lock to be released.
func expectPGUnlock(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
		WithArgs("go-migration:migrations").
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))
}

func TestDryRun_TrackingTableMissing_ErrorsWithoutDDL(t *testing.T) {
	ops := map[string]func(*Migrator) error{
		"Up":       func(m *Migrator) error { return m.Up() },
		"Rollback": func(m *Migrator) error { return m.Rollback(0) },
		"Reset":    func(m *Migrator) error { return m.Reset() },
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			var buf bytes.Buffer
			m := New(db, WithGrammar(grammars.NewPostgresGrammar()), WithDryRun(&buf))
			if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
				t.Fatal(err)
			}

			// Only the lock and the existence check may run: no CREATE
			// TABLE, nothing else.
			expectPGLock(mock)
			expectTrackingTableExists(mock, false)
			expectPGUnlock(mock)

			err := op(m)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, ErrTrackingTable) {
				t.Errorf("errors.Is(err, ErrTrackingTable) = false: %v", err)
			}
			if !strings.Contains(err.Error(), `dry-run: migration tracking table "migrations" does not exist; run migrate:install first`) {
				t.Errorf("unexpected error text: %q", err.Error())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
			if buf.Len() != 0 {
				t.Errorf("nothing should be printed, got %q", buf.String())
			}
		})
	}
}

func TestDryRun_Up_TableExists_PrintsSQLWithoutWrites(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	var buf bytes.Buffer
	m := New(db, WithGrammar(grammars.NewPostgresGrammar()), WithDryRun(&buf))
	if err := m.Register("20240101000000_create_users", &hasTableMigration{}); err != nil {
		t.Fatal(err)
	}

	// Read-only queries only: no CREATE TABLE IF NOT EXISTS, no BEGIN, no
	// INSERT. Dry-run still takes the migration lock.
	expectPGLock(mock)
	expectTrackingTableExists(mock, true)
	expectGetApplied(mock, nil)
	expectMaxBatch(mock, 0)
	expectPGUnlock(mock)

	if err := m.Up(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"-- Migration: 20240101000000_create_users", `CREATE TABLE "users"`} {
		if !strings.Contains(out, want) {
			t.Errorf("%q not in dry-run output %q", want, out)
		}
	}
	if strings.Contains(out, "INSERT INTO") {
		t.Errorf("dry-run output must not contain tracker writes: %q", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDryRun_Rollback_TableExists_NoDelete(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	var buf bytes.Buffer
	m := New(db, WithGrammar(grammars.NewPostgresGrammar()), WithDryRun(&buf))
	if err := m.Register("20240101000000_create_users", &hasTableMigration{}); err != nil {
		t.Fatal(err)
	}

	expectPGLock(mock)
	expectTrackingTableExists(mock, true)
	expectGetApplied(mock, []MigrationRecord{
		{Name: "20240101000000_create_users", Batch: 1, CreatedAt: time.Now()},
	})
	expectPGUnlock(mock)

	if err := m.Rollback(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), `DROP TABLE IF EXISTS "users"`) {
		t.Errorf("expected Down SQL in output, got %q", buf.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestTracker_TableExists(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tr := NewTrackerWithGrammar(db, "migrations", grammars.NewMySQLGrammar())
	mock.ExpectQuery(regexp.QuoteMeta(grammars.NewMySQLGrammar().CompileHasTable("migrations"))).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("information_schema.tables").WillReturnError(errors.New("access denied"))

	ok, err := tr.TableExists()
	if err != nil || !ok {
		t.Fatalf("got (%v, %v), want (true, nil)", ok, err)
	}
	_, err = tr.TableExists()
	if !errors.Is(err, ErrTrackingTable) || !strings.Contains(err.Error(), "access denied") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Fresh discards the connection whose session state it changed ---

func TestDiscardConn_RemovesConnectionFromPool(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	spare, err := db.Conn(ctx) // keeps sqlmock's DSN registered
	if err != nil {
		t.Fatal(err)
	}
	defer spare.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := db.Stats().OpenConnections; got != 2 {
		t.Fatalf("OpenConnections = %d, want 2", got)
	}
	discardConn(conn)
	_ = conn.Close()
	if got := db.Stats().OpenConnections; got != 1 {
		t.Errorf("OpenConnections after discard = %d, want 1 (connection must not return to the pool)", got)
	}
}

func TestFresh_DiscardsDropConnection(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	spare, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer spare.Close()

	m := New(db, WithGrammar(&mockGrammar{}))
	mock.ExpectQuery("LIST TABLES").WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("users"))
	mock.ExpectExec("DROP TABLE users").WillReturnError(errors.New("boom"))

	if err := m.Fresh(); err == nil {
		t.Fatalf("expected error")
	}
	// Only the pinned spare remains: the drop connection was discarded even
	// on the error path, not returned to the idle pool.
	if got := db.Stats().OpenConnections; got != 1 {
		t.Errorf("OpenConnections = %d, want 1", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
