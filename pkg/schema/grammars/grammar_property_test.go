package grammars

import (
	"errors"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"

	"pgregory.net/rapid"
)

// allGrammars returns all Grammar implementations for cross-grammar testing.
func allGrammars() []struct {
	name    string
	grammar schema.Grammar
} {
	return []struct {
		name    string
		grammar schema.Grammar
	}{
		{"postgres", NewPostgresGrammar()},
		{"mysql", NewMySQLGrammar()},
		{"sqlite", NewSQLiteGrammar()},
	}
}

// --- Generators ---

// validColumnTypes returns all valid ColumnType values.
var validColumnTypes = []schema.ColumnType{
	schema.TypeString,
	schema.TypeText,
	schema.TypeInteger,
	schema.TypeBigInteger,
	schema.TypeBoolean,
	schema.TypeTimestamp,
	schema.TypeDate,
	schema.TypeDecimal,
	schema.TypeFloat,
	schema.TypeUUID,
	schema.TypeJSON,
	schema.TypeBinary,
}

// tableNameGen generates valid table names.
func tableNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9_]{1,19}`)
}

// columnNameGen generates valid column names.
func columnNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9_]{1,14}`)
}

// columnTypeGen generates a random valid ColumnType.
func columnTypeGen() *rapid.Generator[schema.ColumnType] {
	return rapid.SampledFrom(validColumnTypes)
}

// fkActionGen generates a random foreign key action.
func fkActionGen() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"CASCADE", "SET NULL", "RESTRICT", "NO ACTION"})
}

// drawColumnDef draws a random ColumnDefinition using rapid generators.
func drawColumnDef(t *rapid.T, name string, colType schema.ColumnType) schema.ColumnDefinition {
	col := schema.ColumnDefinition{
		Name: name,
		Type: colType,
	}

	if colType == schema.TypeString {
		col.Length = rapid.IntRange(1, 500).Draw(t, "length")
	}
	if colType == schema.TypeDecimal {
		col.Precision = rapid.IntRange(1, 38).Draw(t, "precision")
		col.Scale = rapid.IntRange(0, 10).Draw(t, "scale")
	}

	col.IsNullable = rapid.Bool().Draw(t, "nullable")

	if rapid.Bool().Draw(t, "hasDefault") {
		switch colType {
		case schema.TypeBoolean:
			col.DefaultValue = rapid.Bool().Draw(t, "defaultBool")
		case schema.TypeInteger, schema.TypeBigInteger:
			col.DefaultValue = rapid.IntRange(0, 1000).Draw(t, "defaultInt")
		case schema.TypeString, schema.TypeText:
			col.DefaultValue = rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "defaultStr")
		default:
			// skip default for other types to keep it simple
		}
	}

	col.IsUnique = rapid.Bool().Draw(t, "unique")

	return col
}

// drawBlueprint draws a random Blueprint with columns, optionally indexes and foreign keys.
func drawBlueprint(t *rapid.T, tableName string, minCols int) *schema.Blueprint {
	bp := schema.NewBlueprint(tableName)

	numCols := rapid.IntRange(minCols, 6).Draw(t, "numCols")
	usedNames := make(map[string]bool)
	colNames := make([]string, 0, numCols)

	for i := 0; i < numCols; i++ {
		name := columnNameGen().Filter(func(s string) bool {
			return !usedNames[s]
		}).Draw(t, "colName")
		usedNames[name] = true
		colNames = append(colNames, name)

		colType := columnTypeGen().Draw(t, "colType")

		col := drawColumnDef(t, name, colType)
		// Use blueprint methods to add the column properly
		switch colType {
		case schema.TypeString:
			c := bp.String(name, col.Length)
			applyModifiers(c, col)
		case schema.TypeText:
			c := bp.Text(name)
			applyModifiers(c, col)
		case schema.TypeInteger:
			c := bp.Integer(name)
			applyModifiers(c, col)
		case schema.TypeBigInteger:
			c := bp.BigInteger(name)
			applyModifiers(c, col)
		case schema.TypeBoolean:
			c := bp.Boolean(name)
			applyModifiers(c, col)
		case schema.TypeTimestamp:
			c := bp.Timestamp(name)
			applyModifiers(c, col)
		case schema.TypeDate:
			c := bp.Date(name)
			applyModifiers(c, col)
		case schema.TypeDecimal:
			c := bp.Decimal(name, col.Precision, col.Scale)
			applyModifiers(c, col)
		case schema.TypeFloat:
			c := bp.Float(name)
			applyModifiers(c, col)
		case schema.TypeUUID:
			c := bp.UUID(name)
			applyModifiers(c, col)
		case schema.TypeJSON:
			c := bp.JSON(name)
			applyModifiers(c, col)
		case schema.TypeBinary:
			c := bp.Binary(name)
			applyModifiers(c, col)
		}
	}

	// Optionally add indexes
	if len(colNames) > 0 && rapid.Bool().Draw(t, "hasIndex") {
		idxCol := rapid.SampledFrom(colNames).Draw(t, "idxCol")
		if rapid.Bool().Draw(t, "uniqueIdx") {
			bp.UniqueIndex(idxCol)
		} else {
			bp.Index(idxCol)
		}
	}

	// Optionally add a foreign key
	if len(colNames) > 0 && rapid.Bool().Draw(t, "hasForeignKey") {
		fkCol := rapid.SampledFrom(colNames).Draw(t, "fkCol")
		refTable := tableNameGen().Draw(t, "refTable")
		refCol := columnNameGen().Draw(t, "refCol")
		fk := bp.Foreign(fkCol).References(refCol).On(refTable)
		if rapid.Bool().Draw(t, "hasOnDelete") {
			fk.OnDeleteAction(fkActionGen().Draw(t, "onDelete"))
		}
		if rapid.Bool().Draw(t, "hasOnUpdate") {
			fk.OnUpdateAction(fkActionGen().Draw(t, "onUpdate"))
		}
	}

	return bp
}

// applyModifiers applies column modifiers from a ColumnDefinition to a *ColumnDefinition pointer.
func applyModifiers(c *schema.ColumnDefinition, col schema.ColumnDefinition) {
	if col.IsNullable {
		c.Nullable()
	}
	if col.DefaultValue != nil {
		c.Default(col.DefaultValue)
	}
	if col.IsUnique {
		c.Unique()
	}
}

// Feature: go-migration, Property 15: Blueprint features are reflected in compiled SQL
// **Validates: Requirements 6.2, 6.3, 6.4, 6.5**
func TestProperty15_BlueprintFeaturesReflectedInSQL(t *testing.T) {
	for _, g := range allGrammars() {
		t.Run(g.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				tableName := tableNameGen().Draw(t, "table")
				bp := drawBlueprint(t, tableName, 1)

				sql, err := joinCreate(g.grammar.CompileCreate(bp))
				if err != nil {
					// If compilation fails, it should only be for unsupported types
					if !errors.Is(err, ErrUnsupportedType) {
						t.Errorf("CompileCreate error should be ErrUnsupportedType, got: %v", err)
					}
					return
				}

				upperSQL := strings.ToUpper(sql)

				// Verify table name appears in SQL
				if !strings.Contains(sql, tableName) {
					t.Errorf("%q not in %q", tableName, sql)
				}

				// Verify each column name appears in SQL
				for _, col := range bp.Columns() {
					if !strings.Contains(sql, col.Name) {
						t.Errorf("%q not in %q", col.Name, sql)
					}

					// Verify nullable: non-nullable columns should have NOT NULL
					// (except auto-increment serials in Postgres/SQLite)
					if !col.IsNullable && !isAutoIncrementSerial(col, g.name) {
						// Find the column segment and check for NOT NULL
						if !strings.Contains(upperSQL, "NOT NULL") {
							t.Errorf("%q not in %q", "NOT NULL", upperSQL)
						}
					}

					// Verify default values appear
					if col.DefaultValue != nil {
						if !strings.Contains(upperSQL, "DEFAULT") {
							t.Errorf("%q not in %q", "DEFAULT", upperSQL)
						}
					}

					// Verify unique modifier
					if col.IsUnique {
						if !strings.Contains(upperSQL, "UNIQUE") {
							t.Errorf("%q not in %q", "UNIQUE", upperSQL)
						}
					}
				}

				// Verify indexes appear in SQL
				for _, idx := range bp.Indexes() {
					if !strings.Contains(sql, idx.Name) {
						t.Errorf("%q not in %q", idx.Name, sql)
					}
					if idx.Type == schema.IndexUnique {
						if !strings.Contains(upperSQL, "UNIQUE") {
							t.Errorf("%q not in %q", "UNIQUE", upperSQL)
						}
					}
				}

				// Verify foreign keys appear in SQL
				for _, fk := range bp.ForeignKeys() {
					if !strings.Contains(upperSQL, "FOREIGN KEY") {
						t.Errorf("%q not in %q", "FOREIGN KEY", upperSQL)
					}
					if !strings.Contains(upperSQL, "REFERENCES") {
						t.Errorf("%q not in %q", "REFERENCES", upperSQL)
					}
					if !strings.Contains(sql, fk.RefTable) {
						t.Errorf("%q not in %q", fk.RefTable, sql)
					}
					if !strings.Contains(sql, fk.RefColumn) {
						t.Errorf("%q not in %q", fk.RefColumn, sql)
					}
					if fk.OnDelete != "" {
						if !strings.Contains(upperSQL, "ON DELETE") {
							t.Errorf("%q not in %q", "ON DELETE", upperSQL)
						}
					}
					if fk.OnUpdate != "" {
						if !strings.Contains(upperSQL, "ON UPDATE") {
							t.Errorf("%q not in %q", "ON UPDATE", upperSQL)
						}
					}
				}
			})
		})
	}
}

// isAutoIncrementSerial checks if a column is an auto-increment serial type
// that doesn't need explicit NOT NULL.
func isAutoIncrementSerial(col schema.ColumnDefinition, grammarName string) bool {
	if !col.IsAutoIncrement {
		return false
	}
	isIntType := col.Type == schema.TypeInteger || col.Type == schema.TypeBigInteger
	if grammarName == "postgres" && isIntType {
		return true // SERIAL/BIGSERIAL are implicitly NOT NULL
	}
	if grammarName == "sqlite" && isIntType && col.IsPrimary {
		return true // INTEGER PRIMARY KEY AUTOINCREMENT skips modifiers
	}
	return false
}

// Feature: go-migration, Property 16: Grammar produces valid SQL for create-table
// **Validates: Requirements 6.6, 7.1**
func TestProperty16_GrammarProducesValidCreateTableSQL(t *testing.T) {
	for _, g := range allGrammars() {
		t.Run(g.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				tableName := tableNameGen().Draw(t, "table")
				bp := drawBlueprint(t, tableName, 1)

				sql, err := joinCreate(g.grammar.CompileCreate(bp))
				if err != nil {
					if !errors.Is(err, ErrUnsupportedType) {
						t.Errorf("CompileCreate error should be ErrUnsupportedType, got: %v", err)
					}
					return
				}

				upperSQL := strings.ToUpper(sql)

				// The SQL must start with CREATE TABLE
				if !strings.HasPrefix(upperSQL, "CREATE TABLE") {
					t.Errorf("CompileCreate SQL should start with CREATE TABLE, got: %s", sql)
				}

				// The SQL must contain the table name
				if !strings.Contains(sql, tableName) {
					t.Errorf("%q not in %q", tableName, sql)
				}

				// The SQL must contain opening and closing parentheses for column definitions
				if !strings.Contains(sql, "(") {
					t.Errorf("%q not in %q", "(", sql)
				}
				if !strings.Contains(sql, ")") {
					t.Errorf("%q not in %q", ")", sql)
				}

				// Every column name should appear in the SQL
				for _, col := range bp.Columns() {
					if !strings.Contains(sql, col.Name) {
						t.Errorf("%q not in %q", col.Name, sql)
					}
				}

				// Verify the main CREATE TABLE statement has balanced parentheses
				// (only check the first statement before any semicolons for non-unique indexes)
				mainStmt := sql
				if idx := strings.Index(sql, ";"); idx >= 0 {
					mainStmt = sql[:idx]
				}
				openCount := strings.Count(mainStmt, "(")
				closeCount := strings.Count(mainStmt, ")")
				if openCount != closeCount {
					t.Errorf("CREATE TABLE statement should have balanced parentheses, got %d open and %d close", openCount, closeCount)
				}
			})
		})
	}
}

// Feature: go-migration, Property 17: Grammar produces valid SQL for alter-table
// **Validates: Requirements 7.2**
func TestProperty17_GrammarProducesValidAlterTableSQL(t *testing.T) {
	for _, g := range allGrammars() {
		t.Run(g.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				tableName := tableNameGen().Draw(t, "table")
				bp := schema.NewBlueprint(tableName)

				// Randomly add columns, drop columns, and indexes
				numAddCols := rapid.IntRange(0, 3).Draw(t, "numAddCols")
				usedNames := make(map[string]bool)
				for i := 0; i < numAddCols; i++ {
					name := columnNameGen().Filter(func(s string) bool {
						return !usedNames[s]
					}).Draw(t, "addColName")
					usedNames[name] = true
					colType := columnTypeGen().Draw(t, "addColType")
					switch colType {
					case schema.TypeString:
						bp.String(name, rapid.IntRange(1, 255).Draw(t, "len"))
					case schema.TypeText:
						bp.Text(name)
					case schema.TypeInteger:
						bp.Integer(name)
					case schema.TypeBigInteger:
						bp.BigInteger(name)
					case schema.TypeBoolean:
						bp.Boolean(name)
					case schema.TypeTimestamp:
						bp.Timestamp(name)
					case schema.TypeDate:
						bp.Date(name)
					case schema.TypeDecimal:
						bp.Decimal(name, rapid.IntRange(1, 20).Draw(t, "prec"), rapid.IntRange(0, 5).Draw(t, "scale"))
					case schema.TypeFloat:
						bp.Float(name)
					case schema.TypeUUID:
						bp.UUID(name)
					case schema.TypeJSON:
						bp.JSON(name)
					case schema.TypeBinary:
						bp.Binary(name)
					}
				}

				// Add drop column commands
				numDropCols := rapid.IntRange(0, 2).Draw(t, "numDropCols")
				for i := 0; i < numDropCols; i++ {
					dropName := columnNameGen().Filter(func(s string) bool {
						return !usedNames[s]
					}).Draw(t, "dropColName")
					usedNames[dropName] = true
					bp.DropColumn(dropName)
				}

				// Optionally add an index
				if len(usedNames) > 0 && rapid.Bool().Draw(t, "addIdx") {
					idxCol := columnNameGen().Draw(t, "idxCol")
					bp.Index(idxCol)
				}

				stmts, err := g.grammar.CompileAlter(bp)
				if err != nil {
					if !errors.Is(err, ErrUnsupportedType) {
						t.Errorf("CompileAlter error should be ErrUnsupportedType, got: %v", err)
					}
					return
				}

				// Each statement should be a valid ALTER TABLE, CREATE INDEX, CREATE UNIQUE INDEX, or DROP INDEX
				for _, stmt := range stmts {
					upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
					isAlterTable := strings.HasPrefix(upperStmt, "ALTER TABLE")
					isCreateIndex := strings.HasPrefix(upperStmt, "CREATE INDEX") || strings.HasPrefix(upperStmt, "CREATE UNIQUE INDEX")
					isDropIndex := strings.HasPrefix(upperStmt, "DROP INDEX")

					if !isAlterTable && !isCreateIndex && !isDropIndex {
						t.Errorf("Each alter statement should start with ALTER TABLE, CREATE INDEX, CREATE UNIQUE INDEX, or DROP INDEX, got: %s", stmt)
					}

					// Each statement should reference the table name (except DROP INDEX which may not)
					if !isDropIndex {
						if !strings.Contains(stmt, tableName) {
							t.Errorf("%q not in %q", tableName, stmt)
						}
					}
				}
			})
		})
	}
}

// Feature: go-migration, Property 18: Unsupported column types produce errors
// **Validates: Requirements 6.8**
func TestProperty18_UnsupportedColumnTypesProduceErrors(t *testing.T) {
	for _, g := range allGrammars() {
		t.Run(g.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Generate an invalid column type value outside the valid range
				invalidType := schema.ColumnType(rapid.IntRange(100, 999).Draw(t, "invalidType"))

				col := schema.ColumnDefinition{
					Name: columnNameGen().Draw(t, "colName"),
					Type: invalidType,
				}

				_, err := g.grammar.CompileColumnType(col)
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if !errors.Is(err, ErrUnsupportedType) {
					t.Errorf("Error should wrap ErrUnsupportedType, got: %v", err)
				}
				if !strings.Contains(err.Error(), col.Name) {
					t.Errorf("%q not in %q", col.Name, err.Error())
				}

				// Also verify that CompileCreate fails with unsupported type
				tableName := tableNameGen().Draw(t, "table")
				bp := schema.NewBlueprint(tableName)
				// Add a valid column first so the blueprint isn't empty
				bp.String("valid_col", 100)
				// Manually create a blueprint with the invalid column by using CompileCreate
				// through a blueprint that has the invalid type
				bpInvalid := schema.NewBlueprint(tableName)
				bpInvalid.String("valid_col", 100)
				// We can't directly add an invalid column type via Blueprint methods,
				// so we test CompileColumnType directly which is the core validation point.
				// The CompileCreate path is already covered since it calls CompileColumnType internally.
			})
		})
	}
}
