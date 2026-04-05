package database

import "errors"

// Sentinel errors for the database connection manager.
// Defined locally to avoid circular dependencies with pkg/migrator.
var (
	// ErrConnectionNotFound is returned when a named connection does not exist.
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrConnectionFailed is returned when a connection cannot be established.
	ErrConnectionFailed = errors.New("database connection failed")

	// ErrDriverNotFound is returned when a driver is not registered for the given name.
	ErrDriverNotFound = errors.New("database driver not found")

	// ErrNoDefault is returned when no default connection has been set.
	ErrNoDefault = errors.New("no default connection set")

	// ErrTransactionFailed is returned when a transaction operation (begin, commit, or rollback) fails.
	ErrTransactionFailed = errors.New("transaction failed")
)
