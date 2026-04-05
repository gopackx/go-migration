package migrator

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"strings"
)

func TestNewTracker(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	tracker := NewTracker(db, "migrations")
	if tracker == nil {
		t.Errorf("expected non-nil")
	}
}

func TestEnsureTable_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	tracker := NewTracker(db, "migrations")
	err = tracker.EnsureTable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestEnsureTable_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnError(errors.New("db error"))

	tracker := NewTracker(db, "migrations")
	err = tracker.EnsureTable()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}

func TestEnsureTable_CustomTableName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS custom_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	tracker := NewTracker(db, "custom_migrations")
	err = tracker.EnsureTable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetApplied_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
		AddRow("20240115000000_first", 1, now).
		AddRow("20240215000000_second", 1, now)

	mock.ExpectQuery("SELECT migration, batch, created_at FROM migrations ORDER BY migration ASC").
		WillReturnRows(rows)

	tracker := NewTracker(db, "migrations")
	records, err := tracker.GetApplied()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want %d", len(records), 2)
	}
	if "20240115000000_first" != records[0].Name {
		t.Errorf("got %v, want %v", records[0].Name, "20240115000000_first")
	}
	if 1 != records[0].Batch {
		t.Errorf("got %v, want %v", records[0].Batch, 1)
	}
	if "20240215000000_second" != records[1].Name {
		t.Errorf("got %v, want %v", records[1].Name, "20240215000000_second")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetApplied_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"})
	mock.ExpectQuery("SELECT migration, batch, created_at FROM migrations ORDER BY migration ASC").
		WillReturnRows(rows)

	tracker := NewTracker(db, "migrations")
	records, err := tracker.GetApplied()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty, got %v", records)
	}
}

func TestGetApplied_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT migration, batch, created_at FROM migrations").
		WillReturnError(errors.New("db error"))

	tracker := NewTracker(db, "migrations")
	_, err = tracker.GetApplied()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}

func TestGetByBatch_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
		AddRow("20240115000000_first", 2, now).
		AddRow("20240215000000_second", 2, now)

	mock.ExpectQuery("SELECT migration, batch, created_at FROM migrations WHERE batch = \\$1 ORDER BY migration ASC").
		WithArgs(2).
		WillReturnRows(rows)

	tracker := NewTracker(db, "migrations")
	records, err := tracker.GetByBatch(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want %d", len(records), 2)
	}
	if 2 != records[0].Batch {
		t.Errorf("got %v, want %v", records[0].Batch, 2)
	}
	if 2 != records[1].Batch {
		t.Errorf("got %v, want %v", records[1].Batch, 2)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetByBatch_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"migration", "batch", "created_at"})
	mock.ExpectQuery("SELECT migration, batch, created_at FROM migrations WHERE batch = \\$1").
		WithArgs(99).
		WillReturnRows(rows)

	tracker := NewTracker(db, "migrations")
	records, err := tracker.GetByBatch(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty, got %v", records)
	}
}

func TestGetByBatch_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT migration, batch, created_at FROM migrations WHERE batch = \\$1").
		WithArgs(1).
		WillReturnError(errors.New("db error"))

	tracker := NewTracker(db, "migrations")
	_, err = tracker.GetByBatch(1)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}

func TestGetLastBatchNumber_WithRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"max"}).AddRow(3)
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM migrations").
		WillReturnRows(rows)

	tracker := NewTracker(db, "migrations")
	batch, err := tracker.GetLastBatchNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if 3 != batch {
		t.Errorf("got %v, want %v", batch, 3)
	}
}

func TestGetLastBatchNumber_NoRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"max"}).AddRow(0)
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM migrations").
		WillReturnRows(rows)

	tracker := NewTracker(db, "migrations")
	batch, err := tracker.GetLastBatchNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if 0 != batch {
		t.Errorf("got %v, want %v", batch, 0)
	}
}

func TestGetLastBatchNumber_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM migrations").
		WillReturnError(errors.New("db error"))

	tracker := NewTracker(db, "migrations")
	_, err = tracker.GetLastBatchNumber()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}

func TestRecord_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO migrations").
		WithArgs("20240115000000_create_users", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tracker := NewTracker(db, "migrations")
	err = tracker.Record("20240115000000_create_users", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestRecord_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO migrations").
		WithArgs("20240115000000_create_users", 1).
		WillReturnError(errors.New("duplicate key"))

	tracker := NewTracker(db, "migrations")
	err = tracker.Record("20240115000000_create_users", 1)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
	if !strings.Contains(err.Error(), "20240115000000_create_users") {
		t.Errorf("%q not in %q", "20240115000000_create_users", err.Error())
	}
}

func TestRemove_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM migrations WHERE migration = \\$1").
		WithArgs("20240115000000_create_users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	tracker := NewTracker(db, "migrations")
	err = tracker.Remove("20240115000000_create_users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestRemove_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM migrations WHERE migration = \\$1").
		WithArgs("20240115000000_create_users").
		WillReturnError(errors.New("db error"))

	tracker := NewTracker(db, "migrations")
	err = tracker.Remove("20240115000000_create_users")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
	if !strings.Contains(err.Error(), "20240115000000_create_users") {
		t.Errorf("%q not in %q", "20240115000000_create_users", err.Error())
	}
}
