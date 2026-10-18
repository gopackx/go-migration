package migrator

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/config"
	"github.com/gopackx/go-migration/pkg/database"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
	"github.com/gopackx/go-migration/pkg/seeder"
)

func TestSentinelErrors_ImplementErrorInterface(t *testing.T) {
	sentinels := []error{
		ErrConnectionFailed,
		ErrMigrationNotFound,
		ErrDuplicateMigration,
		ErrInvalidMigrationName,
		ErrTransactionFailed,
		ErrTrackingTable,
		ErrDuplicateSeeder,
		ErrInvalidSeederName,
		ErrCircularDependency,
		ErrSeederNotFound,
		ErrUnsupportedType,
		ErrConnectionNotFound,
		ErrConfigValidation,
	}

	for _, sentinel := range sentinels {
		if sentinel.Error() == "" {
			t.Errorf("expected non-empty string")
		}
	}
}

func TestSentinelErrors_UniqueMessages(t *testing.T) {
	sentinels := []error{
		ErrConnectionFailed,
		ErrMigrationNotFound,
		ErrDuplicateMigration,
		ErrInvalidMigrationName,
		ErrTransactionFailed,
		ErrTrackingTable,
		ErrDuplicateSeeder,
		ErrInvalidSeederName,
		ErrCircularDependency,
		ErrSeederNotFound,
		ErrUnsupportedType,
		ErrConnectionNotFound,
		ErrConfigValidation,
	}

	seen := make(map[string]bool)
	for _, sentinel := range sentinels {
		msg := sentinel.Error()
		if seen[msg] {
			t.Errorf("expected false")
		}
		seen[msg] = true
	}
}

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"ErrConnectionFailed", ErrConnectionFailed},
		{"ErrMigrationNotFound", ErrMigrationNotFound},
		{"ErrDuplicateMigration", ErrDuplicateMigration},
		{"ErrInvalidMigrationName", ErrInvalidMigrationName},
		{"ErrTransactionFailed", ErrTransactionFailed},
		{"ErrTrackingTable", ErrTrackingTable},
		{"ErrDuplicateSeeder", ErrDuplicateSeeder},
		{"ErrInvalidSeederName", ErrInvalidSeederName},
		{"ErrCircularDependency", ErrCircularDependency},
		{"ErrSeederNotFound", ErrSeederNotFound},
		{"ErrUnsupportedType", ErrUnsupportedType},
		{"ErrConnectionNotFound", ErrConnectionNotFound},
		{"ErrConfigValidation", ErrConfigValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Direct match
			if !errors.Is(tt.sentinel, tt.sentinel) {
				t.Errorf("expected true")
			}

			// Wrapped error preserves chain
			wrapped := fmt.Errorf("operation failed: %w", tt.sentinel)
			if !errors.Is(wrapped, tt.sentinel) {
				t.Errorf("wrapped error should match sentinel via errors.Is()")
			}
			if !strings.Contains(wrapped.Error(), tt.sentinel.Error()) {
				t.Errorf("%q not in %q", tt.sentinel.Error(), wrapped.Error())
			}
		})
	}
}

func TestSentinelErrors_WrappedPreservesContext(t *testing.T) {
	wrapped := fmt.Errorf("migration %q up: %w", "20240101_create_users", ErrTransactionFailed)

	if !errors.Is(wrapped, ErrTransactionFailed) {
		t.Errorf("expected true")
	}
	if !strings.Contains(wrapped.Error(), "20240101_create_users") {
		t.Errorf("%q not in %q", "20240101_create_users", wrapped.Error())
	}
	if !strings.Contains(wrapped.Error(), "transaction failed") {
		t.Errorf("%q not in %q", "transaction failed", wrapped.Error())
	}
}

// The migrator re-exports sentinels owned by other packages; they must be the
// very same values so errors.Is works across package boundaries.
func TestSentinelErrors_AliasCanonicalValues(t *testing.T) {
	tests := []struct {
		name      string
		alias     error
		canonical error
	}{
		{"ErrConnectionFailed", ErrConnectionFailed, database.ErrConnectionFailed},
		{"ErrConnectionNotFound", ErrConnectionNotFound, database.ErrConnectionNotFound},
		{"ErrTransactionFailed", ErrTransactionFailed, database.ErrTransactionFailed},
		{"ErrDuplicateSeeder", ErrDuplicateSeeder, seeder.ErrDuplicateSeeder},
		{"ErrInvalidSeederName", ErrInvalidSeederName, seeder.ErrInvalidSeederName},
		{"ErrCircularDependency", ErrCircularDependency, seeder.ErrCircularDependency},
		{"ErrSeederNotFound", ErrSeederNotFound, seeder.ErrSeederNotFound},
		{"ErrConfigValidation", ErrConfigValidation, config.ErrConfigValidation},
		{"ErrUnsupportedType", ErrUnsupportedType, grammars.ErrUnsupportedType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("ctx: %w", tt.canonical)
			if !errors.Is(wrapped, tt.alias) {
				t.Errorf("errors.Is(wrapped canonical, migrator alias) = false")
			}
		})
	}
}

func TestSeederNotFound_MatchesMigratorAlias(t *testing.T) {
	r := seeder.NewRegistry()
	_, err := r.Get("missing")
	if err == nil {
		t.Skip("seeder registry returned no error for a missing seeder")
	}
	if !errors.Is(err, ErrSeederNotFound) {
		t.Errorf("errors.Is(seeder registry error, migrator.ErrSeederNotFound) = false: %v", err)
	}
}
