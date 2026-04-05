package commands

import (
	"flag"
	"fmt"
)

// RegisterMigrateInstallCommand registers the "migrate:install" command
// that creates the migration tracking table.
func RegisterMigrateInstallCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext) {
	register("migrate:install", "Create the migration tracking table",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.TrackerEnsurer == nil {
				return fmt.Errorf("database connection not initialized")
			}
			return ctx.TrackerEnsurer.EnsureTable()
		},
		nil,
	)
}
