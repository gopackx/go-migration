package database

import "database/sql"

// applyPoolConfig configures connection pool parameters on a *sql.DB handle.
// Zero values are ignored, leaving the sql.DB defaults in place.
func applyPoolConfig(db *sql.DB, config ConnectionConfig) {
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
