package migrator

import (
	"errors"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
	"strings"
)

// stubMigration is a no-op Migration for testing.
type stubMigration struct{}

func (s *stubMigration) Up(_ *schema.Builder) error   { return nil }
func (s *stubMigration) Down(_ *schema.Builder) error { return nil }

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Errorf("expected non-nil")
	}
	if 0 != r.Count() {
		t.Errorf("got %v, want %v", r.Count(), 0)
	}
	if len(r.GetAll()) != 0 {
		t.Errorf("expected empty, got %v", r.GetAll())
	}
}

func TestRegister_ValidName(t *testing.T) {
	r := NewRegistry()
	err := r.Register("20240215120000_create_users", &stubMigration{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if 1 != r.Count() {
		t.Errorf("got %v, want %v", r.Count(), 1)
	}
}

func TestRegister_DuplicateName(t *testing.T) {
	r := NewRegistry()
	name := "20240215120000_create_users"
	if r.Register(name, &stubMigration{}) != nil {
		t.Fatalf("unexpected error: %v", r.Register(name, &stubMigration{}))
	}

	err := r.Register(name, &stubMigration{})
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrDuplicateMigration) {
		t.Errorf("expected true")
	}
	if !strings.Contains(err.Error(), name) {
		t.Errorf("%q not in %q", name, err.Error())
	}
}

func TestRegister_InvalidNames(t *testing.T) {
	tests := []struct {
		name string
		desc string
	}{
		{"", "empty string"},
		{"create_users", "no timestamp prefix"},
		{"2024021512_create_users", "short timestamp"},
		{"20240215120000", "no description"},
		{"20240215120000_", "empty description"},
		{"20240215120000_Create_users", "uppercase in description"},
		{"20240215120000_1create", "description starts with digit"},
		{"20240215120000_create-users", "hyphen in description"},
		{"20240215120000_create users", "space in description"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			r := NewRegistry()
			err := r.Register(tt.name, &stubMigration{})
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidMigrationName) {
				t.Errorf("expected ErrInvalidMigrationName for %q", tt.name)
			}
		})
	}
}

func TestRegister_MaintainsSortedOrder(t *testing.T) {
	r := NewRegistry()
	// Register out of order.
	names := []string{
		"20240315000000_third",
		"20240115000000_first",
		"20240215000000_second",
	}
	for _, name := range names {
		if r.Register(name, &stubMigration{}) != nil {
			t.Fatalf("unexpected error: %v", r.Register(name, &stubMigration{}))
		}
	}

	all := r.GetAll()
	if len(all) != 3 {
		t.Fatalf("len = %d, want %d", len(all), 3)
	}
	if "20240115000000_first" != all[0].Name {
		t.Errorf("got %v, want %v", all[0].Name, "20240115000000_first")
	}
	if "20240215000000_second" != all[1].Name {
		t.Errorf("got %v, want %v", all[1].Name, "20240215000000_second")
	}
	if "20240315000000_third" != all[2].Name {
		t.Errorf("got %v, want %v", all[2].Name, "20240315000000_third")
	}
}

func TestGet_Existing(t *testing.T) {
	r := NewRegistry()
	m := &stubMigration{}
	name := "20240215120000_create_users"
	if r.Register(name, m) != nil {
		t.Fatalf("unexpected error: %v", r.Register(name, m))
	}

	got, err := r.Get(name)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != got {
		t.Errorf("got %p, want %p", got, m)
	}
}

func TestGet_NotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("20240215120000_nonexistent")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrMigrationNotFound) {
		t.Errorf("expected true")
	}
}

func TestGetAll_ReturnsCopy(t *testing.T) {
	r := NewRegistry()
	if r.Register("20240115000000_first", &stubMigration{}) != nil {
		t.Fatalf("unexpected error: %v", r.Register("20240115000000_first", &stubMigration{}))
	}

	all1 := r.GetAll()
	all2 := r.GetAll()
	// Mutating the returned slice should not affect the registry.
	all1[0].Name = "mutated"
	if "20240115000000_first" != all2[0].Name {
		t.Errorf("got %v, want %v", all2[0].Name, "20240115000000_first")
	}
}

func TestCount(t *testing.T) {
	r := NewRegistry()
	if 0 != r.Count() {
		t.Errorf("got %v, want %v", r.Count(), 0)
	}

	if r.Register("20240115000000_a", &stubMigration{}) != nil {
		t.Fatalf("unexpected error: %v", r.Register("20240115000000_a", &stubMigration{}))
	}
	if 1 != r.Count() {
		t.Errorf("got %v, want %v", r.Count(), 1)
	}

	if r.Register("20240215000000_b", &stubMigration{}) != nil {
		t.Fatalf("unexpected error: %v", r.Register("20240215000000_b", &stubMigration{}))
	}
	if 2 != r.Count() {
		t.Errorf("got %v, want %v", r.Count(), 2)
	}
}
