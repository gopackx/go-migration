package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedTimeGenerator returns a generator with a fixed timestamp and deterministic
// random number for reproducible filenames.
func fixedTimeGenerator(t *testing.T) (*Generator, string) {
	t.Helper()
	dir := t.TempDir()
	g := NewGenerator(dir)
	g.nowFunc = func() time.Time {
		return time.Date(2024, 7, 15, 10, 30, 45, 0, time.UTC)
	}
	g.randFunc = func(n int) int { return 1234 }
	return g, dir
}

func TestMigration_BasicFile(t *testing.T) {
	g, dir := fixedTimeGenerator(t)

	path, err := g.Migration("create_users", MigrationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Join(dir, "2024_07_15_103045_1234_create_users.go") != path {
		t.Errorf("got %v, want %v", path, filepath.Join(dir, "2024_07_15_103045_1234_create_users.go"))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := string(content)
	if !strings.Contains(src, "type CreateUsers struct{}") {
		t.Errorf("%q not in %q", "type CreateUsers struct{}", src)
	}
	if !strings.Contains(src, "func (m *CreateUsers) Up(s *schema.Builder) error") {
		t.Errorf("%q not in %q", "func (m *CreateUsers) Up(s *schema.Builder) error", src)
	}
	if !strings.Contains(src, "func (m *CreateUsers) Down(s *schema.Builder) error") {
		t.Errorf("%q not in %q", "func (m *CreateUsers) Down(s *schema.Builder) error", src)
	}
	if !strings.Contains(src, "package migrations") {
		t.Errorf("%q not in %q", "package migrations", src)
	}
}

func TestMigration_CreateFlag(t *testing.T) {
	g, _ := fixedTimeGenerator(t)

	path, err := g.Migration("create_orders", MigrationOptions{CreateTable: "orders"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := string(content)
	if !strings.Contains(src, "type CreateOrders struct{}") {
		t.Errorf("%q not in %q", "type CreateOrders struct{}", src)
	}
	if !strings.Contains(src, `s.Create("orders"`) {
		t.Errorf("%q not in %q", `s.Create("orders"`, src)
	}
	if !strings.Contains(src, `s.Drop("orders"`) {
		t.Errorf("%q not in %q", `s.Drop("orders"`, src)
	}
	if !strings.Contains(src, "bp.ID()") {
		t.Errorf("%q not in %q", "bp.ID()", src)
	}
	if !strings.Contains(src, "bp.Timestamps()") {
		t.Errorf("%q not in %q", "bp.Timestamps()", src)
	}
}

func TestMigration_TableFlag(t *testing.T) {
	g, _ := fixedTimeGenerator(t)

	path, err := g.Migration("add_email_to_users", MigrationOptions{AlterTable: "users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := string(content)
	if !strings.Contains(src, "type AddEmailToUsers struct{}") {
		t.Errorf("%q not in %q", "type AddEmailToUsers struct{}", src)
	}
	if !strings.Contains(src, `s.Alter("users"`) {
		t.Errorf("%q not in %q", `s.Alter("users"`, src)
	}
}

func TestSeeder_BasicFile(t *testing.T) {
	g, dir := fixedTimeGenerator(t)

	path, err := g.Seeder("user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Join(dir, "user_seeder.go") != path {
		t.Errorf("got %v, want %v", path, filepath.Join(dir, "user_seeder.go"))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := string(content)
	if !strings.Contains(src, "type User struct{}") {
		t.Errorf("%q not in %q", "type User struct{}", src)
	}
	if !strings.Contains(src, "func (s *User) Run(db *sql.DB) error") {
		t.Errorf("%q not in %q", "func (s *User) Run(db *sql.DB) error", src)
	}
	if !strings.Contains(src, "package seeders") {
		t.Errorf("%q not in %q", "package seeders", src)
	}
	// The generated seeder must auto-register itself so db:seed discovers it.
	if !strings.Contains(src, "func init()") {
		t.Errorf("%q not in %q", "func init()", src)
	}
	if !strings.Contains(src, `seeder.AutoRegister("User", &User{})`) {
		t.Errorf("%q not in %q", `seeder.AutoRegister("User", &User{})`, src)
	}
	if !strings.Contains(src, `"github.com/gopackx/go-migration/pkg/seeder"`) {
		t.Errorf("%q not in %q", `"github.com/gopackx/go-migration/pkg/seeder"`, src)
	}
}

// TestPackageNameDerivedFromDir verifies that the generated package
// declaration follows the output directory's base name instead of a
// hardcoded value, so files generated into a custom folder compile.
func TestPackageNameDerivedFromDir(t *testing.T) {
	base := t.TempDir()

	cases := []struct {
		dirName     string
		wantPackage string
	}{
		{"migrations", "package migrations"},
		{"seeders", "package seeders"},
		{"schema", "package schema"},
		{"db-migrations", "package dbmigrations"},
	}

	for _, tc := range cases {
		t.Run(tc.dirName, func(t *testing.T) {
			outDir := filepath.Join(base, tc.dirName)
			g := NewGenerator(outDir)
			g.nowFunc = func() time.Time { return time.Date(2024, 7, 15, 10, 30, 45, 0, time.UTC) }
			g.randFunc = func(n int) int { return 1234 }

			mPath, err := g.Migration("create_things", MigrationOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			mSrc := readFile(t, mPath)
			if !strings.Contains(mSrc, tc.wantPackage) {
				t.Errorf("migration: %q not in %q", tc.wantPackage, mSrc)
			}

			sPath, err := g.Seeder("thing")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sSrc := readFile(t, sPath)
			if !strings.Contains(sSrc, tc.wantPackage) {
				t.Errorf("seeder: %q not in %q", tc.wantPackage, sSrc)
			}
		})
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return string(content)
}

func TestMigration_FilenameFormat(t *testing.T) {
	g, _ := fixedTimeGenerator(t)

	path, err := g.Migration("create_products", MigrationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	filename := filepath.Base(path)
	if !strings.HasSuffix(filename, ".go") {
		t.Errorf("expected true")
	}
	if "2024_07_15_103045_1234_create_products.go" != filename {
		t.Errorf("got %v, want %v", filename, "2024_07_15_103045_1234_create_products.go")
	}
}

func TestSeeder_FilenameFormat(t *testing.T) {
	g, _ := fixedTimeGenerator(t)

	path, err := g.Seeder("product")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	filename := filepath.Base(path)
	if "product_seeder.go" != filename {
		t.Errorf("got %v, want %v", filename, "product_seeder.go")
	}
}

func TestMigration_StructAndMethods(t *testing.T) {
	g, _ := fixedTimeGenerator(t)

	path, err := g.Migration("add_indexes", MigrationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := string(content)
	if !strings.Contains(src, "type AddIndexes struct{}") {
		t.Errorf("%q not in %q", "type AddIndexes struct{}", src)
	}
	if !strings.Contains(src, "func (m *AddIndexes) Up(s *schema.Builder) error") {
		t.Errorf("%q not in %q", "func (m *AddIndexes) Up(s *schema.Builder) error", src)
	}
	if !strings.Contains(src, "func (m *AddIndexes) Down(s *schema.Builder) error") {
		t.Errorf("%q not in %q", "func (m *AddIndexes) Down(s *schema.Builder) error", src)
	}
	if !strings.Contains(src, `"github.com/gopackx/go-migration/pkg/schema"`) {
		t.Errorf("%q not in %q", `"github.com/gopackx/go-migration/pkg/schema"`, src)
	}
}

func TestToStructName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"create_users", "CreateUsers"},
		{"add_email_to_users", "AddEmailToUsers"},
		{"user", "User"},
		{"create_order_items", "CreateOrderItems"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if tt.expected != toStructName(tt.input) {
				t.Errorf("got %v, want %v", toStructName(tt.input), tt.expected)
			}
		})
	}
}

// Descriptions must be lower snake_case; anything else would produce a
// migration name that migrator.AutoRegister rejects with a panic at init().
func TestGenerator_RejectsInvalidDescriptions(t *testing.T) {
	invalid := []string{
		"CreateUsers",
		"create-users",
		"create users",
		"1_create_users",
		"_create_users",
		"",
		"create_Users",
		"create.users",
		"../evil",
	}

	for _, desc := range invalid {
		t.Run(desc, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "out")
			g := NewGenerator(dir)

			calls := map[string]func() (string, error){
				"migration": func() (string, error) { return g.Migration(desc, MigrationOptions{}) },
				"seeder":    func() (string, error) { return g.Seeder(desc) },
				"factory":   func() (string, error) { return g.Factory(desc) },
			}
			for kind, call := range calls {
				path, err := call()
				if err == nil {
					t.Errorf("%s(%q): expected error, got path %q", kind, desc, path)
					continue
				}
				if !errors.Is(err, ErrInvalidDescription) {
					t.Errorf("%s(%q): error %v does not wrap ErrInvalidDescription", kind, desc, err)
				}
				if !strings.Contains(err.Error(), "snake_case") {
					t.Errorf("%s(%q): error should explain the expected format: %v", kind, desc, err)
				}
			}
			// Nothing may be written, not even the output directory.
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Errorf("output dir should not be created for invalid description %q", desc)
			}
		})
	}
}

func TestGenerator_AcceptsSnakeCaseDescriptions(t *testing.T) {
	for _, desc := range []string{"create_users", "add_2fa_to_users", "x", "create_users_table_v2"} {
		g, _ := fixedTimeGenerator(t)
		if _, err := g.Migration(desc, MigrationOptions{}); err != nil {
			t.Errorf("Migration(%q): unexpected error: %v", desc, err)
		}
		if _, err := g.Seeder(desc); err != nil {
			t.Errorf("Seeder(%q): unexpected error: %v", desc, err)
		}
		if _, err := g.Factory(desc); err != nil {
			t.Errorf("Factory(%q): unexpected error: %v", desc, err)
		}
	}
}
