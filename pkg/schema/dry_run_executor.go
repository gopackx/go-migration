package schema

import (
	"database/sql"
	"fmt"
	"io"
)

// Compile-time check that DryRunExecutor implements Executor.
var _ Executor = (*DryRunExecutor)(nil)

// DryRunExecutor implements Executor but writes SQL to a Writer
// instead of executing it against a database.
type DryRunExecutor struct {
	Writer io.Writer
}

func (d *DryRunExecutor) Exec(query string, args ...any) (sql.Result, error) {
	fmt.Fprintf(d.Writer, "%s;\n", query)
	return nil, nil
}

// QueryRow writes the query to the Writer and returns a nil *sql.Row, which
// must not be scanned. Builder.HasTable/HasColumn detect dry-run mode via
// IsDryRun and never scan it; they report the table/column as absent.
func (d *DryRunExecutor) QueryRow(query string, args ...any) *sql.Row {
	fmt.Fprintf(d.Writer, "%s;\n", query)
	return nil
}

// IsDryRun reports that this executor does not touch the database.
func (d *DryRunExecutor) IsDryRun() bool { return true }
