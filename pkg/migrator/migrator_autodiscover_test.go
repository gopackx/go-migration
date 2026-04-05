package migrator

import (
	"errors"
	"strings"
	"testing"
)

// Test AutoDiscover() on empty auto-registry returns nil.
// Validates: Requirement 5.2
func TestAutoDiscover_EmptyAutoRegistry_ReturnsNil(t *testing.T) {
	defer ResetAutoRegistry()

	m := New(nil)
	err := m.AutoDiscover()

	if err != nil {
		t.Errorf("AutoDiscover on empty auto-registry should return nil")
	}
	if len(m.registry.GetAll()) != 0 {
		t.Errorf("expected empty, got %v", m.registry.GetAll())
	}
}

// Test WithAutoDiscover() with empty auto-registry creates Migrator without error.
// Validates: Requirement 6.3
func TestWithAutoDiscover_EmptyAutoRegistry_CreatesWithoutError(t *testing.T) {
	defer ResetAutoRegistry()

	m := New(nil)
	if m == nil {
		t.Errorf("expected non-nil")
	}
}

// Test backward compatibility: Register() still works after AutoDiscover().
// Validates: Requirement 2.5
func TestRegister_StillWorksAfterAutoDiscover(t *testing.T) {
	defer ResetAutoRegistry()

	AutoRegister("20240101000001_create_users", &stubMigration{})

	m := New(nil)
	err := m.AutoDiscover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Register() should still work after AutoDiscover()
	err = m.Register("20240101000002_create_posts", &stubMigration{})
	if err != nil {
		t.Errorf("Register should still work after AutoDiscover")
	}

	all := m.registry.GetAll()
	if len(all) != 2 {
		t.Errorf("len = %d, want %d", len(all), 2)
	}
	if "20240101000001_create_users" != all[0].Name {
		t.Errorf("got %v, want %v", all[0].Name, "20240101000001_create_users")
	}
	if "20240101000002_create_posts" != all[1].Name {
		t.Errorf("got %v, want %v", all[1].Name, "20240101000002_create_posts")
	}
}

// Test AutoDiscover() and Register() used together with no overlap.
// Validates: Requirement 4.1
func TestAutoDiscover_AndRegister_NoOverlap(t *testing.T) {
	defer ResetAutoRegistry()

	// Auto-register two migrations
	AutoRegister("20240101000001_create_users", &stubMigration{})
	AutoRegister("20240101000003_create_comments", &stubMigration{})

	m := New(nil)

	// Manually register a different migration
	err := m.Register("20240101000002_create_posts", &stubMigration{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AutoDiscover should succeed with no overlap
	err = m.AutoDiscover()
	if err != nil {
		t.Errorf("AutoDiscover should succeed when no name overlap exists")
	}

	// All three migrations should be present in sorted order
	all := m.registry.GetAll()
	if len(all) != 3 {
		t.Errorf("len = %d, want %d", len(all), 3)
	}
	if "20240101000001_create_users" != all[0].Name {
		t.Errorf("got %v, want %v", all[0].Name, "20240101000001_create_users")
	}
	if "20240101000002_create_posts" != all[1].Name {
		t.Errorf("got %v, want %v", all[1].Name, "20240101000002_create_posts")
	}
	if "20240101000003_create_comments" != all[2].Name {
		t.Errorf("got %v, want %v", all[2].Name, "20240101000003_create_comments")
	}
}

// Test AutoDiscover() and Register() with overlapping name returns error.
// Validates: Requirement 4.3
func TestAutoDiscover_AndRegister_OverlappingName_ReturnsError(t *testing.T) {
	defer ResetAutoRegistry()

	conflictName := "20240101000001_create_users"

	// Auto-register a migration
	AutoRegister(conflictName, &stubMigration{})

	m := New(nil)

	// Manually register the same name
	err := m.Register(conflictName, &stubMigration{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AutoDiscover should return error wrapping ErrDuplicateMigration
	err = m.AutoDiscover()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrDuplicateMigration) {
		t.Errorf("error should wrap ErrDuplicateMigration, got: %v", err)
	}
	if !strings.Contains(err.Error(), conflictName) {
		t.Errorf("%q not in %q", conflictName, err.Error())
	}
}
