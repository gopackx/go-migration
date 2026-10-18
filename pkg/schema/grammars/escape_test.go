package grammars

import (
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
)

// Identifiers are quoted by doubling the dialect's delimiter, and names used
// as values in catalog lookups are single-quoted with embedded quotes doubled,
// so that names containing quote characters can neither break nor inject SQL.

func TestQuoteIdentifier_EscapesDelimiter(t *testing.T) {
	cases := []struct {
		name string
		g    schema.Grammar
		in   string
		want string
	}{
		{"postgres plain", NewPostgresGrammar(), "users", `"users"`},
		{"postgres double quote", NewPostgresGrammar(), `we"ird`, `"we""ird"`},
		{"postgres only quotes", NewPostgresGrammar(), `""`, `""""""`},
		{"postgres backtick is literal", NewPostgresGrammar(), "a`b", "\"a`b\""},
		{"postgres dot is part of the name", NewPostgresGrammar(), "public.users", `"public.users"`},
		{"sqlite plain", NewSQLiteGrammar(), "users", `"users"`},
		{"sqlite double quote", NewSQLiteGrammar(), `we"ird`, `"we""ird"`},
		{"mysql plain", NewMySQLGrammar(), "users", "`users`"},
		{"mysql backtick", NewMySQLGrammar(), "we`ird", "`we``ird`"},
		{"mysql only backticks", NewMySQLGrammar(), "``", "``````"},
		{"mysql double quote is literal", NewMySQLGrammar(), `a"b`, "`a\"b`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.QuoteIdentifier(c.in); got != c.want {
				t.Errorf("QuoteIdentifier(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

// Every statement built from a blueprint quotes the table, column and index
// names with the escaped delimiter.
func TestCompile_TableNameContainingDelimiter(t *testing.T) {
	cases := []struct {
		name  string
		g     schema.Grammar
		table string
		col   string
		// quoted forms expected in the SQL
		qTable, qCol string
	}{
		{"postgres", NewPostgresGrammar(), `odd"table`, `odd"col`, `"odd""table"`, `"odd""col"`},
		{"sqlite", NewSQLiteGrammar(), `odd"table`, `odd"col`, `"odd""table"`, `"odd""col"`},
		{"mysql", NewMySQLGrammar(), "odd`table", "odd`col", "`odd``table`", "`odd``col`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bp := schema.NewBlueprint(c.table)
			bp.ID()
			bp.String(c.col, 50).Unique()
			bp.Index(c.col)
			stmts, err := c.g.CompileCreate(bp)
			if err != nil {
				t.Fatal(err)
			}
			sql := strings.Join(stmts, "; ")
			if !strings.Contains(sql, "CREATE TABLE "+c.qTable+" (") {
				t.Errorf("table not escaped in %s", sql)
			}
			if !strings.Contains(sql, c.qCol) {
				t.Errorf("column not escaped in %s", sql)
			}

			alter := schema.NewBlueprint(c.table)
			alter.String(c.col+"2", 10)
			alter.DropColumn(c.col)
			alterStmts, err := c.g.CompileAlter(alter)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range alterStmts {
				if !strings.HasPrefix(s, "ALTER TABLE "+c.qTable+" ") {
					t.Errorf("ALTER does not start with the escaped table name: %s", s)
				}
			}

			for _, s := range []string{c.g.CompileDrop(c.table), c.g.CompileDropIfExists(c.table)} {
				if !strings.HasSuffix(s, " "+c.qTable) {
					t.Errorf("DROP not escaped: %s", s)
				}
			}
			if s := c.g.CompileRename(c.table, c.table+"2"); !strings.Contains(s, c.qTable) {
				t.Errorf("RENAME not escaped: %s", s)
			}
			if s := c.g.CompileTrackingTable(c.table); !strings.Contains(s, "CREATE TABLE IF NOT EXISTS "+c.qTable+" (") {
				t.Errorf("tracking table not escaped: %s", s)
			}
		})
	}
}

func TestCompileHasTableAndColumn_EscapeSingleQuotes(t *testing.T) {
	cases := []struct {
		name      string
		g         schema.Grammar
		hasTable  string
		hasColumn string
	}{
		{
			"postgres", NewPostgresGrammar(),
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'o''brien'",
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'o''brien' AND column_name = 'it''s'",
		},
		{
			"mysql", NewMySQLGrammar(),
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'o''brien'",
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'o''brien' AND column_name = 'it''s'",
		},
		{
			"sqlite", NewSQLiteGrammar(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='o''brien'",
			"SELECT COUNT(*) FROM pragma_table_info('o''brien') WHERE name='it''s'",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.CompileHasTable("o'brien"); got != c.hasTable {
				t.Errorf("CompileHasTable =\n  %s\nwant\n  %s", got, c.hasTable)
			}
			if got := c.g.CompileHasColumn("o'brien", "it's"); got != c.hasColumn {
				t.Errorf("CompileHasColumn =\n  %s\nwant\n  %s", got, c.hasColumn)
			}
		})
	}
}

// An injection attempt stays inside the string literal.
func TestCompileHasTable_InjectionStaysInLiteral(t *testing.T) {
	evil := "x' OR '1'='1"
	for name, g := range map[string]schema.Grammar{
		"postgres": NewPostgresGrammar(), "mysql": NewMySQLGrammar(), "sqlite": NewSQLiteGrammar(),
	} {
		q := g.CompileHasTable(evil)
		if !strings.Contains(q, "'x'' OR ''1''=''1'") {
			t.Errorf("%s: name not escaped as one literal: %s", name, q)
		}
	}
}

// MySQL treats a backslash in a string literal as an escape character (unless
// NO_BACKSLASH_ESCAPES is set), so it is doubled too; PostgreSQL and SQLite
// (standard_conforming_strings) take it literally.
func TestCompileHasTable_Backslash(t *testing.T) {
	if got := NewMySQLGrammar().CompileHasTable(`a\'b`); !strings.HasSuffix(got, `table_name = 'a\\''b'`) {
		t.Errorf("mysql: %s", got)
	}
	if got := NewPostgresGrammar().CompileHasTable(`a\b`); !strings.HasSuffix(got, `table_name = 'a\b'`) {
		t.Errorf("postgres: %s", got)
	}
	if got := NewSQLiteGrammar().CompileHasTable(`a\b`); !strings.HasSuffix(got, `name='a\b'`) {
		t.Errorf("sqlite: %s", got)
	}
}

// Enum columns on PostgreSQL and SQLite are emulated with a CHECK constraint
// that names the column; the name must be escaped there as well.
func TestEnumCheck_EscapesColumnName(t *testing.T) {
	for name, g := range map[string]schema.Grammar{"postgres": NewPostgresGrammar(), "sqlite": NewSQLiteGrammar()} {
		col := schema.ColumnDefinition{Name: `st"atus`, Type: schema.TypeEnum, AllowedValues: []string{"a", "b'c"}}
		got, err := g.CompileColumnType(col)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, `CHECK ("st""atus" IN ('a','b''c'))`) {
			t.Errorf("%s: %s", name, got)
		}
	}
}
