package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockDriver is a test double that records calls and returns a controllable *sql.DB.
type mockDriver struct {
	name   string
	openFn func(config ConnectionConfig) (*sql.DB, error)
}

func (d *mockDriver) Open(config ConnectionConfig) (*sql.DB, error) {
	if d.openFn != nil {
		return d.openFn(config)
	}
	return nil, fmt.Errorf("mockDriver.Open not configured")
}

func (d *mockDriver) Name() string {
	return d.name
}

// newTestDriver creates a mock driver that opens a SQLite in-memory DB.
// This avoids needing real database servers for unit tests.
func newTestDriver(name string) *mockDriver {
	return &mockDriver{
		name: name,
		openFn: func(config ConnectionConfig) (*sql.DB, error) {
			// Use sql.Open with a driver that won't actually connect
			// but gives us a valid *sql.DB handle for testing.
			// We use "sqlmock" pattern — just return a stub.
			// Since we can't import sqlmock here, we'll use a simple approach:
			// open with an unknown driver to get a *sql.DB that won't ping.
			db, err := sql.Open("txdb_test", "test")
			if err != nil {
				// Fallback: create a minimal *sql.DB
				return nil, err
			}
			return db, nil
		},
	}
}

// fakeDB creates a minimal *sql.DB for testing purposes.
// We register a fake driver once and reuse it.
func fakeDB(t *testing.T) *sql.DB {
	t.Helper()
	// We can't easily create a *sql.DB without a registered driver,
	// so we'll test the manager logic through its public API.
	return nil
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Errorf("expected non-nil")
	}
	if m.connections == nil {
		t.Errorf("expected non-nil")
	}
	if m.configs == nil {
		t.Errorf("expected non-nil")
	}
	if m.drivers == nil {
		t.Errorf("expected non-nil")
	}
	if len(m.defaultName) != 0 {
		t.Errorf("expected empty, got %v", m.defaultName)
	}
}

func TestRegisterDriver(t *testing.T) {
	m := NewManager()
	d := &mockDriver{name: "postgres"}

	m.RegisterDriver("postgres", d)

	if len(m.drivers) != 1 {
		t.Errorf("len = %d, want %d", len(m.drivers), 1)
	}
	if d != m.drivers["postgres"] {
		t.Errorf("got %v, want %v", m.drivers["postgres"], d)
	}
}

func TestRegisterDriver_Multiple(t *testing.T) {
	m := NewManager()
	pg := &mockDriver{name: "postgres"}
	my := &mockDriver{name: "mysql"}

	m.RegisterDriver("postgres", pg)
	m.RegisterDriver("mysql", my)

	if len(m.drivers) != 2 {
		t.Errorf("len = %d, want %d", len(m.drivers), 2)
	}
}

func TestAddConnection(t *testing.T) {
	m := NewManager()

	err := m.AddConnection("primary", ConnectionConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := m.configs["primary"]; !ok {
		t.Errorf("expected config for %q", "primary")
	}
	// First connection becomes default.
	if "primary" != m.defaultName {
		t.Errorf("got %v, want %v", m.defaultName, "primary")
	}
}

func TestAddConnection_MissingDriver(t *testing.T) {
	m := NewManager()

	err := m.AddConnection("bad", ConnectionConfig{
		Host:     "localhost",
		Database: "testdb",
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "driver is required") {
		t.Errorf("%q not in %q", "driver is required", err.Error())
	}
}

func TestAddConnection_FirstBecomesDefault(t *testing.T) {
	m := NewManager()

	_ = m.AddConnection("first", ConnectionConfig{Driver: "pg"})
	_ = m.AddConnection("second", ConnectionConfig{Driver: "pg"})

	if "first" != m.defaultName {
		t.Errorf("got %v, want %v", m.defaultName, "first")
	}
}

func TestConnection_NotFound(t *testing.T) {
	m := NewManager()

	_, err := m.Connection("nonexistent")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConnectionNotFound) {
		t.Errorf("expected true")
	}
}

func TestConnection_DriverNotFound(t *testing.T) {
	m := NewManager()
	_ = m.AddConnection("test", ConnectionConfig{
		Driver:   "unknown_driver",
		Host:     "localhost",
		Database: "testdb",
	})

	_, err := m.Connection("test")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrDriverNotFound) {
		t.Errorf("expected true")
	}
}

func TestConnection_DriverOpenFails(t *testing.T) {
	m := NewManager()
	m.RegisterDriver("failing", &mockDriver{
		name: "failing",
		openFn: func(config ConnectionConfig) (*sql.DB, error) {
			return nil, fmt.Errorf("connection refused")
		},
	})
	_ = m.AddConnection("test", ConnectionConfig{
		Driver:   "failing",
		Host:     "localhost",
		Database: "testdb",
	})

	_, err := m.Connection("test")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConnectionFailed) {
		t.Errorf("expected true")
	}
}

func TestConnection_LazyOpen(t *testing.T) {
	openCalled := false
	m := NewManager()
	m.RegisterDriver("lazy", &mockDriver{
		name: "lazy",
		openFn: func(config ConnectionConfig) (*sql.DB, error) {
			openCalled = true
			// Return a real *sql.DB via sql.Open with a dummy driver name.
			// This will create a *sql.DB that can't actually connect,
			// but is valid for our manager caching tests.
			db, _ := sql.Open("txdb_nonexist", "")
			return db, nil
		},
	})
	_ = m.AddConnection("test", ConnectionConfig{
		Driver:   "lazy",
		Database: "testdb",
	})

	// Not opened yet after AddConnection.
	if openCalled {
		t.Errorf("expected false")
	}
}

func TestDefault_NoDefault(t *testing.T) {
	m := NewManager()

	_, err := m.Default()

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrNoDefault) {
		t.Errorf("expected true")
	}
}

func TestSetDefault(t *testing.T) {
	m := NewManager()
	_ = m.AddConnection("primary", ConnectionConfig{Driver: "pg"})
	_ = m.AddConnection("secondary", ConnectionConfig{Driver: "pg"})

	err := m.SetDefault("secondary")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "secondary" != m.defaultName {
		t.Errorf("got %v, want %v", m.defaultName, "secondary")
	}
}

func TestSetDefault_NotFound(t *testing.T) {
	m := NewManager()

	err := m.SetDefault("nonexistent")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConnectionNotFound) {
		t.Errorf("expected true")
	}
}

func TestClose_NoConnections(t *testing.T) {
	m := NewManager()

	err := m.Close()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPoolConfig(t *testing.T) {
	config := ConnectionConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// Verify the config values are stored correctly.
	if 25 != config.MaxOpenConns {
		t.Errorf("got %v, want %v", config.MaxOpenConns, 25)
	}
	if 5 != config.MaxIdleConns {
		t.Errorf("got %v, want %v", config.MaxIdleConns, 5)
	}
	if 5*time.Minute != config.ConnMaxLifetime {
		t.Errorf("got %v, want %v", config.ConnMaxLifetime, 5*time.Minute)
	}
}

func TestDriverInterface_Compliance(t *testing.T) {
	// Verify mockDriver implements the Driver interface.
	var _ Driver = (*mockDriver)(nil)
}

func TestConnection_ErrorMessages(t *testing.T) {
	m := NewManager()

	_, err := m.Connection("missing")
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("%q not in %q", "missing", err.Error())
	}
	if !strings.Contains(err.Error(), "connection not found") {
		t.Errorf("%q not in %q", "connection not found", err.Error())
	}
}

func TestSetDefault_ErrorMessage(t *testing.T) {
	m := NewManager()

	err := m.SetDefault("missing")
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("%q not in %q", "missing", err.Error())
	}
	if !strings.Contains(err.Error(), "connection not found") {
		t.Errorf("%q not in %q", "connection not found", err.Error())
	}
}
