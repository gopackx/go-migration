package migrator

import (
	"database/sql"
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
}

// Option configures a Migrator.
type Option func(*Migrator)

// WithTableName sets the migration tracking table name (default: "migrations").
func WithTableName(name string) Option {
	return func(m *Migrator) {
		m.tracker = NewTracker(m.db, name)
		m.batch = NewBatchManager(m.tracker)
	}
}

// WithGrammar sets the SQL grammar used by the Runner and Fresh().
func WithGrammar(g schema.Grammar) Option {
	return func(m *Migrator) {
		m.grammar = g
		m.runner = NewRunner(m.db, g, m.logger)
	}
}

// WithLogger sets the logger used by the Migrator and Runner.
func WithLogger(l Logger) Option {
	return func(m *Migrator) {
		m.logger = l
		// Rebuild runner with the new logger if grammar is already set.
		if m.grammar != nil {
			m.runner = NewRunner(m.db, m.grammar, l)
		}
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
// Defaults: table name "migrations", nil grammar, nil logger.
func New(db *sql.DB, opts ...Option) *Migrator {
	tracker := NewTracker(db, "migrations")
	m := &Migrator{
		db:       db,
		registry: NewRegistry(),
		tracker:  tracker,
		batch:    NewBatchManager(tracker),
		hooks:    NewHookManager(),
	}

	for _, opt := range opts {
		opt(m)
	}

	// Ensure runner exists even if no grammar option was provided.
	if m.runner == nil {
		m.runner = NewRunner(db, m.grammar, m.logger)
	}

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
func (m *Migrator) Up() error {
	if err := m.tracker.EnsureTable(); err != nil {
		return err
	}

	registered := m.registry.GetAll()
	applied, err := m.tracker.GetApplied()
	if err != nil {
		return err
	}

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

	batchNumber, err := m.batch.NextBatchNumber()
	if err != nil {
		return err
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
			if err := m.runner.Execute(p.Migration, "up", p.Name); err != nil {
				return fmt.Errorf("migration %q up: %w", p.Name, err)
			}
		}

		if !m.dryRun {
			if err := m.tracker.Record(p.Name, batchNumber); err != nil {
				return err
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
func (m *Migrator) Rollback(steps int) error {
	if err := m.tracker.EnsureTable(); err != nil {
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

		if m.dryRun {
			if err := m.runner.ExecuteDryRun(migration, "down", rec.Name); err != nil {
				return fmt.Errorf("migration %q down: %w", rec.Name, err)
			}
		} else {
			if err := m.runner.Execute(migration, "down", rec.Name); err != nil {
				return fmt.Errorf("migration %q down: %w", rec.Name, err)
			}
		}

		if !m.dryRun {
			if err := m.tracker.Remove(rec.Name); err != nil {
				return err
			}
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
func (m *Migrator) Reset() error {
	if err := m.tracker.EnsureTable(); err != nil {
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

		start := time.Now()

		if m.dryRun {
			if err := m.runner.ExecuteDryRun(migration, "down", rec.Name); err != nil {
				return fmt.Errorf("migration %q down: %w", rec.Name, err)
			}
		} else {
			if err := m.runner.Execute(migration, "down", rec.Name); err != nil {
				return fmt.Errorf("migration %q down: %w", rec.Name, err)
			}
		}

		if !m.dryRun {
			if err := m.tracker.Remove(rec.Name); err != nil {
				return err
			}
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

// Refresh resets all migrations and then runs them all up again.
func (m *Migrator) Refresh() error {
	if err := m.Reset(); err != nil {
		return fmt.Errorf("refresh reset phase: %w", err)
	}
	if err := m.Up(); err != nil {
		return fmt.Errorf("refresh up phase: %w", err)
	}
	return nil
}

// Fresh drops all tables and then runs all migrations up.
// Requires a grammar to be configured via WithGrammar.
func (m *Migrator) Fresh() error {
	if m.grammar == nil {
		return fmt.Errorf("fresh requires a grammar: configure with WithGrammar")
	}

	dropSQL := m.grammar.CompileDropAllTables()

	if m.dryRun {
		fmt.Fprintf(m.dryRunWriter, "-- Fresh: drop all tables\n%s;\n", dropSQL)
	} else {
		if _, err := m.db.Exec(dropSQL); err != nil {
			return fmt.Errorf("drop all tables: %w", err)
		}
	}

	return m.Up()
}

// Status returns the status of all registered migrations, indicating
// whether each has been applied, its batch number, and applied timestamp.
func (m *Migrator) Status() ([]MigrationStatus, error) {
	if err := m.tracker.EnsureTable(); err != nil {
		return nil, err
	}

	registered := m.registry.GetAll()
	applied, err := m.tracker.GetApplied()
	if err != nil {
		return nil, err
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
