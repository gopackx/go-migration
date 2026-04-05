package commands

import (
	"flag"
	"fmt"
	"io"
)

// RegisterMigrateResetCommand registers the "migrate:reset" command that rolls back all migrations.
func RegisterMigrateResetCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer, in io.Reader) {
	var force *bool

	register("migrate:reset", "Rollback all migrations",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Migrator == nil {
				return fmt.Errorf("migrator not initialized")
			}

			if !*force {
				confirmed, err := confirm(in, out, "Are you sure you want to rollback all migrations?")
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "Operation cancelled.")
					return nil
				}
			}

			return ctx.Migrator.Reset()
		},
		func(fs *flag.FlagSet) {
			force = fs.Bool("force", false, "Force the operation to run without confirmation")
			fs.Bool("dry-run", false, "Show SQL without executing")
		},
	)
}
