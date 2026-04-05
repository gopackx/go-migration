package migrator

import "fmt"

// BatchManager provides batch-level operations on top of the Tracker.
type BatchManager struct {
	tracker *Tracker
}

// NewBatchManager creates a new BatchManager that delegates to the given Tracker.
func NewBatchManager(tracker *Tracker) *BatchManager {
	return &BatchManager{tracker: tracker}
}

// NextBatchNumber returns the next batch number to use for a new migration run.
// It is always one greater than the current highest batch number.
func (b *BatchManager) NextBatchNumber() (int, error) {
	last, err := b.tracker.GetLastBatchNumber()
	if err != nil {
		return 0, fmt.Errorf("next batch number: %w", err)
	}
	return last + 1, nil
}

// GetLastBatch returns all migration records from the most recent batch.
// Returns an empty slice if no migrations have been applied.
func (b *BatchManager) GetLastBatch() ([]MigrationRecord, error) {
	last, err := b.tracker.GetLastBatchNumber()
	if err != nil {
		return nil, fmt.Errorf("get last batch: %w", err)
	}
	if last == 0 {
		return nil, nil
	}
	records, err := b.tracker.GetByBatch(last)
	if err != nil {
		return nil, fmt.Errorf("get last batch: %w", err)
	}
	return records, nil
}

// GetLastNMigrations returns the last N applied migrations in reverse
// timestamp order. If fewer than N migrations exist, all are returned.
func (b *BatchManager) GetLastNMigrations(n int) ([]MigrationRecord, error) {
	if n <= 0 {
		return nil, nil
	}
	applied, err := b.tracker.GetApplied()
	if err != nil {
		return nil, fmt.Errorf("get last %d migrations: %w", n, err)
	}
	if len(applied) == 0 {
		return nil, nil
	}
	if n > len(applied) {
		n = len(applied)
	}
	// Reverse the last N entries so they are in reverse timestamp order.
	result := make([]MigrationRecord, n)
	for i := 0; i < n; i++ {
		result[i] = applied[len(applied)-1-i]
	}
	return result, nil
}
