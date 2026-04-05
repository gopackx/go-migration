package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// envVarPattern matches ${VAR_NAME} placeholders where VAR_NAME is a valid
// identifier (starts with letter or underscore, followed by alphanumerics or underscores).
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// escapePattern matches $${...} escape sequences — a double dollar sign before a brace placeholder.
var escapePattern = regexp.MustCompile(`\$\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// quotedEnvPattern matches a JSON string value that consists entirely of a single
// ${VAR_NAME} placeholder, e.g. "${DB_PORT}". Used to detect cases where the
// resolved value should be unquoted (numbers, booleans, null) for valid JSON.
var quotedEnvPattern = regexp.MustCompile(`"\$\{([A-Za-z_][A-Za-z0-9_]*)\}"`)

// isJSONLiteral reports whether s is a valid JSON number, boolean, or null that
// should appear unquoted in JSON output.
func isJSONLiteral(s string) bool {
	if s == "true" || s == "false" || s == "null" {
		return true
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// InterpolateEnv replaces ${VAR_NAME} placeholders in data with the corresponding
// environment variable values. Escaped sequences ($${VAR_NAME}) are replaced with
// the literal text ${VAR_NAME}. Returns an error listing all unresolved variables.
//
// When a quoted JSON string consists entirely of a single placeholder (e.g.
// "${DB_PORT}") and the resolved value is a number, boolean, or null, the
// surrounding quotes are removed so that the JSON remains valid. For example,
// "port": "${DB_PORT}" with DB_PORT=5433 becomes "port": 5433.
func InterpolateEnv(data []byte) ([]byte, error) {
	s := string(data)

	// Step 1: Replace escape sequences $${VAR} with a temporary sentinel that
	// won't be matched by the interpolation regex.
	const sentinel = "\x00ESCAPED_ENV\x00"
	type escapedEntry struct {
		varName string
	}
	var escaped []escapedEntry

	s = escapePattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := escapePattern.FindStringSubmatch(match)
		escaped = append(escaped, escapedEntry{varName: sub[1]})
		return fmt.Sprintf("%s%d%s", sentinel, len(escaped)-1, sentinel)
	})

	// Step 2: Handle quoted standalone placeholders ("${VAR}") first.
	// If the env value is a JSON literal (number, bool, null), strip the quotes.
	var unresolved []string
	s = quotedEnvPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := quotedEnvPattern.FindStringSubmatch(match)
		varName := sub[1]
		val, ok := os.LookupEnv(varName)
		if !ok {
			unresolved = append(unresolved, varName)
			return match
		}
		if isJSONLiteral(val) {
			return val // unquoted: "port": 5433
		}
		return `"` + val + `"` // keep as string: "host": "localhost"
	})

	// Step 3: Replace remaining ${VAR} placeholders (inside larger strings or unquoted).
	s = envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := envVarPattern.FindStringSubmatch(match)
		varName := sub[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		for _, u := range unresolved {
			if u == varName {
				return match
			}
		}
		unresolved = append(unresolved, varName)
		return match
	})

	// Step 4: Restore escaped sequences to literal ${VAR_NAME}.
	for i, entry := range escaped {
		placeholder := fmt.Sprintf("%s%d%s", sentinel, i, sentinel)
		s = strings.Replace(s, placeholder, "${"+entry.varName+"}", 1)
	}

	if len(unresolved) > 0 {
		return nil, fmt.Errorf("unresolved environment variables: %s", strings.Join(unresolved, ", "))
	}

	return []byte(s), nil
}
