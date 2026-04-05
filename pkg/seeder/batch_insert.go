package seeder

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// Dialect represents a database dialect for SQL generation differences.
type Dialect int

const (
	// DialectPostgres uses $1, $2, ... numbered placeholders and double-quoted identifiers.
	DialectPostgres Dialect = iota
	// DialectMySQL uses ? positional placeholders and backtick-quoted identifiers.
	DialectMySQL
	// DialectSQLite uses ? positional placeholders and double-quoted identifiers.
	DialectSQLite
)

// placeholder returns the appropriate placeholder string for the given dialect
// and 1-based parameter index.
func (d Dialect) placeholder(index int) string {
	switch d {
	case DialectPostgres:
		return fmt.Sprintf("$%d", index)
	default:
		return "?"
	}
}

// quoteIdent returns the identifier quoting style for the dialect.
func (d Dialect) quoteIdent(name string) string {
	switch d {
	case DialectMySQL:
		return "`" + name + "`"
	default:
		return `"` + name + `"`
	}
}

// CreateMany inserts records into the specified table using multi-row INSERT
// statements, each containing at most chunkSize rows. Records are provided as
// []map[string]any where keys are column names and values are column values.
//
// Column names are derived from the first record and sorted for deterministic
// SQL generation. All subsequent records must have the same set of keys.
//
// If chunkSize is zero or negative, a default of 500 is used.
// Uses DialectPostgres by default. Use CreateManyWithDialect for other databases.
func CreateMany(db *sql.DB, table string, records []map[string]any, chunkSize int) error {
	return CreateManyWithDialect(db, table, records, chunkSize, DialectPostgres)
}

// CreateManyWithDialect inserts records using the specified database dialect
// for placeholder and identifier quoting styles.
//
// Supported dialects:
//   - DialectPostgres: $1, $2 placeholders, "double-quoted" identifiers
//   - DialectMySQL: ? placeholders, `backtick-quoted` identifiers
//   - DialectSQLite: ? placeholders, "double-quoted" identifiers
func CreateManyWithDialect(db *sql.DB, table string, records []map[string]any, chunkSize int, dialect Dialect) error {
	if len(records) == 0 {
		return fmt.Errorf("no records to insert")
	}

	if chunkSize <= 0 {
		chunkSize = 500
	}

	// Extract and sort column names from the first record.
	columns := make([]string, 0, len(records[0]))
	for col := range records[0] {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	// Validate that all records have the same keys as the first record.
	if err := validateRecordKeys(columns, records); err != nil {
		return err
	}

	// Process records in chunks.
	for start := 0; start < len(records); start += chunkSize {
		end := start + chunkSize
		if end > len(records) {
			end = len(records)
		}

		chunk := records[start:end]
		if err := insertChunk(db, table, columns, chunk, dialect); err != nil {
			return fmt.Errorf("batch [%d:%d] failed: %w", start, end, err)
		}
	}

	return nil
}

// validateRecordKeys checks that every record has exactly the same keys as the
// sorted column list derived from the first record.
func validateRecordKeys(columns []string, records []map[string]any) error {
	expected := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		expected[col] = struct{}{}
	}

	for i := 1; i < len(records); i++ {
		if len(records[i]) != len(expected) {
			return keyMismatchError(i, columns, records[i])
		}
		for key := range records[i] {
			if _, ok := expected[key]; !ok {
				return keyMismatchError(i, columns, records[i])
			}
		}
	}
	return nil
}

// keyMismatchError builds a descriptive error for a record with mismatched keys.
func keyMismatchError(index int, expected []string, record map[string]any) error {
	actual := make([]string, 0, len(record))
	for k := range record {
		actual = append(actual, k)
	}
	sort.Strings(actual)

	var missing, extra []string
	expectedSet := make(map[string]struct{}, len(expected))
	for _, col := range expected {
		expectedSet[col] = struct{}{}
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, col := range actual {
		actualSet[col] = struct{}{}
	}

	for _, col := range expected {
		if _, ok := actualSet[col]; !ok {
			missing = append(missing, col)
		}
	}
	for _, col := range actual {
		if _, ok := expectedSet[col]; !ok {
			extra = append(extra, col)
		}
	}

	parts := []string{fmt.Sprintf("record %d has mismatched keys", index)}
	if len(missing) > 0 {
		parts = append(parts, fmt.Sprintf("missing: [%s]", strings.Join(missing, ", ")))
	}
	if len(extra) > 0 {
		parts = append(parts, fmt.Sprintf("extra: [%s]", strings.Join(extra, ", ")))
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}

// insertChunk builds and executes a single multi-row INSERT statement.
func insertChunk(db *sql.DB, table string, columns []string, records []map[string]any, dialect Dialect) error {
	if len(records) == 0 {
		return nil
	}

	// Quote column names using dialect-appropriate style.
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = dialect.quoteIdent(col)
	}

	// Build placeholder rows and collect values.
	numCols := len(columns)
	values := make([]any, 0, len(records)*numCols)
	rows := make([]string, 0, len(records))
	paramIdx := 1

	for _, rec := range records {
		placeholders := make([]string, numCols)
		for j, col := range columns {
			placeholders[j] = dialect.placeholder(paramIdx)
			paramIdx++
			values = append(values, rec[col])
		}
		rows = append(rows, "("+strings.Join(placeholders, ", ")+")")
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES %s`,
		dialect.quoteIdent(table),
		strings.Join(quotedCols, ", "),
		strings.Join(rows, ", "),
	)

	_, err := db.Exec(query, values...)
	return err
}
