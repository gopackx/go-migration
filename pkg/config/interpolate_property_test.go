package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// genEnvVarName generates a valid environment variable name with a unique prefix
// to avoid collisions with real system env vars.
func genEnvVarName() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Use uppercase-only names to avoid case-insensitive collisions on Windows,
		// where env var names like "GOMIG_TEST_d" and "GOMIG_TEST_D" are identical.
		suffix := rapid.StringMatching(`[A-Z][A-Z0-9_]{0,12}`).Draw(t, "suffix")
		return "GOMIG_TEST_" + suffix
	})
}

// genEnvVarValue generates a random string value for an environment variable.
// Values avoid containing ${...} patterns to keep assertions clean.
func genEnvVarValue() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		return rapid.StringMatching(`[A-Za-z0-9 _\-\./,:;@#%^&*()]{0,50}`).Draw(t, "value")
	})
}

// genUniqueEnvVarNames generates a slice of unique env var names.
func genUniqueEnvVarNames(minN, maxN int) *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		n := rapid.IntRange(minN, maxN).Draw(t, "varCount")
		seen := make(map[string]bool)
		names := make([]string, 0, n)
		for len(names) < n {
			name := genEnvVarName().Draw(t, "varName")
			// Use case-folded key for uniqueness check (Windows env vars are case-insensitive)
			upper := strings.ToUpper(name)
			if !seen[upper] {
				seen[upper] = true
				names = append(names, name)
			}
		}
		return names
	})
}

// setAndCleanupEnvVars sets environment variables and returns a cleanup function.
func setAndCleanupEnvVars(vars map[string]string) func() {
	for k, v := range vars {
		os.Setenv(k, v)
	}
	return func() {
		for k := range vars {
			os.Unsetenv(k)
		}
	}
}

// Feature: library-improvements, Property 13: Environment variable interpolation replacement
// **Validates: Requirements 8.1**

func TestProperty13_EnvVarInterpolationReplacement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate unique env var names and values.
		names := genUniqueEnvVarNames(1, 5).Draw(t, "names")
		values := make(map[string]string, len(names))
		for _, name := range names {
			val := genEnvVarValue().Draw(t, "val_"+name)
			values[name] = val
		}

		// Set all env vars and defer cleanup.
		cleanup := setAndCleanupEnvVars(values)
		defer cleanup()

		// Build input with ${VAR_NAME} placeholders, optionally with surrounding text.
		var inputParts []string
		for _, name := range names {
			prefix := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "prefix_"+name)
			suffix := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "suffix_"+name)
			inputParts = append(inputParts, fmt.Sprintf("%s${%s}%s", prefix, name, suffix))
		}
		input := strings.Join(inputParts, "\n")

		result, err := InterpolateEnv([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := string(result)

		// Verify each ${VAR_NAME} was replaced with its value.
		for _, name := range names {
			if !strings.Contains(output, values[name]) {
				t.Errorf("%q not in %q", values[name], output)
			}
		}

		// Verify no ${VAR_NAME} patterns remain for the set variables.
		for _, name := range names {
			placeholder := fmt.Sprintf("${%s}", name)
			if strings.Contains(output, placeholder) {
				t.Errorf("%q should not be in %q", placeholder, output)
			}
		}
	})
}

// Feature: library-improvements, Property 14: Unresolved environment variable error
// **Validates: Requirements 8.2**

func TestProperty14_UnresolvedEnvVarError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate unique env var names that we will NOT set.
		names := genUniqueEnvVarNames(1, 5).Draw(t, "names")

		// Ensure none of the generated vars are set.
		for _, name := range names {
			os.Unsetenv(name)
		}

		// Build input with ${VAR_NAME} placeholders.
		var inputParts []string
		for _, name := range names {
			inputParts = append(inputParts, fmt.Sprintf("key: ${%s}", name))
		}
		input := strings.Join(inputParts, "\n")

		_, err := InterpolateEnv([]byte(input))

		// Verify a non-nil error is returned.
		if err == nil {
			t.Fatalf("InterpolateEnv should return an error for unset variables")
		}

		// Verify the error message contains each unresolved variable name.
		for _, name := range names {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("%q not in %q", name, err.Error())
			}
		}
	})
}

// Feature: library-improvements, Property 15: Escaped environment variable literal
// **Validates: Requirements 8.5**

func TestProperty15_EscapedEnvVarLiteral(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate unique env var names.
		names := genUniqueEnvVarNames(1, 5).Draw(t, "names")

		// Ensure the vars are NOT set so we can confirm no lookup is attempted.
		for _, name := range names {
			os.Unsetenv(name)
		}

		// Build input with $${VAR_NAME} escape sequences.
		var inputParts []string
		for _, name := range names {
			prefix := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "prefix_"+name)
			suffix := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "suffix_"+name)
			inputParts = append(inputParts, fmt.Sprintf("%s$${%s}%s", prefix, name, suffix))
		}
		input := strings.Join(inputParts, "\n")

		result, err := InterpolateEnv([]byte(input))

		// Escaped vars should NOT trigger an unresolved error.
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := string(result)

		// Verify each $${VAR_NAME} was replaced with the literal ${VAR_NAME}.
		for _, name := range names {
			literal := fmt.Sprintf("${%s}", name)
			if !strings.Contains(output, literal) {
				t.Errorf("%q not in %q", literal, output)
			}
		}

		// Verify no $${VAR_NAME} escape sequences remain in the output.
		for _, name := range names {
			escaped := fmt.Sprintf("$${%s}", name)
			if strings.Contains(output, escaped) {
				t.Errorf("%q should not be in %q", escaped, output)
			}
		}
	})
}

// Feature: reduce-external-dependencies, Property 7: Env Var Interpolation in JSON
// For any env var name/value, interpolation in a JSON config produces the correct value.
func TestProperty7_EnvVarInterpolationInJSON(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		varName := genEnvVarName().Draw(t, "varName")
		varValue := genEnvVarValue().Draw(t, "varValue")

		os.Setenv(varName, varValue)
		defer os.Unsetenv(varName)

		// Build a minimal JSON config with the env var placeholder as the host value.
		jsonConfig := fmt.Sprintf(`{
  "default": "default",
  "connections": {
    "default": {
      "driver": "postgres",
      "host": "${%s}",
      "database": "testdb"
    }
  }
}`, varName)

		result, err := InterpolateEnv([]byte(jsonConfig))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := string(result)

		// The placeholder should be replaced with the actual value.
		if !strings.Contains(output, varValue) {
			t.Errorf("output should contain %q, got %q", varValue, output)
		}

		// The placeholder should no longer be present.
		placeholder := fmt.Sprintf("${%s}", varName)
		if strings.Contains(output, placeholder) {
			t.Errorf("placeholder %q should not remain in output %q", placeholder, output)
		}
	})
}
