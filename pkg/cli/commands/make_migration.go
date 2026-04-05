package commands

import (
	"flag"
	"fmt"
	"io"

	"github.com/gopackx/go-migration/internal/generator"
)

// RegisterMakeMigrationCommand registers the "make:migration" command.
// It generates a new migration file from a template with the correct
// timestamp prefix and struct scaffolding. Supports --create and --table
// flags for pre-populated schema builder calls.
func RegisterMakeMigrationCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer) {
	var create, table *string

	register("make:migration", "Generate a new migration file",
		func(args []string, flags *flag.FlagSet) error {
			if len(args) < 1 {
				return fmt.Errorf("requires exactly 1 arg(s), only received 0")
			}

			ctx := getCtx()
			if ctx == nil || ctx.Generator == nil {
				return fmt.Errorf("generator not initialized")
			}

			opts := generator.MigrationOptions{
				CreateTable: *create,
				AlterTable:  *table,
			}

			path, err := ctx.Generator.Migration(args[0], opts)
			if err != nil {
				return err
			}

			fmt.Fprintf(out, "Created migration: %s\n", path)
			return nil
		},
		func(fs *flag.FlagSet) {
			create = fs.String("create", "", "table name to create (pre-populates schema Create call)")
			table = fs.String("table", "", "table name to alter (pre-populates schema Alter call)")
		},
	)
}
