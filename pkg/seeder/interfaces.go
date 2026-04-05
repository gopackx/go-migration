package seeder

import "database/sql"

// Seeder defines the contract for a database seeder.
// Implementations populate database tables with data.
type Seeder interface {
	Run(db *sql.DB) error
}

// DependentSeeder extends Seeder with dependency declaration.
// DependsOn returns the names of seeders that must run before this one.
type DependentSeeder interface {
	Seeder
	DependsOn() []string
}

// TaggedSeeder extends Seeder with tag-based grouping.
// Tags returns the list of tags (e.g., "development", "testing", "production")
// that this seeder belongs to. Seeders implementing this interface can be
// selectively executed via Runner.RunByTag.
type TaggedSeeder interface {
	Seeder
	Tags() []string
}

// RollbackableSeeder extends Seeder with rollback capability.
// Rollback undoes the data changes made by the seeder's Run method.
type RollbackableSeeder interface {
	Seeder
	Rollback(db *sql.DB) error
}
