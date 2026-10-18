package commands

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/migrator"
)

// --- NewMigrateCommand ---

func TestMigrateCommand_NilContext(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateCommand(reg, func() *CommandContext { return nil }, &bytes.Buffer{})

	fs := newFlagSet("migrate", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

func TestMigrateCommand_NilMigrator(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateCommand(reg, func() *CommandContext { return &CommandContext{} }, &bytes.Buffer{})

	fs := newFlagSet("migrate", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

// --- MigrateRollback command ---

func TestMigrateRollbackCommand_NilContext(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateRollbackCommand(reg, func() *CommandContext { return nil }, &bytes.Buffer{})

	fs := newFlagSet("migrate:rollback", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

func TestMigrateRollbackCommand_StepFlag(t *testing.T) {
	reg, cmd := captureRegister()
	var capturedStep int
	RegisterMigrateRollbackCommand(reg, func() *CommandContext {
		return &CommandContext{Migrator: stubMigratorFn{
			rollbackFn: func(steps int) error {
				capturedStep = steps
				return nil
			},
		}}
	}, &bytes.Buffer{})

	fs := newFlagSet("migrate:rollback", cmd.setupFlags)
	if err := fs.Parse([]string{"--step", "3"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	err := cmd.handler(fs.Args(), fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStep != 3 {
		t.Errorf("got step %d, want 3", capturedStep)
	}
}

func TestMigrateRollbackCommand_NilMigrator(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateRollbackCommand(reg, func() *CommandContext { return &CommandContext{} }, &bytes.Buffer{})

	fs := newFlagSet("migrate:rollback", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

// --- MigrateReset command ---

func TestMigrateResetCommand_NilContext(t *testing.T) {
	out := &bytes.Buffer{}
	in := bytes.NewBufferString("y\n")
	reg, cmd := captureRegister()
	RegisterMigrateResetCommand(reg, func() *CommandContext { return nil }, out, in)

	fs := newFlagSet("migrate:reset", cmd.setupFlags)
	fs.Parse([]string{"--force"})
	err := cmd.handler(fs.Args(), fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

// --- MigrateRefresh command ---

func TestMigrateRefreshCommand_NilContext(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateRefreshCommand(reg, func() *CommandContext { return nil }, &bytes.Buffer{})

	fs := newFlagSet("migrate:refresh", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

// --- MigrateFresh command ---

func TestMigrateFreshCommand_NilContext(t *testing.T) {
	out := &bytes.Buffer{}
	in := bytes.NewBufferString("y\n")
	reg, cmd := captureRegister()
	RegisterMigrateFreshCommand(reg, func() *CommandContext { return nil }, out, in)

	fs := newFlagSet("migrate:fresh", cmd.setupFlags)
	fs.Parse([]string{"--force"})
	err := cmd.handler(fs.Args(), fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

// --- MigrateStatus command ---

func TestMigrateStatusCommand_NilContext(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateStatusCommand(reg, func() *CommandContext { return nil }, &bytes.Buffer{})

	fs := newFlagSet("migrate:status", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migrator not initialized") {
		t.Errorf("got %q, want 'migrator not initialized'", err.Error())
	}
}

// --- MigrateInstall command ---

func TestMigrateInstallCommand_NilContext(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateInstallCommand(reg, func() *CommandContext { return nil })

	fs := newFlagSet("migrate:install", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "database connection not initialized") {
		t.Errorf("got %q, want 'database connection not initialized'", err.Error())
	}
}

func TestMigrateInstallCommand_NilDB(t *testing.T) {
	reg, cmd := captureRegister()
	RegisterMigrateInstallCommand(reg, func() *CommandContext { return &CommandContext{} })

	fs := newFlagSet("migrate:install", cmd.setupFlags)
	err := cmd.handler(nil, fs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "database connection not initialized") {
		t.Errorf("got %q, want 'database connection not initialized'", err.Error())
	}
}

// --- Help text tests ---

func TestAllMigrateCommands_HaveHelpText(t *testing.T) {
	getCtx := func() *CommandContext { return nil }
	out := &bytes.Buffer{}
	in := &bytes.Buffer{}

	tests := []struct {
		name string
		reg  func(func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet)))
	}{
		{"migrate", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))) {
			RegisterMigrateCommand(r, getCtx, out)
		}},
		{"migrate:rollback", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))) {
			RegisterMigrateRollbackCommand(r, getCtx, out)
		}},
		{"migrate:reset", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))) {
			RegisterMigrateResetCommand(r, getCtx, out, in)
		}},
		{"migrate:refresh", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))) {
			RegisterMigrateRefreshCommand(r, getCtx, out)
		}},
		{"migrate:fresh", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))) {
			RegisterMigrateFreshCommand(r, getCtx, out, in)
		}},
		{"migrate:status", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))) {
			RegisterMigrateStatusCommand(r, getCtx, out)
		}},
		{"migrate:install", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))) {
			RegisterMigrateInstallCommand(r, getCtx)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, cmd := captureRegister()
			tt.reg(reg)
			if cmd.description == "" {
				t.Errorf("command %q has empty description", tt.name)
			}
		})
	}
}

// stubMigratorFn with configurable callbacks for testing
type stubMigratorFn struct {
	rollbackFn func(int) error
	statusFn   func() ([]migrator.MigrationStatus, error)
}

func (s stubMigratorFn) Up() error { return nil }
func (s stubMigratorFn) Rollback(steps int) error {
	if s.rollbackFn != nil {
		return s.rollbackFn(steps)
	}
	return nil
}
func (s stubMigratorFn) Reset() error   { return nil }
func (s stubMigratorFn) Refresh() error { return nil }
func (s stubMigratorFn) Fresh() error   { return nil }
func (s stubMigratorFn) Status() ([]migrator.MigrationStatus, error) {
	if s.statusFn != nil {
		return s.statusFn()
	}
	return nil, nil
}
