package migrator_test

import (
	"testing"

	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
)

// stubMigration is a minimal Migration implementation for testing.
type stubMigration struct {
	upCalled   bool
	downCalled bool
}

func (s *stubMigration) Up(b *schema.Builder) error {
	s.upCalled = true
	return nil
}

func (s *stubMigration) Down(b *schema.Builder) error {
	s.downCalled = true
	return nil
}

// stubNoTxMigration implements both Migration and TransactionOption.
type stubNoTxMigration struct {
	stubMigration
}

func (s *stubNoTxMigration) DisableTransaction() bool {
	return true
}

func TestMigrationInterfaceCompliance(t *testing.T) {
	var m migrator.Migration = &stubMigration{}
	if m == nil {
		t.Errorf("expected non-nil")
	}

	err := m.Up(&schema.Builder{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = m.Down(&schema.Builder{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTransactionOptionInterfaceCompliance(t *testing.T) {
	s := &stubNoTxMigration{}

	// Verify it satisfies both interfaces.
	var m migrator.Migration = s
	var opt migrator.TransactionOption = s

	if m == nil {
		t.Errorf("expected non-nil")
	}
	if opt == nil {
		t.Errorf("expected non-nil")
	}
	if !opt.DisableTransaction() {
		t.Errorf("expected true")
	}
}

func TestMigrationWithoutTransactionOption(t *testing.T) {
	s := &stubMigration{}

	// A plain Migration should NOT satisfy TransactionOption.
	var m migrator.Migration = s
	_, ok := m.(migrator.TransactionOption)
	if ok {
		t.Errorf("expected false")
	}
}
