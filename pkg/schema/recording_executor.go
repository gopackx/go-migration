package schema

import "database/sql"

// Compile-time check that RecordingExecutor implements Executor.
var _ Executor = (*RecordingExecutor)(nil)

// RecordingExecutor wraps an inner Executor, recording the last SQL query
// executed. This is used by the Runner to capture the failing SQL statement
// when wrapping errors in MigrationError.
type RecordingExecutor struct {
	inner   Executor
	LastSQL string
}

// NewRecordingExecutor creates a RecordingExecutor wrapping the given executor.
func NewRecordingExecutor(inner Executor) *RecordingExecutor {
	return &RecordingExecutor{inner: inner}
}

func (r *RecordingExecutor) Exec(query string, args ...any) (sql.Result, error) {
	r.LastSQL = query
	return r.inner.Exec(query, args...)
}

func (r *RecordingExecutor) QueryRow(query string, args ...any) *sql.Row {
	r.LastSQL = query
	return r.inner.QueryRow(query, args...)
}
