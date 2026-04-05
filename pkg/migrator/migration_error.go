package migrator

import "fmt"

const maxSQLLength = 500

// MigrationError menyimpan konteks kegagalan migrasi.
type MigrationError struct {
	MigrationName string // nama migrasi yang gagal
	SQL           string // statement SQL yang menyebabkan kegagalan
	Position      string // informasi posisi dari DB (opsional)
	Cause         error  // error asli dari database driver
}

func (e *MigrationError) Error() string {
	sql := e.SQL
	if len(sql) > maxSQLLength {
		sql = sql[:maxSQLLength] + "... [truncated]"
	}

	msg := fmt.Sprintf("migration %q failed", e.MigrationName)
	if sql != "" {
		msg += fmt.Sprintf("\n  SQL: %s", sql)
	}
	if e.Position != "" {
		msg += fmt.Sprintf("\n  Position: %s", e.Position)
	}
	msg += fmt.Sprintf("\n  Cause: %s", e.Cause)
	return msg
}

func (e *MigrationError) Unwrap() error {
	return e.Cause
}

func wrapMigrationError(migrationName, sql string, err error) *MigrationError {
	return &MigrationError{
		MigrationName: migrationName,
		SQL:           sql,
		Cause:         err,
	}
}
