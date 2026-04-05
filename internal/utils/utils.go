// Package utils provides minimal utility functions for string conversion,
// timestamp generation, and file system helpers.
package utils

import (
	"os"
	"strings"
	"time"
	"unicode"
)

// ToSnakeCase converts a CamelCase or mixed-case string to snake_case.
// For example, "CreateUsers" becomes "create_users" and "getHTTPResponse"
// becomes "get_http_response".
func ToSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				// Insert underscore before an uppercase letter when preceded by
				// a lowercase letter, or when preceded by an uppercase letter
				// that is followed by a lowercase letter (e.g. "HTTPResponse" → "http_response").
				if unicode.IsLower(prev) {
					result.WriteRune('_')
				} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// TimestampPrefix returns the current time formatted as YYYYMMDDHHMMSS.
func TimestampPrefix() string {
	return time.Now().Format("20060102150405")
}

// FileExists reports whether the named file or directory exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates the directory at path (including parents) if it does
// not already exist. Returns nil if the directory already exists.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
