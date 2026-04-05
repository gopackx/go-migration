// Package cli provides the command-line interface for go-migration.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// CommandHandler handles a single CLI command.
// It receives the remaining args after the command name and a FlagSet
// for parsing per-command flags.
type CommandHandler func(args []string, flags *flag.FlagSet) error

// commandEntry holds a handler and its metadata.
type commandEntry struct {
	handler     CommandHandler
	description string
	setupFlags  func(fs *flag.FlagSet)
}

// PreRunFunc is called before each command handler.
// It receives the command name, the parsed global config path, and the
// per-command FlagSet (already parsed).
type PreRunFunc func(cmdName string, configPath string, flags *flag.FlagSet) error

// Router matches os.Args to the correct CommandHandler.
type Router struct {
	commands map[string]commandEntry
	out      io.Writer
	errOut   io.Writer
	stdin    io.Reader
	preRun   PreRunFunc
}

// NewRouter creates a Router that writes to the given writers.
// If out or errOut is nil, os.Stdout / os.Stderr is used.
func NewRouter(out, errOut io.Writer, stdin io.Reader) *Router {
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	if stdin == nil {
		stdin = os.Stdin
	}
	return &Router{
		commands: make(map[string]commandEntry),
		out:      out,
		errOut:   errOut,
		stdin:    stdin,
	}
}

// SetPreRun sets a function that runs before every command handler.
func (r *Router) SetPreRun(fn PreRunFunc) {
	r.preRun = fn
}

// Register adds a command to the router.
func (r *Router) Register(name, description string, handler CommandHandler, setupFlags func(fs *flag.FlagSet)) {
	r.commands[name] = commandEntry{
		handler:     handler,
		description: description,
		setupFlags:  setupFlags,
	}
}

// Out returns the stdout writer.
func (r *Router) Out() io.Writer { return r.out }

// ErrOut returns the stderr writer.
func (r *Router) ErrOut() io.Writer { return r.errOut }

// Stdin returns the stdin reader.
func (r *Router) Stdin() io.Reader { return r.stdin }

// Run parses os.Args (or the provided args) and dispatches to the matching handler.
// If args is nil, os.Args[1:] is used.
func (r *Router) Run(args []string) error {
	if args == nil {
		args = os.Args[1:]
	}

	// Parse global flags first. We scan for --config before the command name.
	configPath := "migration.json"
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--config" {
			if i+1 < len(args) {
				configPath = args[i+1]
				i++ // skip value
				continue
			}
			// --config without value
			fmt.Fprintf(r.errOut, "Error: --config requires a value\n")
			return fmt.Errorf("--config requires a value")
		}
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
			continue
		}
		remaining = append(remaining, arg)
	}

	// No args or --help → show help
	if len(remaining) == 0 {
		r.printHelp()
		return nil
	}

	if remaining[0] == "--help" || remaining[0] == "-h" || remaining[0] == "help" {
		r.printHelp()
		return nil
	}

	cmdName := remaining[0]
	cmdArgs := remaining[1:]

	entry, ok := r.commands[cmdName]
	if !ok {
		fmt.Fprintf(r.errOut, "Error: unknown command %q\n\n", cmdName)
		r.printAvailableCommands()
		return fmt.Errorf("unknown command %q", cmdName)
	}

	// Set up per-command flags.
	fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	fs.SetOutput(r.errOut)
	if entry.setupFlags != nil {
		entry.setupFlags(fs)
	}

	// Check for --help in command args before parsing
	for _, a := range cmdArgs {
		if a == "--help" || a == "-h" {
			fmt.Fprintf(r.out, "Usage: go-migration %s [flags]\n\n", cmdName)
			fmt.Fprintf(r.out, "  %s\n\n", entry.description)
			fmt.Fprintf(r.out, "Flags:\n")
			fs.SetOutput(r.out)
			fs.PrintDefaults()
			return nil
		}
	}

	if err := fs.Parse(cmdArgs); err != nil {
		return fmt.Errorf("flag parse error: %w", err)
	}

	// Run pre-run hook if set.
	if r.preRun != nil {
		if err := r.preRun(cmdName, configPath, fs); err != nil {
			return err
		}
	}

	return entry.handler(fs.Args(), fs)
}

// printHelp prints the full help text.
func (r *Router) printHelp() {
	fmt.Fprintln(r.out, "go-migration is a Laravel-inspired database migration and seeding tool for Go.")
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "Usage:")
	fmt.Fprintln(r.out, "  go-migration <command> [flags]")
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "Global Flags:")
	fmt.Fprintln(r.out, "  --config string   path to configuration file (default \"migration.json\")")
	fmt.Fprintln(r.out)
	r.printAvailableCommands()
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "Usage Examples:")
	fmt.Fprintln(r.out, "  go-migration migrate                  Run all pending migrations")
	fmt.Fprintln(r.out, "  go-migration migrate:rollback         Rollback the last batch of migrations")
	fmt.Fprintln(r.out, "  go-migration migrate:rollback --step 2  Rollback the last 2 migrations")
	fmt.Fprintln(r.out, "  go-migration migrate:status           Show migration status")
	fmt.Fprintln(r.out, "  go-migration migrate:reset            Rollback all migrations")
	fmt.Fprintln(r.out, "  go-migration migrate:refresh          Reset and re-run all migrations")
	fmt.Fprintln(r.out, "  go-migration migrate:fresh            Drop all tables and re-run migrations")
	fmt.Fprintln(r.out, "  go-migration migrate:install          Create the migration tracking table")
	fmt.Fprintln(r.out, "  go-migration make:migration create_users --create=users")
	fmt.Fprintln(r.out, "  go-migration make:seeder users")
	fmt.Fprintln(r.out, "  go-migration make:factory users")
	fmt.Fprintln(r.out, "  go-migration db:seed                  Run all seeders")
	fmt.Fprintln(r.out, "  go-migration db:seed --class=users    Run a specific seeder")
}

// printAvailableCommands prints the sorted list of registered commands.
func (r *Router) printAvailableCommands() {
	fmt.Fprintln(r.out, "Available Commands:")

	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	maxLen := 0
	for _, n := range names {
		if len(n) > maxLen {
			maxLen = len(n)
		}
	}

	for _, name := range names {
		fmt.Fprintf(r.out, "  %-*s  %s\n", maxLen, name, r.commands[name].description)
	}
}
