// Package app is the go-migration command-line application: it parses the
// command line, loads the configuration, opens the database connections,
// registers the auto-discovered migrations and seeders, and dispatches the
// command.
//
// It lives outside pkg/migrator so that importing the migrator library does
// not pull in the CLI, the config loader, the code generator or the YAML
// parser. A project's migration binary is typically just:
//
//	package main
//
//	import (
//	    _ "github.com/go-sql-driver/mysql"
//	    _ "github.com/lib/pq"
//	    _ "modernc.org/sqlite"
//
//	    _ "myapp/database/migrations"
//	    _ "myapp/database/seeders"
//
//	    "github.com/gopackx/go-migration/pkg/cli/app"
//	)
//
//	func main() {
//	    app.Run()
//	}
package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gopackx/go-migration/internal/generator"
	"github.com/gopackx/go-migration/internal/logger"
	"github.com/gopackx/go-migration/pkg/cli"
	"github.com/gopackx/go-migration/pkg/cli/commands"
	"github.com/gopackx/go-migration/pkg/config"
	"github.com/gopackx/go-migration/pkg/database"
	"github.com/gopackx/go-migration/pkg/database/drivers"
	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/seeder"
)

// commandsNeedingDB lists commands that require a database connection.
// Commands not in this set (version, help, init, make:migration, make:seeder,
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

// migrationLocker adapts Migrator.Lock to commands.Locker for the commands
// that do not go through the Migrator's own locking (migrate:install and the
// db:seed* commands), so they are serialised with each other and with
// migrations. As in the Migrator's own operations, a failure to release the
// lock is only logged: the lock connection has then been discarded, which
// releases the lock on the server.
type migrationLocker struct {
	m   *migrator.Migrator
	log logger.Logger
}

func (l *migrationLocker) Lock() (func() error, error) {
	release, err := l.m.Lock(context.Background())
	if err != nil {
		return nil, err
	}
	return func() error {
		if err := release(); err != nil && l.log != nil {
			l.log.Error("%v", err)
		}
		return nil
	}, nil
}

// Run is the all-in-one entry point for the go-migration CLI.
// It handles the full lifecycle: parse CLI args, load config, connect DB,
// auto-discover migrations and seeders, and dispatch the command.
// On error it prints to stderr and exits with code 1.
func Run() {
	if err := run(); err != nil {
		if errors.Is(err, commands.ErrPendingMigrations) {
			// migrate:status --check found pending migrations: report the
			// count without the "Error:" prefix; the exit status is still 1.
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

// run implements the full CLI lifecycle and returns an error for testability.
func run() error {
	cliApp := cli.NewApp(nil)
	router := cliApp.Router()

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
		// config accepts both "sqlite" and "sqlite3" as the driver name, so
		// register the same SQLite driver under both.
		sqliteDriver := drivers.NewSQLiteDriver()
		connManager.RegisterDriver("sqlite", sqliteDriver)
		connManager.RegisterDriver("sqlite3", sqliteDriver)

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
		grammar, err := migrator.ResolveGrammar(connCfg.Driver)
		if err != nil {
			return fmt.Errorf("resolve grammar for driver %q: %w", connCfg.Driver, err)
		}

		// Build Migrator options.
		opts := []migrator.Option{
			migrator.WithTableName(cfg.MigrationTable),
			migrator.WithLogger(log),
			migrator.WithGrammar(grammar),
		}

		// Enable dry-run mode if the command has --dry-run flag set.
		dryRunFlag := flags.Lookup("dry-run")
		if dryRunFlag != nil && dryRunFlag.Value.String() == "true" {
			opts = append(opts, migrator.WithDryRun(os.Stdout))
		}

		// Migration lock settings (--lock-timeout, --no-lock) on the commands
		// that change the database.
		lock, err := lockSettingsFromFlags(flags)
		if err != nil {
			return err
		}
		opts = append(opts, lock.options()...)

		// Create Migrator with auto-discover.
		m := migrator.New(db, opts...)

		// Auto-discover migrations from global auto-registry.
		autoMigrations := migrator.GetAutoRegistered()
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
		seederRunner := seeder.NewRunner(seederRegistry, db, log,
			seeder.WithDialect(seederDialect(connCfg.Driver)))

		// Create Generator.
		gen := generator.NewGenerator(cfg.MigrationDir)

		// Create Tracker for migrate:install command, using the connection's
		// dialect for the tracking table DDL.
		tracker := migrator.NewTrackerWithGrammar(db, cfg.MigrationTable, grammar)

		cmdCtx = &commands.CommandContext{
			DB:             db,
			Migrator:       m,
			Seeder:         seederRunner,
			Generator:      gen,
			TrackerEnsurer: tracker,
			Locker:         &migrationLocker{m: m, log: log},
		}

		return nil
	})

	// Run the CLI.
	err := cliApp.Run(nil)

	// Cleanup: close all database connections.
	if connManager != nil {
		connManager.Close()
	}

	return err
}

// lockSettings holds the --lock-timeout and --no-lock values of a command.
type lockSettings struct {
	// timeout is the --lock-timeout value; nil when the command has no lock
	// flags (the Migrator default, migrator.DefaultLockTimeout, then applies).
	timeout *time.Duration
	// disabled is true when --no-lock was given.
	disabled bool
}

// lockSettingsFromFlags reads the --lock-timeout and --no-lock flags, when
// the command defines them.
func lockSettingsFromFlags(flags *flag.FlagSet) (lockSettings, error) {
	var s lockSettings
	if f := flags.Lookup(commands.LockTimeoutFlag); f != nil {
		getter, ok := f.Value.(flag.Getter)
		if !ok {
			return s, fmt.Errorf("--%s: not a duration flag", commands.LockTimeoutFlag)
		}
		d, ok := getter.Get().(time.Duration)
		if !ok {
			return s, fmt.Errorf("--%s: not a duration flag", commands.LockTimeoutFlag)
		}
		s.timeout = &d
	}
	if f := flags.Lookup(commands.NoLockFlag); f != nil && f.Value.String() == "true" {
		s.disabled = true
	}
	return s, nil
}

// options converts the settings into Migrator options.
func (s lockSettings) options() []migrator.Option {
	var opts []migrator.Option
	if s.timeout != nil {
		opts = append(opts, migrator.WithLockTimeout(*s.timeout))
	}
	if s.disabled {
		opts = append(opts, migrator.WithoutLock())
	}
	return opts
}

// seederDialect maps a configured driver name to the seeder SQL dialect.
func seederDialect(driver string) seeder.Dialect {
	switch driver {
	case "mysql":
		return seeder.DialectMySQL
	case "sqlite", "sqlite3":
		return seeder.DialectSQLite
	default:
		return seeder.DialectPostgres
	}
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
