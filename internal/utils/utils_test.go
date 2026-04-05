package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CreateUsers", "create_users"},
		{"getHTTPResponse", "get_http_response"},
		{"simpleTest", "simple_test"},
		{"already_snake", "already_snake"},
		{"A", "a"},
		{"", ""},
		{"JSONParser", "json_parser"},
		{"myURL", "my_url"},
		{"ID", "id"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if tc.expected != ToSnakeCase(tc.input) {
				t.Errorf("got %v, want %v", ToSnakeCase(tc.input), tc.expected)
			}
		})
	}
}

func TestTimestampPrefix(t *testing.T) {
	ts := TimestampPrefix()
	// Must be exactly 14 digits matching YYYYMMDDHHMMSS.
	if len(ts) != 14 {
		t.Errorf("len = %d, want %d", len(ts), 14)
	}
	if !regexp.MustCompile(`^\d{14}$`).MatchString(ts) {
		t.Errorf("%q does not match pattern", ts)
	}
}

func TestFileExists(t *testing.T) {
	// Existing file
	tmp := t.TempDir()
	f := filepath.Join(tmp, "exists.txt")
	if os.WriteFile(f, []byte("hi"), 0o644) != nil {
		t.Fatalf("unexpected error: %v", os.WriteFile(f, []byte("hi"), 0o644))
	}
	if !FileExists(f) {
		t.Errorf("expected true")
	}

	// Non-existing file
	if FileExists(filepath.Join(tmp, "nope.txt")) {
		t.Errorf("expected false")
	}

	// Directory counts as existing
	if !FileExists(tmp) {
		t.Errorf("expected true")
	}
}

func TestEnsureDir(t *testing.T) {
	tmp := t.TempDir()
	nested := filepath.Join(tmp, "a", "b", "c")

	if EnsureDir(nested) != nil {
		t.Fatalf("unexpected error: %v", EnsureDir(nested))
	}
	if !FileExists(nested) {
		t.Errorf("expected true")
	}

	// Calling again is idempotent
	if EnsureDir(nested) != nil {
		t.Fatalf("unexpected error: %v", EnsureDir(nested))
	}
}
