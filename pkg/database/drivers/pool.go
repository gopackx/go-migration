package drivers

import (
	"database/sql"

	"github.com/gopackx/go-migration/pkg/database"
)

// configurePool applies connection pool settings from the config to the database handle.
func configurePool(db *sql.DB, config database.ConnectionConfig) {
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
}
