package migrator

import (
	"fmt"

	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
)

// grammarMap maps database driver names to grammar constructor functions.
var grammarMap = map[string]func() schema.Grammar{
	"postgres": func() schema.Grammar { return grammars.NewPostgresGrammar() },
	"mysql":    func() schema.Grammar { return grammars.NewMySQLGrammar() },
	"sqlite":   func() schema.Grammar { return grammars.NewSQLiteGrammar() },
	"sqlite3":  func() schema.Grammar { return grammars.NewSQLiteGrammar() },
}

// ResolveGrammar returns the schema.Grammar for the given database driver name.
// It returns an error if the driver is not recognized.
func ResolveGrammar(driver string) (schema.Grammar, error) {
	fn, ok := grammarMap[driver]
	if !ok {
		return nil, fmt.Errorf("unsupported driver %q", driver)
	}
	return fn(), nil
}
