package commands

import (
	"flag"
	"fmt"
	"io"
)

// RegisterMigrateCommand registers the "migrate" command on the router.
func RegisterMigrateCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer) {
	register("migrate", "Run all pending migrations",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Migrator == nil {
				return fmt.Errorf("migrator not initialized")
			}
			return ctx.Migrator.Up()
		},
		func(fs *flag.FlagSet) {
			fs.Bool("dry-run", false, "Show SQL without executing")
			AddLockFlags(fs)
		},
	)
}
