package commands

import (
	"flag"
	"fmt"
)

// RegisterMigrateInstallCommand registers the "migrate:install" command
// that creates the migration tracking table. The table is created while
// holding the migration lock (ctx.Locker, configured with --lock-timeout and
// --no-lock), so that concurrent installs and migrations do not race on
// CREATE TABLE IF NOT EXISTS.
func RegisterMigrateInstallCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext) {
	register("migrate:install", "Create the migration tracking table",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.TrackerEnsurer == nil {
				return fmt.Errorf("database connection not initialized")
			}
			return withLock(ctx, ctx.TrackerEnsurer.EnsureTable)
		},
		func(fs *flag.FlagSet) {
			AddLockFlags(fs)
		},
	)
}
