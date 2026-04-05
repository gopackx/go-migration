package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// snakeCaseDescriptionGen generates valid snake_case migration/seeder descriptions.
func snakeCaseDescriptionGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		numParts := rapid.IntRange(1, 4).Draw(t, "numParts")
		parts := make([]string, numParts)
		for i := 0; i < numParts; i++ {
			parts[i] = rapid.StringMatching(`[a-z]{2,8}`).Draw(t, "part")
		}
		return strings.Join(parts, "_")
	})
}

// tableNameGen generates valid table names.
func tableNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z]{3,12}`)
}

// Feature: go-migration, Property 31: Generated migration files contain valid scaffolding
// **Validates: Requirements 16.1, 16.3**
func TestProperty31_GeneratedMigrationFilesContainValidScaffolding(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		description := snakeCaseDescriptionGen().Draw(t, "description")

		dir, err := os.MkdirTemp("", "gen-test-*")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.RemoveAll(dir)

		fixedTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
		gen := NewGenerator(dir)
		gen.nowFunc = func() time.Time { return fixedTime }
		gen.randFunc = func(n int) int { return 5678 }

		outPath, err := gen.Migration(description, MigrationOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify filename pattern: YYYY_MM_DD_HHMMSS_RRRR_description.go
		filename := filepath.Base(outPath)
		filenamePattern := regexp.MustCompile(`^\d{4}_\d{2}_\d{2}_\d{6}_\d{4}_` + regexp.QuoteMeta(description) + `\.go$`)
		if !filenamePattern.MatchString(filename) {
			t.Errorf("filename %q does not match expected pattern", filename)
		}

		// Verify timestamp prefix
		if !strings.HasPrefix(filename, "2024_06_15_103045_5678_") {
			t.Errorf("Filename should start with the correct timestamp and random segment")
		}

		// Read generated content
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		contentStr := string(content)

		// Verify struct name (PascalCase conversion of description)
		expectedStruct := toStructName(description)
		if !strings.Contains(contentStr, "type "+expectedStruct+" struct{}") {
			t.Errorf("%q not in %q", "type "+expectedStruct+" struct{}", contentStr)
		}

		// Verify Up() method
		if !strings.Contains(contentStr, "func (m *"+expectedStruct+") Up(s *schema.Builder) error") {
			t.Errorf("%q not in %q", "func (m *"+expectedStruct+") Up(s *schema.Builder) error", contentStr)
		}

		// Verify Down() method
		if !strings.Contains(contentStr, "func (m *"+expectedStruct+") Down(s *schema.Builder) error") {
			t.Errorf("%q not in %q", "func (m *"+expectedStruct+") Down(s *schema.Builder) error", contentStr)
		}

		// Verify package declaration
		if !strings.Contains(contentStr, "package migrations") {
			t.Errorf("%q not in %q", "package migrations", contentStr)
		}
	})
}

// Feature: go-migration, Property 32: Generated seeder files contain valid scaffolding
// **Validates: Requirements 16.2, 16.3**
func TestProperty32_GeneratedSeederFilesContainValidScaffolding(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		description := snakeCaseDescriptionGen().Draw(t, "description")

		dir, err := os.MkdirTemp("", "gen-test-*")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.RemoveAll(dir)

		gen := NewGenerator(dir)

		outPath, err := gen.Seeder(description)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify filename pattern: description_seeder.go
		filename := filepath.Base(outPath)
		expectedFilename := description + "_seeder.go"
		if expectedFilename != filename {
			t.Errorf("Filename should match description_seeder.go pattern")
		}

		// Read generated content
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		contentStr := string(content)

		// Verify struct name (PascalCase conversion of description)
		expectedStruct := toStructName(description)
		if !strings.Contains(contentStr, "type "+expectedStruct+" struct{}") {
			t.Errorf("%q not in %q", "type "+expectedStruct+" struct{}", contentStr)
		}

		// Verify Run() method
		if !strings.Contains(contentStr, "func (s *"+expectedStruct+") Run(db *sql.DB) error") {
			t.Errorf("%q not in %q", "func (s *"+expectedStruct+") Run(db *sql.DB) error", contentStr)
		}

		// Verify package declaration
		if !strings.Contains(contentStr, "package seeders") {
			t.Errorf("%q not in %q", "package seeders", contentStr)
		}
	})
}

// Feature: go-migration, Property 33: Create flag pre-populates schema create call
// **Validates: Requirements 16.4**
func TestProperty33_CreateFlagPrePopulatesSchemaCreateCall(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		description := snakeCaseDescriptionGen().Draw(t, "description")
		tableName := tableNameGen().Draw(t, "tableName")

		dir, err := os.MkdirTemp("", "gen-test-*")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.RemoveAll(dir)

		gen := NewGenerator(dir)
		gen.nowFunc = func() time.Time { return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) }
		gen.randFunc = func(n int) int { return 0 }

		outPath, err := gen.Migration(description, MigrationOptions{
			CreateTable: tableName,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		contentStr := string(content)

		// Up() should contain s.Create("tableName", ...)
		if !strings.Contains(contentStr, `s.Create("`+tableName+`"`) {
			t.Errorf("%q not in %q", `s.Create("`+tableName+`"`, contentStr)
		}
	})
}
