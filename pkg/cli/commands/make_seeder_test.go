package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/internal/generator"
)

func TestMakeSeederCommand_NilContext(t *testing.T) {
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()
	RegisterMakeSeederCommand(reg, func() *CommandContext { return nil }, out)

	fs := newFlagSet("make:seeder", cmd.setupFlags)
	err := cmd.handler([]string{"user"}, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "generator not initialized") {
		t.Errorf("got %q, want 'generator not initialized'", err.Error())
	}
}

func TestMakeSeederCommand_NilGenerator(t *testing.T) {
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()
	RegisterMakeSeederCommand(reg, func() *CommandContext { return &CommandContext{} }, out)

	fs := newFlagSet("make:seeder", cmd.setupFlags)
	err := cmd.handler([]string{"user"}, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "generator not initialized") {
		t.Errorf("got %q, want 'generator not initialized'", err.Error())
	}
}

func TestMakeSeederCommand_MissingArg(t *testing.T) {
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()
	RegisterMakeSeederCommand(reg, func() *CommandContext { return nil }, out)

	fs := newFlagSet("make:seeder", cmd.setupFlags)
	err := cmd.handler([]string{}, fs)
	if err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestMakeSeederCommand_Success(t *testing.T) {
	tmpDir := t.TempDir()
	gen := generator.NewGenerator(tmpDir)
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()

	RegisterMakeSeederCommand(reg, func() *CommandContext {
		return &CommandContext{Generator: gen}
	}, out)

	fs := newFlagSet("make:seeder", cmd.setupFlags)
	err := cmd.handler([]string{"user"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Created seeder:") {
		t.Errorf("expected 'Created seeder:' in output, got %q", out.String())
	}

	// Verify file was actually created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	if entries[0].Name() != "user_seeder.go" {
		t.Errorf("got %q, want %q", entries[0].Name(), "user_seeder.go")
	}
}
