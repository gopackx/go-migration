package grammars

import (
	"fmt"
	"strings"

	"github.com/gopackx/go-migration/pkg/schema"
)

// SQLiteGrammar compiles Blueprint definitions into SQLite-specific SQL.
type SQLiteGrammar struct{}

// NewSQLiteGrammar creates a new SQLiteGrammar.
func NewSQLiteGrammar() *SQLiteGrammar {
	return &SQLiteGrammar{}
}

// CompileCreate generates a CREATE TABLE statement from the given Blueprint.
func (g *SQLiteGrammar) CompileCreate(bp *schema.Blueprint) (string, error) {
	columns := bp.Columns()
	if len(columns) == 0 {
		return "", fmt.Errorf("table %q: no columns defined", bp.Table())
	}

	var parts []string

	// Compile columns
	for _, col := range columns {
		colSQL, err := g.compileColumnDef(col)
		if err != nil {
			return "", err
		}
		parts = append(parts, colSQL)
	}

	// Collect primary key columns that are NOT auto-increment INTEGER
	// (auto-increment INTEGER PRIMARY KEY is handled inline in compileColumnDef)
	var pkCols []string
	for _, col := range columns {
		if col.IsPrimary && !g.isInlineAutoIncrementPK(col) {
			pkCols = append(pkCols, quote(col.Name))
		}
	}
	if len(pkCols) > 0 {
		parts = append(parts, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}

	// Compile indexes as separate constraints for unique indexes
	for _, idx := range bp.Indexes() {
		switch idx.Type {
		case schema.IndexFulltext:
			return "", fmt.Errorf("fulltext indexes are not supported by SQLite")
		case schema.IndexSpatial:
			return "", fmt.Errorf("spatial indexes are not supported by SQLite")
		case schema.IndexUnique:
			quotedCols := quoteSlice(idx.Columns)
			parts = append(parts, fmt.Sprintf("CONSTRAINT %s UNIQUE (%s)", quote(idx.Name), strings.Join(quotedCols, ", ")))
		}
	}

	// Compile foreign keys
	for _, fk := range bp.ForeignKeys() {
		fkSQL := g.compileForeignKey(fk)
		parts = append(parts, fkSQL)
	}

	sql := fmt.Sprintf("CREATE TABLE %s (%s)", quote(bp.Table()), strings.Join(parts, ", "))

	// Append non-unique index CREATE INDEX statements separated by semicolons
	for _, idx := range bp.Indexes() {
		switch idx.Type {
		case schema.IndexFulltext:
			return "", fmt.Errorf("fulltext indexes are not supported by SQLite")
		case schema.IndexSpatial:
			return "", fmt.Errorf("spatial indexes are not supported by SQLite")
		case schema.IndexRegular:
			quotedCols := quoteSlice(idx.Columns)
			sql += fmt.Sprintf("; CREATE INDEX %s ON %s (%s)", quote(idx.Name), quote(bp.Table()), strings.Join(quotedCols, ", "))
		}
	}

	return sql, nil
}

// CompileAlter generates ALTER TABLE statements from the given Blueprint.
// SQLite has limited ALTER TABLE support: only ADD COLUMN is fully supported.
func (g *SQLiteGrammar) CompileAlter(bp *schema.Blueprint) ([]string, error) {
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
			// SQLite does not support dropping foreign keys directly;
			// emit the statement for compatibility but it may not work on all versions.
			stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", table, quote(cmd.Name)))
		}
	}

	// Add new indexes
	for _, idx := range bp.Indexes() {
		switch idx.Type {
		case schema.IndexFulltext:
			return nil, fmt.Errorf("fulltext indexes are not supported by SQLite")
		case schema.IndexSpatial:
			return nil, fmt.Errorf("spatial indexes are not supported by SQLite")
		case schema.IndexUnique:
			quotedCols := quoteSlice(idx.Columns)
			stmts = append(stmts, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)", quote(idx.Name), table, strings.Join(quotedCols, ", ")))
		default:
			quotedCols := quoteSlice(idx.Columns)
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
func (g *SQLiteGrammar) CompileDrop(table string) string {
	return fmt.Sprintf("DROP TABLE %s", quote(table))
}

// CompileDropIfExists generates a DROP TABLE IF EXISTS statement.
func (g *SQLiteGrammar) CompileDropIfExists(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table))
}

// CompileRename generates an ALTER TABLE ... RENAME TO statement.
func (g *SQLiteGrammar) CompileRename(from, to string) string {
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quote(from), quote(to))
}

// CompileHasTable generates a query to check if a table exists using sqlite_master.
func (g *SQLiteGrammar) CompileHasTable(table string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)
}

// CompileHasColumn generates a PRAGMA table_info query to check column existence.
func (g *SQLiteGrammar) CompileHasColumn(table, column string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name='%s'", table, column)
}

// CompileDropAllTables generates a query to retrieve all table names for dropping.
// SQLite doesn't support dropping all tables in a single statement, so this returns
// a query to select all table names from sqlite_master.
func (g *SQLiteGrammar) CompileDropAllTables() string {
	return "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
}

// CompileColumnType returns the SQLite-specific SQL type string for a column.
func (g *SQLiteGrammar) CompileColumnType(col schema.ColumnDefinition) (string, error) {
	switch col.Type {
	case schema.TypeString:
		return "TEXT", nil
	case schema.TypeText:
		return "TEXT", nil
	case schema.TypeInteger:
		if col.IsAutoIncrement && col.IsPrimary {
			return "INTEGER PRIMARY KEY AUTOINCREMENT", nil
		}
		return "INTEGER", nil
	case schema.TypeBigInteger:
		if col.IsAutoIncrement && col.IsPrimary {
			return "INTEGER PRIMARY KEY AUTOINCREMENT", nil
		}
		return "INTEGER", nil
	case schema.TypeBoolean:
		return "INTEGER", nil
	case schema.TypeTimestamp:
		return "TEXT", nil
	case schema.TypeDate:
		return "TEXT", nil
	case schema.TypeDecimal:
		return "REAL", nil
	case schema.TypeFloat:
		return "REAL", nil
	case schema.TypeUUID:
		return "TEXT", nil
	case schema.TypeJSON:
		return "TEXT", nil
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
		return fmt.Sprintf(`TEXT CHECK ("%s" IN (%s))`, col.Name, strings.Join(quoted, ",")), nil
	case schema.TypeChar:
		return "TEXT", nil
	case schema.TypeLongText:
		return "TEXT", nil
	case schema.TypeMediumText:
		return "TEXT", nil
	case schema.TypeTinyInt:
		return "INTEGER", nil
	case schema.TypeSmallInt:
		return "INTEGER", nil
	default:
		return "", fmt.Errorf("column %q: type %q: %w", col.Name, col.Type.String(), ErrUnsupportedType)
	}
}

// isInlineAutoIncrementPK returns true if the column is an auto-increment
// integer primary key, which SQLite handles inline (INTEGER PRIMARY KEY AUTOINCREMENT).
func (g *SQLiteGrammar) isInlineAutoIncrementPK(col schema.ColumnDefinition) bool {
	return col.IsPrimary && col.IsAutoIncrement &&
		(col.Type == schema.TypeInteger || col.Type == schema.TypeBigInteger)
}

// compileColumnDef compiles a single column definition into SQL.
func (g *SQLiteGrammar) compileColumnDef(col schema.ColumnDefinition) (string, error) {
	colType, err := g.CompileColumnType(col)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(quote(col.Name))
	sb.WriteString(" ")
	sb.WriteString(colType)

	// For INTEGER PRIMARY KEY AUTOINCREMENT, skip NOT NULL and other modifiers
	// since the type string already includes PRIMARY KEY AUTOINCREMENT.
	if g.isInlineAutoIncrementPK(col) {
		return sb.String(), nil
	}

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
func (g *SQLiteGrammar) compileForeignKey(fk schema.ForeignKeyDefinition) string {
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
