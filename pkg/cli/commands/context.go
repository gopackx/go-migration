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

// Locker acquires the cross-process migration lock. Lock blocks until the
// lock is held (or fails, e.g. on timeout) and returns a function releasing
// it. It is satisfied by an adapter over migrator.Migrator.Lock; the seed
// commands use it so that seeders never run concurrently with each other or
// with migrations.
type Locker interface {
	Lock() (release func() error, err error)
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
	// Locker, when non-nil, is held by the db:seed* commands while they run.
	Locker Locker
}

// withLock runs fn while holding ctx.Locker (if any). If fn succeeds but
// releasing the lock fails, the release error is returned.
func withLock(ctx *CommandContext, fn func() error) (err error) {
	if ctx.Locker == nil {
		return fn()
	}
	release, err := ctx.Locker.Lock()
	if err != nil {
		return err
	}
	defer func() {
		if rerr := release(); rerr != nil && err == nil {
			err = rerr
		}
	}()
	return fn()
}
