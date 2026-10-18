package migrator

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/schema"
	"strings"
)

// --- Test helpers ---

// mockGrammar implements schema.Grammar for testing purposes.
type mockGrammar struct{}

func (g *mockGrammar) CompileCreate(bp *schema.Blueprint) ([]string, error) {
	return []string{fmt.Sprintf("CREATE TABLE %s ()", bp.Table())}, nil
}
func (g *mockGrammar) CompileAlter(bp *schema.Blueprint) ([]string, error) {
	return []string{fmt.Sprintf("ALTER TABLE %s", bp.Table())}, nil
}
func (g *mockGrammar) CompileDrop(table string) string { return "DROP TABLE " + table }
func (g *mockGrammar) CompileDropIfExists(table string) string {
	return "DROP TABLE IF EXISTS " + table
}
func (g *mockGrammar) CompileRename(from, to string) string {
	return "ALTER TABLE " + from + " RENAME TO " + to
}
func (g *mockGrammar) CompileHasTable(table string) string       { return "SELECT 1" }
func (g *mockGrammar) CompileHasTableIn(s, table string) string  { return "SELECT 1" }
func (g *mockGrammar) CompileHasColumn(table, col string) string { return "SELECT 1" }
func (g *mockGrammar) CompileListTables() string                 { return "LIST TABLES" }

// CompileDropAllTables returns one "DROP TABLE <name>" statement per table.
func (g *mockGrammar) CompileDropAllTables(tables []string) []string {
	stmts := make([]string, 0, len(tables))
	for _, t := range tables {
		stmts = append(stmts, "DROP TABLE "+t)
	}
	return stmts
}

// CompileTrackingTable, QuoteIdentifier and Placeholder mimic PostgreSQL so
// sqlmock expectations ("CREATE TABLE IF NOT EXISTS", `"migrations"`, "$1")
// match the SQL the tracker generates in production.
func (g *mockGrammar) CompileTrackingTable(table string) string {
	return "CREATE TABLE IF NOT EXISTS " + schema.QuoteQualified(g, table) + " (id SERIAL PRIMARY KEY, migration VARCHAR(255) NOT NULL UNIQUE, batch INTEGER NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)"
}
func (g *mockGrammar) QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
func (g *mockGrammar) Placeholder(index int) string { return fmt.Sprintf("$%d", index) }

// CompileTryLock and CompileUnlock return "" (no lock, like SQLite) so that
// the many sqlmock tests built on mockGrammar need no lock expectations. Lock
// behaviour is tested with lockingGrammar.
func (g *mockGrammar) CompileTryLock(key string) string { return "" }
func (g *mockGrammar) CompileUnlock(key string) string  { return "" }
func (g *mockGrammar) CompileColumnType(col schema.ColumnDefinition) (string, error) {
	return "TEXT", nil
}

// noopMigration succeeds on both Up and Down.
type noopMigration struct {
	upCalled   bool
	downCalled bool
}

func (m *noopMigration) Up(s *schema.Builder) error   { m.upCalled = true; return nil }
func (m *noopMigration) Down(s *schema.Builder) error { m.downCalled = true; return nil }

// failingMigration returns an error on both Up and Down.
type failingMigration struct{}

func (m *failingMigration) Up(s *schema.Builder) error   { return errors.New("up failed") }
func (m *failingMigration) Down(s *schema.Builder) error { return errors.New("down failed") }

// noTxMigration opts out of transaction wrapping.
type noTxMigration struct {
	upCalled   bool
	downCalled bool
}

func (m *noTxMigration) Up(s *schema.Builder) error   { m.upCalled = true; return nil }
func (m *noTxMigration) Down(s *schema.Builder) error { m.downCalled = true; return nil }
func (m *noTxMigration) DisableTransaction() bool     { return true }

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return db, mock
}

// --- Tests ---

func TestExecuteInTransaction_Up(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	runner := NewRunner(db, &mockGrammar{}, nil)
	m := &noopMigration{}

	err := runner.Execute(m, "up")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !m.upCalled {
		t.Errorf("expected true")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteInTransaction_Down(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	runner := NewRunner(db, &mockGrammar{}, nil)
	m := &noopMigration{}

	err := runner.Execute(m, "down")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !m.downCalled {
		t.Errorf("expected true")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteWithoutTransaction_OptOut(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	// No Begin/Commit expected — migration runs directly on db
	runner := NewRunner(db, &mockGrammar{}, nil)
	m := &noTxMigration{}

	err := runner.Execute(m, "up")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !m.upCalled {
		t.Errorf("expected true")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteInTransaction_MigrationError_TriggersRollback(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	runner := NewRunner(db, &mockGrammar{}, nil)
	m := &failingMigration{}

	err := runner.Execute(m, "up")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "up failed") {
		t.Errorf("%q not in %q", "up failed", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteInTransaction_CommitFailure_ReturnsTransactionFailed(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	runner := NewRunner(db, &mockGrammar{}, nil)
	m := &noopMigration{}

	err := runner.Execute(m, "up")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTransactionFailed) {
		t.Errorf("expected true")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteInTransaction_UnknownDirection(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	runner := NewRunner(db, &mockGrammar{}, nil)
	m := &noopMigration{}

	err := runner.Execute(m, "sideways")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown migration direction") {
		t.Errorf("%q not in %q", "unknown migration direction", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteInTransaction_BeginError(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	runner := NewRunner(db, &mockGrammar{}, nil)
	m := &noopMigration{}

	err := runner.ExecuteInTransaction(m, "up")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "begin transaction") {
		t.Errorf("%q not in %q", "begin transaction", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// --- Dry-run tests ---

// createTableMigration creates a table in Up and drops it in Down.
type createTableMigration struct{}

func (m *createTableMigration) Up(b *schema.Builder) error {
	return b.Create("users", func(bp *schema.Blueprint) {
		bp.String("name", 255)
	})
}

func (m *createTableMigration) Down(b *schema.Builder) error {
	return b.Drop("users")
}

func TestExecuteDryRun_WritesSQL_NoTransaction(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	var buf bytes.Buffer
	runner := NewRunner(db, &mockGrammar{}, nil)
	runner.SetDryRun(&buf)

	m := &noopMigration{}
	err := runner.Execute(m, "up")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !m.upCalled {
		t.Errorf("expected true")
	}
	// No Begin/Commit should have been called on the mock DB.
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteDryRun_WritesMigrationNamePrefix(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	var buf bytes.Buffer
	runner := NewRunner(db, &mockGrammar{}, nil)
	runner.SetDryRun(&buf)

	m := &createTableMigration{}
	err := runner.ExecuteDryRun(m, "up", "20240101_create_users")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "-- Migration: 20240101_create_users") {
		t.Errorf("%q not in %q", "-- Migration: 20240101_create_users", output)
	}
	if !strings.Contains(output, "CREATE TABLE") {
		t.Errorf("%q not in %q", "CREATE TABLE", output)
	}
	// No DB interactions expected.
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteDryRun_Down_WritesDropSQL(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	var buf bytes.Buffer
	runner := NewRunner(db, &mockGrammar{}, nil)
	runner.SetDryRun(&buf)

	m := &createTableMigration{}
	err := runner.ExecuteDryRun(m, "down", "20240101_create_users")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "-- Migration: 20240101_create_users") {
		t.Errorf("%q not in %q", "-- Migration: 20240101_create_users", output)
	}
	if !strings.Contains(output, "DROP TABLE") {
		t.Errorf("%q not in %q", "DROP TABLE", output)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestExecuteDryRun_NoTransactionCommitted(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	var buf bytes.Buffer
	runner := NewRunner(db, &mockGrammar{}, nil)
	runner.SetDryRun(&buf)

	// Even a migration that would normally use transactions should not
	// start any transaction in dry-run mode.
	m := &noopMigration{}
	err := runner.Execute(m, "up")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// mock has no expectations set — if Begin/Commit were called, it would fail.
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}
