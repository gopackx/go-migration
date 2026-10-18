package schema

import "database/sql"

// Executor abstracts database execution so that both *sql.DB and *sql.Tx
// can be used interchangeably with the Builder.
type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Builder provides a fluent API for defining database schema changes.
// It compiles Blueprint definitions through a Grammar and executes the
// resulting SQL against the database.
type Builder struct {
	executor Executor
	grammar  Grammar
}

// NewBuilder creates a new Builder with the given executor and grammar.
// The executor can be a *sql.DB or *sql.Tx.
func NewBuilder(executor Executor, grammar Grammar) *Builder {
	return &Builder{
		executor: executor,
		grammar:  grammar,
	}
}

// Create creates a new table by building a Blueprint via the callback,
// compiling it through the Grammar, and executing each resulting SQL
// statement (CREATE TABLE followed by any separate CREATE INDEX statements).
func (b *Builder) Create(table string, fn func(*Blueprint)) error {
	bp := NewBlueprint(table)
	fn(bp)

	stmts, err := b.grammar.CompileCreate(bp)
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		if _, err := b.executor.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// Alter modifies an existing table by building a Blueprint via the callback,
// compiling it through the Grammar, and executing each resulting SQL statement.
func (b *Builder) Alter(table string, fn func(*Blueprint)) error {
	bp := NewBlueprint(table)
	fn(bp)

	stmts, err := b.grammar.CompileAlter(bp)
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		if _, err := b.executor.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

// Drop drops the given table.
func (b *Builder) Drop(table string) error {
	sqlStr := b.grammar.CompileDrop(table)
	_, err := b.executor.Exec(sqlStr)
	return err
}

// DropIfExists drops the given table if it exists.
func (b *Builder) DropIfExists(table string) error {
	sqlStr := b.grammar.CompileDropIfExists(table)
	_, err := b.executor.Exec(sqlStr)
	return err
}

// Rename renames a table from one name to another.
func (b *Builder) Rename(from, to string) error {
	sqlStr := b.grammar.CompileRename(from, to)
	_, err := b.executor.Exec(sqlStr)
	return err
}

// HasTable checks whether the given table exists in the database.
//
// In dry-run mode (see DryRunExecutor) no query is run: the query is written
// to the dry-run output and the table is reported as absent (false, nil).
func (b *Builder) HasTable(table string) (bool, error) {
	return b.queryExists(b.grammar.CompileHasTable(table))
}

// RawExec executes one or more raw SQL statements directly against the database.
// Use this for operations not covered by the fluent API, such as creating
// materialized views, custom indexes, or database-specific DDL.
func (b *Builder) RawExec(statements ...string) error {
	for _, stmt := range statements {
		if _, err := b.executor.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// HasColumn checks whether the given column exists in the given table.
//
// In dry-run mode (see DryRunExecutor) no query is run: the query is written
// to the dry-run output and the column is reported as absent (false, nil).
func (b *Builder) HasColumn(table, column string) (bool, error) {
	return b.queryExists(b.grammar.CompileHasColumn(table, column))
}

// queryExists runs a COUNT(*) style query and reports whether the count is
// greater than zero. Dry-run executors return no row, so in dry-run mode the
// query is only recorded and false is returned.
func (b *Builder) queryExists(sqlStr string) (bool, error) {
	if isDryRun(b.executor) {
		// DryRunExecutor.QueryRow writes the query and returns a nil row.
		_ = b.executor.QueryRow(sqlStr)
		return false, nil
	}
	var count int
	if err := b.executor.QueryRow(sqlStr).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// dryRunner is implemented by executors that do not touch the database
// (DryRunExecutor, and RecordingExecutor when wrapping one).
type dryRunner interface {
	IsDryRun() bool
}

// isDryRun reports whether ex is a dry-run executor.
func isDryRun(ex Executor) bool {
	d, ok := ex.(dryRunner)
	return ok && d.IsDryRun()
}
