package cli

import (
	"bytes"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// validCommands is the set of supported CLI command names.
var validCommands = []string{
	"migrate", "migrate:rollback", "migrate:reset", "migrate:refresh",
	"migrate:fresh", "migrate:install", "migrate:status",
	"make:migration", "make:seeder", "make:factory",
	"db:seed", "db:seed:rollback", "db:seed:truncate", "init",
}

// newTestRouter creates a Router with all valid commands registered.
// Each handler records which command was invoked into the provided map.
func newTestRouter(invoked map[string]bool) *Router {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	r := NewRouter(out, errOut, strings.NewReader(""))
	for _, name := range validCommands {
		cmd := name // capture
		r.Register(cmd, "description for "+cmd, func(args []string, fs *flag.FlagSet) error {
			invoked[cmd] = true
			return nil
		}, nil)
	}
	return r
}

// Feature: reduce-external-dependencies, Property 1: Valid Command Routing
// For any valid command name, the router dispatches to the correct handler.
func TestProperty1_ValidCommandRouting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmdName := rapid.SampledFrom(validCommands).Draw(t, "command")

		invoked := make(map[string]bool)
		r := newTestRouter(invoked)

		err := r.Run([]string{cmdName})
		if err != nil {
			t.Fatalf("unexpected error for command %q: %v", cmdName, err)
		}
		if !invoked[cmdName] {
			t.Errorf("handler for %q was not invoked", cmdName)
		}
		// Verify no other handler was invoked.
		for _, other := range validCommands {
			if other != cmdName && invoked[other] {
				t.Errorf("handler for %q was unexpectedly invoked when running %q", other, cmdName)
			}
		}
	})
}

// Feature: reduce-external-dependencies, Property 2: Unknown Command Error
// For any string that is not a valid command, the router returns an error.
func TestProperty2_UnknownCommandError(t *testing.T) {
	validSet := make(map[string]bool)
	for _, c := range validCommands {
		validSet[c] = true
	}

	rapid.Check(t, func(t *rapid.T) {
		input := rapid.StringMatching(`[a-z:_]{1,20}`).Draw(t, "input")
		// Skip if it happens to be a valid command or help alias.
		if validSet[input] || input == "help" || input == "--help" || input == "-h" {
			return
		}

		invoked := make(map[string]bool)
		r := newTestRouter(invoked)

		err := r.Run([]string{input})
		if err == nil {
			t.Fatalf("expected error for unknown command %q, got nil", input)
		}
		if !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("error %q should contain 'unknown command'", err.Error())
		}
	})
}

// Feature: reduce-external-dependencies, Property 3: Flag Parsing Correctness
// For any valid flag value, the router parses it correctly.
func TestProperty3_FlagParsingCorrectness(t *testing.T) {
	t.Run("integer_flags", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			stepVal := rapid.IntRange(1, 1000).Draw(t, "step")

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			r := NewRouter(out, errOut, strings.NewReader(""))

			var parsedStep int
			r.Register("migrate:rollback", "rollback", func(args []string, fs *flag.FlagSet) error {
				parsedStep, _ = strconv.Atoi(fs.Lookup("step").Value.String())
				return nil
			}, func(fs *flag.FlagSet) {
				fs.Int("step", 1, "number of steps")
			})

			err := r.Run([]string{"migrate:rollback", fmt.Sprintf("--step=%d", stepVal)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsedStep != stepVal {
				t.Errorf("got step=%d, want %d", parsedStep, stepVal)
			}
		})
	})

	t.Run("string_flags", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableVal := rapid.StringMatching(`[a-z_]{1,20}`).Draw(t, "table")

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			r := NewRouter(out, errOut, strings.NewReader(""))

			var parsedTable string
			r.Register("make:migration", "make migration", func(args []string, fs *flag.FlagSet) error {
				parsedTable = fs.Lookup("table").Value.String()
				return nil
			}, func(fs *flag.FlagSet) {
				fs.String("table", "", "table name")
			})

			err := r.Run([]string{"make:migration", fmt.Sprintf("--table=%s", tableVal)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsedTable != tableVal {
				t.Errorf("got table=%q, want %q", parsedTable, tableVal)
			}
		})
	})

	t.Run("config_flag", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			configVal := rapid.StringMatching(`[a-z_/]{1,20}\.(json|yaml|yml)`).Draw(t, "config")

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			r := NewRouter(out, errOut, strings.NewReader(""))

			var receivedConfig string
			r.SetPreRun(func(cmdName string, configPath string, fs *flag.FlagSet) error {
				receivedConfig = configPath
				return nil
			})
			r.Register("migrate", "run migrations", func(args []string, fs *flag.FlagSet) error {
				return nil
			}, nil)

			err := r.Run([]string{"--config", configVal, "migrate"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if receivedConfig != configVal {
				t.Errorf("got config=%q, want %q", receivedConfig, configVal)
			}
		})
	})
}
