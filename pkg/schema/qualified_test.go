package schema_test

import (
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
)

func TestQuoteQualified(t *testing.T) {
	pg, my := grammars.NewPostgresGrammar(), grammars.NewMySQLGrammar()
	cases := []struct {
		name string
		g    schema.Grammar
		in   string
		want string
	}{
		{"pg plain", pg, "migrations", `"migrations"`},
		{"pg qualified", pg, "app.migrations", `"app"."migrations"`},
		{"pg three parts", pg, "db.app.migrations", `"db"."app"."migrations"`},
		{"pg empty parts dropped", pg, ".app..migrations.", `"app"."migrations"`},
		{"pg quote in part", pg, `a"pp.mig"rations`, `"a""pp"."mig""rations"`},
		{"pg only dots", pg, "..", `".."`},
		{"mysql plain", my, "migrations", "`migrations`"},
		{"mysql qualified", my, "app.migrations", "`app`.`migrations`"},
		{"mysql backtick in part", my, "a`pp.migrations", "`a``pp`.`migrations`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := schema.QuoteQualified(c.g, c.in); got != c.want {
				t.Errorf("QuoteQualified(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
	// A plain name is quoted exactly as QuoteIdentifier would.
	if schema.QuoteQualified(pg, `x"y`) != pg.QuoteIdentifier(`x"y`) {
		t.Error("plain name not quoted like QuoteIdentifier")
	}
}

func TestSplitQualified(t *testing.T) {
	cases := []struct{ in, schema, table string }{
		{"migrations", "", "migrations"},
		{"app.migrations", "app", "migrations"},
		{"db.app.migrations", "app", "migrations"},
		{".migrations", "", "migrations"},
		{"app.migrations.", "app", "migrations"},
		{"", "", ""},
	}
	for _, c := range cases {
		s, tbl := schema.SplitQualified(c.in)
		if s != c.schema || tbl != c.table {
			t.Errorf("SplitQualified(%q) = (%q, %q), want (%q, %q)", c.in, s, tbl, c.schema, c.table)
		}
	}
}
