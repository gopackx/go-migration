package grammars

import (
	"fmt"
	"strings"

	"github.com/gopackx/go-migration/pkg/schema"
)

// MySQLGrammar compiles Blueprint definitions into MySQL-specific SQL.
type MySQLGrammar struct{}

// NewMySQLGrammar creates a new MySQLGrammar.
func NewMySQLGrammar() *MySQLGrammar {
	return &MySQLGrammar{}
}

// CompileCreate generates a CREATE TABLE statement from the given Blueprint.
// MySQL declares every index inline, so exactly one statement is returned.
func (g *MySQLGrammar) CompileCreate(bp *schema.Blueprint) ([]string, error) {
	columns := bp.Columns()
	if len(columns) == 0 {
		return nil, fmt.Errorf("table %q: no columns defined", bp.Table())
	}

	var parts []string

	// Compile columns
	for _, col := range columns {
		colSQL, err := g.compileColumnDef(col)
		if err != nil {
			return nil, err
		}
		parts = append(parts, colSQL)
	}

	// Collect primary key columns
	var pkCols []string
	for _, col := range columns {
		if col.IsPrimary {
			pkCols = append(pkCols, mysqlQuote(col.Name))
		}
	}
	if len(pkCols) > 0 {
		parts = append(parts, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}

	// Compile indexes
	for _, idx := range bp.Indexes() {
		quotedCols := mysqlQuoteSlice(idx.Columns)
		switch idx.Type {
		case schema.IndexUnique:
			parts = append(parts, fmt.Sprintf("UNIQUE KEY %s (%s)", mysqlQuote(idx.Name), strings.Join(quotedCols, ", ")))
		case schema.IndexFulltext:
			parts = append(parts, fmt.Sprintf("FULLTEXT KEY %s (%s)", mysqlQuote(idx.Name), strings.Join(quotedCols, ", ")))
		case schema.IndexSpatial:
			parts = append(parts, fmt.Sprintf("SPATIAL KEY %s (%s)", mysqlQuote(idx.Name), strings.Join(quotedCols, ", ")))
		default:
			parts = append(parts, fmt.Sprintf("KEY %s (%s)", mysqlQuote(idx.Name), strings.Join(quotedCols, ", ")))
		}
	}

	// Compile foreign keys
	for _, fk := range bp.ForeignKeys() {
		fkSQL := g.compileForeignKey(fk)
		parts = append(parts, fkSQL)
	}

	sql := fmt.Sprintf("CREATE TABLE %s (%s)", mysqlQuote(bp.Table()), strings.Join(parts, ", "))
	return []string{sql}, nil
}

// CompileAlter generates ALTER TABLE statements from the given Blueprint.
func (g *MySQLGrammar) CompileAlter(bp *schema.Blueprint) ([]string, error) {
	var stmts []string
	table := mysqlQuote(bp.Table())

	// Add new columns
	for _, col := range bp.Columns() {
		colSQL, err := g.compileColumnDef(col)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, colSQL))
	}

	// Process commands
	for _, cmd := range bp.Commands() {
		switch cmd.Type {
		case schema.CommandDropColumn:
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, mysqlQuote(cmd.Name)))
		case schema.CommandRenameColumn:
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", table, mysqlQuote(cmd.Name), mysqlQuote(cmd.To)))
		case schema.CommandDropIndex:
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", table, mysqlQuote(cmd.Name)))
		case schema.CommandDropForeign:
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", table, mysqlQuote(cmd.Name)))
		}
	}

	// Add new indexes
	for _, idx := range bp.Indexes() {
		quotedCols := mysqlQuoteSlice(idx.Columns)
		switch idx.Type {
		case schema.IndexUnique:
			stmts = append(stmts, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)", mysqlQuote(idx.Name), table, strings.Join(quotedCols, ", ")))
		case schema.IndexFulltext:
			stmts = append(stmts, fmt.Sprintf("CREATE FULLTEXT INDEX %s ON %s (%s)", mysqlQuote(idx.Name), table, strings.Join(quotedCols, ", ")))
		case schema.IndexSpatial:
			stmts = append(stmts, fmt.Sprintf("CREATE SPATIAL INDEX %s ON %s (%s)", mysqlQuote(idx.Name), table, strings.Join(quotedCols, ", ")))
		default:
			stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s (%s)", mysqlQuote(idx.Name), table, strings.Join(quotedCols, ", ")))
		}
	}

	// Add new foreign keys
	for _, fk := range bp.ForeignKeys() {
		fkSQL := g.compileForeignKey(fk)
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD %s", table, fkSQL))
	}

	return stmts, nil
}

// CompileDrop generates a DROP TABLE statement.
func (g *MySQLGrammar) CompileDrop(table string) string {
	return fmt.Sprintf("DROP TABLE %s", mysqlQuote(table))
}

// CompileDropIfExists generates a DROP TABLE IF EXISTS statement.
func (g *MySQLGrammar) CompileDropIfExists(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", mysqlQuote(table))
}

// CompileRename generates a RENAME TABLE statement.
func (g *MySQLGrammar) CompileRename(from, to string) string {
	return fmt.Sprintf("RENAME TABLE %s TO %s", mysqlQuote(from), mysqlQuote(to))
}

// CompileHasTable generates a query to check if a table exists.
func (g *MySQLGrammar) CompileHasTable(table string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '%s'", table)
}

// CompileHasColumn generates a query to check if a column exists in a table.
func (g *MySQLGrammar) CompileHasColumn(table, column string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '%s' AND column_name = '%s'", table, column)
}

// CompileListTables generates a query listing all base tables in the
// current database.
func (g *MySQLGrammar) CompileListTables() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'"
}

// CompileDropAllTables generates statements to drop every given table with
// foreign key checks disabled. FOREIGN_KEY_CHECKS is a session variable, so
// the statements must run in order on the same connection.
func (g *MySQLGrammar) CompileDropAllTables(tables []string) []string {
	stmts := make([]string, 0, len(tables)+2)
	stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 0")
	for _, t := range tables {
		stmts = append(stmts, g.CompileDropIfExists(t))
	}
	stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 1")
	return stmts
}

// CompileTrackingTable generates the CREATE TABLE IF NOT EXISTS statement for
// the migration tracking table.
func (g *MySQLGrammar) CompileTrackingTable(table string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id         INT AUTO_INCREMENT PRIMARY KEY,
		migration  VARCHAR(255) NOT NULL UNIQUE,
		batch      INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`, table)
}

// Placeholder returns the MySQL placeholder ("?").
func (g *MySQLGrammar) Placeholder(index int) string {
	return "?"
}

// CompileTryLock returns a query that tries to take a named lock without
// waiting. GET_LOCK names are server-wide, so the key is prefixed with the
// current database name. The result is 1 (acquired), 0 (held elsewhere) or
// NULL (error); callers treat NULL as not acquired.
func (g *MySQLGrammar) CompileTryLock(key string) string {
	return "SELECT GET_LOCK(CONCAT(DATABASE(), ':', ?), 0)"
}

// CompileUnlock returns a query that releases the named lock taken by
// CompileTryLock.
func (g *MySQLGrammar) CompileUnlock(key string) string {
	return "SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':', ?))"
}

// CompileColumnType returns the MySQL-specific SQL type string for a column.
func (g *MySQLGrammar) CompileColumnType(col schema.ColumnDefinition) (string, error) {
	switch col.Type {
	case schema.TypeString:
		length := col.Length
		if length <= 0 {
			length = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", length), nil
	case schema.TypeText:
		return "TEXT", nil
	case schema.TypeInteger:
		base := "INT"
		if col.IsUnsigned {
			base += " UNSIGNED"
		}
		if col.IsAutoIncrement {
			base += " AUTO_INCREMENT"
		}
		return base, nil
	case schema.TypeBigInteger:
		base := "BIGINT"
		if col.IsUnsigned {
			base += " UNSIGNED"
		}
		if col.IsAutoIncrement {
			base += " AUTO_INCREMENT"
		}
		return base, nil
	case schema.TypeBoolean:
		return "TINYINT(1)", nil
	case schema.TypeTimestamp:
		return "TIMESTAMP", nil
	case schema.TypeDate:
		return "DATE", nil
	case schema.TypeDecimal:
		precision := col.Precision
		scale := col.Scale
		if precision <= 0 {
			precision = 10
		}
		if scale < 0 {
			scale = 0
		}
		return fmt.Sprintf("DECIMAL(%d, %d)", precision, scale), nil
	case schema.TypeFloat:
		return "DOUBLE", nil
	case schema.TypeUUID:
		return "CHAR(36)", nil
	case schema.TypeJSON:
		return "JSON", nil
	case schema.TypeBinary:
		return "BLOB", nil
	case schema.TypeEnum:
		if len(col.AllowedValues) == 0 {
			return "", fmt.Errorf("column %q: enum column requires at least one allowed value", col.Name)
		}
		quoted := make([]string, len(col.AllowedValues))
		for i, v := range col.AllowedValues {
			quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		}
		return fmt.Sprintf("ENUM(%s)", strings.Join(quoted, ",")), nil
	case schema.TypeChar:
		length := col.Length
		if length <= 0 {
			length = 255
		}
		return fmt.Sprintf("CHAR(%d)", length), nil
	case schema.TypeLongText:
		return "LONGTEXT", nil
	case schema.TypeMediumText:
		return "MEDIUMTEXT", nil
	case schema.TypeTinyInt:
		return "TINYINT", nil
	case schema.TypeSmallInt:
		return "SMALLINT", nil
	default:
		return "", fmt.Errorf("column %q: type %q: %w", col.Name, col.Type.String(), ErrUnsupportedType)
	}
}

// compileColumnDef compiles a single column definition into SQL.
func (g *MySQLGrammar) compileColumnDef(col schema.ColumnDefinition) (string, error) {
	colType, err := g.CompileColumnType(col)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(mysqlQuote(col.Name))
	sb.WriteString(" ")
	sb.WriteString(colType)

	if !col.IsNullable {
		sb.WriteString(" NOT NULL")
	}

	if col.DefaultValue != nil {
		sb.WriteString(fmt.Sprintf(" DEFAULT %s", formatDefault(col.DefaultValue)))
	}

	if col.IsUnique {
		sb.WriteString(" UNIQUE")
	}

	return sb.String(), nil
}

// compileForeignKey compiles a foreign key constraint into SQL.
func (g *MySQLGrammar) compileForeignKey(fk schema.ForeignKeyDefinition) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		mysqlQuote(fk.Name), mysqlQuote(fk.Column), mysqlQuote(fk.RefTable), mysqlQuote(fk.RefColumn)))

	if fk.OnDelete != "" {
		sb.WriteString(fmt.Sprintf(" ON DELETE %s", fk.OnDelete))
	}
	if fk.OnUpdate != "" {
		sb.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate))
	}

	return sb.String()
}

// mysqlQuote wraps an identifier in backticks for MySQL.
func mysqlQuote(name string) string {
	return fmt.Sprintf("`%s`", name)
}

// mysqlQuoteSlice quotes each element in a string slice with backticks.
func mysqlQuoteSlice(names []string) []string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = mysqlQuote(n)
	}
	return quoted
}
