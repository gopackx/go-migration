package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gopackx/go-migration/pkg/database"
)

// SQLiteDriver implements the Driver interface for SQLite.
type SQLiteDriver struct{}

// NewSQLiteDriver creates a new SQLiteDriver.
func NewSQLiteDriver() *SQLiteDriver {
	return &SQLiteDriver{}
}

// Name returns the driver name used for sql.Open registration. The pure-Go
// modernc.org/sqlite driver registers itself as "sqlite"; applications must
// blank-import it (_ "modernc.org/sqlite").
func (d *SQLiteDriver) Name() string {
	return "sqlite"
}

// Open creates a SQLite database connection using the provided config.
// It uses the Database field as the file path (e.g. "./data.db" or ":memory:").
func (d *SQLiteDriver) Open(config database.ConnectionConfig) (*sql.DB, error) {
	dsn := d.buildDSN(config)

	db, err := sql.Open(d.Name(), dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}

	configurePool(db, config)

	return db, nil
}

// buildDSN constructs a SQLite connection string from the config.
func (d *SQLiteDriver) buildDSN(config database.ConnectionConfig) string {
	dsn := config.Database

	// Append options as query parameters
	if len(config.Options) > 0 {
		params := make([]string, 0, len(config.Options))
		for key, val := range config.Options {
			params = append(params, fmt.Sprintf("%s=%s", key, val))
		}
		dsn += "?" + strings.Join(params, "&")
	}

	return dsn
}
