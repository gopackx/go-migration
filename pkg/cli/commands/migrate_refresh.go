package commands

import (
	"flag"
	"fmt"
	"io"
)

// RegisterMigrateRefreshCommand registers the "migrate:refresh" command
// that resets all migrations and re-runs them.
func RegisterMigrateRefreshCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer) {
	register("migrate:refresh", "Reset and re-run all migrations",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Migrator == nil {
				return fmt.Errorf("migrator not initialized")
			}
			return ctx.Migrator.Refresh()
		},
		func(fs *flag.FlagSet) {
			fs.Bool("dry-run", false, "Show SQL without executing")
		},
	)
}
