package config

import (
	"os"
	"strings"
	"testing"
)

func TestInterpolateEnv_BasicReplacement(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")

	input := []byte("host: ${DB_HOST}\nport: ${DB_PORT}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "host: localhost\nport: 5432" != string(result) {
		t.Errorf("got %v, want %v", string(result), "host: localhost\nport: 5432")
	}
}

func TestInterpolateEnv_PartialStringReplacement(t *testing.T) {
	t.Setenv("REGION", "us-east-1")

	input := []byte(`host: "db-${REGION}.example.com"`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if `host: "db-us-east-1.example.com"` != string(result) {
		t.Errorf("got %v, want %v", string(result), `host: "db-us-east-1.example.com"`)
	}
}

func TestInterpolateEnv_UnresolvedVariable(t *testing.T) {
	// Ensure the variable is not set
	os.Unsetenv("MISSING_VAR")

	input := []byte("password: ${MISSING_VAR}")
	_, err := InterpolateEnv(input)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "MISSING_VAR") {
		t.Errorf("%q not in %q", "MISSING_VAR", err.Error())
	}
	if !strings.Contains(err.Error(), "unresolved environment variables") {
		t.Errorf("%q not in %q", "unresolved environment variables", err.Error())
	}
}

func TestInterpolateEnv_MultipleUnresolved(t *testing.T) {
	os.Unsetenv("VAR_A")
	os.Unsetenv("VAR_B")

	input := []byte("a: ${VAR_A}\nb: ${VAR_B}")
	_, err := InterpolateEnv(input)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "VAR_A") {
		t.Errorf("%q not in %q", "VAR_A", err.Error())
	}
	if !strings.Contains(err.Error(), "VAR_B") {
		t.Errorf("%q not in %q", "VAR_B", err.Error())
	}
}

func TestInterpolateEnv_EscapedPlaceholder(t *testing.T) {
	// $${VAR} should become literal ${VAR}, not interpolated
	t.Setenv("SECRET", "should_not_appear")

	input := []byte("literal: $${SECRET}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "literal: ${SECRET}" != string(result) {
		t.Errorf("got %v, want %v", string(result), "literal: ${SECRET}")
	}
}

func TestInterpolateEnv_EscapedAndRegularMixed(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")

	input := []byte("host: ${DB_HOST}\nexample: $${DB_HOST}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "host: localhost\nexample: ${DB_HOST}" != string(result) {
		t.Errorf("got %v, want %v", string(result), "host: localhost\nexample: ${DB_HOST}")
	}
}

func TestInterpolateEnv_NoPlaceholders(t *testing.T) {
	input := []byte("host: localhost\nport: 5432")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "host: localhost\nport: 5432" != string(result) {
		t.Errorf("got %v, want %v", string(result), "host: localhost\nport: 5432")
	}
}

func TestInterpolateEnv_EmptyInput(t *testing.T) {
	result, err := InterpolateEnv([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "" != string(result) {
		t.Errorf("got %v, want %v", string(result), "")
	}
}

func TestInterpolateEnv_EmptyValue(t *testing.T) {
	t.Setenv("EMPTY_VAR", "")

	input := []byte("val: ${EMPTY_VAR}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "val: " != string(result) {
		t.Errorf("got %v, want %v", string(result), "val: ")
	}
}

func TestInterpolateEnv_UnderscoreVarName(t *testing.T) {
	t.Setenv("_PRIVATE_VAR", "secret")

	input := []byte("val: ${_PRIVATE_VAR}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "val: secret" != string(result) {
		t.Errorf("got %v, want %v", string(result), "val: secret")
	}
}

func TestInterpolateEnv_MultipleEscaped(t *testing.T) {
	input := []byte("a: $${FOO}\nb: $${BAR}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "a: ${FOO}\nb: ${BAR}" != string(result) {
		t.Errorf("got %v, want %v", string(result), "a: ${FOO}\nb: ${BAR}")
	}
}

func TestInterpolateEnv_EscapedUnsetVarNoError(t *testing.T) {
	// Escaped placeholders should not trigger unresolved errors
	os.Unsetenv("UNSET_VAR")

	input := []byte("val: $${UNSET_VAR}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "val: ${UNSET_VAR}" != string(result) {
		t.Errorf("got %v, want %v", string(result), "val: ${UNSET_VAR}")
	}
}

func TestInterpolateEnv_SameVarMultipleTimes(t *testing.T) {
	t.Setenv("REPEATED", "value")

	input := []byte("a: ${REPEATED} b: ${REPEATED}")
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "a: value b: value" != string(result) {
		t.Errorf("got %v, want %v", string(result), "a: value b: value")
	}
}

func TestInterpolateEnv_JSONNumericUnquoting(t *testing.T) {
	t.Setenv("DB_PORT", "5432")

	input := []byte(`{"port": "${DB_PORT}"}`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"port": 5432}`
	if expected != string(result) {
		t.Errorf("got %v, want %v", string(result), expected)
	}
}

func TestInterpolateEnv_JSONBooleanUnquoting(t *testing.T) {
	t.Setenv("ENABLED", "true")

	input := []byte(`{"enabled": "${ENABLED}"}`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"enabled": true}`
	if expected != string(result) {
		t.Errorf("got %v, want %v", string(result), expected)
	}
}

func TestInterpolateEnv_JSONNullUnquoting(t *testing.T) {
	t.Setenv("NULLABLE", "null")

	input := []byte(`{"value": "${NULLABLE}"}`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"value": null}`
	if expected != string(result) {
		t.Errorf("got %v, want %v", string(result), expected)
	}
}

func TestInterpolateEnv_JSONStringStaysQuoted(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")

	input := []byte(`{"host": "${DB_HOST}"}`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"host": "localhost"}`
	if expected != string(result) {
		t.Errorf("got %v, want %v", string(result), expected)
	}
}

func TestInterpolateEnv_JSONMixedTypes(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "mydb")

	input := []byte(`{"host": "${DB_HOST}", "port": "${DB_PORT}", "database": "${DB_NAME}"}`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"host": "localhost", "port": 5432, "database": "mydb"}`
	if expected != string(result) {
		t.Errorf("got %v, want %v", string(result), expected)
	}
}

func TestInterpolateEnv_JSONFloatUnquoting(t *testing.T) {
	t.Setenv("TIMEOUT", "3.14")

	input := []byte(`{"timeout": "${TIMEOUT}"}`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"timeout": 3.14}`
	if expected != string(result) {
		t.Errorf("got %v, want %v", string(result), expected)
	}
}

func TestInterpolateEnv_JSONPartialStringNoUnquoting(t *testing.T) {
	t.Setenv("REGION", "us-east-1")

	// Placeholder is part of a larger string — should NOT unquote
	input := []byte(`{"host": "db-${REGION}.example.com"}`)
	result, err := InterpolateEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"host": "db-us-east-1.example.com"}`
	if expected != string(result) {
		t.Errorf("got %v, want %v", string(result), expected)
	}
}
