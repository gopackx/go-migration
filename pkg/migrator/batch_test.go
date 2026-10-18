package migrator

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewBatchManager(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	tracker := NewTracker(db, "migrations")
	bm := NewBatchManager(tracker)
	if bm == nil {
		t.Errorf("expected non-nil")
	}
}

// --- NextBatchNumber ---

func TestNextBatchNumber_FirstBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM \"migrations\"").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(0))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	next, err := bm.NextBatchNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if 1 != next {
		t.Errorf("got %v, want %v", next, 1)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestNextBatchNumber_SubsequentBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM \"migrations\"").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	next, err := bm.NextBatchNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if 6 != next {
		t.Errorf("got %v, want %v", next, 6)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestNextBatchNumber_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM \"migrations\"").
		WillReturnError(errors.New("db error"))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	_, err = bm.NextBatchNumber()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}

// --- GetLastBatch ---

func TestGetLastBatch_WithRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)

	// GetLastBatchNumber
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM \"migrations\"").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(3))

	// GetByBatch(3)
	mock.ExpectQuery("SELECT migration, batch, created_at FROM \"migrations\" WHERE batch = \\$1").
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
			AddRow("20240315000000_add_posts", 3, now).
			AddRow("20240316000000_add_comments", 3, now))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	records, err := bm.GetLastBatch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want %d", len(records), 2)
	}
	if "20240315000000_add_posts" != records[0].Name {
		t.Errorf("got %v, want %v", records[0].Name, "20240315000000_add_posts")
	}
	if "20240316000000_add_comments" != records[1].Name {
		t.Errorf("got %v, want %v", records[1].Name, "20240316000000_add_comments")
	}
	if 3 != records[0].Batch {
		t.Errorf("got %v, want %v", records[0].Batch, 3)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetLastBatch_NoRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM \"migrations\"").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(0))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	records, err := bm.GetLastBatch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil, got %v", records)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetLastBatch_BatchNumberError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM \"migrations\"").
		WillReturnError(errors.New("db error"))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	_, err = bm.GetLastBatch()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}

func TestGetLastBatch_GetByBatchError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(batch\\), 0\\) FROM \"migrations\"").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(2))

	mock.ExpectQuery("SELECT migration, batch, created_at FROM \"migrations\" WHERE batch = \\$1").
		WithArgs(2).
		WillReturnError(errors.New("db error"))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	_, err = bm.GetLastBatch()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}

// --- GetLastNMigrations ---

func TestGetLastNMigrations_ReturnsReversed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)
	mock.ExpectQuery("SELECT migration, batch, created_at FROM \"migrations\" ORDER BY migration ASC").
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
			AddRow("20240101000000_first", 1, now).
			AddRow("20240201000000_second", 1, now).
			AddRow("20240301000000_third", 2, now).
			AddRow("20240401000000_fourth", 2, now))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	records, err := bm.GetLastNMigrations(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want %d", len(records), 2)
	}
	// Should be in reverse timestamp order
	if "20240401000000_fourth" != records[0].Name {
		t.Errorf("got %v, want %v", records[0].Name, "20240401000000_fourth")
	}
	if "20240301000000_third" != records[1].Name {
		t.Errorf("got %v, want %v", records[1].Name, "20240301000000_third")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetLastNMigrations_NExceedsTotal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)
	mock.ExpectQuery("SELECT migration, batch, created_at FROM \"migrations\" ORDER BY migration ASC").
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}).
			AddRow("20240101000000_first", 1, now).
			AddRow("20240201000000_second", 1, now))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	records, err := bm.GetLastNMigrations(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want %d", len(records), 2)
	}
	if "20240201000000_second" != records[0].Name {
		t.Errorf("got %v, want %v", records[0].Name, "20240201000000_second")
	}
	if "20240101000000_first" != records[1].Name {
		t.Errorf("got %v, want %v", records[1].Name, "20240101000000_first")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetLastNMigrations_ZeroN(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	bm := NewBatchManager(NewTracker(db, "migrations"))
	records, err := bm.GetLastNMigrations(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil, got %v", records)
	}
}

func TestGetLastNMigrations_NegativeN(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	bm := NewBatchManager(NewTracker(db, "migrations"))
	records, err := bm.GetLastNMigrations(-1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil, got %v", records)
	}
}

func TestGetLastNMigrations_NoApplied(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT migration, batch, created_at FROM \"migrations\" ORDER BY migration ASC").
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	records, err := bm.GetLastNMigrations(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil, got %v", records)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestGetLastNMigrations_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT migration, batch, created_at FROM \"migrations\" ORDER BY migration ASC").
		WillReturnError(errors.New("db error"))

	bm := NewBatchManager(NewTracker(db, "migrations"))
	_, err = bm.GetLastNMigrations(3)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTrackingTable) {
		t.Errorf("expected true")
	}
}
