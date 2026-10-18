package commands

import (
	"flag"
	"fmt"
)

// RegisterSeedRollbackCommand registers the "db:seed:rollback" command that rolls back
// a specific seeder by name. The --class flag is required and specifies which
// seeder to roll back.
func RegisterSeedRollbackCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext) {
	var class *string

	register("db:seed:rollback", "Rollback a specific database seeder",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Seeder == nil {
				return fmt.Errorf("seeder runner not initialized")
			}
			if *class == "" {
				return fmt.Errorf("--class flag is required")
			}
			return withLock(ctx, func() error { return ctx.Seeder.Rollback(*class) })
		},
		func(fs *flag.FlagSet) {
			class = fs.String("class", "", "seeder class to rollback (required)")
			AddLockFlags(fs)
		},
	)
}
