# go-migration - Folder Structure

This is the actual layout of the repository (generated from `git ls-files`).
Test files (`*_test.go`, including rapid property tests named
`*_property_test.go`) live next to the code they test and are omitted from the
tree below unless noted.

```
go-migration/
│
├── .github/
│   └── workflows/
│       └── ci.yml                    # go vet, unit tests, integration tests (Postgres 16, MySQL 8, SQLite)
│
├── cmd/
│   └── migrator/                     # CLI entry point (blank-imports the DB drivers, calls app.Run)
│       ├── main.go
│       └── version.go                # Version / BuildDate, set via -ldflags
│
├── pkg/                              # Public API
│   ├── migrator/                     # Core migration engine
│   │   ├── migrator.go               # Migrator: Up, Rollback, Reset, Refresh, Fresh, Status, options
│   │   ├── lock.go                   # Cross-process migration lock (Lock, WithLockTimeout, WithoutLock)
│   │   ├── interfaces.go             # Migration and TransactionOption interfaces
│   │   ├── registry.go               # Migration registry (name validation, sorted order)
│   │   ├── auto_registry.go          # Global AutoRegister used by generated migrations' init()
│   │   ├── runner.go                 # Executes Up/Down in a transaction (or not), dry-run
│   │   ├── tracker.go                # Tracking table access (dialect-aware via Grammar)
│   │   ├── batch.go                  # Batch numbering and last-batch / last-N lookups
│   │   ├── hooks.go                  # Before/after migration hooks
│   │   ├── grammar_resolver.go       # Driver name -> Grammar (used by the CLI)
│   │   ├── errors.go                 # Sentinel errors (and aliases of database/seeder/grammar sentinels)
│   │   └── migration_error.go        # MigrationError (failing migration, SQL, cause)
│   │
│   ├── schema/                       # Schema builder
│   │   ├── schema.go                 # Builder: Create, Alter, Drop, Rename, HasTable, HasColumn, RawExec
│   │   ├── blueprint.go              # Blueprint: columns, indexes, foreign keys, alter commands
│   │   ├── column.go                 # Column types and modifiers
│   │   ├── index.go                  # Index definitions (.Named)
│   │   ├── foreign_key.go            # Foreign key definitions
│   │   ├── raw_expression.go         # Raw SQL default values (DefaultRaw)
│   │   ├── grammar.go                # Grammar interface
│   │   ├── qualified.go              # QuoteQualified / SplitQualified (schema-qualified tracking table)
│   │   ├── dry_run_executor.go       # Executor that prints SQL instead of running it
│   │   ├── recording_executor.go     # Executor wrapper that remembers the last SQL (for errors)
│   │   └── grammars/                 # Dialect implementations
│   │       ├── postgres.go
│   │       ├── mysql.go
│   │       └── sqlite.go
│   │
│   ├── seeder/                       # Seeding system
│   │   ├── interfaces.go             # Seeder, DependentSeeder, TaggedSeeder, RollbackableSeeder
│   │   ├── registry.go               # Seeder registry
│   │   ├── auto_registry.go          # Global AutoRegister used by generated seeders' init()
│   │   ├── runner.go                 # Dependency-ordered execution, tags, rollback, truncate
│   │   ├── batch_insert.go           # CreateMany / CreateManyWithDialect (chunked multi-row INSERT)
│   │   ├── errors.go                 # Sentinel errors
│   │   └── factory/                  # Generic factories
│   │       ├── factory.go            # Factory[T]: Make, MakeMany, states
│   │       ├── faker.go              # Faker interface
│   │       └── faker_impl.go         # Default Faker implementation
│   │
│   ├── database/                     # Connection management
│   │   ├── connection.go             # ConnectionConfig
│   │   ├── manager.go                # Named multi-connection manager
│   │   ├── pool.go                   # Pool settings helper
│   │   ├── transaction.go            # Transaction helper
│   │   ├── errors.go                 # Sentinel errors
│   │   └── drivers/                  # DSN builders / sql.Open wrappers
│   │       ├── driver.go             # Driver interface
│   │       ├── pool.go               # Pool settings helper
│   │       ├── postgres.go           # lib/pq ("postgres")
│   │       ├── mysql.go              # go-sql-driver/mysql ("mysql", adds parseTime=true by default)
│   │       └── sqlite.go             # modernc.org/sqlite ("sqlite", pure Go)
│   │
│   ├── config/                       # Configuration (JSON, deprecated YAML, env interpolation)
│   │   ├── config.go                 # Config structs, loading, defaults, validation
│   │   └── interpolate.go            # ${ENV_VAR} interpolation
│   │
│   └── cli/                          # Command-line interface (hand-written router on the flag package)
│       ├── app.go                    # App: router holder
│       ├── router.go                 # Command router / help output
│       ├── context.go                # Shared CLI context
│       ├── app/
│       │   └── app.go                # app.Run(): config loading, connections, auto-discovery, command wiring
│       └── commands/                 # One file per command
│           ├── context.go            # CommandContext passed to handlers (Migrator, Locker, ...)
│           ├── lock_flags.go         # --lock-timeout / --no-lock flags
│           ├── confirm.go            # Confirmation prompt for destructive commands
│           ├── init.go               # init
│           ├── migrate.go            # migrate
│           ├── migrate_status.go     # migrate:status (read-only; --check)
│           ├── migrate_rollback.go   # migrate:rollback
│           ├── migrate_reset.go      # migrate:reset
│           ├── migrate_refresh.go    # migrate:refresh
│           ├── migrate_fresh.go      # migrate:fresh
│           ├── migrate_install.go    # migrate:install (takes the migration lock)
│           ├── make_migration.go     # make:migration
│           ├── make_seeder.go        # make:seeder
│           ├── make_factory.go       # make:factory
│           ├── seed.go               # db:seed
│           ├── seed_rollback.go      # db:seed:rollback
│           └── seed_truncate.go      # db:seed:truncate
│
├── internal/                         # Private implementation
│   ├── generator/                    # Code generation for make:* and init
│   │   ├── generator.go              # Migration / seeder / factory generators
│   │   ├── init.go                   # Project scaffolding (init)
│   │   └── templates/                # Embedded (go:embed) templates — the only templates used
│   │       ├── migration.go.tmpl
│   │       ├── migration_create.go.tmpl
│   │       ├── migration_alter.go.tmpl
│   │       ├── seeder.go.tmpl
│   │       ├── factory.go.tmpl
│   │       ├── init_main.go.tmpl
│   │       ├── init_config.json.tmpl
│   │       └── init_config.yaml.tmpl
│   ├── logger/
│   │   └── logger.go                 # Leveled console/file logger
│   ├── scanner/
│   │   └── scanner.go                # Discovers migration/seeder files by name pattern
│   └── utils/
│       └── utils.go                  # String, timestamp and file helpers
│
├── tests/
│   ├── integration/                  # Integration tests (build tag "integration")
│   │   ├── backend_test.go           # Backends (SQLite always; Postgres/MySQL via DSN env vars) and helpers
│   │   ├── migrations_test.go        # Migrations used by the scenarios
│   │   ├── migrator_test.go          # Up/Status/Rollback/Reset/Refresh/Fresh, hooks, dry-run, transactions
│   │   ├── lock_test.go              # Concurrent migrators, lock timeout, read-only Status
│   │   ├── quoting_test.go           # Table/column/tracking-table names containing quote characters
│   │   ├── qualified_test.go         # Tracking table in another schema/database (Postgres, MySQL)
│   │   └── seeder_test.go            # Seeder dependency order, CreateManyWithDialect, Truncate
│   ├── e2e/                          # (placeholder, empty)
│   ├── fixtures/                     # (placeholder, empty)
│   ├── mocks/                        # (placeholder, empty)
│   ├── testutil/                     # (placeholder, empty)
│   └── unit/                         # (placeholder, empty — unit tests live next to the code)
│
├── examples/                         # Framework integration examples
│   ├── echo/      (main.go, README.md)
│   ├── fiber/     (main.go, README.md)
│   ├── gin/       (main.go, README.md)
│   └── nethttp/   (main.go, README.md)
│
├── .gitignore
├── .goreleaser.yml                   # Release builds (CGO_ENABLED=0)
├── FOLDER_STRUCTURE.md
├── LICENSE
├── Makefile                          # build, install, clean, test, test-integration
├── README.md
├── go.mod
└── go.sum
```

Some packages also contain an empty `.gitkeep` file.

## Directory roles

### `/cmd`
The CLI binary. It only blank-imports the database drivers (and, in a user's
project, the migrations/seeders packages) and calls `app.Run()` from
`pkg/cli/app`. No business logic lives here. `migrator init` generates an equivalent
`cmd/migrator/main.go` in a user's project.

### `/pkg`
The public, importable API: `migrator`, `schema` (+ `grammars`), `seeder`
(+ `factory`), `database` (+ `drivers`), `config` and `cli` (+ `app`,
`commands`).

Dependencies point from the CLI to the library, never the other way:
`pkg/cli/app` and `pkg/cli/commands` import `pkg/migrator`, while
`pkg/migrator` imports only `schema`, `grammars`, `seeder` and `database`.
Importing the migrator as a library therefore does not pull in the CLI, the
configuration loader, the code generator or the YAML parser
(`go list -deps ./pkg/migrator` shows none of them).

### `/internal`
Implementation details that other modules cannot import: the code generator
and its embedded templates, the logger, the file scanner and small utilities.
Generator templates live only in `internal/generator/templates` and are
compiled into the binary with `go:embed`.

### `/tests`
`tests/integration` holds the integration suite, which runs only with
`-tags integration` (see README, "Testing"). Unit and property tests are next
to the code in each package. The other subdirectories are empty placeholders.

### `/examples`
Minimal programs showing how to run migrations from Echo, Fiber, Gin and
net/http applications.

## File naming conventions

```
# Migration files (generated by make:migration; description is lower snake_case,
# RRRR is a random 4-digit suffix)
YYYY_MM_DD_HHMMSS_RRRR_description.go
2024_02_15_120000_4821_create_users_table.go
# Hand-written migrations may also use YYYYMMDDHHMMSS_description

# Seeder / factory files (generated by make:seeder / make:factory)
<name>_seeder.go
<name>_factory.go

# Tests
*_test.go                            # unit tests
*_property_test.go                   # property-based tests (pgregory.net/rapid)
tests/integration/*_test.go          # integration tests (//go:build integration)
```

Registered migration names must match
`^(\d{14}|\d{4}_\d{2}_\d{2}_\d{6}_\d{4})_[a-z][a-z0-9_]*$`.

## Import paths

```go
import "github.com/gopackx/go-migration/pkg/migrator"
import "github.com/gopackx/go-migration/pkg/schema"
import "github.com/gopackx/go-migration/pkg/schema/grammars"
import "github.com/gopackx/go-migration/pkg/seeder"
import "github.com/gopackx/go-migration/pkg/seeder/factory"
import "github.com/gopackx/go-migration/pkg/database"
import "github.com/gopackx/go-migration/pkg/database/drivers"
import "github.com/gopackx/go-migration/pkg/cli/app" // the CLI: app.Run()
```
