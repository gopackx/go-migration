package migrator

import (
	"database/sql"
	"fmt"
	"time"
)

// MigrationRecord represents a single row in the migration tracking table.
type MigrationRecord struct {
	Name      string
	Batch     int
	CreatedAt time.Time
}

// Tracker manages the migrations tracking table in the database.
type Tracker struct {
	db        *sql.DB
	tableName string
}

// NewTracker creates a new Tracker that uses the given database connection
// and stores records in the specified table.
func NewTracker(db *sql.DB, tableName string) *Tracker {
	return &Tracker{
		db:        db,
		tableName: tableName,
	}
}

// EnsureTable creates the migration tracking table if it does not already exist.
// This operation is idempotent — calling it multiple times has no effect.
func (t *Tracker) EnsureTable() error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id         SERIAL PRIMARY KEY,
		migration  VARCHAR(255) NOT NULL UNIQUE,
		batch      INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`, t.tableName)

	if _, err := t.db.Exec(query); err != nil {
		return fmt.Errorf("ensure tracking table %q: %w", t.tableName, ErrTrackingTable)
	}
	return nil
}

// GetApplied returns all migration records ordered by name ascending.
func (t *Tracker) GetApplied() ([]MigrationRecord, error) {
	query := fmt.Sprintf(
		`SELECT migration, batch, created_at FROM %s ORDER BY migration ASC`,
		t.tableName,
	)

	rows, err := t.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("get applied migrations: %w", ErrTrackingTable)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var r MigrationRecord
		if err := rows.Scan(&r.Name, &r.Batch, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan migration record: %w", ErrTrackingTable)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration records: %w", ErrTrackingTable)
	}
	return records, nil
}

// GetByBatch returns all migration records for the given batch number,
// ordered by name ascending.
func (t *Tracker) GetByBatch(batch int) ([]MigrationRecord, error) {
	query := fmt.Sprintf(
		`SELECT migration, batch, created_at FROM %s WHERE batch = $1 ORDER BY migration ASC`,
		t.tableName,
	)

	rows, err := t.db.Query(query, batch)
	if err != nil {
		return nil, fmt.Errorf("get migrations for batch %d: %w", batch, ErrTrackingTable)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var r MigrationRecord
		if err := rows.Scan(&r.Name, &r.Batch, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan migration record: %w", ErrTrackingTable)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration records: %w", ErrTrackingTable)
	}
	return records, nil
}

// GetLastBatchNumber returns the highest batch number in the tracking table.
// Returns 0 if no records exist.
func (t *Tracker) GetLastBatchNumber() (int, error) {
	query := fmt.Sprintf(
		`SELECT COALESCE(MAX(batch), 0) FROM %s`,
		t.tableName,
	)

	var batch int
	if err := t.db.QueryRow(query).Scan(&batch); err != nil {
		return 0, fmt.Errorf("get last batch number: %w", ErrTrackingTable)
	}
	return batch, nil
}

// Record inserts a new migration record with the given name and batch number.
func (t *Tracker) Record(name string, batch int) error {
	query := fmt.Sprintf(
		`INSERT INTO %s (migration, batch) VALUES ($1, $2)`,
		t.tableName,
	)

	if _, err := t.db.Exec(query, name, batch); err != nil {
		return fmt.Errorf("record migration %q: %w", name, ErrTrackingTable)
	}
	return nil
}

// Remove deletes a migration record by name.
func (t *Tracker) Remove(name string) error {
	query := fmt.Sprintf(
		`DELETE FROM %s WHERE migration = $1`,
		t.tableName,
	)

	if _, err := t.db.Exec(query, name); err != nil {
		return fmt.Errorf("remove migration %q: %w", name, ErrTrackingTable)
	}
	return nil
}
