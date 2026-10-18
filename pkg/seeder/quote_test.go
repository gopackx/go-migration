package seeder

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDialect_QuoteIdent_EscapesDelimiter(t *testing.T) {
	cases := []struct {
		name string
		d    Dialect
		in   string
		want string
	}{
		{"postgres plain", DialectPostgres, "users", `"users"`},
		{"postgres double quote", DialectPostgres, `we"ird`, `"we""ird"`},
		{"postgres backtick is literal", DialectPostgres, "a`b", "\"a`b\""},
		{"sqlite double quote", DialectSQLite, `we"ird`, `"we""ird"`},
		{"mysql plain", DialectMySQL, "users", "`users`"},
		{"mysql backtick", DialectMySQL, "we`ird", "`we``ird`"},
		{"mysql double quote is literal", DialectMySQL, `a"b`, "`a\"b`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.d.quoteIdent(c.in); got != c.want {
				t.Errorf("quoteIdent(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

func TestCreateManyWithDialect_EscapesTableAndColumns(t *testing.T) {
	cases := []struct {
		name    string
		d       Dialect
		table   string
		col     string
		wantSQL string
	}{
		{"postgres", DialectPostgres, `odd"table`, `odd"col`, `INSERT INTO "odd""table" ("odd""col") VALUES ($1), ($2)`},
		{"sqlite", DialectSQLite, `odd"table`, `odd"col`, `INSERT INTO "odd""table" ("odd""col") VALUES (?), (?)`},
		{"mysql", DialectMySQL, "odd`table", "odd`col", "INSERT INTO `odd``table` (`odd``col`) VALUES (?), (?)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectExec("^"+regexp.QuoteMeta(c.wantSQL)+"$").
				WithArgs("a", "b").
				WillReturnResult(sqlmock.NewResult(0, 2))

			records := []map[string]any{{c.col: "a"}, {c.col: "b"}}
			if err := CreateManyWithDialect(db, c.table, records, 10, c.d); err != nil {
				t.Fatalf("CreateManyWithDialect: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRunner_Truncate_EscapesTableName(t *testing.T) {
	cases := []struct {
		name  string
		d     Dialect
		table string
		want  string
	}{
		{"postgres", DialectPostgres, `odd"table`, `DELETE FROM "odd""table"`},
		{"sqlite", DialectSQLite, `odd"table`, `DELETE FROM "odd""table"`},
		{"mysql", DialectMySQL, "odd`table", "DELETE FROM `odd``table`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectExec("^" + regexp.QuoteMeta(c.want) + "$").WillReturnResult(sqlmock.NewResult(0, 1))
			if err := NewRunner(NewRegistry(), db, nil, WithDialect(c.d)).Truncate(c.table); err != nil {
				t.Fatalf("Truncate: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}
