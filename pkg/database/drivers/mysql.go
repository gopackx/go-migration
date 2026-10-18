package drivers

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/gopackx/go-migration/pkg/database"
)

// MySQLDriver implements the Driver interface for MySQL.
type MySQLDriver struct{}

// NewMySQLDriver creates a new MySQLDriver.
func NewMySQLDriver() *MySQLDriver {
	return &MySQLDriver{}
}

// Name returns the driver name used for sql.Open registration.
func (d *MySQLDriver) Name() string {
	return "mysql"
}

// Open creates a MySQL database connection using the provided config.
// It builds a DSN in the format: user:password@tcp(host:port)/dbname?params
func (d *MySQLDriver) Open(config database.ConnectionConfig) (*sql.DB, error) {
	dsn := d.buildDSN(config)

	db, err := sql.Open(d.Name(), dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}

	configurePool(db, config)

	return db, nil
}

// buildDSN constructs a MySQL connection string from the config.
//
// parseTime=true is added unless the caller set parseTime explicitly in
// Options: without it go-sql-driver/mysql returns DATETIME/TIMESTAMP columns
// (including the migrations table's created_at) as []byte instead of
// time.Time, which breaks scanning into time.Time.
func (d *MySQLDriver) buildDSN(config database.ConnectionConfig) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		config.Username, config.Password, config.Host, config.Port, config.Database,
	)

	hasParseTime := false
	params := make([]string, 0, len(config.Options)+1)
	for key, val := range config.Options {
		if strings.EqualFold(key, "parseTime") {
			hasParseTime = true
		}
		params = append(params, fmt.Sprintf("%s=%s", key, val))
	}
	if !hasParseTime {
		params = append(params, "parseTime=true")
	}
	// Sort for a deterministic DSN (map iteration order is random).
	sort.Strings(params)

	return dsn + "?" + strings.Join(params, "&")
}
