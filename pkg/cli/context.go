// Package cli provides the command-line interface for go-migration.
package cli

import (
	"database/sql"

	"github.com/gopackx/go-migration/internal/generator"
	"github.com/gopackx/go-migration/internal/logger"
	"github.com/gopackx/go-migration/pkg/cli/commands"
	"github.com/gopackx/go-migration/pkg/config"
	"github.com/gopackx/go-migration/pkg/seeder"
)

// CLIContext bundles the dependencies needed by CLI command handlers.
// It is constructed during app initialization and passed to each command.
type CLIContext struct {
	Config    *config.Config
	DB        *sql.DB
	Migrator  commands.Migrator
	Seeder    *seeder.Runner
	Generator *generator.Generator
	Logger    logger.Logger
}

// NewCLIContext creates a CLIContext with the given dependencies.
func NewCLIContext(cfg *config.Config, db *sql.DB, m commands.Migrator, s *seeder.Runner, g *generator.Generator, l logger.Logger) *CLIContext {
	return &CLIContext{
		Config:    cfg,
		DB:        db,
		Migrator:  m,
		Seeder:    s,
		Generator: g,
		Logger:    l,
	}
}
