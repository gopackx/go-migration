package grammars

import (
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
)

// A schema-qualified tracking table name is quoted per part in the DDL.
func TestCompileTrackingTable_Qualified(t *testing.T) {
	cases := []struct {
		name string
		g    schema.Grammar
		in   string
		want string
	}{
		{"postgres", NewPostgresGrammar(), "app.migrations", `CREATE TABLE IF NOT EXISTS "app"."migrations" (`},
		{"postgres plain", NewPostgresGrammar(), "migrations", `CREATE TABLE IF NOT EXISTS "migrations" (`},
		{"postgres quote in part", NewPostgresGrammar(), `a"pp.migrations`, `CREATE TABLE IF NOT EXISTS "a""pp"."migrations" (`},
		{"mysql", NewMySQLGrammar(), "app.migrations", "CREATE TABLE IF NOT EXISTS `app`.`migrations` ("},
		{"mysql plain", NewMySQLGrammar(), "migrations", "CREATE TABLE IF NOT EXISTS `migrations` ("},
		{"sqlite", NewSQLiteGrammar(), "app.migrations", `CREATE TABLE IF NOT EXISTS "app"."migrations" (`},
		{"sqlite plain", NewSQLiteGrammar(), "migrations", `CREATE TABLE IF NOT EXISTS "migrations" (`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.CompileTrackingTable(c.in); !strings.HasPrefix(got, c.want) {
				t.Errorf("CompileTrackingTable(%q) = %s, want prefix %s", c.in, got, c.want)
			}
		})
	}
}

func TestCompileHasTableIn(t *testing.T) {
	cases := []struct {
		name string
		g    schema.Grammar
		want string
	}{
		{"postgres", NewPostgresGrammar(),
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'app' AND table_name = 'migrations'"},
		{"mysql", NewMySQLGrammar(),
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'app' AND table_name = 'migrations'"},
		{"sqlite", NewSQLiteGrammar(),
			`SELECT COUNT(*) FROM "app".sqlite_master WHERE type='table' AND name='migrations'`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.CompileHasTableIn("app", "migrations"); got != c.want {
				t.Errorf("CompileHasTableIn =\n  %s\nwant\n  %s", got, c.want)
			}
		})
	}
}

func TestCompileHasTableIn_Escapes(t *testing.T) {
	cases := []struct {
		name string
		g    schema.Grammar
		want string
	}{
		{"postgres", NewPostgresGrammar(), "table_schema = 'o''app' AND table_name = 'mig''s'"},
		{"mysql", NewMySQLGrammar(), "table_schema = 'o''app' AND table_name = 'mig''s'"},
		{"sqlite", NewSQLiteGrammar(), `FROM "o'app".sqlite_master WHERE type='table' AND name='mig''s'`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.CompileHasTableIn("o'app", "mig's"); !strings.Contains(got, c.want) {
				t.Errorf("CompileHasTableIn = %s, want it to contain %s", got, c.want)
			}
		})
	}
	if got := NewSQLiteGrammar().CompileHasTableIn(`a"b`, "t"); !strings.Contains(got, `FROM "a""b".sqlite_master`) {
		t.Errorf("sqlite schema not escaped: %s", got)
	}
}
