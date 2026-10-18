package database

import "database/sql"

// applyPoolConfig configures connection pool parameters on a *sql.DB handle.
// Zero values are ignored, leaving the sql.DB defaults in place.
//
// The built-in drivers in pkg/database/drivers already apply the same
// settings (drivers.configurePool) inside Driver.Open, so for them this call
// is redundant. It is kept deliberately: it is idempotent, and it guarantees
// the pool settings for third-party Driver implementations that do not
// configure the pool themselves.
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
