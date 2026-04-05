package seeder

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// mockSeeder is a simple Seeder implementation for testing.
type mockSeeder struct {
	name string
}

func (m *mockSeeder) Run(db *sql.DB) error {
	return nil
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Errorf("expected non-nil")
	}
	if len(r.GetAll()) != 0 {
		t.Errorf("expected empty, got %v", r.GetAll())
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	s := &mockSeeder{name: "users"}

	err := r.Register("users", s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := r.Get("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != got {
		t.Errorf("got %v, want %v", got, s)
	}
}

func TestRegisterDuplicateNameReturnsError(t *testing.T) {
	r := NewRegistry()
	s1 := &mockSeeder{name: "users"}
	s2 := &mockSeeder{name: "users2"}

	err := r.Register("users", s1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = r.Register("users", s2)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrDuplicateSeeder) {
		t.Errorf("expected true")
	}
	if !strings.Contains(err.Error(), "users") {
		t.Errorf("%q not in %q", "users", err.Error())
	}
}

func TestRegisterEmptyNameReturnsError(t *testing.T) {
	r := NewRegistry()
	s := &mockSeeder{name: "test"}

	tests := []struct {
		name string
	}{
		{""},
		{" "},
		{"  "},
		{"\t"},
	}

	for _, tc := range tests {
		err := r.Register(tc.name, s)
		if err == nil {
			t.Fatalf("expected error for name %q", tc.name)
		}
		if !errors.Is(err, ErrInvalidSeederName) {
			t.Errorf("expected true")
		}
	}
}

func TestGetNonExistentReturnsError(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrSeederNotFound) {
		t.Errorf("expected true")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("%q not in %q", "nonexistent", err.Error())
	}
}

func TestGetAllReturnsAllRegistered(t *testing.T) {
	r := NewRegistry()
	s1 := &mockSeeder{name: "users"}
	s2 := &mockSeeder{name: "posts"}
	s3 := &mockSeeder{name: "comments"}

	if r.Register("users", s1) != nil {
		t.Fatalf("unexpected error: %v", r.Register("users", s1))
	}
	if r.Register("posts", s2) != nil {
		t.Fatalf("unexpected error: %v", r.Register("posts", s2))
	}
	if r.Register("comments", s3) != nil {
		t.Fatalf("unexpected error: %v", r.Register("comments", s3))
	}

	all := r.GetAll()
	if len(all) != 3 {
		t.Errorf("len = %d, want %d", len(all), 3)
	}
	if s1 != all["users"] {
		t.Errorf("got %v, want %v", all["users"], s1)
	}
	if s2 != all["posts"] {
		t.Errorf("got %v, want %v", all["posts"], s2)
	}
	if s3 != all["comments"] {
		t.Errorf("got %v, want %v", all["comments"], s3)
	}
}

func TestGetAllReturnsCopy(t *testing.T) {
	r := NewRegistry()
	s := &mockSeeder{name: "users"}
	if r.Register("users", s) != nil {
		t.Fatalf("unexpected error: %v", r.Register("users", s))
	}

	all := r.GetAll()
	// Mutating the returned map should not affect the registry.
	delete(all, "users")

	got, err := r.Get("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != got {
		t.Errorf("got %v, want %v", got, s)
	}
}
