package commands

import (
	"flag"
	"fmt"
)

// RegisterSeedCommand registers the "db:seed" command that runs database seeders.
// It supports an optional --class flag to run a specific seeder by name.
// When --class is empty, all registered seeders are executed.
func RegisterSeedCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext) {
	var class, tag *string

	register("db:seed", "Run database seeders",
		func(args []string, flags *flag.FlagSet) error {
			ctx := getCtx()
			if ctx == nil || ctx.Seeder == nil {
				return fmt.Errorf("seeder runner not initialized")
			}
			return withLock(ctx, func() error {
				if *class != "" {
					return ctx.Seeder.Run(*class)
				}
				if *tag != "" {
					return ctx.Seeder.RunByTag(*tag)
				}
				return ctx.Seeder.RunAll()
			})
		},
		func(fs *flag.FlagSet) {
			class = fs.String("class", "", "specific seeder class to run")
			tag = fs.String("tag", "", "run only seeders with the specified tag")
			AddLockFlags(fs)
		},
	)
}
