package schema

// Grammar defines the contract for compiling Blueprint definitions into
// database-specific SQL statements. Each supported database engine (PostgreSQL,
// MySQL, SQLite) provides its own Grammar implementation.
type Grammar interface {
	// CompileCreate generates the statements needed to create a table from the
	// given Blueprint: a CREATE TABLE statement, optionally followed by one
	// CREATE INDEX statement per index the dialect cannot declare inline.
	// Each element is a single statement and must be executed separately, since
	// many drivers (pgx stdlib, go-sql-driver/mysql) reject multi-statement Exec.
	CompileCreate(blueprint *Blueprint) ([]string, error)

	// CompileAlter generates ALTER TABLE statements from the given Blueprint.
	// Multiple statements may be returned (e.g., one per column add/drop).
	CompileAlter(blueprint *Blueprint) ([]string, error)

	// CompileDrop generates a DROP TABLE statement for the given table.
	CompileDrop(table string) string

	// CompileDropIfExists generates a DROP TABLE IF EXISTS statement for the given table.
	CompileDropIfExists(table string) string

	// CompileRename generates a RENAME TABLE statement.
	CompileRename(from, to string) string

	// CompileHasTable generates a query to check if a table exists. The name
	// is a single identifier, looked up in the default schema.
	CompileHasTable(table string) string

	// CompileHasTableIn generates a query to check if table exists in the
	// given schema (PostgreSQL schema, MySQL database, SQLite attached
	// database name such as "main"). It is used for a schema-qualified
	// tracking table name (see QuoteQualified / SplitQualified).
	CompileHasTableIn(schema, table string) string

	// CompileHasColumn generates a query to check if a column exists in a table.
	CompileHasColumn(table, column string) string

	// CompileListTables generates a query returning one row per user table in
	// the current database/schema, with the table name as the only column.
	CompileListTables() string

	// CompileDropAllTables generates the statements that drop every table in
	// the given list (as returned by the CompileListTables query). Each element
	// is a single statement and must be executed separately, in order, on the
	// same connection. Grammars that can drop everything without the list
	// (e.g. PostgreSQL via DROP SCHEMA ... CASCADE) may ignore it.
	CompileDropAllTables(tables []string) []string

	// CompileTrackingTable generates a CREATE TABLE IF NOT EXISTS statement for
	// the migration tracking table with the columns id, migration (unique),
	// batch and created_at. The table name may be schema-qualified
	// ("app.migrations") and must be quoted with QuoteQualified.
	CompileTrackingTable(table string) string

	// QuoteIdentifier returns name quoted as a single identifier (table or
	// column name) for the dialect, escaping any embedded quote character by
	// doubling it: "a""b" on PostgreSQL and SQLite, `a``b` on MySQL. The name
	// is treated as one identifier: a dot is part of the name, not a schema
	// separator.
	QuoteIdentifier(name string) string

	// Placeholder returns the bind-parameter placeholder for the given 1-based
	// parameter index (e.g. "$1" for PostgreSQL, "?" for MySQL and SQLite).
	Placeholder(index int) string

	// CompileTryLock generates a non-blocking query that tries to acquire a
	// named, session-level lock used to serialise migration processes. The
	// query takes the lock key as its single bind parameter (Placeholder(1))
	// and returns one row with one boolean-ish column: 1/true when the lock was
	// acquired, 0/false (or NULL) when it is held by another session.
	//
	// The lock must belong to the database session (connection), not to a
	// transaction, so that it can be held across several transactions and is
	// released by the server if the connection is closed.
	//
	// An empty string means the dialect needs no explicit lock (SQLite, whose
	// file lock already serialises writers); callers then skip locking.
	CompileTryLock(key string) string

	// CompileUnlock generates the query releasing the lock taken by
	// CompileTryLock. It takes the key as its single bind parameter and returns
	// one row with one boolean-ish column (1/true = released). It must run on
	// the same connection that acquired the lock. An empty string means no
	// lock is used (see CompileTryLock).
	CompileUnlock(key string) string

	// CompileColumnType returns the database-specific SQL type string for a column.
	CompileColumnType(col ColumnDefinition) (string, error)
}
