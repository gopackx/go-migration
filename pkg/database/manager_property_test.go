package database

import (
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"pgregory.net/rapid"
)

// mockPropertyDriver is a test driver that returns valid *sql.DB handles via sqlmock.
type mockPropertyDriver struct {
	driverName string
}

func (d *mockPropertyDriver) Open(config ConnectionConfig) (*sql.DB, error) {
	db, _, err := sqlmock.New()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *mockPropertyDriver) Name() string {
	return d.driverName
}

// connNameGen generates non-empty alphanumeric connection names.
func connNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,19}`)
}

// Feature: go-migration, Property 21: Named connections are retrievable
// Validates: Requirements 9.1, 9.4
func TestProperty21_NamedConnectionsAreRetrievable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 1..10 unique connection names.
		count := rapid.IntRange(1, 10).Draw(t, "count")
		nameSet := make(map[string]bool)
		names := make([]string, 0, count)
		for len(names) < count {
			name := connNameGen().Draw(t, "name")
			if !nameSet[name] {
				nameSet[name] = true
				names = append(names, name)
			}
		}

		m := NewManager()
		driver := &mockPropertyDriver{driverName: "mock"}
		m.RegisterDriver("mock", driver)

		// Register all named connections.
		for _, name := range names {
			err := m.AddConnection(name, ConnectionConfig{
				Driver:   "mock",
				Host:     "localhost",
				Database: name + "_db",
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}

		// Each registered name should return a non-nil connection without error.
		for _, name := range names {
			db, err := m.Connection(name)
			if err != nil {
				t.Errorf("Connection(%q) should not error", name)
			}
			if db == nil {
				t.Errorf("expected non-nil")
			}
		}
	})
}

// Feature: go-migration, Property 22: Unknown connection names produce errors
// Validates: Requirements 9.1, 9.4
func TestProperty22_UnknownConnectionNamesProduceErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Register 0..5 named connections.
		regCount := rapid.IntRange(0, 5).Draw(t, "regCount")
		registered := make(map[string]bool)
		regNames := make([]string, 0, regCount)
		for len(regNames) < regCount {
			name := connNameGen().Draw(t, "regName")
			if !registered[name] {
				registered[name] = true
				regNames = append(regNames, name)
			}
		}

		m := NewManager()
		driver := &mockPropertyDriver{driverName: "mock"}
		m.RegisterDriver("mock", driver)

		for _, name := range regNames {
			_ = m.AddConnection(name, ConnectionConfig{
				Driver:   "mock",
				Host:     "localhost",
				Database: name + "_db",
			})
		}

		// Generate an unknown name that is NOT in the registered set.
		unknownName := connNameGen().Filter(func(s string) bool {
			return !registered[s]
		}).Draw(t, "unknownName")

		// Requesting an unknown name should return an error wrapping ErrConnectionNotFound.
		db, err := m.Connection(unknownName)
		if db != nil {
			t.Errorf("expected nil, got %v", db)
		}
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !errors.Is(err, ErrConnectionNotFound) {
			t.Errorf("error should wrap ErrConnectionNotFound, got: %v", err)
		}
	})
}
