package migrator

import "errors"

// Sentinel errors for common failure scenarios across the migration system.
// All errors support errors.Is() and errors.As() for inspection.
// Wrap these with fmt.Errorf("context: %w", ErrXxx) to preserve the error chain.
var (
	ErrConnectionFailed     = errors.New("database connection failed")
	ErrMigrationNotFound    = errors.New("migration not found")
	ErrDuplicateMigration   = errors.New("duplicate migration name")
	ErrInvalidMigrationName = errors.New("invalid migration name")
	ErrTransactionFailed    = errors.New("transaction failed")
	ErrTrackingTable        = errors.New("migration tracking table error")
	ErrDuplicateSeeder      = errors.New("duplicate seeder name")
	ErrInvalidSeederName    = errors.New("invalid seeder name")
	ErrCircularDependency   = errors.New("circular seeder dependency")
	ErrSeederNotFound       = errors.New("seeder not found")
	ErrUnsupportedType      = errors.New("unsupported column type")
	ErrConnectionNotFound   = errors.New("connection not found")
	ErrConfigValidation     = errors.New("configuration validation failed")
)
