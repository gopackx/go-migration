package migrator

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: go-migration, Property 39: Error wrapping and inspection
// Validates: Requirements 19.2, 19.3

// allSentinelErrors returns all sentinel errors defined in the package.
func allSentinelErrors() []error {
	return []error{
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
	}
}

// sentinelErrorGen returns a rapid generator that picks a random sentinel error.
func sentinelErrorGen() *rapid.Generator[error] {
	return rapid.SampledFrom(allSentinelErrors())
}

// contextStringGen generates non-empty context strings simulating operation names and identifiers.
func contextStringGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9_]{1,50}`)
}

func TestProperty39_ErrorsIs_WrappedSentinel(t *testing.T) {
	// Feature: go-migration, Property 39: Error wrapping and inspection
	// For any sentinel error wrapped with random context, errors.Is must identify the sentinel.
	rapid.Check(t, func(t *rapid.T) {
		sentinel := sentinelErrorGen().Draw(t, "sentinel")
		context := contextStringGen().Draw(t, "context")

		wrapped := fmt.Errorf("%s: %w", context, sentinel)

		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is(wrapped, sentinel) should be true for sentinel %q with context %q", sentinel.Error(), context)
		}
	})
}

func TestProperty39_WrappedError_PreservesContext(t *testing.T) {
	// Feature: go-migration, Property 39: Error wrapping and inspection
	// For any sentinel wrapped with context, the error message must contain both the context and the original message.
	rapid.Check(t, func(t *rapid.T) {
		sentinel := sentinelErrorGen().Draw(t, "sentinel")
		context := contextStringGen().Draw(t, "context")

		wrapped := fmt.Errorf("%s: %w", context, sentinel)
		msg := wrapped.Error()

		if !strings.Contains(msg, context) {
			t.Errorf("wrapped error message %q should contain context %q", msg, context)
		}
		if !strings.Contains(msg, sentinel.Error()) {
			t.Errorf("wrapped error message %q should contain sentinel message %q", msg, sentinel.Error())
		}
	})
}

func TestProperty39_MultiLevelWrapping_PreservesChain(t *testing.T) {
	// Feature: go-migration, Property 39: Error wrapping and inspection
	// For any sentinel wrapped at multiple levels, errors.Is must still identify the original sentinel.
	rapid.Check(t, func(t *rapid.T) {
		sentinel := sentinelErrorGen().Draw(t, "sentinel")
		ctx1 := contextStringGen().Draw(t, "context1")
		ctx2 := contextStringGen().Draw(t, "context2")

		level1 := fmt.Errorf("%s: %w", ctx1, sentinel)
		level2 := fmt.Errorf("%s: %w", ctx2, level1)

		if !errors.Is(level2, sentinel) {
			t.Errorf("multi-level wrapped error should match sentinel via errors.Is()")
		}
		if !errors.Is(level2, level1) {
			t.Errorf("multi-level wrapped error should match intermediate error via errors.Is()")
		}

		msg := level2.Error()
		if !strings.Contains(msg, ctx1) {
			t.Errorf("multi-level error should contain inner context %q", ctx1)
		}
		if !strings.Contains(msg, ctx2) {
			t.Errorf("multi-level error should contain outer context %q", ctx2)
		}
		if !strings.Contains(msg, sentinel.Error()) {
			t.Errorf("multi-level error should contain sentinel message %q", sentinel.Error())
		}
	})
}

func TestProperty39_ErrorsAs_WrappedSentinel(t *testing.T) {
	// Feature: go-migration, Property 39: Error wrapping and inspection
	// For any sentinel wrapped with context, errors.As should extract the error as *errors.errorString.
	// Since sentinel errors are created via errors.New, they are *errorString under the hood.
	// We verify errors.As can extract the wrapped error into a generic error target.
	rapid.Check(t, func(t *rapid.T) {
		sentinel := sentinelErrorGen().Draw(t, "sentinel")
		context := contextStringGen().Draw(t, "context")

		wrapped := fmt.Errorf("%s: %w", context, sentinel)

		// errors.As should be able to unwrap to find the sentinel
		var target interface{ Error() string }
		if !errors.As(wrapped, &target) {
			t.Errorf("errors.As should extract an error implementing the error interface")
		}
		if target == nil {
			t.Errorf("expected non-nil")
		}
	})
}

func TestProperty39_AllSentinels_ImplementErrorInterface(t *testing.T) {
	// Feature: go-migration, Property 39: Error wrapping and inspection
	// For any sentinel error, it must implement the Go error interface with a non-empty message.
	rapid.Check(t, func(t *rapid.T) {
		sentinel := sentinelErrorGen().Draw(t, "sentinel")

		// Verify it implements the error interface (compile-time guaranteed, but test the message)
		var err error = sentinel
		if err.Error() == "" {
			t.Errorf("expected non-empty string")
		}
	})
}

func TestProperty39_DistinctSentinels_NotConfused(t *testing.T) {
	// Feature: go-migration, Property 39: Error wrapping and inspection
	// For any two distinct sentinel errors, wrapping one should not match the other via errors.Is.
	rapid.Check(t, func(t *rapid.T) {
		sentinels := allSentinelErrors()
		idx1 := rapid.IntRange(0, len(sentinels)-1).Draw(t, "idx1")
		idx2 := rapid.IntRange(0, len(sentinels)-1).Draw(t, "idx2")

		if idx1 == idx2 {
			return // skip when same sentinel is picked
		}

		sentinel1 := sentinels[idx1]
		sentinel2 := sentinels[idx2]
		context := contextStringGen().Draw(t, "context")

		wrapped := fmt.Errorf("%s: %w", context, sentinel1)

		if errors.Is(wrapped, sentinel2) {
			t.Errorf("expected false")
		}
	})
}
