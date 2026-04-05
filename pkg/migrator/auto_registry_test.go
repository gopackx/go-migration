package migrator

import (
	"strings"
	"testing"
)

func TestAutoRegister_ValidName(t *testing.T) {
	defer ResetAutoRegistry()

	AutoRegister("20240101000001_create_users", &stubMigration{})

	result := GetAutoRegistered()
	if len(result) != 1 {
		t.Errorf("len = %d, want %d", len(result), 1)
	}
	if "20240101000001_create_users" != result[0].Name {
		t.Errorf("got %v, want %v", result[0].Name, "20240101000001_create_users")
	}
}

func TestAutoRegister_SortedOrder(t *testing.T) {
	defer ResetAutoRegistry()

	// Register out of order
	AutoRegister("20240101000003_create_posts", &stubMigration{})
	AutoRegister("20240101000001_create_users", &stubMigration{})
	AutoRegister("20240101000002_create_comments", &stubMigration{})

	result := GetAutoRegistered()
	if len(result) != 3 {
		t.Errorf("len = %d, want %d", len(result), 3)
	}
	if "20240101000001_create_users" != result[0].Name {
		t.Errorf("got %v, want %v", result[0].Name, "20240101000001_create_users")
	}
	if "20240101000002_create_comments" != result[1].Name {
		t.Errorf("got %v, want %v", result[1].Name, "20240101000002_create_comments")
	}
	if "20240101000003_create_posts" != result[2].Name {
		t.Errorf("got %v, want %v", result[2].Name, "20240101000003_create_posts")
	}
}

func TestAutoRegister_InvalidNamePanics(t *testing.T) {
	defer ResetAutoRegistry()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic for invalid name")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "invalid_name") {
			t.Errorf("panic message %q should contain %q", msg, "invalid_name")
		}
	}()
	AutoRegister("invalid_name", &stubMigration{})
}

func TestAutoRegister_DuplicateNamePanics(t *testing.T) {
	defer ResetAutoRegistry()

	AutoRegister("20240101000001_create_users", &stubMigration{})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic for duplicate name")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "20240101000001_create_users") {
			t.Errorf("panic message %q should contain %q", msg, "20240101000001_create_users")
		}
	}()
	AutoRegister("20240101000001_create_users", &stubMigration{})
}

func TestGetAutoRegistered_ReturnsCopy(t *testing.T) {
	defer ResetAutoRegistry()

	AutoRegister("20240101000001_create_users", &stubMigration{})

	result1 := GetAutoRegistered()
	result2 := GetAutoRegistered()

	// Modifying the returned slice should not affect the registry
	result1[0].Name = "modified"
	if "20240101000001_create_users" != result2[0].Name {
		t.Errorf("got %v, want %v", result2[0].Name, "20240101000001_create_users")
	}
}

func TestGetAutoRegistered_EmptyRegistry(t *testing.T) {
	defer ResetAutoRegistry()

	result := GetAutoRegistered()
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
	if result == nil {
		t.Errorf("expected non-nil")
	}
}

func TestResetAutoRegistry(t *testing.T) {
	AutoRegister("20240101000001_create_users", &stubMigration{})
	if len(GetAutoRegistered()) != 1 {
		t.Errorf("len = %d, want %d", len(GetAutoRegistered()), 1)
	}

	ResetAutoRegistry()
	if len(GetAutoRegistered()) != 0 {
		t.Errorf("expected empty, got %v", GetAutoRegistered())
	}
}
