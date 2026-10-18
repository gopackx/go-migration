package grammars

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
)

// joinCreate adapts CompileCreate's ([]string, error) result to a single
// "; "-joined string so assertions written against the full DDL text keep
// working now that each statement is returned separately.
func joinCreate(stmts []string, err error) (string, error) {
	return strings.Join(stmts, "; "), err
}

// --- CompileCreate returns separate statements ---

func TestPostgres_CompileCreate_IndexesAreSeparateStatements(t *testing.T) {
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("email", 255)
	bp.String("name", 255)
	bp.Index("email")
	bp.Index("name")

	stmts, err := NewPostgresGrammar().CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 3 {
		t.Fatalf("len = %d, want 3: %q", len(stmts), stmts)
	}
	if !strings.HasPrefix(stmts[0], `CREATE TABLE "users"`) {
		t.Errorf("stmts[0] = %q, want CREATE TABLE", stmts[0])
	}
	for i, s := range stmts {
		if strings.Contains(s, ";") {
			t.Errorf("stmts[%d] contains ';': %q", i, s)
		}
	}
	for _, s := range stmts[1:] {
		if !strings.HasPrefix(s, "CREATE INDEX ") {
			t.Errorf("want CREATE INDEX, got %q", s)
		}
	}
}

func TestSQLite_CompileCreate_IndexesAreSeparateStatements(t *testing.T) {
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("email", 255)
	bp.Index("email")

	stmts, err := NewSQLiteGrammar().CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 2 {
		t.Fatalf("len = %d, want 2: %q", len(stmts), stmts)
	}
	if !strings.HasPrefix(stmts[0], `CREATE TABLE "users"`) {
		t.Errorf("stmts[0] = %q", stmts[0])
	}
	if !strings.HasPrefix(stmts[1], "CREATE INDEX ") || strings.Contains(stmts[1], ";") {
		t.Errorf("stmts[1] = %q", stmts[1])
	}
}

func TestMySQL_CompileCreate_SingleStatement(t *testing.T) {
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("email", 255)
	bp.Index("email")

	stmts, err := NewMySQLGrammar().CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want 1: %q", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "KEY `") {
		t.Errorf("expected inline index in %q", stmts[0])
	}
}

// --- CompileListTables ---

func TestCompileListTables(t *testing.T) {
	tests := []struct {
		name    string
		grammar schema.Grammar
		want    []string
	}{
		{"postgres", NewPostgresGrammar(), []string{"pg_tables", "schemaname = 'public'"}},
		{"mysql", NewMySQLGrammar(), []string{"information_schema.tables", "table_schema = DATABASE()", "table_type = 'BASE TABLE'"}},
		{"sqlite", NewSQLiteGrammar(), []string{"sqlite_master", "type='table'", "NOT LIKE 'sqlite_%'"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.grammar.CompileListTables()
			if !strings.HasPrefix(q, "SELECT ") {
				t.Errorf("expected SELECT query, got %q", q)
			}
			for _, w := range tt.want {
				if !strings.Contains(q, w) {
					t.Errorf("%q not in %q", w, q)
				}
			}
		})
	}
}

// --- CompileDropAllTables ---

func TestPostgres_CompileDropAllTables(t *testing.T) {
	got := NewPostgresGrammar().CompileDropAllTables([]string{"users", "posts"})
	want := []string{"DROP SCHEMA public CASCADE", "CREATE SCHEMA public"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMySQL_CompileDropAllTables(t *testing.T) {
	got := NewMySQLGrammar().CompileDropAllTables([]string{"users", "posts"})
	want := []string{
		"SET FOREIGN_KEY_CHECKS = 0",
		"DROP TABLE IF EXISTS `users`",
		"DROP TABLE IF EXISTS `posts`",
		"SET FOREIGN_KEY_CHECKS = 1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMySQL_CompileDropAllTables_Empty(t *testing.T) {
	got := NewMySQLGrammar().CompileDropAllTables(nil)
	want := []string{"SET FOREIGN_KEY_CHECKS = 0", "SET FOREIGN_KEY_CHECKS = 1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSQLite_CompileDropAllTables(t *testing.T) {
	got := NewSQLiteGrammar().CompileDropAllTables([]string{"users", "migrations"})
	want := []string{
		"PRAGMA foreign_keys = OFF",
		`DROP TABLE IF EXISTS "users"`,
		`DROP TABLE IF EXISTS "migrations"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompileDropAllTables_NoJoinedStatements(t *testing.T) {
	for _, g := range []schema.Grammar{NewPostgresGrammar(), NewMySQLGrammar(), NewSQLiteGrammar()} {
		for _, s := range g.CompileDropAllTables([]string{"a", "b"}) {
			if strings.Contains(s, ";") {
				t.Errorf("%T: statement contains ';': %q", g, s)
			}
		}
	}
}

// --- CompileTrackingTable ---

func TestCompileTrackingTable(t *testing.T) {
	tests := []struct {
		name    string
		grammar schema.Grammar
		want    []string
		notWant []string
	}{
		{
			"postgres", NewPostgresGrammar(),
			[]string{`CREATE TABLE IF NOT EXISTS "schema_migrations" (`, "id         SERIAL PRIMARY KEY", "created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"},
			[]string{"AUTO_INCREMENT", "AUTOINCREMENT"},
		},
		{
			"mysql", NewMySQLGrammar(),
			[]string{"CREATE TABLE IF NOT EXISTS `schema_migrations` (", "id         INT AUTO_INCREMENT PRIMARY KEY", "created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"},
			[]string{"SERIAL", "AUTOINCREMENT"},
		},
		{
			"sqlite", NewSQLiteGrammar(),
			[]string{`CREATE TABLE IF NOT EXISTS "schema_migrations" (`, "id         INTEGER PRIMARY KEY AUTOINCREMENT", "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP"},
			[]string{"SERIAL", "AUTO_INCREMENT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.grammar.CompileTrackingTable("schema_migrations")
			common := []string{
				"migration  VARCHAR(255) NOT NULL UNIQUE",
				"batch      INTEGER NOT NULL",
			}
			for _, w := range append(common, tt.want...) {
				if !strings.Contains(q, w) {
					t.Errorf("%q not in %q", w, q)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(q, nw) {
					t.Errorf("%q unexpectedly in %q", nw, q)
				}
			}
		})
	}
}

// --- Placeholder ---

func TestPlaceholder(t *testing.T) {
	pg, my, lite := NewPostgresGrammar(), NewMySQLGrammar(), NewSQLiteGrammar()
	for i := 1; i <= 3; i++ {
		if got, want := pg.Placeholder(i), "$"+string(rune('0'+i)); got != want {
			t.Errorf("postgres Placeholder(%d) = %q, want %q", i, got, want)
		}
		if got := my.Placeholder(i); got != "?" {
			t.Errorf("mysql Placeholder(%d) = %q, want ?", i, got)
		}
		if got := lite.Placeholder(i); got != "?" {
			t.Errorf("sqlite Placeholder(%d) = %q, want ?", i, got)
		}
	}
}

// --- Migration lock queries ---

func TestCompileLock(t *testing.T) {
	cases := []struct {
		name         string
		g            schema.Grammar
		lock, unlock string
	}{
		{"postgres", NewPostgresGrammar(), "SELECT pg_try_advisory_lock(hashtext($1))", "SELECT pg_advisory_unlock(hashtext($1))"},
		{"mysql", NewMySQLGrammar(), "SELECT GET_LOCK(CONCAT(DATABASE(), ':', ?), 0)", "SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':', ?))"},
		{"sqlite", NewSQLiteGrammar(), "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.CompileTryLock("go-migration:migrations"); got != c.lock {
				t.Errorf("CompileTryLock = %q, want %q", got, c.lock)
			}
			if got := c.g.CompileUnlock("go-migration:migrations"); got != c.unlock {
				t.Errorf("CompileUnlock = %q, want %q", got, c.unlock)
			}
			// The key is a bind parameter, never interpolated.
			if strings.Contains(c.g.CompileTryLock("x'y"), "x'y") {
				t.Errorf("CompileTryLock interpolates the key")
			}
		})
	}
}
