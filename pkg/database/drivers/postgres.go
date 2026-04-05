package drivers

import (
	"database/sql"
	"fmt"

	"github.com/gopackx/go-migration/pkg/database"
)

// PostgresDriver implements the Driver interface for PostgreSQL.
type PostgresDriver struct{}

// NewPostgresDriver creates a new PostgresDriver.
func NewPostgresDriver() *PostgresDriver {
	return &PostgresDriver{}
}

// Name returns the driver name used for sql.Open registration.
func (d *PostgresDriver) Name() string {
	return "postgres"
}

// Open creates a PostgreSQL database connection using the provided config.
// It builds a DSN in the format: host=X port=Y user=Z password=W dbname=D sslmode=disable
func (d *PostgresDriver) Open(config database.ConnectionConfig) (*sql.DB, error) {
	dsn := d.buildDSN(config)

	db, err := sql.Open(d.Name(), dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}

	configurePool(db, config)

	return db, nil
}

// buildDSN constructs a PostgreSQL connection string from the config.
func (d *PostgresDriver) buildDSN(config database.ConnectionConfig) string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database,
	)

	// Default sslmode to disable if not specified in options
	sslmode := "disable"
	if config.Options != nil {
		if val, ok := config.Options["sslmode"]; ok {
			sslmode = val
		}
	}
	dsn += fmt.Sprintf(" sslmode=%s", sslmode)

	// Append any additional options
	if config.Options != nil {
		for key, val := range config.Options {
			if key == "sslmode" {
				continue // already handled
			}
			dsn += fmt.Sprintf(" %s=%s", key, val)
		}
	}

	return dsn
}
