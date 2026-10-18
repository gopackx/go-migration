package commands

import (
	"flag"
	"fmt"
)

// RegisterSeedTruncateCommand registers the "db:seed:truncate" command that truncates
// (deletes all rows from) a specific seeder table. The --table flag is required
// and specifies which table to truncate.
func RegisterSeedTruncateCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext) {
	var table *string

	register("db:seed:truncate", "Truncate a specific seeder table",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Seeder == nil {
				return fmt.Errorf("seeder runner not initialized")
			}
			if *table == "" {
				return fmt.Errorf("--table flag is required")
			}
			return withLock(ctx, func() error { return ctx.Seeder.Truncate(*table) })
		},
		func(fs *flag.FlagSet) {
			table = fs.String("table", "", "table to truncate (required)")
			AddLockFlags(fs)
		},
	)
}
