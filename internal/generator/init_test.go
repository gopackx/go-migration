package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModulePath(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantPath  string
		wantError bool
		errMsg    string
	}{
		{
			name:     "standard module directive",
			content:  "module github.com/user/myapp\n\ngo 1.21\n",
			wantPath: "github.com/user/myapp",
		},
		{
			name:     "module directive with extra whitespace",
			content:  "  module   github.com/user/myapp  \n\ngo 1.21\n",
			wantPath: "github.com/user/myapp",
		},
		{
			name:     "module directive not on first line",
			content:  "// comment\nmodule github.com/org/project\n\ngo 1.22\n",
			wantPath: "github.com/org/project",
		},
		{
			name:      "no module directive",
			content:   "go 1.21\n\nrequire (\n)\n",
			wantError: true,
			errMsg:    "no module directive found",
		},
		{
			name:      "empty file",
			content:   "",
			wantError: true,
			errMsg:    "no module directive found",
		},
		{
			name:      "malformed module line - no path",
			content:   "module\n",
			wantError: true,
			errMsg:    "no module directive found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			goModPath := filepath.Join(dir, "go.mod")
			if err := os.WriteFile(goModPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test go.mod: %v", err)
			}

			got, err := ParseModulePath(goModPath)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantPath {
				t.Errorf("got %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestParseModulePath_FileNotFound(t *testing.T) {
	_, err := ParseModulePath("/nonexistent/path/go.mod")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "failed to read go.mod") {
		t.Errorf("error %q does not contain expected message", err.Error())
	}
}

func TestNewInitScaffolder(t *testing.T) {
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	s := NewInitScaffolder("/tmp/myproject", "github.com/user/myapp", true, stdout, stderr)

	if s == nil {
		t.Fatal("expected non-nil scaffolder")
	}
	if s.baseDir != "/tmp/myproject" {
		t.Errorf("baseDir = %q, want %q", s.baseDir, "/tmp/myproject")
	}
	if s.modulePath != "github.com/user/myapp" {
		t.Errorf("modulePath = %q, want %q", s.modulePath, "github.com/user/myapp")
	}
	if s.force != true {
		t.Errorf("force = %v, want true", s.force)
	}
	if s.stdout != stdout {
		t.Error("stdout writer not set correctly")
	}
	if s.stderr != stderr {
		t.Error("stderr writer not set correctly")
	}
}

func TestInitResult_ZeroValue(t *testing.T) {
	result := &InitResult{}
	if result.DirsCreated != nil {
		t.Error("expected nil DirsCreated")
	}
	if result.FilesCreated != nil {
		t.Error("expected nil FilesCreated")
	}
	if result.FilesSkipped != nil {
		t.Error("expected nil FilesSkipped")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestScaffold_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)

	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDirs := []string{filepath.Join("database", "migrations"), filepath.Join("database", "seeders"), filepath.Join("database", "factories"), filepath.Join("cmd", "migrator")}
	if len(result.DirsCreated) != len(expectedDirs) {
		t.Fatalf("DirsCreated = %v, want %v", result.DirsCreated, expectedDirs)
	}
	for i, d := range expectedDirs {
		if result.DirsCreated[i] != d {
			t.Errorf("DirsCreated[%d] = %q, want %q", i, result.DirsCreated[i], d)
		}
		info, err := os.Stat(filepath.Join(dir, d))
		if err != nil {
			t.Errorf("directory %q not created: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", d)
		}
	}
}

func TestScaffold_IdempotentDirectoryCreation(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)

	// First run
	_, err := s.Scaffold()
	if err != nil {
		t.Fatalf("first scaffold: unexpected error: %v", err)
	}

	// Second run should not error
	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("second scaffold: unexpected error: %v", err)
	}

	expectedDirs := []string{filepath.Join("database", "migrations"), filepath.Join("database", "seeders"), filepath.Join("database", "factories"), filepath.Join("cmd", "migrator")}
	if len(result.DirsCreated) != len(expectedDirs) {
		t.Fatalf("DirsCreated = %v, want %v", result.DirsCreated, expectedDirs)
	}
}

func TestScaffold_DirectoryPermissions(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping permission test in CI")
	}
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)

	_, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, d := range []string{filepath.Join("database", "migrations"), filepath.Join("database", "seeders"), filepath.Join("database", "factories"), filepath.Join("cmd", "migrator")} {
		info, err := os.Stat(filepath.Join(dir, d))
		if err != nil {
			t.Errorf("stat %q: %v", d, err)
			continue
		}
		// On most systems MkdirAll with 0755 produces 0755 (modulo umask).
		// We just verify the directory is accessible.
		if info.Mode().Perm()&0700 != 0700 {
			t.Errorf("directory %q perm = %o, expected owner rwx", d, info.Mode().Perm())
		}
	}
}

func TestScaffold_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)

	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := []string{
		filepath.Join("cmd", "migrator", "main.go"),
		"migration.json",
	}
	if len(result.FilesCreated) != len(expectedFiles) {
		t.Fatalf("FilesCreated = %v, want %v", result.FilesCreated, expectedFiles)
	}
	for i, f := range expectedFiles {
		if result.FilesCreated[i] != f {
			t.Errorf("FilesCreated[%d] = %q, want %q", i, result.FilesCreated[i], f)
		}
	}

	// Verify main.go content
	mainContent, err := os.ReadFile(filepath.Join(dir, "cmd", "migrator", "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	mainStr := string(mainContent)
	if !strings.Contains(mainStr, `_ "github.com/user/myapp/database/migrations"`) {
		t.Error("main.go missing migrations import")
	}
	if !strings.Contains(mainStr, `_ "github.com/user/myapp/database/seeders"`) {
		t.Error("main.go missing seeders import")
	}
	if !strings.Contains(mainStr, "migrator.Run()") {
		t.Error("main.go missing migrator.Run() call")
	}

	// Verify config content
	cfgContent, err := os.ReadFile(filepath.Join(dir, "migration.json"))
	if err != nil {
		t.Fatalf("failed to read migration.json: %v", err)
	}
	cfgStr := string(cfgContent)
	if !strings.Contains(cfgStr, `"default": "default"`) {
		t.Error("config missing default field")
	}
	if !strings.Contains(cfgStr, `"migration_dir": "database/migrations"`) {
		t.Error("config missing migration_dir field")
	}
	if !strings.Contains(cfgStr, "${DB_HOST}") {
		t.Error("config missing DB_HOST placeholder")
	}
}

func TestScaffold_SkipsExistingFilesWithoutForce(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	// Create directories and pre-existing files
	os.MkdirAll(filepath.Join(dir, "cmd", "migrator"), 0755)
	originalMain := []byte("// original main.go content\n")
	originalConfig := []byte(`{"default":"default"}`)
	os.WriteFile(filepath.Join(dir, "cmd", "migrator", "main.go"), originalMain, 0644)
	os.WriteFile(filepath.Join(dir, "migration.json"), originalConfig, 0644)

	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)
	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both files should be skipped
	if len(result.FilesSkipped) != 2 {
		t.Fatalf("FilesSkipped = %v, want 2 entries", result.FilesSkipped)
	}
	if len(result.FilesCreated) != 0 {
		t.Fatalf("FilesCreated = %v, want empty", result.FilesCreated)
	}

	// Verify original content is preserved
	gotMain, _ := os.ReadFile(filepath.Join(dir, "cmd", "migrator", "main.go"))
	if string(gotMain) != string(originalMain) {
		t.Error("main.go content was modified without --force")
	}
	gotConfig, _ := os.ReadFile(filepath.Join(dir, "migration.json"))
	if string(gotConfig) != string(originalConfig) {
		t.Error("migration.json content was modified without --force")
	}

	// Verify stderr warnings
	stderrStr := stderr.String()
	mainRelPath := filepath.Join("cmd", "migrator", "main.go")
	if !strings.Contains(stderrStr, "skipped: "+mainRelPath+" already exists") {
		t.Errorf("stderr missing skip warning for main.go, got: %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "skipped: migration.json already exists") {
		t.Errorf("stderr missing skip warning for config, got: %q", stderrStr)
	}
}

func TestScaffold_ForceOverwritesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	// Create directories and pre-existing files
	os.MkdirAll(filepath.Join(dir, "cmd", "migrator"), 0755)
	os.WriteFile(filepath.Join(dir, "cmd", "migrator", "main.go"), []byte("// old"), 0644)
	os.WriteFile(filepath.Join(dir, "migration.json"), []byte("{}"), 0644)

	s := NewInitScaffolder(dir, "github.com/user/myapp", true, stdout, stderr)
	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both files should be created (overwritten), none skipped
	if len(result.FilesCreated) != 2 {
		t.Fatalf("FilesCreated = %v, want 2 entries", result.FilesCreated)
	}
	if len(result.FilesSkipped) != 0 {
		t.Fatalf("FilesSkipped = %v, want empty", result.FilesSkipped)
	}

	// Verify content was overwritten with template output
	mainContent, _ := os.ReadFile(filepath.Join(dir, "cmd", "migrator", "main.go"))
	if !strings.Contains(string(mainContent), "migrator.Run()") {
		t.Error("main.go was not overwritten with template content")
	}

	cfgContent, _ := os.ReadFile(filepath.Join(dir, "migration.json"))
	if !strings.Contains(string(cfgContent), `"default": "default"`) {
		t.Error("migration.json was not overwritten with template content")
	}

	// No warnings should be printed to stderr
	if stderr.String() != "" {
		t.Errorf("expected no stderr output with --force, got: %q", stderr.String())
	}
}

func TestScaffold_WarningMessagesToStderr(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	// Create only main.go, not config
	os.MkdirAll(filepath.Join(dir, "cmd", "migrator"), 0755)
	os.WriteFile(filepath.Join(dir, "cmd", "migrator", "main.go"), []byte("// existing"), 0644)

	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)
	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// main.go skipped, config created
	if len(result.FilesSkipped) != 1 {
		t.Fatalf("FilesSkipped = %v, want 1 entry", result.FilesSkipped)
	}
	if len(result.FilesCreated) != 1 {
		t.Fatalf("FilesCreated = %v, want 1 entry", result.FilesCreated)
	}
	if result.FilesCreated[0] != "migration.json" {
		t.Errorf("FilesCreated[0] = %q, want %q", result.FilesCreated[0], "migration.json")
	}

	mainRelPath := filepath.Join("cmd", "migrator", "main.go")
	if result.FilesSkipped[0] != mainRelPath {
		t.Errorf("FilesSkipped[0] = %q, want %q", result.FilesSkipped[0], mainRelPath)
	}

	// Verify warning message format
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "skipped: "+mainRelPath+" already exists") {
		t.Errorf("stderr = %q, want skip warning for main.go", stderrStr)
	}
	// No warning for config since it was created
	if strings.Contains(stderrStr, "migration.json") {
		t.Errorf("stderr should not mention migration.json, got: %q", stderrStr)
	}
}

func TestScaffold_SummaryOutput_AllCreated(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)
	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()

	// Verify created directories are listed
	if !strings.Contains(out, "Created directories:") {
		t.Error("summary missing 'Created directories:' header")
	}
	for _, d := range result.DirsCreated {
		if !strings.Contains(out, d+"/") {
			t.Errorf("summary missing directory %q", d)
		}
	}

	// Verify created files are listed
	if !strings.Contains(out, "Created files:") {
		t.Error("summary missing 'Created files:' header")
	}
	for _, f := range result.FilesCreated {
		if !strings.Contains(out, f) {
			t.Errorf("summary missing file %q", f)
		}
	}

	// No skipped section when nothing was skipped
	if strings.Contains(out, "Skipped files:") {
		t.Error("summary should not contain 'Skipped files:' when nothing was skipped")
	}
}

func TestScaffold_SummaryOutput_WithSkippedFiles(t *testing.T) {
	dir := t.TempDir()
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	// Pre-create both files so they get skipped
	os.MkdirAll(filepath.Join(dir, "cmd", "migrator"), 0755)
	os.WriteFile(filepath.Join(dir, "cmd", "migrator", "main.go"), []byte("// existing"), 0644)
	os.WriteFile(filepath.Join(dir, "migration.json"), []byte("{}"), 0644)

	s := NewInitScaffolder(dir, "github.com/user/myapp", false, stdout, stderr)
	result, err := s.Scaffold()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()

	// Directories are still created
	if !strings.Contains(out, "Created directories:") {
		t.Error("summary missing 'Created directories:' header")
	}

	// No created files section
	if strings.Contains(out, "Created files:") {
		t.Error("summary should not contain 'Created files:' when no files were created")
	}

	// Skipped files section present
	if !strings.Contains(out, "Skipped files:") {
		t.Error("summary missing 'Skipped files:' header")
	}
	for _, f := range result.FilesSkipped {
		if !strings.Contains(out, f+" (already exists)") {
			t.Errorf("summary missing skipped file %q", f)
		}
	}
}
