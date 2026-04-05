package generator

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// InitScaffolder handles all file system operations for the init command.
type InitScaffolder struct {
	baseDir    string
	modulePath string
	force      bool
	stdout     io.Writer
	stderr     io.Writer
}

// InitResult summarizes what the scaffolder created or skipped.
type InitResult struct {
	DirsCreated  []string
	FilesCreated []string
	FilesSkipped []string
}

// initTemplateData is passed to init templates.
type initTemplateData struct {
	ModulePath string
}

// NewInitScaffolder creates a new InitScaffolder with the given configuration.
func NewInitScaffolder(baseDir, modulePath string, force bool, stdout, stderr io.Writer) *InitScaffolder {
	return &InitScaffolder{
		baseDir:    baseDir,
		modulePath: modulePath,
		force:      force,
		stdout:     stdout,
		stderr:     stderr,
	}
}

// ParseModulePath reads a go.mod file and extracts the module path
// from the module directive. It returns an error if the file cannot
// be read or the module directive is missing or malformed.
func ParseModulePath(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	return "", fmt.Errorf("failed to parse module path from go.mod: no module directive found")
}

// initDirs lists the directories the scaffolder creates.
var initDirs = []string{
	filepath.Join("database", "migrations"),
	filepath.Join("database", "seeders"),
	filepath.Join("database", "factories"),
	filepath.Join("cmd", "migrator"),
}

// Scaffold creates the project directory structure and generated files.
// It returns an InitResult summarizing what was created or skipped.
func (s *InitScaffolder) Scaffold() (*InitResult, error) {
	result := &InitResult{}

	// Step 1: Create directories.
	for _, dir := range initDirs {
		dirPath := filepath.Join(s.baseDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dirPath, err)
		}
		result.DirsCreated = append(result.DirsCreated, dir)
	}

	// Step 2: Render and write template files.
	data := initTemplateData{ModulePath: s.modulePath}

	type fileSpec struct {
		tmplName string
		relPath  string
	}

	files := []fileSpec{
		{tmplName: "init_main.go.tmpl", relPath: filepath.Join("cmd", "migrator", "main.go")},
		{tmplName: "init_config.json.tmpl", relPath: "migration.json"},
	}

	for _, f := range files {
		filePath := filepath.Join(s.baseDir, f.relPath)

		// Check if file exists and skip unless --force is set.
		if _, err := os.Stat(filePath); err == nil && !s.force {
			fmt.Fprintf(s.stderr, "skipped: %s already exists\n", f.relPath)
			result.FilesSkipped = append(result.FilesSkipped, f.relPath)
			continue
		}

		// Parse and execute the template.
		tmplPath := "templates/" + f.tmplName
		content, err := templateFS.ReadFile(tmplPath)
		if err != nil {
			return nil, fmt.Errorf("execute template %s: %w", f.tmplName, err)
		}

		tmpl, err := template.New(f.tmplName).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("execute template %s: %w", f.tmplName, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("execute template %s: %w", f.tmplName, err)
		}

		// Write the rendered content to disk.
		if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
			return nil, fmt.Errorf("write file %s: %w", filePath, err)
		}
		result.FilesCreated = append(result.FilesCreated, f.relPath)
	}

	// Step 3: Print summary to stdout.
	if len(result.DirsCreated) > 0 {
		fmt.Fprintln(s.stdout, "Created directories:")
		for _, d := range result.DirsCreated {
			fmt.Fprintf(s.stdout, "  %s/\n", d)
		}
	}
	if len(result.FilesCreated) > 0 {
		if len(result.DirsCreated) > 0 {
			fmt.Fprintln(s.stdout)
		}
		fmt.Fprintln(s.stdout, "Created files:")
		for _, f := range result.FilesCreated {
			fmt.Fprintf(s.stdout, "  %s\n", f)
		}
	}
	if len(result.FilesSkipped) > 0 {
		if len(result.DirsCreated) > 0 || len(result.FilesCreated) > 0 {
			fmt.Fprintln(s.stdout)
		}
		fmt.Fprintln(s.stdout, "Skipped files:")
		for _, f := range result.FilesSkipped {
			fmt.Fprintf(s.stdout, "  %s (already exists)\n", f)
		}
	}

	return result, nil
}
