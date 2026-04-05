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

func (d *DryRunExecutor) QueryRow(query string, args ...any) *sql.Row {
	fmt.Fprintf(d.Writer, "%s;\n", query)
	return nil
}
