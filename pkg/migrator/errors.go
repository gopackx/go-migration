package migrator

import (
	"errors"

	"github.com/gopackx/go-migration/pkg/database"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
	"github.com/gopackx/go-migration/pkg/seeder"
)

// Sentinel errors for common failure scenarios across the migration system.
// All errors support errors.Is() and errors.As() for inspection.
// Wrap these with fmt.Errorf("context: %w", ErrXxx) to preserve the error chain.
//
// Errors owned by another package are re-exported here as aliases of the
// canonical value, so errors.Is(err, migrator.ErrSeederNotFound) and
// errors.Is(err, seeder.ErrSeederNotFound) are equivalent. Configuration
// errors are not re-exported: use config.ErrConfigValidation, so that the
// library does not depend on the configuration loader.
var (
	ErrMigrationNotFound    = errors.New("migration not found")
	ErrDuplicateMigration   = errors.New("duplicate migration name")
	ErrInvalidMigrationName = errors.New("invalid migration name")
	ErrTrackingTable        = errors.New("migration tracking table error")

	// ErrLockTimeout is returned when the cross-process migration lock could
	// not be acquired within the lock timeout (see WithLockTimeout), usually
	// because another process is running migrations or seeders.
	ErrLockTimeout = errors.New("could not acquire migration lock")

	// Aliases of pkg/database sentinels.
	ErrConnectionFailed   = database.ErrConnectionFailed
	ErrConnectionNotFound = database.ErrConnectionNotFound
	ErrTransactionFailed  = database.ErrTransactionFailed

	// Aliases of pkg/seeder sentinels.
	ErrDuplicateSeeder    = seeder.ErrDuplicateSeeder
	ErrInvalidSeederName  = seeder.ErrInvalidSeederName
	ErrCircularDependency = seeder.ErrCircularDependency
	ErrSeederNotFound     = seeder.ErrSeederNotFound

	// Alias of the grammar sentinel returned for unsupported column types.
	ErrUnsupportedType = grammars.ErrUnsupportedType
)
