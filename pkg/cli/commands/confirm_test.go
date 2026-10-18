package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/migrator"
)

// stubMigrator is a minimal Migrator for tests that need a non-nil Migrator.
type stubMigrator struct {
	rollbackFn func(int) error
}

func (s stubMigrator) Up() error { return nil }
func (s stubMigrator) Rollback(steps int) error {
	if s.rollbackFn != nil {
		return s.rollbackFn(steps)
	}
	return nil
}
func (stubMigrator) Reset() error                                { return nil }
func (stubMigrator) Refresh() error                              { return nil }
func (stubMigrator) Fresh() error                                { return nil }
func (stubMigrator) Status() ([]migrator.MigrationStatus, error) { return nil, nil }

func TestConfirm_AcceptsY(t *testing.T) {
	in := bytes.NewBufferString("y\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Errorf("expected true for 'y' input")
	}
	if !strings.Contains(out.String(), "Proceed?") {
		t.Errorf("expected prompt in output, got %q", out.String())
	}
}

func TestConfirm_AcceptsYes(t *testing.T) {
	in := bytes.NewBufferString("yes\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Errorf("expected true for 'yes' input")
	}
}

func TestConfirm_AcceptsUpperY(t *testing.T) {
	in := bytes.NewBufferString("Y\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Errorf("expected true for 'Y' input")
	}
}

func TestConfirm_AcceptsUpperYES(t *testing.T) {
	in := bytes.NewBufferString("YES\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Errorf("expected true for 'YES' input")
	}
}

func TestConfirm_RejectsN(t *testing.T) {
	in := bytes.NewBufferString("n\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Errorf("expected false for 'n' input")
	}
}

func TestConfirm_RejectsNo(t *testing.T) {
	in := bytes.NewBufferString("no\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Errorf("expected false for 'no' input")
	}
}

func TestConfirm_RejectsRandom(t *testing.T) {
	in := bytes.NewBufferString("random\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Errorf("expected false for 'random' input")
	}
}

func TestConfirm_RejectsEmptyInput(t *testing.T) {
	in := bytes.NewBufferString("\n")
	out := &bytes.Buffer{}

	result, err := confirm(in, out, "Proceed?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Errorf("expected false for empty input")
	}
}

func TestConfirm_PromptAppearsInOutput(t *testing.T) {
	in := bytes.NewBufferString("n\n")
	out := &bytes.Buffer{}

	_, err := confirm(in, out, "Are you sure?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Are you sure?") {
		t.Errorf("expected 'Are you sure?' in output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "[y/N]") {
		t.Errorf("expected '[y/N]' in output, got %q", out.String())
	}
}

func TestMigrateFresh_ForceSkipsPrompt(t *testing.T) {
	out := &bytes.Buffer{}
	in := &bytes.Buffer{} // empty stdin — would block if prompt were shown
	reg, cmd := captureRegister()
	RegisterMigrateFreshCommand(reg, func() *CommandContext {
		return &CommandContext{}
	}, out, in)

	fs := newFlagSet("migrate:fresh", cmd.setupFlags)
	fs.Parse([]string{"--force"})
	_ = cmd.handler(fs.Args(), fs)

	// The prompt should NOT appear because --force was set
	if strings.Contains(out.String(), "[y/N]") {
		t.Errorf("expected no prompt with --force, got %q", out.String())
	}
}

func TestMigrateFresh_NoForcePromptsAndCancels(t *testing.T) {
	out := &bytes.Buffer{}
	in := bytes.NewBufferString("n\n")
	reg, cmd := captureRegister()
	RegisterMigrateFreshCommand(reg, func() *CommandContext {
		return &CommandContext{Migrator: stubMigrator{}}
	}, out, in)

	fs := newFlagSet("migrate:fresh", cmd.setupFlags)
	err := cmd.handler(fs.Args(), fs)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "[y/N]") {
		t.Errorf("expected prompt in output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Operation cancelled.") {
		t.Errorf("expected 'Operation cancelled.' in output, got %q", out.String())
	}
}

func TestMigrateReset_ForceSkipsPrompt(t *testing.T) {
	out := &bytes.Buffer{}
	in := &bytes.Buffer{}
	reg, cmd := captureRegister()
	RegisterMigrateResetCommand(reg, func() *CommandContext {
		return &CommandContext{}
	}, out, in)

	fs := newFlagSet("migrate:reset", cmd.setupFlags)
	fs.Parse([]string{"--force"})
	_ = cmd.handler(fs.Args(), fs)

	if strings.Contains(out.String(), "[y/N]") {
		t.Errorf("expected no prompt with --force, got %q", out.String())
	}
}

func TestMigrateReset_NoForcePromptsAndCancels(t *testing.T) {
	out := &bytes.Buffer{}
	in := bytes.NewBufferString("n\n")
	reg, cmd := captureRegister()
	RegisterMigrateResetCommand(reg, func() *CommandContext {
		return &CommandContext{Migrator: stubMigrator{}}
	}, out, in)

	fs := newFlagSet("migrate:reset", cmd.setupFlags)
	err := cmd.handler(fs.Args(), fs)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "[y/N]") {
		t.Errorf("expected prompt in output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Operation cancelled.") {
		t.Errorf("expected 'Operation cancelled.' in output, got %q", out.String())
	}
}
