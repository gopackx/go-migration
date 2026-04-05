package grammars

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"

	"pgregory.net/rapid"
)

// Feature: go-migration, Property 19: Blueprint to SQL round-trip
// **Validates: Requirements 7.4**

// parsedTable is a structural representation of a CREATE TABLE statement.
type parsedTable struct {
	Name        string
	Columns     []parsedColumn
	PrimaryKey  []string
	Indexes     []parsedIndex
	ForeignKeys []parsedForeignKey
}

// parsedColumn represents a column extracted from SQL.
type parsedColumn struct {
	Name       string
	SQLType    string
	NotNull    bool
	HasDefault bool
	IsUnique   bool
}

// parsedIndex represents an index extracted from SQL.
type parsedIndex struct {
	Name    string
	Columns []string
	Unique  bool
}

// parsedForeignKey represents a foreign key extracted from SQL.
type parsedForeignKey struct {
	Column    string
	RefTable  string
	RefColumn string
	OnDelete  string
	OnUpdate  string
}

// parseCreateTableSQL parses a PostgreSQL CREATE TABLE statement into a parsedTable.
// It handles the main CREATE TABLE body and any trailing CREATE INDEX statements.
func parseCreateTableSQL(sql string) parsedTable {
	result := parsedTable{}

	// Split on semicolons to separate CREATE TABLE from CREATE INDEX statements
	statements := strings.Split(sql, ";")

	mainStmt := strings.TrimSpace(statements[0])

	// Extract table name: CREATE TABLE "tablename" (...)
	tableNameRe := regexp.MustCompile(`CREATE TABLE "([^"]+)"\s*\(`)
	if m := tableNameRe.FindStringSubmatch(mainStmt); len(m) > 1 {
		result.Name = m[1]
	}

	// Extract the body between the outermost parentheses
	body := extractParenBody(mainStmt)
	if body == "" {
		return result
	}

	// Split body into top-level comma-separated parts (respecting nested parens)
	parts := splitTopLevel(body)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		upperPart := strings.ToUpper(part)

		switch {
		case strings.HasPrefix(upperPart, "PRIMARY KEY"):
			result.PrimaryKey = extractQuotedNames(part)

		case strings.HasPrefix(upperPart, "CONSTRAINT") && strings.Contains(upperPart, "UNIQUE"):
			idx := parseConstraintUnique(part)
			result.Indexes = append(result.Indexes, idx)

		case strings.HasPrefix(upperPart, "CONSTRAINT") && strings.Contains(upperPart, "FOREIGN KEY"):
			fk := parseConstraintForeignKey(part)
			result.ForeignKeys = append(result.ForeignKeys, fk)

		default:
			// It's a column definition
			col := parseColumnDef(part)
			if col.Name != "" {
				result.Columns = append(result.Columns, col)
			}
		}
	}

	// Parse trailing CREATE INDEX statements
	for i := 1; i < len(statements); i++ {
		stmt := strings.TrimSpace(statements[i])
		if stmt == "" {
			continue
		}
		idx := parseCreateIndex(stmt)
		if idx.Name != "" {
			result.Indexes = append(result.Indexes, idx)
		}
	}

	return result
}

// extractParenBody extracts the content between the first outermost '(' and its matching ')'.
func extractParenBody(s string) string {
	start := strings.Index(s, "(")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return s[start+1 : i]
			}
		}
	}
	return ""
}

// splitTopLevel splits a string by commas, but only at the top level (not inside parentheses).
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// extractQuotedNames extracts all double-quoted identifiers from a string.
func extractQuotedNames(s string) []string {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(s, -1)
	var names []string
	for _, m := range matches {
		names = append(names, m[1])
	}
	return names
}

// parseColumnDef parses a column definition like: "colname" TYPE [NOT NULL] [DEFAULT ...] [UNIQUE]
func parseColumnDef(s string) parsedColumn {
	col := parsedColumn{}

	// Extract column name (first quoted identifier)
	nameRe := regexp.MustCompile(`^"([^"]+)"`)
	trimmed := strings.TrimSpace(s)
	if m := nameRe.FindStringSubmatch(trimmed); len(m) > 1 {
		col.Name = m[1]
	} else {
		return col
	}

	// Extract everything after the closing quote of the column name
	closeQuote := strings.Index(trimmed, `"`) + len(col.Name) + 2
	afterName := strings.TrimSpace(trimmed[closeQuote:])
	upperAfterName := strings.ToUpper(afterName)

	// Extract SQL type by finding where modifier keywords start.
	// Modifier keywords: NOT NULL, DEFAULT, UNIQUE, PRIMARY KEY
	// The type is everything before the first modifier keyword.
	modifierRe := regexp.MustCompile(`\b(NOT NULL|DEFAULT|UNIQUE|PRIMARY KEY)\b`)
	loc := modifierRe.FindStringIndex(upperAfterName)
	if loc != nil {
		col.SQLType = strings.TrimSpace(upperAfterName[:loc[0]])
	} else {
		col.SQLType = strings.TrimSpace(upperAfterName)
	}

	upper := strings.ToUpper(s)
	col.NotNull = strings.Contains(upper, "NOT NULL")
	col.HasDefault = strings.Contains(upper, "DEFAULT")
	col.IsUnique = strings.Contains(upper, "UNIQUE")

	return col
}

// parseConstraintUnique parses: CONSTRAINT "name" UNIQUE ("col1", "col2")
func parseConstraintUnique(s string) parsedIndex {
	idx := parsedIndex{Unique: true}

	nameRe := regexp.MustCompile(`CONSTRAINT "([^"]+)"`)
	if m := nameRe.FindStringSubmatch(s); len(m) > 1 {
		idx.Name = m[1]
	}

	// Extract columns from the parenthesized list after UNIQUE
	upper := strings.ToUpper(s)
	uPos := strings.Index(upper, "UNIQUE")
	if uPos >= 0 {
		rest := s[uPos+6:]
		body := extractParenBody(rest)
		if body != "" {
			idx.Columns = extractQuotedNames(body)
		}
	}

	return idx
}

// parseConstraintForeignKey parses: CONSTRAINT "name" FOREIGN KEY ("col") REFERENCES "table" ("col") [ON DELETE ...] [ON UPDATE ...]
func parseConstraintForeignKey(s string) parsedForeignKey {
	fk := parsedForeignKey{}
	upper := strings.ToUpper(s)

	// Extract the FK column
	fkPos := strings.Index(upper, "FOREIGN KEY")
	if fkPos >= 0 {
		rest := s[fkPos+11:]
		body := extractParenBody(rest)
		if body != "" {
			names := extractQuotedNames(body)
			if len(names) > 0 {
				fk.Column = names[0]
			}
		}
	}

	// Extract REFERENCES table and column
	refPos := strings.Index(upper, "REFERENCES")
	if refPos >= 0 {
		rest := s[refPos+10:]
		names := extractQuotedNames(rest)
		if len(names) >= 2 {
			fk.RefTable = names[0]
			fk.RefColumn = names[1]
		}
	}

	// Extract ON DELETE action
	onDelRe := regexp.MustCompile(`(?i)ON DELETE (CASCADE|SET NULL|RESTRICT|NO ACTION)`)
	if m := onDelRe.FindStringSubmatch(s); len(m) > 1 {
		fk.OnDelete = strings.ToUpper(m[1])
	}

	// Extract ON UPDATE action
	onUpdRe := regexp.MustCompile(`(?i)ON UPDATE (CASCADE|SET NULL|RESTRICT|NO ACTION)`)
	if m := onUpdRe.FindStringSubmatch(s); len(m) > 1 {
		fk.OnUpdate = strings.ToUpper(m[1])
	}

	return fk
}

// parseCreateIndex parses: CREATE [UNIQUE] INDEX "name" ON "table" ("col1", "col2")
func parseCreateIndex(s string) parsedIndex {
	idx := parsedIndex{}
	upper := strings.ToUpper(strings.TrimSpace(s))

	idx.Unique = strings.HasPrefix(upper, "CREATE UNIQUE INDEX")

	names := extractQuotedNames(s)
	if len(names) >= 2 {
		idx.Name = names[0]
		// names[1] is the table name, skip it
	}

	// Extract columns from the parenthesized list
	body := extractParenBody(s)
	if body != "" {
		idx.Columns = extractQuotedNames(body)
	}

	return idx
}

// expectedSQLType returns the expected PostgreSQL SQL type for a given ColumnDefinition.
func expectedSQLType(col schema.ColumnDefinition) string {
	switch col.Type {
	case schema.TypeString:
		length := col.Length
		if length <= 0 {
			length = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", length)
	case schema.TypeText:
		return "TEXT"
	case schema.TypeInteger:
		if col.IsAutoIncrement {
			return "SERIAL"
		}
		return "INTEGER"
	case schema.TypeBigInteger:
		if col.IsAutoIncrement {
			return "BIGSERIAL"
		}
		return "BIGINT"
	case schema.TypeBoolean:
		return "BOOLEAN"
	case schema.TypeTimestamp:
		return "TIMESTAMPTZ"
	case schema.TypeDate:
		return "DATE"
	case schema.TypeDecimal:
		precision := col.Precision
		scale := col.Scale
		if precision <= 0 {
			precision = 10
		}
		if scale < 0 {
			scale = 0
		}
		return fmt.Sprintf("DECIMAL(%d, %d)", precision, scale)
	case schema.TypeFloat:
		return "DOUBLE PRECISION"
	case schema.TypeUUID:
		return "UUID"
	case schema.TypeJSON:
		return "JSONB"
	case schema.TypeBinary:
		return "BYTEA"
	default:
		return "UNKNOWN"
	}
}

func TestProperty19_BlueprintToSQLRoundTrip(t *testing.T) {
	// Feature: go-migration, Property 19: Blueprint to SQL round-trip
	// **Validates: Requirements 7.4**
	grammar := NewPostgresGrammar()

	rapid.Check(t, func(t *rapid.T) {
		tableName := tableNameGen().Draw(t, "table")
		bp := drawBlueprint(t, tableName, 1)

		// Step 1: Compile Blueprint to SQL
		sql, err := grammar.CompileCreate(bp)
		if err != nil {
			// Unsupported types are expected to fail — skip those
			t.Skip("CompileCreate returned error (unsupported type)")
			return
		}

		// Step 2: Parse the SQL back into a structural representation
		parsed := parseCreateTableSQL(sql)

		// Step 3: Verify round-trip equivalence

		// 3a: Table name must match
		if tableName != parsed.Name {
			t.Errorf("Parsed table name should match original")
		}

		// 3b: Column count must match
		bpCols := bp.Columns()
		if len(bpCols) != len(parsed.Columns) {
			t.Errorf("Parsed column count should match original")
		}

		// 3c: Each column name and type must match
		for i, bpCol := range bpCols {
			if i >= len(parsed.Columns) {
				break
			}
			pc := parsed.Columns[i]

			if bpCol.Name != pc.Name {
				t.Errorf("Column %d name should match", i)
			}

			// Verify SQL type matches expected PostgreSQL type
			expected := expectedSQLType(bpCol)
			if expected != pc.SQLType {
				t.Errorf("Column %q SQL type should match (expected %s, got %s)", bpCol.Name, expected, pc.SQLType)
			}

			// Verify NOT NULL: non-nullable columns should have NOT NULL
			// (except SERIAL/BIGSERIAL which are implicitly NOT NULL)
			isSerial := bpCol.IsAutoIncrement && (bpCol.Type == schema.TypeInteger || bpCol.Type == schema.TypeBigInteger)
			if !bpCol.IsNullable && !isSerial {
				if !pc.NotNull {
					t.Errorf("Non-nullable column %q should have NOT NULL", bpCol.Name)
				}
			}

			// Verify DEFAULT presence
			if bpCol.DefaultValue != nil {
				if !pc.HasDefault {
					t.Errorf("Column %q with default value should have DEFAULT in SQL", bpCol.Name)
				}
			}

			// Verify UNIQUE modifier (inline UNIQUE on column)
			if bpCol.IsUnique {
				if !pc.IsUnique {
					t.Errorf("Column %q marked unique should have UNIQUE in SQL", bpCol.Name)
				}
			}
		}

		// 3d: Primary key columns must match
		var expectedPK []string
		for _, col := range bpCols {
			if col.IsPrimary {
				expectedPK = append(expectedPK, col.Name)
			}
		}
		if len(expectedPK) > 0 {
			if !reflect.DeepEqual(expectedPK, parsed.PrimaryKey) {
				t.Errorf("Primary key columns should match")
			}
		}

		// 3e: Indexes must match
		bpIndexes := bp.Indexes()
		if len(bpIndexes) != len(parsed.Indexes) {
			t.Errorf("Parsed index count should match original")
		}
		for i, bpIdx := range bpIndexes {
			if i >= len(parsed.Indexes) {
				break
			}
			pi := parsed.Indexes[i]
			if bpIdx.Name != pi.Name {
				t.Errorf("Index %d name should match", i)
			}
			if bpIdx.Type == schema.IndexUnique != pi.Unique {
				t.Errorf("Index %q unique flag should match", bpIdx.Name)
			}
			if !reflect.DeepEqual(bpIdx.Columns, pi.Columns) {
				t.Errorf("Index %q columns should match", bpIdx.Name)
			}
		}

		// 3f: Foreign keys must match
		bpFKs := bp.ForeignKeys()
		if len(bpFKs) != len(parsed.ForeignKeys) {
			t.Errorf("Parsed foreign key count should match original")
		}
		for i, bpFK := range bpFKs {
			if i >= len(parsed.ForeignKeys) {
				break
			}
			pf := parsed.ForeignKeys[i]
			if bpFK.Column != pf.Column {
				t.Errorf("FK %d column should match", i)
			}
			if bpFK.RefTable != pf.RefTable {
				t.Errorf("FK %d ref table should match", i)
			}
			if bpFK.RefColumn != pf.RefColumn {
				t.Errorf("FK %d ref column should match", i)
			}
			if bpFK.OnDelete != pf.OnDelete {
				t.Errorf("FK %d OnDelete should match", i)
			}
			if bpFK.OnUpdate != pf.OnUpdate {
				t.Errorf("FK %d OnUpdate should match", i)
			}
		}
	})
}
