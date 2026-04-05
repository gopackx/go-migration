package commands

import (
	"flag"
	"testing"
)

// capturedCommand holds a registered command for testing.
type capturedCommand struct {
	name        string
	description string
	handler     func(args []string, flags *flag.FlagSet) error
	setupFlags  func(fs *flag.FlagSet)
}

// captureRegister returns a register function that captures the command.
func captureRegister() (func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet)), *capturedCommand) {
	cmd := &capturedCommand{}
	reg := func(name, desc string, handler func([]string, *flag.FlagSet) error, setupFlags func(*flag.FlagSet)) {
		cmd.name = name
		cmd.description = desc
		cmd.handler = handler
		cmd.setupFlags = setupFlags
	}
	return reg, cmd
}

// newFlagSet creates a FlagSet and runs setupFlags if provided.
func newFlagSet(name string, setupFlags func(*flag.FlagSet)) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	if setupFlags != nil {
		setupFlags(fs)
	}
	return fs
}

func TestRegisterMigrateCommand_Name(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateCommand(reg, func() *CommandContext { return nil }, nil)

	if cmd.name != "migrate" {
		t.Errorf("got name %q, want %q", cmd.name, "migrate")
	}
	if cmd.description == "" {
		t.Error("expected non-empty description")
	}
}

func TestRegisterMigrateRollbackCommand_Name(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateRollbackCommand(reg, func() *CommandContext { return nil }, nil)

	if cmd.name != "migrate:rollback" {
		t.Errorf("got name %q, want %q", cmd.name, "migrate:rollback")
	}
}

func TestRegisterAllCommands_HaveDescriptions(t *testing.T) {
	getCtx := func() *CommandContext { return nil }

	commands := []struct {
		name     string
		register func()
		cmd      *capturedCommand
	}{}

	names := []string{
		"migrate", "migrate:rollback", "migrate:reset", "migrate:refresh",
		"migrate:fresh", "migrate:status", "migrate:install",
		"make:migration", "make:seeder", "make:factory",
		"db:seed", "db:seed:rollback", "db:seed:truncate", "init",
	}

	for _, n := range names {
		reg, cmd := captureRegister()
		switch n {
		case "migrate":
			RegisterMigrateCommand(reg, getCtx, nil)
		case "migrate:rollback":
			RegisterMigrateRollbackCommand(reg, getCtx, nil)
		case "migrate:reset":
			RegisterMigrateResetCommand(reg, getCtx, nil, nil)
		case "migrate:refresh":
			RegisterMigrateRefreshCommand(reg, getCtx, nil)
		case "migrate:fresh":
			RegisterMigrateFreshCommand(reg, getCtx, nil, nil)
		case "migrate:status":
			RegisterMigrateStatusCommand(reg, getCtx, nil)
		case "migrate:install":
			RegisterMigrateInstallCommand(reg, getCtx)
		case "make:migration":
			RegisterMakeMigrationCommand(reg, getCtx, nil)
		case "make:seeder":
			RegisterMakeSeederCommand(reg, getCtx, nil)
		case "make:factory":
			RegisterMakeFactoryCommand(reg, getCtx, nil)
		case "db:seed":
			RegisterSeedCommand(reg, getCtx)
		case "db:seed:rollback":
			RegisterSeedRollbackCommand(reg, getCtx)
		case "db:seed:truncate":
			RegisterSeedTruncateCommand(reg, getCtx)
		case "init":
			RegisterInitCommand(reg, nil, nil)
		}
		commands = append(commands, struct {
			name     string
			register func()
			cmd      *capturedCommand
		}{name: n, cmd: cmd})
	}

	for _, c := range commands {
		if c.cmd.name != c.name {
			t.Errorf("expected command name %q, got %q", c.name, c.cmd.name)
		}
		if c.cmd.description == "" {
			t.Errorf("command %q has empty description", c.name)
		}
	}
}
