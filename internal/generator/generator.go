// Package generator provides migration and seeder file generation
// using Go text/template and embedded template files.
package generator

import (
	"embed"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// MigrationOptions configures optional flags for migration generation.
type MigrationOptions struct {
	// CreateTable pre-populates Up() with a Schema_Builder Create call.
	CreateTable string
	// AlterTable pre-populates Up() with a Schema_Builder Alter call.
	AlterTable string
}

// templateData holds the data passed to templates.
type templateData struct {
	StructName    string
	TableName     string
	MigrationName string // e.g., "2024_02_15_120405_4827_create_users_table"
}

// Generator generates migration and seeder files from templates.
type Generator struct {
	outputDir string
	// nowFunc allows overriding time for testing.
	nowFunc func() time.Time
	// randFunc allows overriding random number generation for testing.
	randFunc func(int) int
}

// NewGenerator creates a new Generator that writes files to outputDir.
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
		nowFunc:   time.Now,
		randFunc:  rand.Intn,
	}
}

// Migration generates a migration file and returns the full filepath.
// The filename follows the pattern YYYY_MM_DD_HHMMSS_RRRR_description.go.
// The opts parameter controls whether the template includes pre-populated
// Schema_Builder calls for --create or --table flags.
func (g *Generator) Migration(description string, opts MigrationOptions) (string, error) {
	timestamp := g.nowFunc().Format("2006_01_02_150405")
	random := fmt.Sprintf("%04d", g.randFunc(10000))
	filename := fmt.Sprintf("%s_%s_%s.go", timestamp, random, description)
	migrationName := strings.TrimSuffix(filename, ".go")
	structName := toStructName(description)

	tmplName := "templates/migration.go.tmpl"
	data := templateData{StructName: structName, MigrationName: migrationName}

	if opts.CreateTable != "" {
		tmplName = "templates/migration_create.go.tmpl"
		data.TableName = opts.CreateTable
	} else if opts.AlterTable != "" {
		tmplName = "templates/migration_alter.go.tmpl"
		data.TableName = opts.AlterTable
	}

	content, err := templateFS.ReadFile(tmplName)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", tmplName, err)
	}

	tmpl, err := template.New(filepath.Base(tmplName)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", tmplName, err)
	}

	outPath := filepath.Join(g.outputDir, filename)
	if err := os.MkdirAll(g.outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create file %s: %w", outPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return outPath, nil
}

// Seeder generates a seeder file and returns the full filepath.
// The filename follows the pattern description_seeder.go.
func (g *Generator) Seeder(description string) (string, error) {
	filename := fmt.Sprintf("%s_seeder.go", description)
	structName := toStructName(description)

	content, err := templateFS.ReadFile("templates/seeder.go.tmpl")
	if err != nil {
		return "", fmt.Errorf("read seeder template: %w", err)
	}

	tmpl, err := template.New("seeder.go.tmpl").Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse seeder template: %w", err)
	}

	data := templateData{StructName: structName}

	outPath := filepath.Join(g.outputDir, filename)
	if err := os.MkdirAll(g.outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create file %s: %w", outPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return outPath, nil
}

// Factory generates a factory file and returns the full filepath.
// The filename follows the pattern description_factory.go.
func (g *Generator) Factory(description string) (string, error) {
	filename := fmt.Sprintf("%s_factory.go", description)
	structName := toStructName(description)

	content, err := templateFS.ReadFile("templates/factory.go.tmpl")
	if err != nil {
		return "", fmt.Errorf("read factory template: %w", err)
	}

	tmpl, err := template.New("factory.go.tmpl").Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse factory template: %w", err)
	}

	data := templateData{StructName: structName}

	outPath := filepath.Join(g.outputDir, filename)
	if err := os.MkdirAll(g.outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create file %s: %w", outPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return outPath, nil
}

// toStructName converts a snake_case description to PascalCase.
// For example, "create_users" becomes "CreateUsers".
func toStructName(description string) string {
	parts := strings.Split(description, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}
