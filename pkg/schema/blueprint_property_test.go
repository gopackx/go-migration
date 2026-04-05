package schema_test

import (
	"reflect"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"

	"pgregory.net/rapid"
)

// Feature: library-improvements, Property 1: New column type blueprint methods preserve type and attributes
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5, 1.6**
//
// For any new column type (Enum, Char, LongText, MediumText, TinyInt, SmallInt) and any valid
// arguments (name string, length int, allowed values), calling the corresponding Blueprint method
// SHALL add a ColumnDefinition with the correct ColumnType, and for Enum the AllowedValues SHALL
// match, and for Char the Length SHALL match.
func TestProperty1_NewColumnTypeBlueprintMethodsPreserveTypeAndAttributes(t *testing.T) {
	// Generator for valid column names (non-empty, alphanumeric with underscores)
	colNameGen := rapid.StringMatching(`[a-z][a-z0-9_]{0,19}`)

	t.Run("Enum preserves type and allowed values", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			bp := schema.NewBlueprint("test_table")
			name := colNameGen.Draw(t, "name")
			numValues := rapid.IntRange(1, 10).Draw(t, "numValues")
			allowed := make([]string, numValues)
			for i := range allowed {
				allowed[i] = rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,9}`).Draw(t, "allowedValue")
			}

			col := bp.Enum(name, allowed)

			if col == nil {
				t.Fatalf("expected non-nil")
			}
			if name != col.Name {
				t.Errorf("got %v, want %v", col.Name, name)
			}
			if schema.TypeEnum != col.Type {
				t.Errorf("got %v, want %v", col.Type, schema.TypeEnum)
			}
			if !reflect.DeepEqual(allowed, col.AllowedValues) {
				t.Errorf("got %v, want %v", col.AllowedValues, allowed)
			}

			// Verify it was added to the blueprint
			cols := bp.Columns()
			if len(cols) != 1 {
				t.Fatalf("len = %d, want %d", len(cols), 1)
			}
			if schema.TypeEnum != cols[0].Type {
				t.Errorf("got %v, want %v", cols[0].Type, schema.TypeEnum)
			}
			if !reflect.DeepEqual(allowed, cols[0].AllowedValues) {
				t.Errorf("got %v, want %v", cols[0].AllowedValues, allowed)
			}
		})
	})

	t.Run("Char preserves type and length", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			bp := schema.NewBlueprint("test_table")
			name := colNameGen.Draw(t, "name")
			length := rapid.IntRange(1, 255).Draw(t, "length")

			col := bp.Char(name, length)

			if col == nil {
				t.Fatalf("expected non-nil")
			}
			if name != col.Name {
				t.Errorf("got %v, want %v", col.Name, name)
			}
			if schema.TypeChar != col.Type {
				t.Errorf("got %v, want %v", col.Type, schema.TypeChar)
			}
			if length != col.Length {
				t.Errorf("got %v, want %v", col.Length, length)
			}

			cols := bp.Columns()
			if len(cols) != 1 {
				t.Fatalf("len = %d, want %d", len(cols), 1)
			}
			if schema.TypeChar != cols[0].Type {
				t.Errorf("got %v, want %v", cols[0].Type, schema.TypeChar)
			}
			if length != cols[0].Length {
				t.Errorf("got %v, want %v", cols[0].Length, length)
			}
		})
	})

	t.Run("LongText preserves type", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			bp := schema.NewBlueprint("test_table")
			name := colNameGen.Draw(t, "name")

			col := bp.LongText(name)

			if col == nil {
				t.Fatalf("expected non-nil")
			}
			if name != col.Name {
				t.Errorf("got %v, want %v", col.Name, name)
			}
			if schema.TypeLongText != col.Type {
				t.Errorf("got %v, want %v", col.Type, schema.TypeLongText)
			}

			cols := bp.Columns()
			if len(cols) != 1 {
				t.Fatalf("len = %d, want %d", len(cols), 1)
			}
			if schema.TypeLongText != cols[0].Type {
				t.Errorf("got %v, want %v", cols[0].Type, schema.TypeLongText)
			}
		})
	})

	t.Run("MediumText preserves type", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			bp := schema.NewBlueprint("test_table")
			name := colNameGen.Draw(t, "name")

			col := bp.MediumText(name)

			if col == nil {
				t.Fatalf("expected non-nil")
			}
			if name != col.Name {
				t.Errorf("got %v, want %v", col.Name, name)
			}
			if schema.TypeMediumText != col.Type {
				t.Errorf("got %v, want %v", col.Type, schema.TypeMediumText)
			}

			cols := bp.Columns()
			if len(cols) != 1 {
				t.Fatalf("len = %d, want %d", len(cols), 1)
			}
			if schema.TypeMediumText != cols[0].Type {
				t.Errorf("got %v, want %v", cols[0].Type, schema.TypeMediumText)
			}
		})
	})

	t.Run("TinyInt preserves type", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			bp := schema.NewBlueprint("test_table")
			name := colNameGen.Draw(t, "name")

			col := bp.TinyInt(name)

			if col == nil {
				t.Fatalf("expected non-nil")
			}
			if name != col.Name {
				t.Errorf("got %v, want %v", col.Name, name)
			}
			if schema.TypeTinyInt != col.Type {
				t.Errorf("got %v, want %v", col.Type, schema.TypeTinyInt)
			}

			cols := bp.Columns()
			if len(cols) != 1 {
				t.Fatalf("len = %d, want %d", len(cols), 1)
			}
			if schema.TypeTinyInt != cols[0].Type {
				t.Errorf("got %v, want %v", cols[0].Type, schema.TypeTinyInt)
			}
		})
	})

	t.Run("SmallInt preserves type", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			bp := schema.NewBlueprint("test_table")
			name := colNameGen.Draw(t, "name")

			col := bp.SmallInt(name)

			if col == nil {
				t.Fatalf("expected non-nil")
			}
			if name != col.Name {
				t.Errorf("got %v, want %v", col.Name, name)
			}
			if schema.TypeSmallInt != col.Type {
				t.Errorf("got %v, want %v", col.Type, schema.TypeSmallInt)
			}

			cols := bp.Columns()
			if len(cols) != 1 {
				t.Fatalf("len = %d, want %d", len(cols), 1)
			}
			if schema.TypeSmallInt != cols[0].Type {
				t.Errorf("got %v, want %v", cols[0].Type, schema.TypeSmallInt)
			}
		})
	})
}

// Feature: library-improvements, Property 3: New index type blueprint methods set correct IndexType
// **Validates: Requirements 2.1, 2.2**
//
// For any set of column names, calling FulltextIndex(columns...) SHALL produce an IndexDefinition
// with Type == IndexFulltext, and calling SpatialIndex(columns...) SHALL produce an IndexDefinition
// with Type == IndexSpatial. The columns in the definition SHALL match the input columns.
func TestProperty3_NewIndexTypeBlueprintMethodsSetCorrectIndexType(t *testing.T) {
	// Generator for valid column names
	colNameGen := rapid.StringMatching(`[a-z][a-z0-9_]{0,19}`)

	// Generator for a non-empty slice of unique column names
	colsGen := rapid.Custom(func(t *rapid.T) []string {
		n := rapid.IntRange(1, 5).Draw(t, "numCols")
		seen := make(map[string]bool)
		cols := make([]string, 0, n)
		for len(cols) < n {
			c := colNameGen.Draw(t, "col")
			if !seen[c] {
				seen[c] = true
				cols = append(cols, c)
			}
		}
		return cols
	})

	t.Run("FulltextIndex sets IndexFulltext and preserves columns", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := colNameGen.Draw(t, "table")
			columns := colsGen.Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			idx := bp.FulltextIndex(columns...)

			if idx == nil {
				t.Fatalf("expected non-nil")
			}
			if schema.IndexFulltext != idx.Type {
				t.Errorf("got %v, want %v", idx.Type, schema.IndexFulltext)
			}
			if !reflect.DeepEqual(columns, idx.Columns) {
				t.Errorf("got %v, want %v", idx.Columns, columns)
			}

			// Verify it was added to the blueprint
			indexes := bp.Indexes()
			if len(indexes) != 1 {
				t.Fatalf("len = %d, want %d", len(indexes), 1)
			}
			if schema.IndexFulltext != indexes[0].Type {
				t.Errorf("got %v, want %v", indexes[0].Type, schema.IndexFulltext)
			}
			if !reflect.DeepEqual(columns, indexes[0].Columns) {
				t.Errorf("got %v, want %v", indexes[0].Columns, columns)
			}
		})
	})

	t.Run("SpatialIndex sets IndexSpatial and preserves columns", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tableName := colNameGen.Draw(t, "table")
			columns := colsGen.Draw(t, "columns")

			bp := schema.NewBlueprint(tableName)
			idx := bp.SpatialIndex(columns...)

			if idx == nil {
				t.Fatalf("expected non-nil")
			}
			if schema.IndexSpatial != idx.Type {
				t.Errorf("got %v, want %v", idx.Type, schema.IndexSpatial)
			}
			if !reflect.DeepEqual(columns, idx.Columns) {
				t.Errorf("got %v, want %v", idx.Columns, columns)
			}

			// Verify it was added to the blueprint
			indexes := bp.Indexes()
			if len(indexes) != 1 {
				t.Fatalf("len = %d, want %d", len(indexes), 1)
			}
			if schema.IndexSpatial != indexes[0].Type {
				t.Errorf("got %v, want %v", indexes[0].Type, schema.IndexSpatial)
			}
			if !reflect.DeepEqual(columns, indexes[0].Columns) {
				t.Errorf("got %v, want %v", indexes[0].Columns, columns)
			}
		})
	})
}
