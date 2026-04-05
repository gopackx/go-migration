package commands

import (
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/seeder"
)

func TestSeedRollbackCommand_NilContext(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterSeedRollbackCommand(reg, func() *CommandContext { return nil })

	fs := newFlagSet("db:seed:rollback", cmd.setupFlags)
	fs.Parse([]string{"--class", "UserSeeder"})
	err := cmd.handler(fs.Args(), fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "seeder runner not initialized") {
		t.Errorf("got %q, want 'seeder runner not initialized'", err.Error())
	}
}

func TestSeedRollbackCommand_NilSeeder(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterSeedRollbackCommand(reg, func() *CommandContext { return &CommandContext{} })

	fs := newFlagSet("db:seed:rollback", cmd.setupFlags)
	fs.Parse([]string{"--class", "UserSeeder"})
	err := cmd.handler(fs.Args(), fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "seeder runner not initialized") {
		t.Errorf("got %q, want 'seeder runner not initialized'", err.Error())
	}
}

func TestSeedRollbackCommand_EmptyClass(t *testing.T) {
	runner := seeder.NewRunner(seeder.NewRegistry(), nil, nil)
	reg, cmd := captureRegister()
	RegisterSeedRollbackCommand(reg, func() *CommandContext {
		return &CommandContext{Seeder: runner}
	})

	fs := newFlagSet("db:seed:rollback", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--class flag is required") {
		t.Errorf("got %q, want '--class flag is required'", err.Error())
	}
}

func TestSeedRollbackCommand_HasClassFlag(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterSeedRollbackCommand(reg, func() *CommandContext { return nil })

	fs := newFlagSet("db:seed:rollback", cmd.setupFlags)
	classFlag := fs.Lookup("class")
	if classFlag == nil {
		t.Fatal("--class flag should be registered")
	}
	if classFlag.DefValue != "" {
		t.Errorf("got default %q, want empty", classFlag.DefValue)
	}
}
