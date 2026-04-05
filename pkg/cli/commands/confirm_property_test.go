package commands

import (
	"bytes"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: library-improvements, Property 6: Confirmation input parsing
// **Validates: Requirements 3.7**
//
// For any string, the confirm parser SHALL return true if and only if the
// trimmed, lowercased input equals "y" or "yes". All other inputs SHALL return false.
func TestProperty6_ConfirmationInputParsing(t *testing.T) {
	t.Run("returns true for inputs that trim+lowercase to y or yes", func(t *testing.T) {
		// Generator for strings that, after trimming and lowercasing, equal "y" or "yes"
		positiveGen := rapid.Custom(func(t *rapid.T) string {
			base := rapid.SampledFrom([]string{"y", "Y", "yes", "Yes", "YES", "yEs", "yeS", "YeS", "yES", "YEs"}).Draw(t, "base")
			leadingSpaces := rapid.StringMatching(`[ \t]{0,5}`).Draw(t, "leading")
			trailingSpaces := rapid.StringMatching(`[ \t]{0,5}`).Draw(t, "trailing")
			return leadingSpaces + base + trailingSpaces
		})

		rapid.Check(t, func(t *rapid.T) {
			input := positiveGen.Draw(t, "input")

			in := bytes.NewBufferString(input + "\n")
			out := &bytes.Buffer{}

			result, err := confirm(in, out, "proceed?")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result {
				t.Errorf("expected true for input %q (trimmed+lowered=%q)", input, strings.TrimSpace(strings.ToLower(input)))
			}
		})
	})

	t.Run("returns false for inputs that do not trim+lowercase to y or yes", func(t *testing.T) {
		// Generator for arbitrary strings that are NOT "y" or "yes" after trim+lower
		negativeGen := rapid.Custom(func(t *rapid.T) string {
			s := rapid.String().Draw(t, "input")
			// Filter out strings that would be accepted
			normalized := strings.TrimSpace(strings.ToLower(s))
			if normalized == "y" || normalized == "yes" {
				// Replace with something definitely negative
				return "no"
			}
			return s
		})

		rapid.Check(t, func(t *rapid.T) {
			input := negativeGen.Draw(t, "input")

			// Ensure the input doesn't contain a newline (which would break ReadString('\n'))
			sanitized := strings.ReplaceAll(input, "\n", "")
			sanitized = strings.ReplaceAll(sanitized, "\r", "")

			in := bytes.NewBufferString(sanitized + "\n")
			out := &bytes.Buffer{}

			result, err := confirm(in, out, "proceed?")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			normalized := strings.TrimSpace(strings.ToLower(sanitized))
			if result {
				t.Errorf("expected false for input %q (trimmed+lowered=%q)", sanitized, normalized)
			}
		})
	})

	t.Run("arbitrary string matches expected confirm logic", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			input := rapid.String().Draw(t, "input")

			// Remove embedded newlines so ReadString('\n') reads the whole thing
			sanitized := strings.ReplaceAll(input, "\n", "")
			sanitized = strings.ReplaceAll(sanitized, "\r", "")

			in := bytes.NewBufferString(sanitized + "\n")
			out := &bytes.Buffer{}

			result, err := confirm(in, out, "proceed?")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			normalized := strings.TrimSpace(strings.ToLower(sanitized))
			expected := normalized == "y" || normalized == "yes"
			if result != expected {
				t.Errorf("for input %q (normalized=%q), expected %v but got %v", sanitized, normalized, expected, result)
			}
		})
	})
}
