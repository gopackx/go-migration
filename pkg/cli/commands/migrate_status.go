package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
)

// ErrPendingMigrations is returned by "migrate:status --check" when at least
// one registered migration has not been applied. The returned error reads
// "N migration(s) pending" and wraps this sentinel; the CLI exits with
// status 1 for it.
var ErrPendingMigrations = errors.New("migration(s) pending")

// RegisterMigrateStatusCommand registers the "migrate:status" command
// that displays the status of all registered migrations. The command is
// read-only. With --check it returns ErrPendingMigrations (after printing the
// table) when any migration is pending, for use as a readiness gate.
func RegisterMigrateStatusCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer) {
	var check *bool

	register("migrate:status", "Show the status of each migration",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Migrator == nil {
				return fmt.Errorf("migrator not initialized")
			}
			statuses, err := ctx.Migrator.Status()
			if err != nil {
				return err
			}

			pending := 0
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "Migration\tStatus\tBatch\tApplied At")
			for _, s := range statuses {
				status := "Pending"
				batch := ""
				appliedAt := ""
				if s.Applied {
					status = "Applied"
					batch = fmt.Sprintf("%d", s.Batch)
					if s.AppliedAt != nil {
						appliedAt = s.AppliedAt.Format("2006-01-02 15:04:05")
					}
				} else {
					pending++
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, status, batch, appliedAt)
			}
			if err := w.Flush(); err != nil {
				return err
			}

			if check != nil && *check && pending > 0 {
				return fmt.Errorf("%d %w", pending, ErrPendingMigrations)
			}
			return nil
		},
		func(fs *flag.FlagSet) {
			check = fs.Bool("check", false, "exit with status 1 if any migration is pending")
		},
	)
}
