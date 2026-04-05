package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestSeedCommand_NilContext(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterSeedCommand(reg, func() *CommandContext { return nil })

	fs := newFlagSet("db:seed", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "seeder runner not initialized") {
		t.Errorf("got %q, want 'seeder runner not initialized'", err.Error())
	}
}

func TestSeedCommand_NilSeeder(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterSeedCommand(reg, func() *CommandContext { return &CommandContext{} })

	fs := newFlagSet("db:seed", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "seeder runner not initialized") {
		t.Errorf("got %q, want 'seeder runner not initialized'", err.Error())
	}
}

func TestSeedCommand_HasDescription(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterSeedCommand(reg, func() *CommandContext { return nil })

	if cmd.description == "" {
		t.Error("expected non-empty description")
	}
	if !strings.Contains(cmd.description, "seed") {
		t.Errorf("expected description to contain 'seed', got %q", cmd.description)
	}
}

func TestSeedCommand_ClassFlag(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterSeedCommand(reg, func() *CommandContext { return nil })

	fs := newFlagSet("db:seed", cmd.setupFlags)
	classFlag := fs.Lookup("class")
	if classFlag == nil {
		t.Fatal("--class flag should be registered")
	}
	if classFlag.DefValue != "" {
		t.Errorf("got default %q, want empty", classFlag.DefValue)
	}

	_ = &bytes.Buffer{} // suppress unused import
}
