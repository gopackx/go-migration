package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gopackx/go-migration/internal/generator"
)

// RegisterInitCommand registers the "init" command.
// It scaffolds a new go-migration project with directories, a pre-wired
// cmd/migrator/main.go entry point, and a default config file.
// This command does not require a CommandContext or database connection.
func RegisterInitCommand(register func(name, desc string, handler func(args []string, flags *flag.FlagSet) error, setupFlags func(fs *flag.FlagSet)), out io.Writer, errOut io.Writer) {
	var modulePath *string
	var force *bool

	register("init", "Scaffold a new go-migration project",
		func(args []string, flags *flag.FlagSet) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			actualModule := *modulePath
			if actualModule == "" {
				goModPath := filepath.Join(cwd, "go.mod")
				actualModule, err = generator.ParseModulePath(goModPath)
				if err != nil {
					return fmt.Errorf("no go.mod found; run 'go mod init' or provide --module flag")
				}
			}

			scaffolder := generator.NewInitScaffolder(cwd, actualModule, *force, out, errOut)
			_, err = scaffolder.Scaffold()
			return err
		},
		func(fs *flag.FlagSet) {
			modulePath = fs.String("module", "", "Go module path (overrides go.mod detection)")
			force = fs.Bool("force", false, "Overwrite existing files")
		},
	)
}
