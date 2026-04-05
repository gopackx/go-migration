package seeder

import "errors"

// Sentinel errors for the seeder system.
// Defined locally to avoid circular dependencies with pkg/migrator.
var (
	ErrDuplicateSeeder    = errors.New("duplicate seeder name")
	ErrInvalidSeederName  = errors.New("invalid seeder name")
	ErrSeederNotFound     = errors.New("seeder not found")
	ErrCircularDependency = errors.New("circular seeder dependency")
)
