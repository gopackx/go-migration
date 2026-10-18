package commands

import (
	"flag"
	"time"
)

// Names of the per-command flags controlling the cross-process migration
// lock. They are registered on every command that changes the database
// (migrate, migrate:rollback, migrate:reset, migrate:refresh, migrate:fresh,
// db:seed, db:seed:rollback, db:seed:truncate) and read by the CLI pre-run
// hook, which configures the Migrator accordingly.
const (
	LockTimeoutFlag = "lock-timeout"
	NoLockFlag      = "no-lock"
)

// DefaultLockTimeout is the default of --lock-timeout. It mirrors
// migrator.DefaultLockTimeout (this package cannot import migrator).
const DefaultLockTimeout = 30 * time.Second

// AddLockFlags registers --lock-timeout and --no-lock on fs.
func AddLockFlags(fs *flag.FlagSet) {
	fs.Duration(LockTimeoutFlag, DefaultLockTimeout, "how long to wait for the migration lock held by another process (e.g. 30s, 2m)")
	fs.Bool(NoLockFlag, false, "do not take the cross-process migration lock")
}
