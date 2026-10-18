package commands

import (
	"flag"
	"fmt"
	"io"
)

// RegisterMigrateRollbackCommand registers the "migrate:rollback" command.
// It supports a --step flag to roll back a specific number of migrations.
// When --step is 0 (default), it rolls back the last batch.
func RegisterMigrateRollbackCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer) {
	var step *int

	register("migrate:rollback", "Rollback the last batch of migrations",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Migrator == nil {
				return fmt.Errorf("migrator not initialized")
			}
			return ctx.Migrator.Rollback(*step)
		},
		func(fs *flag.FlagSet) {
			step = fs.Int("step", 0, "number of migrations to roll back (0 = last batch)")
			fs.Bool("dry-run", false, "Show SQL without executing")
			AddLockFlags(fs)
		},
	)
}
