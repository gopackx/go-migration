package commands

import (
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
)

// RegisterMigrateStatusCommand registers the "migrate:status" command
// that displays the status of all registered migrations.
func RegisterMigrateStatusCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer) {
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
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, status, batch, appliedAt)
			}
			return w.Flush()
		},
		nil,
	)
}
