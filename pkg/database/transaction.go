package database

import (
	"database/sql"
	"fmt"
)

// TxFunc is a function that executes within a database transaction.
type TxFunc func(tx *sql.Tx) error

// WithTransaction begins a transaction, calls fn, commits on nil return,
// and rolls back on error. If commit fails, it attempts a rollback and
// returns a descriptive error.
func WithTransaction(db *sql.DB, fn TxFunc) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w: %v", ErrTransactionFailed, err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback after error: %w: fn error: %v, rollback error: %v", ErrTransactionFailed, err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil && rbErr != sql.ErrTxDone {
			return fmt.Errorf("rollback after commit failure: %w: commit error: %v, rollback error: %v", ErrTransactionFailed, err, rbErr)
		}
		return fmt.Errorf("commit: %w: %v", ErrTransactionFailed, err)
	}

	return nil
}
