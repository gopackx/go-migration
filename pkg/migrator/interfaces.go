package migrator

import "github.com/gopackx/go-migration/pkg/schema"

// Migration defines the contract for a database migration.
// Each migration must implement Up and Down methods that receive
// a schema Builder to define schema changes.
type Migration interface {
	Up(schema *schema.Builder) error
	Down(schema *schema.Builder) error
}

// TransactionOption allows a migration to opt out of transaction wrapping.
// If a migration implements this interface and DisableTransaction returns true,
// the runner will execute it without a surrounding database transaction.
type TransactionOption interface {
	DisableTransaction() bool
}
