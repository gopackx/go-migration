package commands

import (
	"database/sql"

	"github.com/gopackx/go-migration/internal/generator"
	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/seeder"
)

// Migrator is the set of migration operations the CLI commands use. It is
// satisfied directly by *migrator.Migrator; it is an interface only so that
// the command tests can substitute fakes.
type Migrator interface {
	Up() error
	Rollback(steps int) error
	Reset() error
	Refresh() error
	Fresh() error
	Status() ([]migrator.MigrationStatus, error)
}

// *migrator.Migrator must satisfy Migrator without an adapter.
var _ Migrator = (*migrator.Migrator)(nil)

// TrackerCreator creates the migration tracking table. It is satisfied by
// *migrator.Tracker.
type TrackerCreator interface {
	EnsureTable() error
}

var _ TrackerCreator = (*migrator.Tracker)(nil)

// Locker acquires the cross-process migration lock. Lock blocks until the
// lock is held (or fails, e.g. on timeout) and returns a function releasing
// it. The CLI implements it on top of migrator.Migrator.Lock; migrate:install
// and the db:seed* commands hold it while they run, so that they never run
// concurrently with each other or with migrations.
type Locker interface {
	Lock() (release func() error, err error)
}

// CommandContext holds the dependencies needed by CLI command handlers.
// It mirrors the fields of cli.CLIContext but lives in the commands package
// to avoid an import cycle between pkg/cli and pkg/cli/commands.
type CommandContext struct {
	DB             *sql.DB
	Migrator       Migrator
	Seeder         *seeder.Runner
	Generator      *generator.Generator
	TrackerEnsurer TrackerCreator
	// Locker, when non-nil, is held by migrate:install and the db:seed*
	// commands while they run.
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
