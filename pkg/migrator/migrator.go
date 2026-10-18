package migrator

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"time"

	"github.com/gopackx/go-migration/pkg/schema"
)

// MigrationStatus represents the status of a single registered migration.
type MigrationStatus struct {
	Name      string
	Applied   bool
	Batch     int
	AppliedAt *time.Time
}

// Migrator is the top-level orchestrator that wires together the Registry,
// Runner, Tracker, BatchManager, and HookManager to execute migration
// lifecycle operations.
// ProgressEvent describes the completion of a single migration during
// an Up, Rollback, or Reset operation.
type ProgressEvent struct {
	Name      string
	Index     int // 0-based position
	Total     int // total migrations in this operation
	Duration  time.Duration
	Direction string // "up" or "down"
}

// ProgressFunc is a callback invoked after each migration completes.
type ProgressFunc func(event ProgressEvent)

type Migrator struct {
	db           *sql.DB
	tableName    string
	registry     *Registry
	runner       *Runner
	tracker      *Tracker
	batch        *BatchManager
	hooks        *HookManager
	grammar      schema.Grammar
	logger       Logger
	progressFn   ProgressFunc
	dryRun       bool
	dryRunWriter io.Writer
	lockTimeout  time.Duration
	lockDisabled bool
}

// Option configures a Migrator.
type Option func(*Migrator)

// WithTableName sets the migration tracking table name (default: "migrations").
func WithTableName(name string) Option {
	return func(m *Migrator) {
		m.tableName = name
	}
}

// WithGrammar sets the SQL grammar used by the Runner, the migration Tracker
// and Fresh(). Without it the Tracker falls back to PostgreSQL syntax.
func WithGrammar(g schema.Grammar) Option {
	return func(m *Migrator) {
		m.grammar = g
	}
}

// WithLogger sets the logger used by the Migrator and Runner.
func WithLogger(l Logger) Option {
	return func(m *Migrator) {
		m.logger = l
	}
}

// WithAutoDiscover returns an Option that loads auto-registered migrations.
// Panics if AutoDiscover() returns an error (consistent with option pattern).
func WithAutoDiscover() Option {
	return func(m *Migrator) {
		if err := m.AutoDiscover(); err != nil {
			panic(fmt.Sprintf("WithAutoDiscover: %v", err))
		}
	}
}

// WithProgress sets a callback that is invoked after each migration completes
// during Up, Rollback, or Reset operations.
func WithProgress(fn ProgressFunc) Option {
	return func(m *Migrator) {
		m.progressFn = fn
	}
}

// WithDryRun enables dry-run mode: SQL statements are written to w
// instead of being executed against the database.
func WithDryRun(w io.Writer) Option {
	return func(m *Migrator) {
		m.dryRun = true
		m.dryRunWriter = w
	}
}

// New creates a new Migrator with the given database connection and options.
// Defaults: table name "migrations", nil grammar, nil logger, migration lock
// enabled with DefaultLockTimeout.
//
// Options only record settings; the Runner, Tracker and BatchManager are built
// once all options have been applied, so the result does not depend on the
// order of WithTableName / WithGrammar / WithLogger.
func New(db *sql.DB, opts ...Option) *Migrator {
	m := &Migrator{
		db:          db,
		tableName:   "migrations",
		registry:    NewRegistry(),
		hooks:       NewHookManager(),
		lockTimeout: DefaultLockTimeout,
	}

	for _, opt := range opts {
		opt(m)
	}

	m.tracker = NewTrackerWithGrammar(db, m.tableName, m.grammar)
	m.batch = NewBatchManager(m.tracker)
	m.runner = NewRunner(db, m.grammar, m.logger)

	// Propagate dry-run settings to the runner.
	if m.dryRun && m.dryRunWriter != nil {
		m.runner.SetDryRun(m.dryRunWriter)
	}

	return m
}

// Register adds a migration to the registry.
func (m *Migrator) Register(name string, migration Migration) error {
	return m.registry.Register(name, migration)
}

// AutoDiscover loads all migrations from the global auto-registry
// into the Migrator's internal registry.
func (m *Migrator) AutoDiscover() error {
	autoMigrations := GetAutoRegistered()
	for _, am := range autoMigrations {
		if err := m.registry.Register(am.Name, am.Migration); err != nil {
			return fmt.Errorf("auto-discover: %w", err)
		}
	}
	return nil
}

// BeforeMigrate registers a before-migration hook.
func (m *Migrator) BeforeMigrate(fn HookFunc) {
	m.hooks.RegisterBefore(fn)
}

// AfterMigrate registers an after-migration hook.
func (m *Migrator) AfterMigrate(fn AfterHookFunc) {
	m.hooks.RegisterAfter(fn)
}

// Up runs all pending migrations in timestamp order.
// All migrations executed in a single Up() call share the same batch number.
// The migration lock is held for the whole operation (see Lock), so the
// pending list is computed only once no other process can change it.
func (m *Migrator) Up() error {
	return m.withLock(func() error { return m.up(false) })
}

// up runs all pending migrations. afterDropAll is set only by a dry-run
// Fresh: the tables were (notionally) dropped but still exist, so the
// tracking table is not consulted, every registered migration is pending and
// the batch number is 1.
func (m *Migrator) up(afterDropAll bool) error {
	var applied []MigrationRecord
	if !afterDropAll {
		if err := m.prepareTrackingTable(); err != nil {
			return err
		}
		var err error
		applied, err = m.tracker.GetApplied()
		if err != nil {
			return err
		}
	}

	registered := m.registry.GetAll()

	// Build a set of applied migration names for fast lookup.
	appliedSet := make(map[string]struct{}, len(applied))
	for _, rec := range applied {
		appliedSet[rec.Name] = struct{}{}
	}

	// Compute pending migrations (registered but not applied).
	var pending []registeredMigration
	for _, reg := range registered {
		if _, ok := appliedSet[reg.Name]; !ok {
			pending = append(pending, reg)
		}
	}

	if len(pending) == 0 {
		return nil
	}

	batchNumber := 1
	if !afterDropAll {
		var err error
		batchNumber, err = m.batch.NextBatchNumber()
		if err != nil {
			return err
		}
	}

	total := len(pending)
	for i, p := range pending {
		if err := m.hooks.RunBefore(p.Name, "up"); err != nil {
			return fmt.Errorf("before hook for %q: %w", p.Name, err)
		}

		start := time.Now()

		if m.dryRun {
			if err := m.runner.ExecuteDryRun(p.Migration, "up", p.Name); err != nil {
				return fmt.Errorf("migration %q up: %w", p.Name, err)
			}
		} else {
			// The tracking row is written on the migration's own transaction
			// (or on the DB after success for non-transactional migrations).
			name := p.Name
			record := func(ex schema.Executor) error {
				return m.tracker.RecordTx(ex, name, batchNumber)
			}
			if err := m.runner.ExecuteWithHook(p.Migration, "up", p.Name, record); err != nil {
				return fmt.Errorf("migration %q up: %w", p.Name, err)
			}
		}

		duration := time.Since(start)
		_ = m.hooks.RunAfter(p.Name, "up", duration)

		if m.progressFn != nil {
			m.progressFn(ProgressEvent{
				Name:      p.Name,
				Index:     i,
				Total:     total,
				Duration:  duration,
				Direction: "up",
			})
		}
	}

	return nil
}

// Rollback rolls back migrations.
// If steps == 0, rolls back the last batch.
// If steps > 0, rolls back the last N individual migrations.
// The migration lock is held for the whole operation.
func (m *Migrator) Rollback(steps int) error {
	return m.withLock(func() error { return m.rollback(steps) })
}

func (m *Migrator) rollback(steps int) error {
	if err := m.prepareTrackingTable(); err != nil {
		return err
	}

	var records []MigrationRecord
	var err error

	if steps == 0 {
		records, err = m.batch.GetLastBatch()
	} else {
		records, err = m.batch.GetLastNMigrations(steps)
	}
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil
	}

	// GetLastBatch returns in ascending order; we need reverse timestamp order.
	// GetLastNMigrations already returns in reverse order.
	if steps == 0 {
		reverseRecords(records)
	}

	total := len(records)
	for i, rec := range records {
		migration, err := m.registry.Get(rec.Name)
		if err != nil {
			return err
		}

		if err := m.hooks.RunBefore(rec.Name, "down"); err != nil {
			return fmt.Errorf("before hook for %q: %w", rec.Name, err)
		}

		start := time.Now()

		if err := m.runDown(migration, rec.Name); err != nil {
			return err
		}

		duration := time.Since(start)
		_ = m.hooks.RunAfter(rec.Name, "down", duration)

		if m.progressFn != nil {
			m.progressFn(ProgressEvent{
				Name:      rec.Name,
				Index:     i,
				Total:     total,
				Duration:  duration,
				Direction: "down",
			})
		}
	}

	return nil
}

// Reset rolls back all applied migrations in reverse order.
// The migration lock is held for the whole operation.
func (m *Migrator) Reset() error {
	return m.withLock(m.reset)
}

func (m *Migrator) reset() error {
	if err := m.prepareTrackingTable(); err != nil {
		return err
	}

	applied, err := m.tracker.GetApplied()
	if err != nil {
		return err
	}

	// Reverse to execute Down() in reverse timestamp order.
	reverseRecords(applied)

	total := len(applied)
	for i, rec := range applied {
		migration, err := m.registry.Get(rec.Name)
		if err != nil {
			return err
		}

		if err := m.hooks.RunBefore(rec.Name, "down"); err != nil {
			return fmt.Errorf("before hook for %q: %w", rec.Name, err)
		}

		start := time.Now()

		if err := m.runDown(migration, rec.Name); err != nil {
			return err
		}

		duration := time.Since(start)
		_ = m.hooks.RunAfter(rec.Name, "down", duration)

		if m.progressFn != nil {
			m.progressFn(ProgressEvent{
				Name:      rec.Name,
				Index:     i,
				Total:     total,
				Duration:  duration,
				Direction: "down",
			})
		}
	}

	return nil
}

// prepareTrackingTable makes sure the tracking table can be used. Normally it
// creates the table if needed. In dry-run mode no DDL may be executed, so it
// only checks that the table exists and fails with ErrTrackingTable if not;
// all later tracker calls in dry-run mode are read-only queries.
func (m *Migrator) prepareTrackingTable() error {
	if !m.dryRun {
		return m.tracker.EnsureTable()
	}
	exists, err := m.tracker.TableExists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("dry-run: migration tracking table %q does not exist; run migrate:install first: %w",
			m.tableName, ErrTrackingTable)
	}
	return nil
}

// runDown executes a migration's Down and removes its tracking row. For
// transactional migrations the DELETE runs on the migration's transaction, so
// a failed Down never loses the record and a failed DELETE rolls Down back.
func (m *Migrator) runDown(migration Migration, name string) error {
	if m.dryRun {
		if err := m.runner.ExecuteDryRun(migration, "down", name); err != nil {
			return fmt.Errorf("migration %q down: %w", name, err)
		}
		return nil
	}
	remove := func(ex schema.Executor) error {
		return m.tracker.RemoveTx(ex, name)
	}
	if err := m.runner.ExecuteWithHook(migration, "down", name, remove); err != nil {
		return fmt.Errorf("migration %q down: %w", name, err)
	}
	return nil
}

// Refresh resets all migrations and then runs them all up again. One
// migration lock is held across both phases, so no other process can run in
// between.
func (m *Migrator) Refresh() error {
	return m.withLock(func() error {
		if err := m.reset(); err != nil {
			return fmt.Errorf("refresh reset phase: %w", err)
		}
		if err := m.up(false); err != nil {
			return fmt.Errorf("refresh up phase: %w", err)
		}
		return nil
	})
}

// Fresh drops all tables and then runs all migrations up.
// Requires a grammar to be configured via WithGrammar.
//
// The table list is read with the grammar's CompileListTables query and the
// drop statements from CompileDropAllTables are executed one by one on a
// single dedicated connection, because session settings such as MySQL's
// FOREIGN_KEY_CHECKS or SQLite's PRAGMA foreign_keys are per connection.
// In dry-run mode the (read-only) list query still runs so the printed DROP
// statements are accurate, but nothing is dropped.
//
// Only the default schema is cleared; a schema-qualified tracking table
// (WithTableName("app.migrations")) is dropped explicitly afterwards.
//
// The migration lock is acquired before anything is dropped and held until
// the migrations have run.
func (m *Migrator) Fresh() error {
	if m.grammar == nil {
		return fmt.Errorf("fresh requires a grammar: configure with WithGrammar")
	}

	return m.withLock(func() error {
		if err := m.dropAllTables(); err != nil {
			return err
		}

		// A schema-qualified tracking table ("app.migrations") lives outside
		// the default schema that dropAllTables clears, so it would survive
		// with its rows and Up would find nothing pending. Drop it as well.
		if !m.dryRun && m.tracker.isQualified() {
			if err := m.tracker.dropTable(m.db); err != nil {
				return err
			}
		}

		// In dry-run nothing was dropped, so the existing tracking table must
		// be ignored: after a real Fresh every migration would run in batch 1.
		return m.up(m.dryRun)
	})
}

// dropAllTables lists and drops every table on one dedicated connection. The
// connection is released before returning so that a pool limited to a single
// open connection (MaxOpenConns=1, common for SQLite) does not deadlock in Up.
// Once any drop statement has run, the physical connection is discarded rather
// than returned to the pool (see discardConn).
func (m *Migrator) dropAllTables() error {
	ctx := context.Background()
	conn, err := m.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("drop all tables: acquire connection: %w", err)
	}
	sessionTouched := false
	defer func() {
		if sessionTouched {
			discardConn(conn)
		}
		// After discardConn, Close returns sql.ErrConnDone; nothing to do.
		_ = conn.Close()
	}()

	tables, err := listTables(ctx, conn, m.grammar.CompileListTables())
	if err != nil {
		return fmt.Errorf("drop all tables: list tables: %w", err)
	}

	stmts := m.grammar.CompileDropAllTables(tables)

	if m.dryRun {
		fmt.Fprintln(m.dryRunWriter, "-- Fresh: drop all tables")
		for _, stmt := range stmts {
			fmt.Fprintf(m.dryRunWriter, "%s;\n", stmt)
		}
		return nil
	}

	sessionTouched = true
	for _, stmt := range stmts {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("drop all tables: %s: %w", stmt, err)
		}
	}
	return nil
}

// discardConn makes database/sql close the physical connection behind conn
// instead of returning it to the pool. The drop-all statements change
// per-connection session state (SQLite "PRAGMA foreign_keys = OFF"; MySQL
// "SET FOREIGN_KEY_CHECKS = 0", whose trailing "= 1" never runs if a DROP
// fails), and a pooled connection carrying that state would silently disable
// foreign key enforcement for whatever later borrows it. Returning
// driver.ErrBadConn from Conn.Raw's callback is the documented way to have
// database/sql drop the connection.
func discardConn(conn *sql.Conn) {
	_ = conn.Raw(func(any) error { return driver.ErrBadConn })
}

// listTables runs the grammar's list-tables query and returns the names.
func listTables(ctx context.Context, conn *sql.Conn, query string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

// Status returns the status of all registered migrations, indicating
// whether each has been applied, its batch number, and applied timestamp.
//
// Status is read-only: it never creates the tracking table and does not take
// the migration lock, so it is safe to use as a readiness check. If the
// tracking table does not exist, every registered migration is pending.
func (m *Migrator) Status() ([]MigrationStatus, error) {
	registered := m.registry.GetAll()

	exists, err := m.tracker.TableExists()
	if err != nil {
		return nil, err
	}
	var applied []MigrationRecord
	if exists {
		applied, err = m.tracker.GetApplied()
		if err != nil {
			return nil, err
		}
	}

	// Build lookup map from applied records.
	appliedMap := make(map[string]MigrationRecord, len(applied))
	for _, rec := range applied {
		appliedMap[rec.Name] = rec
	}

	statuses := make([]MigrationStatus, 0, len(registered))
	for _, reg := range registered {
		status := MigrationStatus{Name: reg.Name}
		if rec, ok := appliedMap[reg.Name]; ok {
			status.Applied = true
			status.Batch = rec.Batch
			t := rec.CreatedAt
			status.AppliedAt = &t
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// reverseRecords reverses a slice of MigrationRecord in place.
func reverseRecords(records []MigrationRecord) {
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
}
