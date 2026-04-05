package commands

import (
	"flag"
	"fmt"
	"io"
)

// RegisterMakeFactoryCommand registers the "make:factory" command.
// It generates a new factory file from a template with the correct
// struct scaffolding.
func RegisterMakeFactoryCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), getCtx func() *CommandContext, out io.Writer) {
	register("make:factory", "Generate a new factory file",
		func(args []string, flags *flag.FlagSet) error {
			if len(args) < 1 {
				return fmt.Errorf("requires exactly 1 arg(s), only received 0")
			}

			ctx := getCtx()
			if ctx == nil || ctx.Generator == nil {
				return fmt.Errorf("generator not initialized")
			}

			path, err := ctx.Generator.Factory(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(out, "Created factory: %s\n", path)
			return nil
		},
		nil,
	)
}
