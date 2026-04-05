package commands

import (
	"database/sql"
	"time"

	"github.com/gopackx/go-migration/internal/generator"
	"github.com/gopackx/go-migration/pkg/seeder"
)

// MigratorRunner defines the migration operations needed by CLI commands.
// It is satisfied by *migrator.Migrator (via MigratorAdapter) but defined
// here as an interface to avoid an import cycle between commands and migrator.
type MigratorRunner interface {
	Up() error
	Rollback(steps int) error
	Reset() error
	Refresh() error
	Fresh() error
	Status() ([]MigrationStatusInfo, error)
}

// MigrationStatusInfo holds the status of a single migration.
// It mirrors migrator.MigrationStatus without importing the package.
type MigrationStatusInfo struct {
	Name      string
	Applied   bool
	Batch     int
	AppliedAt *time.Time
}

// TrackerCreator creates a migration tracker for the given DB.
type TrackerCreator interface {
	EnsureTable() error
}

// CommandContext holds the dependencies needed by CLI command handlers.
// It mirrors the fields of cli.CLIContext but lives in the commands package
// to avoid an import cycle between pkg/cli and pkg/cli/commands.
type CommandContext struct {
	DB             *sql.DB
	Migrator       MigratorRunner
	Seeder         *seeder.Runner
	Generator      *generator.Generator
	TrackerEnsurer TrackerCreator
}
