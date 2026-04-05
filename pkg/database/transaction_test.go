package database

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"strings"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

// Req 10.1: Begin a database transaction before calling the migration method
// Req 10.2: Commit the transaction when the migration method returns nil
func TestWithTransaction_Success(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectCommit()

	called := false
	err := WithTransaction(db, func(tx *sql.Tx) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Errorf("fn should have been called")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// Req 10.3: Roll back the transaction when the migration method returns an error
func TestWithTransaction_FnError_Rollback(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	fnErr := errors.New("migration failed")
	err := WithTransaction(db, func(tx *sql.Tx) error {
		return fnErr
	})

	if !errors.Is(err, fnErr) {
		t.Errorf("got %v, want %v", err, fnErr)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// Req 10.5: If the transaction commit fails, attempt a rollback and return a descriptive error.
// Note: In Go's database/sql, a failed Commit marks the tx as done, so the
// subsequent Rollback returns sql.ErrTxDone (never reaches the driver).
// The implementation still attempts rollback as required by the spec.
func TestWithTransaction_CommitFails_ReturnsDescriptiveError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(fmt.Errorf("commit error"))

	err := WithTransaction(db, func(tx *sql.Tx) error {
		return nil
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTransactionFailed) {
		t.Errorf("got %v, want %v", err, ErrTransactionFailed)
	}
	if !strings.Contains(err.Error(), "commit") {
		t.Errorf("%q not in %q", "commit", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestWithTransaction_BeginFails(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin error"))

	err := WithTransaction(db, func(tx *sql.Tx) error {
		t.Fatal("fn should not be called when begin fails")
		return nil
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTransactionFailed) {
		t.Errorf("got %v, want %v", err, ErrTransactionFailed)
	}
	if !strings.Contains(err.Error(), "begin") {
		t.Errorf("%q not in %q", "begin", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestWithTransaction_FnError_RollbackFails(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(fmt.Errorf("rollback error"))

	fnErr := errors.New("migration failed")
	err := WithTransaction(db, func(tx *sql.Tx) error {
		return fnErr
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrTransactionFailed) {
		t.Errorf("got %v, want %v", err, ErrTransactionFailed)
	}
	if !strings.Contains(err.Error(), "migration failed") {
		t.Errorf("%q not in %q", "migration failed", err.Error())
	}
	if !strings.Contains(err.Error(), "rollback error") {
		t.Errorf("%q not in %q", "rollback error", err.Error())
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

// Verify the fn receives a valid *sql.Tx that can execute queries
func TestWithTransaction_FnReceivesTx(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO test").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := WithTransaction(db, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test VALUES (1)")
		return err
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}
