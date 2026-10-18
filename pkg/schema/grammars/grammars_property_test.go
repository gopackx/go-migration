package grammars

import (
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"

	"pgregory.net/rapid"
)

// allowedValueGen generates a non-empty allowed value string suitable for enum values.
// Avoids single quotes to keep SQL quoting straightforward.
func allowedValueGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,19}`)
}

// allowedValuesGen generates a non-empty slice of unique allowed values for enum columns.
func allowedValuesGen() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		count := rapid.IntRange(1, 10).Draw(t, "numValues")
		seen := make(map[string]bool)
		values := make([]string, 0, count)
		for len(values) < count {
			v := allowedValueGen().Filter(func(s string) bool {
				return !seen[s]
			}).Draw(t, "allowedValue")
			seen[v] = true
			values = append(values, v)
		}
		return values
	})
}

// Feature: library-improvements, Property 2: Enum compilation produces grammar-appropriate SQL with all allowed values
// **Validates: Requirements 1.7, 1.8, 1.9**
func TestProperty2_EnumCompilationProducesGrammarAppropriateSQL(t *testing.T) {
	grammars := allGrammars()

	for _, g := range grammars {
		t.Run(g.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				colName := columnNameGen().Draw(t, "colName")
				allowedValues := allowedValuesGen().Draw(t, "allowedValues")

				col := schema.ColumnDefinition{
					Name:          colName,
					Type:          schema.TypeEnum,
					AllowedValues: allowedValues,
				}

				sql, err := g.grammar.CompileColumnType(col)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Every allowed value must appear in the output SQL
				for _, v := range allowedValues {
					if !strings.Contains(sql, v) {
						t.Errorf("%q not in %q", v, sql)
					}
				}

				// Grammar-specific structural checks
				switch g.name {
				case "postgres":
					if !strings.Contains(sql, "VARCHAR(255)") {
						t.Errorf("%q not in %q", "VARCHAR(255)", sql)
					}
					if !strings.Contains(sql, "CHECK") {
						t.Errorf("%q not in %q", "CHECK", sql)
					}
					if !strings.Contains(sql, colName) {
						t.Errorf("%q not in %q", colName, sql)
					}

				case "mysql":
					if !strings.Contains(sql, "ENUM") {
						t.Errorf("%q not in %q", "ENUM", sql)
					}
				}
			})
		})
	}
}

// columnsGen generates a non-empty slice of unique column names for index testing.
func columnsGen() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		count := rapid.IntRange(1, 5).Draw(t, "numColumns")
		seen := make(map[string]bool)
		cols := make([]string, 0, count)
		for len(cols) < count {
			c := columnNameGen().Filter(func(s string) bool {
				return !seen[s]
			}).Draw(t, "col")
			seen[c] = true
			cols = append(cols, c)
		}
		return cols
	})
}

// Feature: library-improvements, Property 4: Fulltext and spatial index compilation produces grammar-appropriate SQL
// **Validates: Requirements 2.3, 2.4, 2.5, 2.7, 2.8**
func TestProperty4_FulltextSpatialIndexCompilationProducesGrammarAppropriateSQL(t *testing.T) {
	pgGrammar := NewPostgresGrammar()
	myGrammar := NewMySQLGrammar()

	t.Run("Fulltext_Postgres_CompileCreate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.ID()
			for _, c := range cols {
				bp.Text(c)
			}
			bp.FulltextIndex(cols...)

			sql, err := joinCreate(pgGrammar.CompileCreate(bp))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(sql, "USING GIN") {
				t.Errorf("%q not in %q", "USING GIN", sql)
			}
			if !strings.Contains(sql, "to_tsvector") {
				t.Errorf("%q not in %q", "to_tsvector", sql)
			}
		})
	})

	t.Run("Fulltext_Postgres_CompileAlter", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.FulltextIndex(cols...)

			stmts, err := pgGrammar.CompileAlter(bp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			joined := strings.Join(stmts, "; ")
			if !strings.Contains(joined, "USING GIN") {
				t.Errorf("%q not in %q", "USING GIN", joined)
			}
			if !strings.Contains(joined, "to_tsvector") {
				t.Errorf("%q not in %q", "to_tsvector", joined)
			}
		})
	})

	t.Run("Fulltext_MySQL_CompileCreate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.ID()
			for _, c := range cols {
				bp.Text(c)
			}
			bp.FulltextIndex(cols...)

			sql, err := joinCreate(myGrammar.CompileCreate(bp))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(sql, "FULLTEXT") {
				t.Errorf("%q not in %q", "FULLTEXT", sql)
			}
		})
	})

	t.Run("Fulltext_MySQL_CompileAlter", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.FulltextIndex(cols...)

			stmts, err := myGrammar.CompileAlter(bp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			joined := strings.Join(stmts, "; ")
			if !strings.Contains(joined, "FULLTEXT") {
				t.Errorf("%q not in %q", "FULLTEXT", joined)
			}
		})
	})

	t.Run("Spatial_Postgres_CompileCreate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.ID()
			for _, c := range cols {
				bp.Text(c)
			}
			bp.SpatialIndex(cols...)

			sql, err := joinCreate(pgGrammar.CompileCreate(bp))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(sql, "USING GIST") {
				t.Errorf("%q not in %q", "USING GIST", sql)
			}
		})
	})

	t.Run("Spatial_Postgres_CompileAlter", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.SpatialIndex(cols...)

			stmts, err := pgGrammar.CompileAlter(bp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			joined := strings.Join(stmts, "; ")
			if !strings.Contains(joined, "USING GIST") {
				t.Errorf("%q not in %q", "USING GIST", joined)
			}
		})
	})

	t.Run("Spatial_MySQL_CompileCreate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.ID()
			for _, c := range cols {
				bp.Text(c)
			}
			bp.SpatialIndex(cols...)

			sql, err := joinCreate(myGrammar.CompileCreate(bp))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(sql, "SPATIAL") {
				t.Errorf("%q not in %q", "SPATIAL", sql)
			}
		})
	})

	t.Run("Spatial_MySQL_CompileAlter", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.SpatialIndex(cols...)

			stmts, err := myGrammar.CompileAlter(bp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			joined := strings.Join(stmts, "; ")
			if !strings.Contains(joined, "SPATIAL") {
				t.Errorf("%q not in %q", "SPATIAL", joined)
			}
		})
	})
}

// Feature: library-improvements, Property 5: SQLite grammar rejects fulltext and spatial indexes
// **Validates: Requirements 2.6, 2.9**
func TestProperty5_SQLiteGrammarRejectsFulltextAndSpatialIndexes(t *testing.T) {
	sqliteGrammar := NewSQLiteGrammar()

	t.Run("Fulltext_CompileCreate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.ID()
			for _, c := range cols {
				bp.Text(c)
			}
			bp.FulltextIndex(cols...)

			_, err := joinCreate(sqliteGrammar.CompileCreate(bp))
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	})

	t.Run("Fulltext_CompileAlter", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.FulltextIndex(cols...)

			_, err := sqliteGrammar.CompileAlter(bp)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	})

	t.Run("Spatial_CompileCreate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.ID()
			for _, c := range cols {
				bp.Text(c)
			}
			bp.SpatialIndex(cols...)

			_, err := joinCreate(sqliteGrammar.CompileCreate(bp))
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	})

	t.Run("Spatial_CompileAlter", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := tableNameGen().Draw(t, "tableName")
			cols := columnsGen().Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			bp.SpatialIndex(cols...)

			_, err := sqliteGrammar.CompileAlter(bp)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	})
}
