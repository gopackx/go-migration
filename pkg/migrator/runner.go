package migrator

import (
	"database/sql"
	"fmt"
	"io"
	"regexp"

	"github.com/gopackx/go-migration/pkg/schema"
)

// Logger defines a minimal logging interface for the runner.
// This will be replaced by the full logger package once implemented.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Runner executes migration Up/Down methods within transactions.
type Runner struct {
	db           *sql.DB
	grammar      schema.Grammar
	logger       Logger
	dryRun       bool
	dryRunWriter io.Writer
}

// NewRunner creates a new Runner with the given database connection, grammar, and logger.
// The logger parameter may be nil, in which case logging is silently skipped.
func NewRunner(db *sql.DB, grammar schema.Grammar, logger Logger) *Runner {
	return &Runner{
		db:      db,
		grammar: grammar,
		logger:  logger,
	}
}

// SetDryRun enables dry-run mode on the Runner. When active, SQL statements
// are written to the given writer instead of being executed against the database.
func (r *Runner) SetDryRun(w io.Writer) {
	r.dryRun = true
	r.dryRunWriter = w
}

// positionRe matches PostgreSQL-style "at character N" position info in error messages.
var positionRe = regexp.MustCompile(`at character (\d+)`)

// extractPosition attempts to extract position information from a database error.
// It looks for patterns like "at character N" commonly found in PostgreSQL errors.
func extractPosition(err error) string {
	if err == nil {
		return ""
	}
	if m := positionRe.FindString(err.Error()); m != "" {
		return m
	}
	return ""
}

// Execute runs a migration in the given direction ("up" or "down").
// migrationName is used to provide context in error messages via MigrationError.
// If dry-run mode is active, it uses a DryRunExecutor instead of the real database.
// If the migration implements TransactionOption and DisableTransaction() returns true,
// it executes without a transaction. Otherwise it delegates to ExecuteInTransaction.
func (r *Runner) Execute(m Migration, direction string, migrationName ...string) error {
	name := ""
	if len(migrationName) > 0 {
		name = migrationName[0]
	}
	return r.ExecuteWithHook(m, direction, name, nil)
}

// AfterMigrateFunc runs after a migration's Up/Down succeeded. For
// transactional migrations ex is the migration's *sql.Tx and the callback runs
// before commit, so its writes commit or roll back together with the
// migration. For migrations that disable transactions ex is the *sql.DB.
type AfterMigrateFunc func(ex schema.Executor) error

// ExecuteWithHook behaves like Execute, and additionally calls after (when
// non-nil) once the migration has succeeded, using the same executor the
// migration ran on. If after returns an error, a transactional migration is
// rolled back and the error is returned. In dry-run mode after is not called.
func (r *Runner) ExecuteWithHook(m Migration, direction string, migrationName string, after AfterMigrateFunc) error {
	if r.dryRun {
		return r.executeDryRun(m, direction)
	}
	if opt, ok := m.(TransactionOption); ok && opt.DisableTransaction() {
		if err := r.executeWithoutTransaction(m, direction, migrationName); err != nil {
			return err
		}
		if after != nil {
			return after(r.db)
		}
		return nil
	}
	return r.executeInTransaction(m, direction, migrationName, after)
}

// ExecuteDryRun runs a migration in the given direction ("up" or "down")
// with a migration name prefix written to the dry-run writer.
func (r *Runner) ExecuteDryRun(m Migration, direction string, migrationName string) error {
	fmt.Fprintf(r.dryRunWriter, "-- Migration: %s\n", migrationName)
	return r.executeDryRun(m, direction)
}

// executeDryRun runs a migration using a DryRunExecutor, writing SQL to the
// configured writer instead of executing against the database.
func (r *Runner) executeDryRun(m Migration, direction string) error {
	executor := &schema.DryRunExecutor{Writer: r.dryRunWriter}
	builder := schema.NewBuilder(executor, r.grammar)
	return r.runMigration(m, builder, direction)
}

// ExecuteInTransaction runs a migration within a database transaction.
// On success the transaction is committed. On migration error the transaction
// is rolled back. If commit fails, ErrTransactionFailed (wrapping the commit error)
// is returned. SQL errors are wrapped in MigrationError when migrationName is provided.
func (r *Runner) ExecuteInTransaction(m Migration, direction string, migrationName ...string) error {
	name := ""
	if len(migrationName) > 0 {
		name = migrationName[0]
	}
	return r.executeInTransaction(m, direction, name, nil)
}

// executeInTransaction runs the migration in a transaction and, on success,
// calls after (if non-nil) with the transaction before committing.
func (r *Runner) executeInTransaction(m Migration, direction string, name string, after AfterMigrateFunc) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	recorder := schema.NewRecordingExecutor(tx)
	builder := schema.NewBuilder(recorder, r.grammar)

	if err := r.runMigration(m, builder, direction); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback after migration error: %v, original: %w", rbErr, err)
		}
		if name != "" {
			migErr := wrapMigrationError(name, recorder.LastSQL, err)
			migErr.Position = extractPosition(err)
			return migErr
		}
		return err
	}

	if after != nil {
		if err := after(tx); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("rollback after post-migration error: %v, original: %w", rbErr, err)
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		// A failed Commit already ends the transaction (a further Rollback
		// would only return sql.ErrTxDone), so report the commit error itself.
		return fmt.Errorf("commit: %w: %w", ErrTransactionFailed, err)
	}

	return nil
}

// executeWithoutTransaction runs a migration directly against the database
// connection without wrapping it in a transaction. SQL errors are wrapped
// in MigrationError when migrationName is provided.
func (r *Runner) executeWithoutTransaction(m Migration, direction string, migrationName string) error {
	recorder := schema.NewRecordingExecutor(r.db)
	builder := schema.NewBuilder(recorder, r.grammar)

	if err := r.runMigration(m, builder, direction); err != nil {
		if migrationName != "" {
			migErr := wrapMigrationError(migrationName, recorder.LastSQL, err)
			migErr.Position = extractPosition(err)
			return migErr
		}
		return err
	}
	return nil
}

// runMigration calls the appropriate migration method based on direction.
func (r *Runner) runMigration(m Migration, builder *schema.Builder, direction string) error {
	switch direction {
	case "up":
		return m.Up(builder)
	case "down":
		return m.Down(builder)
	default:
		return fmt.Errorf("unknown migration direction: %q", direction)
	}
}
