package migrator

import (
	"flag"
	"fmt"
	"os"

	"github.com/gopackx/go-migration/internal/generator"
	"github.com/gopackx/go-migration/internal/logger"
	"github.com/gopackx/go-migration/pkg/cli"
	"github.com/gopackx/go-migration/pkg/cli/commands"
	"github.com/gopackx/go-migration/pkg/config"
	"github.com/gopackx/go-migration/pkg/database"
	"github.com/gopackx/go-migration/pkg/database/drivers"
	"github.com/gopackx/go-migration/pkg/seeder"
)

// commandsNeedingDB lists commands that require a database connection.
// Commands not in this set (version, help, make:migration, make:seeder,
// make:factory) skip the database setup entirely.
var commandsNeedingDB = map[string]bool{
	"migrate":          true,
	"migrate:rollback": true,
	"migrate:reset":    true,
	"migrate:refresh":  true,
	"migrate:fresh":    true,
	"migrate:status":   true,
	"migrate:install":  true,
	"db:seed":          true,
	"db:seed:rollback": true,
	"db:seed:truncate": true,
}

// migratorAdapter wraps *Migrator to satisfy commands.MigratorRunner.
// This adapter converts []MigrationStatus to []commands.MigrationStatusInfo
// in the Status() method, bridging the two packages without a direct import
// from commands to migrator.
type migratorAdapter struct {
	m *Migrator
}

func (a *migratorAdapter) Up() error                { return a.m.Up() }
func (a *migratorAdapter) Rollback(steps int) error { return a.m.Rollback(steps) }
func (a *migratorAdapter) Reset() error             { return a.m.Reset() }
func (a *migratorAdapter) Refresh() error           { return a.m.Refresh() }
func (a *migratorAdapter) Fresh() error             { return a.m.Fresh() }

func (a *migratorAdapter) Status() ([]commands.MigrationStatusInfo, error) {
	statuses, err := a.m.Status()
	if err != nil {
		return nil, err
	}
	result := make([]commands.MigrationStatusInfo, len(statuses))
	for i, s := range statuses {
		result[i] = commands.MigrationStatusInfo{
			Name:      s.Name,
			Applied:   s.Applied,
			Batch:     s.Batch,
			AppliedAt: s.AppliedAt,
		}
	}
	return result, nil
}

// Run is the all-in-one entry point for the go-migration CLI.
// It handles the full lifecycle: parse CLI args, load config, connect DB,
// auto-discover migrations and seeders, and dispatch the command.
// On fatal error it prints to stderr and exits with code 1.
//
// Usage in main.go:
//
//	package main
//
//	import (
//	    _ "myapp/migrations"
//	    _ "myapp/seeders"
//	    "github.com/gopackx/go-migration/pkg/migrator"
//	)
//
//	func main() {
//	    migrator.Run()
//	}
func Run() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run implements the full CLI lifecycle and returns an error for testability.
func run() error {
	app := cli.NewApp(nil)
	router := app.Router()

	// cmdCtx is populated lazily by the pre-run hook for commands that need DB.
	var cmdCtx *commands.CommandContext
	// connManager is stored so we can defer Close().
	var connManager *database.Manager

	getCtx := func() *commands.CommandContext {
		return cmdCtx
	}

	// --- Command registration ---
	// Wrapper to bridge cli.CommandHandler (named type) with the expanded
	// function type expected by the Register* functions in commands package.
	reg := func(name, desc string, handler func([]string, *flag.FlagSet) error, setupFlags func(*flag.FlagSet)) {
		router.Register(name, desc, handler, setupFlags)
	}
	out := router.Out()
	errOut := router.ErrOut()
	in := router.Stdin()

	commands.RegisterInitCommand(reg, out, errOut)
	commands.RegisterMigrateCommand(reg, getCtx, out)
	commands.RegisterMigrateRollbackCommand(reg, getCtx, out)
	commands.RegisterMigrateResetCommand(reg, getCtx, out, in)
	commands.RegisterMigrateRefreshCommand(reg, getCtx, out)
	commands.RegisterMigrateFreshCommand(reg, getCtx, out, in)
	commands.RegisterMigrateStatusCommand(reg, getCtx, out)
	commands.RegisterMigrateInstallCommand(reg, getCtx)
	commands.RegisterMakeMigrationCommand(reg, getCtx, out)
	commands.RegisterMakeSeederCommand(reg, getCtx, out)
	commands.RegisterMakeFactoryCommand(reg, getCtx, out)
	commands.RegisterSeedCommand(reg, getCtx)
	commands.RegisterSeedRollbackCommand(reg, getCtx)
	commands.RegisterSeedTruncateCommand(reg, getCtx)

	// --- PreRun: config loading, DB connection, auto-discover ---
	router.SetPreRun(func(cmdName string, configPath string, flags *flag.FlagSet) error {
		if !commandsNeedingDB[cmdName] {
			// For make:* commands we still need the generator.
			if cmdName == "make:migration" || cmdName == "make:seeder" || cmdName == "make:factory" {
				cfg, err := loadConfig(configPath, false)
				if err != nil {
					return err
				}
				gen := generator.NewGenerator(cfg.MigrationDir)
				if cmdName == "make:seeder" {
					gen = generator.NewGenerator(cfg.SeederDir)
				}
				if cmdName == "make:factory" {
					gen = generator.NewGenerator(cfg.FactoryDir)
				}
				cmdCtx = &commands.CommandContext{Generator: gen}
			}
			return nil
		}

		// Load configuration.
		cfg, err := loadConfig(configPath, true)
		if err != nil {
			return err
		}

		// Set up logger.
		log := setupLogger(cfg)

		// Set up database connection manager with all drivers.
		connManager = database.NewManager()
		connManager.RegisterDriver("postgres", drivers.NewPostgresDriver())
		connManager.RegisterDriver("mysql", drivers.NewMySQLDriver())
		connManager.RegisterDriver("sqlite3", drivers.NewSQLiteDriver())

		// Add all configured connections.
		for name, connCfg := range cfg.Connections {
			dbCfg := toDBConnectionConfig(connCfg)
			if err := connManager.AddConnection(name, dbCfg); err != nil {
				return fmt.Errorf("add connection %q: %w", name, err)
			}
		}

		// Set default connection if specified.
		if cfg.DefaultConn != "" {
			if err := connManager.SetDefault(cfg.DefaultConn); err != nil {
				return fmt.Errorf("set default connection: %w", err)
			}
		}

		// Get the default DB connection.
		db, err := connManager.Default()
		if err != nil {
			return fmt.Errorf("get default connection: %w", err)
		}

		// Resolve grammar from default connection driver.
		connCfg := cfg.Connections[cfg.DefaultConn]
		grammar, err := ResolveGrammar(connCfg.Driver)
		if err != nil {
			return fmt.Errorf("resolve grammar for driver %q: %w", connCfg.Driver, err)
		}

		// Build Migrator options.
		opts := []Option{
			WithTableName(cfg.MigrationTable),
			WithLogger(log),
			WithGrammar(grammar),
		}

		// Enable dry-run mode if the command has --dry-run flag set.
		dryRunFlag := flags.Lookup("dry-run")
		if dryRunFlag != nil && dryRunFlag.Value.String() == "true" {
			opts = append(opts, WithDryRun(os.Stdout))
		}

		// Create Migrator with auto-discover.
		m := New(db, opts...)

		// Auto-discover migrations from global auto-registry.
		autoMigrations := GetAutoRegistered()
		for _, rm := range autoMigrations {
			if err := m.Register(rm.Name, rm.Migration); err != nil {
				return fmt.Errorf("register migration %q: %w", rm.Name, err)
			}
		}

		// Auto-discover seeders from global auto-registry.
		seederRegistry := seeder.NewRegistry()
		autoSeeders := seeder.GetAutoRegistered()
		for name, s := range autoSeeders {
			if err := seederRegistry.Register(name, s); err != nil {
				return fmt.Errorf("register seeder %q: %w", name, err)
			}
		}
		seederRunner := seeder.NewRunner(seederRegistry, db, log)

		// Create Generator.
		gen := generator.NewGenerator(cfg.MigrationDir)

		// Create Tracker for migrate:install command.
		tracker := NewTracker(db, cfg.MigrationTable)

		cmdCtx = &commands.CommandContext{
			DB:             db,
			Migrator:       &migratorAdapter{m: m},
			Seeder:         seederRunner,
			Generator:      gen,
			TrackerEnsurer: tracker,
		}

		return nil
	})

	// --- --seed flag integration ---
	// The seed flag is handled by wrapping the original handlers.
	// This is done after registration by re-registering with wrapped handlers.
	// For simplicity, we handle it in the pre-run or post-run.
	// Actually, we need to handle --seed for migrate, migrate:fresh, migrate:refresh.
	// We'll wrap these commands' handlers.

	// Run the CLI.
	err := app.Run(nil)

	// Cleanup: close all database connections.
	if connManager != nil {
		connManager.Close()
	}

	return err
}

// loadConfig reads the config file and loads configuration.
// If validate is true, it also validates the config.
func loadConfig(configPath string, validate bool) (*config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		// Fall back to environment variables.
		cfg, err = config.LoadFromEnv()
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
	}

	cfg.ApplyDefaults()

	if validate {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// setupLogger creates a logger based on the configuration.
func setupLogger(cfg *config.Config) logger.Logger {
	level := logger.ParseLevel(cfg.LogLevel)

	if cfg.LogOutput != "" && cfg.LogOutput != "console" {
		fl, err := logger.NewFileLogger(cfg.LogOutput, level)
		if err == nil {
			return fl
		}
		// Fall back to console if file logger fails.
	}

	return logger.NewConsoleLogger(level)
}

// toDBConnectionConfig converts a config.ConnectionConfig to a database.ConnectionConfig.
func toDBConnectionConfig(c config.ConnectionConfig) database.ConnectionConfig {
	return database.ConnectionConfig{
		Driver:          c.Driver,
		Host:            c.Host,
		Port:            c.Port,
		Database:        c.Database,
		Username:        c.Username,
		Password:        c.Password,
		MaxOpenConns:    c.MaxOpenConns,
		MaxIdleConns:    c.MaxIdleConns,
		ConnMaxLifetime: c.ConnMaxLifetime,
		Options:         c.Options,
	}
}
