package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/internal/generator"
)

func TestMakeMigrationCommand_NilContext(t *testing.T) {
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()
	RegisterMakeMigrationCommand(reg, func() *CommandContext { return nil }, out)

	fs := newFlagSet("make:migration", cmd.setupFlags)
	err := cmd.handler([]string{"create_users"}, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "generator not initialized") {
		t.Errorf("got %q, want 'generator not initialized'", err.Error())
	}
}

func TestMakeMigrationCommand_NilGenerator(t *testing.T) {
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()
	RegisterMakeMigrationCommand(reg, func() *CommandContext { return &CommandContext{} }, out)

	fs := newFlagSet("make:migration", cmd.setupFlags)
	err := cmd.handler([]string{"create_users"}, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "generator not initialized") {
		t.Errorf("got %q, want 'generator not initialized'", err.Error())
	}
}

func TestMakeMigrationCommand_MissingArg(t *testing.T) {
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()
	RegisterMakeMigrationCommand(reg, func() *CommandContext { return nil }, out)

	fs := newFlagSet("make:migration", cmd.setupFlags)
	err := cmd.handler([]string{}, fs)
	if err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestMakeMigrationCommand_Success(t *testing.T) {
	tmpDir := t.TempDir()
	gen := generator.NewGenerator(tmpDir)
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()

	RegisterMakeMigrationCommand(reg, func() *CommandContext {
		return &CommandContext{Generator: gen}
	}, out)

	fs := newFlagSet("make:migration", cmd.setupFlags)
	err := cmd.handler([]string{"create_users"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Created migration:") {
		t.Errorf("expected 'Created migration:' in output, got %q", out.String())
	}

	// Verify file was actually created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Name(), "create_users.go") {
		t.Errorf("expected filename containing 'create_users.go', got %q", entries[0].Name())
	}
}

func TestMakeMigrationCommand_CreateFlag(t *testing.T) {
	tmpDir := t.TempDir()
	gen := generator.NewGenerator(tmpDir)
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()

	RegisterMakeMigrationCommand(reg, func() *CommandContext {
		return &CommandContext{Generator: gen}
	}, out)

	fs := newFlagSet("make:migration", cmd.setupFlags)
	fs.Parse([]string{"--create", "users"})
	err := cmd.handler([]string{"create_users"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the generated file contains a Create call
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(content), "Create") {
		t.Errorf("expected 'Create' in generated file")
	}
	if !strings.Contains(string(content), "users") {
		t.Errorf("expected 'users' in generated file")
	}
}

func TestMakeMigrationCommand_TableFlag(t *testing.T) {
	tmpDir := t.TempDir()
	gen := generator.NewGenerator(tmpDir)
	out := &bytes.Buffer{}
	reg, cmd := captureRegister()

	RegisterMakeMigrationCommand(reg, func() *CommandContext {
		return &CommandContext{Generator: gen}
	}, out)

	fs := newFlagSet("make:migration", cmd.setupFlags)
	fs.Parse([]string{"--table", "users"})
	err := cmd.handler([]string{"add_email_to_users"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the generated file contains an Alter call
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(content), "Alter") {
		t.Errorf("expected 'Alter' in generated file")
	}
	if !strings.Contains(string(content), "users") {
		t.Errorf("expected 'users' in generated file")
	}
}
