package grammars

import (
	"fmt"
	"strings"

	"github.com/gopackx/go-migration/pkg/schema"
)

// ErrUnsupportedType is returned when a column type is not supported by the grammar.
var ErrUnsupportedType = fmt.Errorf("unsupported column type")

// PostgresGrammar compiles Blueprint definitions into PostgreSQL-specific SQL.
type PostgresGrammar struct{}

// NewPostgresGrammar creates a new PostgresGrammar.
func NewPostgresGrammar() *PostgresGrammar {
	return &PostgresGrammar{}
}

// CompileCreate generates a CREATE TABLE statement from the given Blueprint,
// followed by one CREATE INDEX statement per non-unique index.
func (g *PostgresGrammar) CompileCreate(bp *schema.Blueprint) ([]string, error) {
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
			pkCols = append(pkCols, quote(col.Name))
		}
	}
	if len(pkCols) > 0 {
		parts = append(parts, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}

	// Compile indexes as separate constraints for unique indexes
	for _, idx := range bp.Indexes() {
		if idx.Type == schema.IndexUnique {
			quotedCols := quoteSlice(idx.Columns)
			parts = append(parts, fmt.Sprintf("CONSTRAINT %s UNIQUE (%s)", quote(idx.Name), strings.Join(quotedCols, ", ")))
		}
	}

	// Compile foreign keys
	for _, fk := range bp.ForeignKeys() {
		fkSQL := g.compileForeignKey(fk)
		parts = append(parts, fkSQL)
	}

	stmts := []string{fmt.Sprintf("CREATE TABLE %s (%s)", quote(bp.Table()), strings.Join(parts, ", "))}

	// Non-unique indexes cannot be declared inline in PostgreSQL; emit one
	// CREATE INDEX statement each, to be executed separately.
	for _, idx := range bp.Indexes() {
		switch idx.Type {
		case schema.IndexRegular:
			quotedCols := quoteSlice(idx.Columns)
			stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s (%s)", quote(idx.Name), quote(bp.Table()), strings.Join(quotedCols, ", ")))
		case schema.IndexFulltext:
			for _, col := range idx.Columns {
				stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s USING GIN (to_tsvector('english', %s))", quote(idx.Name), quote(bp.Table()), quote(col)))
			}
		case schema.IndexSpatial:
			quotedCols := quoteSlice(idx.Columns)
			stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s USING GIST (%s)", quote(idx.Name), quote(bp.Table()), strings.Join(quotedCols, ", ")))
		}
	}

	return stmts, nil
}

// CompileAlter generates ALTER TABLE statements from the given Blueprint.
func (g *PostgresGrammar) CompileAlter(bp *schema.Blueprint) ([]string, error) {
	var stmts []string
	table := quote(bp.Table())

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
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, quote(cmd.Name)))
		case schema.CommandRenameColumn:
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", table, quote(cmd.Name), quote(cmd.To)))
		case schema.CommandDropIndex:
			stmts = append(stmts, fmt.Sprintf("DROP INDEX %s", quote(cmd.Name)))
		case schema.CommandDropForeign:
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", table, quote(cmd.Name)))
		}
	}

	// Add new indexes
	for _, idx := range bp.Indexes() {
		quotedCols := quoteSlice(idx.Columns)
		switch idx.Type {
		case schema.IndexUnique:
			stmts = append(stmts, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)", quote(idx.Name), table, strings.Join(quotedCols, ", ")))
		case schema.IndexFulltext:
			for _, col := range idx.Columns {
				stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s USING GIN (to_tsvector('english', %s))", quote(idx.Name), table, quote(col)))
			}
		case schema.IndexSpatial:
			stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s USING GIST (%s)", quote(idx.Name), table, strings.Join(quotedCols, ", ")))
		default:
			stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s (%s)", quote(idx.Name), table, strings.Join(quotedCols, ", ")))
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
func (g *PostgresGrammar) CompileDrop(table string) string {
	return fmt.Sprintf("DROP TABLE %s", quote(table))
}

// CompileDropIfExists generates a DROP TABLE IF EXISTS statement.
func (g *PostgresGrammar) CompileDropIfExists(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table))
}

// CompileRename generates an ALTER TABLE ... RENAME TO statement.
func (g *PostgresGrammar) CompileRename(from, to string) string {
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quote(from), quote(to))
}

// CompileHasTable generates a query to check if a table exists.
func (g *PostgresGrammar) CompileHasTable(table string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '%s'", table)
}

// CompileHasColumn generates a query to check if a column exists in a table.
func (g *PostgresGrammar) CompileHasColumn(table, column string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = '%s' AND column_name = '%s'", table, column)
}

// CompileListTables generates a query listing all tables in the public schema.
func (g *PostgresGrammar) CompileListTables() string {
	return "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
}

// CompileDropAllTables generates statements to drop all tables by dropping and
// recreating the public schema. The table list is not needed and is ignored.
func (g *PostgresGrammar) CompileDropAllTables(tables []string) []string {
	return []string{
		"DROP SCHEMA public CASCADE",
		"CREATE SCHEMA public",
	}
}

// CompileTrackingTable generates the CREATE TABLE IF NOT EXISTS statement for
// the migration tracking table.
func (g *PostgresGrammar) CompileTrackingTable(table string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id         SERIAL PRIMARY KEY,
		migration  VARCHAR(255) NOT NULL UNIQUE,
		batch      INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`, table)
}

// Placeholder returns the PostgreSQL positional placeholder ($1, $2, ...).
func (g *PostgresGrammar) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// CompileTryLock returns a query that tries to take a session-level advisory
// lock on hashtext(key). Advisory locks are scoped to the current database.
func (g *PostgresGrammar) CompileTryLock(key string) string {
	return "SELECT pg_try_advisory_lock(hashtext($1))"
}

// CompileUnlock returns a query that releases the advisory lock taken by
// CompileTryLock.
func (g *PostgresGrammar) CompileUnlock(key string) string {
	return "SELECT pg_advisory_unlock(hashtext($1))"
}

// CompileColumnType returns the PostgreSQL-specific SQL type string for a column.
func (g *PostgresGrammar) CompileColumnType(col schema.ColumnDefinition) (string, error) {
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
		if col.IsAutoIncrement {
			return "SERIAL", nil
		}
		return "INTEGER", nil
	case schema.TypeBigInteger:
		if col.IsAutoIncrement {
			return "BIGSERIAL", nil
		}
		return "BIGINT", nil
	case schema.TypeBoolean:
		return "BOOLEAN", nil
	case schema.TypeTimestamp:
		return "TIMESTAMPTZ", nil
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
		return "DOUBLE PRECISION", nil
	case schema.TypeUUID:
		return "UUID", nil
	case schema.TypeJSON:
		return "JSONB", nil
	case schema.TypeBinary:
		return "BYTEA", nil
	case schema.TypeEnum:
		if len(col.AllowedValues) == 0 {
			return "", fmt.Errorf("column %q: enum column requires at least one allowed value", col.Name)
		}
		quoted := make([]string, len(col.AllowedValues))
		for i, v := range col.AllowedValues {
			quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		}
		return fmt.Sprintf(`VARCHAR(255) CHECK ("%s" IN (%s))`, col.Name, strings.Join(quoted, ",")), nil
	case schema.TypeChar:
		length := col.Length
		if length <= 0 {
			length = 255
		}
		return fmt.Sprintf("CHAR(%d)", length), nil
	case schema.TypeLongText:
		return "TEXT", nil
	case schema.TypeMediumText:
		return "TEXT", nil
	case schema.TypeTinyInt:
		return "SMALLINT", nil
	case schema.TypeSmallInt:
		return "SMALLINT", nil
	default:
		return "", fmt.Errorf("column %q: type %q: %w", col.Name, col.Type.String(), ErrUnsupportedType)
	}
}

// compileColumnDef compiles a single column definition into SQL.
func (g *PostgresGrammar) compileColumnDef(col schema.ColumnDefinition) (string, error) {
	colType, err := g.CompileColumnType(col)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(quote(col.Name))
	sb.WriteString(" ")
	sb.WriteString(colType)

	// For SERIAL/BIGSERIAL, skip NOT NULL (they are implicitly NOT NULL)
	isSerial := col.IsAutoIncrement && (col.Type == schema.TypeInteger || col.Type == schema.TypeBigInteger)

	if !col.IsNullable && !isSerial {
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
func (g *PostgresGrammar) compileForeignKey(fk schema.ForeignKeyDefinition) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		quote(fk.Name), quote(fk.Column), quote(fk.RefTable), quote(fk.RefColumn)))

	if fk.OnDelete != "" {
		sb.WriteString(fmt.Sprintf(" ON DELETE %s", fk.OnDelete))
	}
	if fk.OnUpdate != "" {
		sb.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate))
	}

	return sb.String()
}

// quote wraps an identifier in double quotes for PostgreSQL.
func quote(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// quoteSlice quotes each element in a string slice.
func quoteSlice(names []string) []string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = quote(n)
	}
	return quoted
}

// formatDefault formats a default value for SQL.
func formatDefault(value any) string {
	switch v := value.(type) {
	case schema.RawExpression:
		return v.Expression
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", v)
	}
}
