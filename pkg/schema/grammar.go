package schema

// Grammar defines the contract for compiling Blueprint definitions into
// database-specific SQL statements. Each supported database engine (PostgreSQL,
// MySQL, SQLite) provides its own Grammar implementation.
type Grammar interface {
	// CompileCreate generates a CREATE TABLE statement from the given Blueprint.
	CompileCreate(blueprint *Blueprint) (string, error)

	// CompileAlter generates ALTER TABLE statements from the given Blueprint.
	// Multiple statements may be returned (e.g., one per column add/drop).
	CompileAlter(blueprint *Blueprint) ([]string, error)

	// CompileDrop generates a DROP TABLE statement for the given table.
	CompileDrop(table string) string

	// CompileDropIfExists generates a DROP TABLE IF EXISTS statement for the given table.
	CompileDropIfExists(table string) string

	// CompileRename generates a RENAME TABLE statement.
	CompileRename(from, to string) string

	// CompileHasTable generates a query to check if a table exists.
	CompileHasTable(table string) string

	// CompileHasColumn generates a query to check if a column exists in a table.
	CompileHasColumn(table, column string) string

	// CompileDropAllTables generates a statement to drop all tables in the database.
	CompileDropAllTables() string

	// CompileColumnType returns the database-specific SQL type string for a column.
	CompileColumnType(col ColumnDefinition) (string, error)
}
