package commands

import (
	"flag"
	"fmt"
	"io"
)

// RegisterMigrateFreshCommand registers the "migrate:fresh" command
// that drops all tables and re-runs all migrations.
func RegisterMigrateFreshCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer, in io.Reader) {
	var force *bool

	register("migrate:fresh", "Drop all tables and re-run all migrations",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Migrator == nil {
				return fmt.Errorf("migrator not initialized")
			}

			if !*force {
				confirmed, err := confirm(in, out, "Are you sure you want to drop all tables and re-run all migrations?")
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "Operation cancelled.")
					return nil
				}
			}

			return ctx.Migrator.Fresh()
		},
		func(fs *flag.FlagSet) {
			force = fs.Bool("force", false, "Force the operation to run without confirmation")
			fs.Bool("dry-run", false, "Show SQL without executing")
			AddLockFlags(fs)
		},
	)
}
