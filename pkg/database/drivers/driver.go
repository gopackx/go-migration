package drivers

import (
	"database/sql"

	"github.com/gopackx/go-migration/pkg/database"
)

// Driver defines the contract for a database driver that can open connections.
type Driver interface {
	// Open creates a new database connection using the provided configuration.
	Open(config database.ConnectionConfig) (*sql.DB, error)
	// Name returns the driver name (e.g. "postgres", "mysql", "sqlite3").
	Name() string
}
