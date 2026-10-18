# go-migration

[![Go Reference](https://pkg.go.dev/badge/github.com/gopackx/go-migration.svg)](https://pkg.go.dev/github.com/gopackx/go-migration)
[![Go Report Card](https://goreportcard.com/badge/github.com/gopackx/go-migration)](https://goreportcard.com/report/github.com/gopackx/go-migration)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Laravel-inspired database migration and seeding system for Go. Struct-based migrations, fluent schema builder, factory-based seeders with faker, batch tracking, hooks, and multi-database support — works with any framework or standalone.

## Features

- Struct-based migrations with `Up()` / `Down()` methods — type-safe, no raw SQL files
- Fluent schema builder for tables, columns, indexes, and foreign keys
- Per-migration transactions with opt-out support
- Batch tracking and granular rollback (by batch or step count)
- Before/after migration hooks
- Seeder system with dependency resolution and circular dependency detection
- Generic factory pattern with faker for realistic test data
- Multi-database connection management with pooling
- CLI with Laravel-style commands (`migrate`, `migrate:rollback`, `make:migration`, `db:seed`, etc.)
- Grammars for PostgreSQL, MySQL, and SQLite
- Framework-agnostic — depends only on `database/sql`

## Installation

```bash
go get github.com/gopackx/go-migration
```

## Quick Start

### Define a migration

```go
package main

import (
    "github.com/gopackx/go-migration/pkg/schema"
)

type CreateUsersTable struct{}

func (m *CreateUsersTable) Up(s *schema.Builder) error {
    return s.Create("users", func(bp *schema.Blueprint) {
        bp.ID()
        bp.String("name", 255)
        bp.String("email", 255).Unique()
        bp.Boolean("active").Default(true)
        bp.Timestamps()
    })
}

func (m *CreateUsersTable) Down(s *schema.Builder) error {
    return s.Drop("users")
}
```

### Register and run migrations

```go
package main

import (
    "database/sql"
    "log"

    _ "github.com/lib/pq"
    "github.com/gopackx/go-migration/pkg/migrator"
    "github.com/gopackx/go-migration/pkg/schema/grammars"
)

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    m := migrator.New(db, migrator.WithGrammar(&grammars.PostgresGrammar{}))

    // Register migrations
    m.Register("20260101120000_create_users", &CreateUsersTable{})

    // Run all pending migrations
    if err := m.Up(); err != nil {
        log.Fatal(err)
    }
}
```

## Schema Builder

The schema builder provides a fluent, database-agnostic API for defining tables:

```go
// Create table
s.Create("posts", func(bp *schema.Blueprint) {
    bp.ID()
    bp.String("title", 255)
    bp.Text("body").Nullable()
    bp.BigInteger("author_id").Unsigned()
    bp.Decimal("rating", 3, 2).Default(0)
    bp.Boolean("published").Default(false)
    bp.JSON("metadata").Nullable()
    bp.Timestamps()
    bp.SoftDeletes()

    // Indexes
    bp.Index("title")
    bp.UniqueIndex("title", "author_id")

    // Foreign keys
    bp.Foreign("author_id").References("id").On("users").OnDeleteAction("CASCADE")
})

// Alter table
s.Alter("posts", func(bp *schema.Blueprint) {
    bp.String("slug", 255).Unique()
    bp.DropColumn("metadata")
})

// Other operations
s.Drop("posts")
s.DropIfExists("posts")
s.Rename("posts", "articles")
s.HasTable("posts")       // bool, error
s.HasColumn("posts", "id") // bool, error
```

### Supported column types

`ID`, `String`, `Text`, `Integer`, `BigInteger`, `Boolean`, `Timestamp`, `Date`, `Decimal`, `Float`, `UUID`, `JSON`, `Binary`

### Column modifiers

`.Nullable()`, `.Default(value)`, `.DefaultRaw(expression)`, `.Primary()`, `.Unique()`, `.Unsigned()`, `.AutoIncrement()`

```go
// Default(value) quotes the value as a literal; DefaultRaw(expression)
// injects a SQL expression verbatim (no quoting) for DB-evaluated defaults.
bp.Timestamp("created_at").DefaultRaw("CURRENT_TIMESTAMP")
bp.UUID("id").DefaultRaw("gen_random_uuid()")
```

> `.Unsigned()` is only emitted by MySQL (`INT UNSIGNED`). PostgreSQL and SQLite
> have no unsigned types, so the modifier is silently ignored on those grammars —
> add a `CHECK (col >= 0)` constraint yourself if you need it.

### Indexes

```go
bp.Index("title")                       // idx_posts_title
bp.UniqueIndex("title", "author_id")    // uniq_posts_title_author_id
bp.FulltextIndex("title", "body")       // ft_posts_title_body (MySQL / Postgres GIN)
bp.SpatialIndex("location")             // sp_posts_location  (MySQL / Postgres GIST)

// Override the auto-generated name with .Named(...)
bp.Index("title").Named("idx_posts_title_custom")
```

Index names are generated automatically (`<prefix>_<table>_<columns>`); use
`.Named("...")` to set your own. Fulltext and spatial indexes are unsupported on
SQLite and return an error there.

## Migrator API

```go
m := migrator.New(db,
    migrator.WithGrammar(&grammars.PostgresGrammar{}),
    migrator.WithTableName("migrations"),  // default: "migrations"
    migrator.WithLogger(myLogger),
    migrator.WithLockTimeout(time.Minute),  // default: 30s; see "Concurrency and locking"
)

m.Register("20260101120000_create_users", &CreateUsersTable{})

m.Up()              // Run all pending migrations
m.Rollback(0)       // Rollback last batch
m.Rollback(3)       // Rollback last 3 migrations
m.Reset()           // Rollback all migrations
m.Refresh()         // Reset + Up
m.Fresh()           // Drop all tables + Up (requires grammar)
m.Status()          // []MigrationStatus (read-only, takes no lock)
```

`Up`, `Rollback`, `Reset`, `Refresh` and `Fresh` hold a cross-process lock while
they run, so several processes can call them at the same time safely. See
[Concurrency and locking](#concurrency-and-locking).

### Transaction opt-out

By default every migration runs in a transaction. To opt out, implement `TransactionOption`:

```go
type LargeDataMigration struct{}

func (m *LargeDataMigration) DisableTransaction() bool { return true }
func (m *LargeDataMigration) Up(s *schema.Builder) error   { /* ... */ return nil }
func (m *LargeDataMigration) Down(s *schema.Builder) error { /* ... */ return nil }
```

### Hooks

```go
m.BeforeMigrate(func(name, direction string) error {
    log.Printf("About to run %s (%s)", name, direction)
    return nil
})

m.AfterMigrate(func(name, direction string, duration time.Duration) error {
    log.Printf("Completed %s (%s) in %v", name, direction, duration)
    return nil
})
```

### Progress reporting

Pass `WithProgress` to receive a structured event after each migration runs —
handy for progress bars or logging on long batches:

```go
m := migrator.New(db,
    migrator.WithGrammar(&grammars.PostgresGrammar{}),
    migrator.WithProgress(func(e migrator.ProgressEvent) {
        // Index is 0-based; +1 for a human-friendly "N of Total".
        log.Printf("[%d/%d] %s %s (%s)", e.Index+1, e.Total, e.Direction, e.Name, e.Duration)
    }),
)
```

`ProgressEvent` carries `Name`, `Index`, `Total`, `Duration`, and `Direction`
(`"up"` or `"down"`). Events fire for `Up()`, `Rollback()`, and `Reset()`.

### Error handling

The migrator exposes typed sentinel errors so you can branch on failure modes
with `errors.Is` instead of matching strings:

```go
import "errors"

if err := m.Up(); err != nil {
    switch {
    case errors.Is(err, migrator.ErrMigrationNotFound):
        log.Println("a referenced migration is not registered")
    case errors.Is(err, migrator.ErrDuplicateMigration):
        log.Println("two migrations share a name")
    case errors.Is(err, migrator.ErrTransactionFailed):
        log.Printf("migration transaction failed: %v", err)
    default:
        log.Fatalf("unexpected: %v", err)
    }
}
```

Exported sentinels include `ErrMigrationNotFound`, `ErrDuplicateMigration`,
`ErrInvalidMigrationName`, `ErrTransactionFailed`, `ErrTrackingTable`,
`ErrLockTimeout`, `ErrConnectionFailed`, `ErrSeederNotFound`,
`ErrCircularDependency`, and more.
For richer context (failing SQL, position), use `errors.As` with
`*migrator.MigrationError`.

## Seeder System

```go
import "github.com/gopackx/go-migration/pkg/seeder"

type UserSeeder struct{}

func (s *UserSeeder) Run(db *sql.DB) error {
    _, err := db.Exec("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")
    return err
}

// Optional: declare dependencies
type PostSeeder struct{}

func (s *PostSeeder) Run(db *sql.DB) error { /* ... */ return nil }
func (s *PostSeeder) DependsOn() []string  { return []string{"UserSeeder"} }
```

```go
reg := seeder.NewRegistry()
reg.Register("UserSeeder", &UserSeeder{})
reg.Register("PostSeeder", &PostSeeder{})

runner := seeder.NewRunner(reg, db, nil)
runner.RunAll()           // Runs all seeders in dependency order
runner.Run("PostSeeder")  // Runs PostSeeder and its dependencies
```

### Factory + Faker

```go
import "github.com/gopackx/go-migration/pkg/seeder/factory"

type User struct {
    Name  string
    Email string
    Age   int
}

f := factory.NewFactory(func(fake factory.Faker) User {
    return User{
        Name:  fake.Name(),
        Email: fake.Email(),
        Age:   fake.IntBetween(18, 65),
    }
})

user := f.Make()          // Single instance
users := f.MakeMany(10)   // Slice of 10

// Named states
f.State("admin", func(fake factory.Faker, base User) User {
    base.Name = "Admin " + base.Name
    return base
})
admin := f.WithState("admin").Make()
```

## Multi-Database Connections

```go
import "github.com/gopackx/go-migration/pkg/database"
import "github.com/gopackx/go-migration/pkg/database/drivers"

// The database/sql drivers must be blank-imported somewhere in your program:
//   _ "github.com/lib/pq"
//   _ "github.com/go-sql-driver/mysql"
//   _ "modernc.org/sqlite"

mgr := database.NewManager()
mgr.RegisterDriver("postgres", &drivers.PostgresDriver{})
mgr.RegisterDriver("mysql", &drivers.MySQLDriver{})
mgr.RegisterDriver("sqlite", &drivers.SQLiteDriver{})

mgr.AddConnection("primary", database.ConnectionConfig{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    Database: "myapp",
    Username: "user",
    Password: "pass",
})

mgr.AddConnection("analytics", database.ConnectionConfig{
    Driver:   "mysql",
    Host:     "localhost",
    Port:     3306,
    Database: "analytics",
    Username: "user",
    Password: "pass",
})

mgr.SetDefault("primary")

db, _ := mgr.Connection("primary")
analyticsDB, _ := mgr.Connection("analytics")
defer mgr.Close()
```

## CLI

The package includes a CLI. It has no third-party CLI dependency: commands are
dispatched by a small hand-written router in `pkg/cli` on top of the standard
`flag` package. Build it from `cmd/migrator/`:

```bash
go build -o migrator ./cmd/migrator
```

In your own project the binary is a few lines (this is what `migrator init`
generates in `cmd/migrator/main.go`): blank-import the database drivers and
your migrations/seeders packages, and call `app.Run()`:

```go
package main

import (
    "github.com/gopackx/go-migration/pkg/cli/app"

    _ "github.com/go-sql-driver/mysql"
    _ "github.com/lib/pq"
    _ "modernc.org/sqlite"

    _ "myapp/database/migrations"
    _ "myapp/database/seeders"
)

func main() {
    app.Run()
}
```

The CLI lives in `pkg/cli/app`, separate from the library: importing
`pkg/migrator` does not pull in the CLI, the config loader or the YAML parser.

Available commands:

| Command | Description |
|---|---|
| `init` | Scaffold a new project (`migration.json`, `cmd/migrator/main.go`, directories) |
| `migrate` | Run all pending migrations (`--dry-run`) |
| `migrate:rollback` | Rollback last batch (use `--step N` for N migrations; `--dry-run`) |
| `migrate:reset` | Rollback all migrations (`--force`, `--dry-run`) |
| `migrate:refresh` | Reset + migrate up (`--dry-run`) |
| `migrate:fresh` | Drop all tables + migrate up (`--force`, `--dry-run`) |
| `migrate:status` | Show migration status; read-only (`--check` exits 1 if any migration is pending) |
| `migrate:install` | Create the migration tracking table (under the migration lock) |
| `make:migration` | Generate a migration file (`--create` or `--table` flags) |
| `make:seeder` | Generate a seeder file |
| `make:factory` | Generate a factory file |
| `db:seed` | Run seeders (`--class` for a specific seeder, `--tag` to filter) |
| `db:seed:rollback` | Roll back one seeder (`--class`, seeder must implement `Rollback`) |
| `db:seed:truncate` | Delete all rows from a table (`--table`) |

The commands that change the database (`migrate`, `migrate:rollback`,
`migrate:reset`, `migrate:refresh`, `migrate:fresh`, `migrate:install`,
`db:seed`, `db:seed:rollback`, `db:seed:truncate`) also accept
`--lock-timeout <duration>` (default `30s`) and `--no-lock`; see
[Concurrency and locking](#concurrency-and-locking).

`migrate:status` never writes: if the tracking table does not exist it reports
every migration as pending without creating it, and it does not wait for the
migration lock. With `--check` it prints the status table as usual and then, if
anything is pending, prints `N migration(s) pending` to stderr and exits with
status 1, so it can be used as a readiness gate (e.g. a Kubernetes
initContainer that waits for the migration Job):

```bash
until ./migrator migrate:status --check; do sleep 5; done
```

Names passed to `make:migration`, `make:seeder` and `make:factory` must be lower
snake_case (`^[a-z][a-z0-9_]*$`, e.g. `create_orders_table`); the generator
derives the timestamped file name and the Go type name from it.

`--dry-run` needs the tracking table to exist; run `migrate:install` once
first (see [Database-specific behaviour](#database-specific-behaviour)).

```bash
# Run migrations
./migrator migrate --config config.yaml

# Generate a migration with create-table scaffolding
./migrator make:migration create_orders --create orders

# Generate a migration with alter-table scaffolding
./migrator make:migration add_status_to_orders --table orders

# Rollback last 2 migrations
./migrator migrate:rollback --step 2

# Seed the database
./migrator db:seed
./migrator db:seed --class UserSeeder
```

## Configuration

Configuration is loaded from JSON, environment variables, or — for backward
compatibility — YAML. The CLI looks for `migration.json` by default (use
`--config` to point at another file). The format is detected from the file
extension.

```json
// migration.json
{
  "default": "primary",
  "migration_table": "migrations",
  "migration_dir": "database/migrations",
  "seeder_dir": "database/seeders",
  "factory_dir": "database/factories",
  "log_level": "info",
  "log_output": "console",
  "connections": {
    "primary": {
      "driver": "postgres",
      "host": "localhost",
      "port": 5432,
      "database": "myapp",
      "username": "user",
      "password": "pass",
      "max_open_conns": 25,
      "max_idle_conns": 5,
      "conn_max_lifetime": "5m"
    }
  }
}
```

> **Note:** YAML config (`.yaml`/`.yml`) is still accepted but **deprecated** —
> loading a YAML file prints a deprecation warning to stderr. Migrate to JSON.
> Run `migrator init` to scaffold a project with a ready-to-edit `migration.json`.

Values support `${ENV_VAR}` interpolation, so secrets can come from the
environment:

```json
"password": "${DB_PASSWORD}"
```

## Framework Integration

go-migration works with any Go framework — it only depends on `database/sql`. See the [examples/](examples/) directory:

- [Gin](examples/gin/) — pass `*sql.DB` from your Gin setup
- [Echo](examples/echo/) — pass `*sql.DB` from your Echo setup
- [Fiber](examples/fiber/) — pass `*sql.DB` from your Fiber setup
- [net/http](examples/nethttp/) — standard library, no framework

The pattern is always the same: open a `*sql.DB`, create a `migrator.New(db)`, register migrations, call `m.Up()`.

## Supported Databases

| Database   | `driver` in config     | Grammar                       | `database/sql` driver (blank import)  |
|------------|------------------------|-------------------------------|---------------------------------------|
| PostgreSQL | `postgres`             | `grammars.NewPostgresGrammar()` | `_ "github.com/lib/pq"`             |
| MySQL      | `mysql`                | `grammars.NewMySQLGrammar()`    | `_ "github.com/go-sql-driver/mysql"` |
| SQLite     | `sqlite` or `sqlite3`  | `grammars.NewSQLiteGrammar()`   | `_ "modernc.org/sqlite"`            |

- **SQLite** uses [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), a
  pure-Go driver registered as `"sqlite"`, so the CLI builds with
  `CGO_ENABLED=0`. In the config both `"sqlite"` and `"sqlite3"` are accepted as
  the driver name; `host`, `port`, `username` and `password` are not required —
  `database` is the file path (or `:memory:`), and `options` are appended to the
  DSN as query parameters, e.g. `"options": {"_pragma": "foreign_keys(1)"}`.
  With `sql.Open` directly, use `sql.Open("sqlite", "app.db?_pragma=foreign_keys(1)")`.
  SQLite allows one writer at a time, so `db.SetMaxOpenConns(1)` is recommended.
- **MySQL**: the driver adds `parseTime=true` to the DSN unless you set
  `parseTime` yourself in `options`, so `DATETIME`/`TIMESTAMP` columns scan into
  `time.Time`.

### Database-specific behaviour

**`Fresh()` / `migrate:fresh`** drops every table and then runs all migrations
in batch 1:

- PostgreSQL: `DROP SCHEMA public CASCADE` followed by `CREATE SCHEMA public`.
  Everything else in the `public` schema (views, sequences, functions, types)
  is dropped too, and privileges granted on the schema are reset.
- MySQL: lists the base tables of the current database and drops each one with
  `SET FOREIGN_KEY_CHECKS = 0` on the same connection.
- SQLite: lists the tables in `sqlite_master` and drops each one with
  `PRAGMA foreign_keys = OFF` on the same connection.

The connection used for the MySQL/SQLite drops is closed afterwards rather than
returned to the pool, so the disabled foreign key checks never leak into later
queries.

Only the default schema (PostgreSQL `public`, the current MySQL database, the
SQLite main database) is cleared. If the tracking table is schema-qualified
(`WithTableName("app.migrations")`), Fresh also drops that table explicitly, so
every migration runs again in batch 1; other tables in that schema are left
alone.

**Transactions.** Each migration runs in a transaction (unless it implements
`TransactionOption`), and its row in the tracking table is written on that same
transaction, so a failed migration leaves no tracking row behind. On PostgreSQL
and SQLite, DDL is transactional too, so a failed migration is rolled back
completely. **MySQL commits DDL implicitly** (`CREATE`, `ALTER`, `DROP`, …): only
DML (`INSERT`/`UPDATE`/`DELETE`) is rolled back with the tracking row, and a
migration that fails after some DDL leaves that DDL applied. Keep MySQL
migrations small (ideally one DDL statement each) so a failure is easy to repair.

**Identifier quoting.** Table, column, index and tracking-table names are
always quoted, with the delimiter escaped by doubling it: `"a""b"` on
PostgreSQL and SQLite, `` `a``b` `` on MySQL. Names compared as values in the
`HasTable`/`HasColumn` catalog queries are single-quoted with `'` doubled (and,
on MySQL, `\` doubled). Any name the database accepts can therefore be used,
including ones containing quote characters. Because names are quoted, they are
case-sensitive on PostgreSQL. In schema-builder names (`Create`, `Alter`,
`HasTable`, ...) a dot is part of the name. The tracking table name
(`migration_table` / `WithTableName`) is the exception: it may be
schema-qualified, and each dot-separated part is quoted separately —
`app.migrations` becomes `"app"."migrations"` on PostgreSQL (schema `app`),
`` `app`.`migrations` `` on MySQL (database `app`) and `"app"."migrations"` on
SQLite (attached database `app`). The schema must already exist.

**Dry-run** (`WithDryRun(w)`, `--dry-run`) prints the SQL instead of executing
it and never writes to the database — not even the tracking table. It still
reads the tracking table to find pending migrations, so the table must already
exist: run `migrate:install` (or `migrator.NewTrackerWithGrammar(db, table,
grammar).EnsureTable()`) first; otherwise dry-run fails with
`ErrTrackingTable`. `HasTable`/`HasColumn` report `false` in dry-run mode.

## Concurrency and locking

Running migrations from several processes at once — for example a Kubernetes
Job whose pods start together, or every replica of an app migrating on boot —
is safe: `Up`, `Rollback`, `Reset`, `Refresh` (one lock across both phases) and
`Fresh` (locked before anything is dropped) take a cross-process lock before
touching the tracking table, and compute the pending/applied migrations only
once they hold it. A second process waits, then finds nothing left to do. The
CLI's `migrate:install`, `db:seed`, `db:seed:rollback` and `db:seed:truncate`
take the same lock, so concurrent installs do not race on `CREATE TABLE IF NOT
EXISTS` (which is not atomic on PostgreSQL) and seeders do not run concurrently
with each other or with migrations.
Dry-run takes the lock too; `Status` / `migrate:status` does not.

How the lock works per database:

- **PostgreSQL**: a session-level advisory lock,
  `pg_try_advisory_lock(hashtext('go-migration:<table>'))`. Advisory locks are
  scoped to the current database.
- **MySQL**: a named lock, `GET_LOCK(CONCAT(DATABASE(), ':', 'go-migration:<table>'), 0)`.
  `GET_LOCK` names are server-wide, so the name includes the database.
- **SQLite**: no explicit lock; SQLite's database file lock already serialises
  writers.

`<table>` is the tracking table name, so migration sets that use different
tracking tables (`WithTableName`) do not block each other.

The lock is taken without blocking and retried with a short backoff (100 ms,
doubling up to 1 s) until it is acquired or the lock timeout elapses; the
operation then fails with `migrator.ErrLockTimeout` (`could not acquire
migration lock within 30s ...`) without having changed anything.

- Library: `migrator.WithLockTimeout(d)` (default 30s; `0` means a single
  attempt) and `migrator.WithoutLock()` to disable locking. `m.Lock(ctx)`
  acquires the lock and returns a `release` function, if you need to
  serialise your own work with migrations.
- CLI: `--lock-timeout 2m` and `--no-lock` on the commands listed above.

The lock is held on a dedicated connection taken from the pool for the
duration of the operation, so with locking enabled on PostgreSQL/MySQL the
pool needs **at least two connections** (`MaxOpenConns >= 2`, or `0` for
unlimited): one holds the lock, the other runs the migrations. A pool limited
to one connection is rejected with an error suggesting a larger pool or
`WithoutLock` / `--no-lock`. SQLite takes no lock, so `SetMaxOpenConns(1)` is
fine there. If releasing the lock fails, its connection is closed instead of
being returned to the pool, which makes the server drop the lock.

Session-level locks need a real session: behind a transaction-pooling proxy
(e.g. PgBouncer in `transaction` mode) connect the migrator directly to the
database or use session pooling.

## Upgrading to v1.2.0

### CLI entry point and package layout

- `migrator.Run()` is gone; use `app.Run()` from
  `github.com/gopackx/go-migration/pkg/cli/app`. In your `cmd/migrator/main.go`
  replace the import `github.com/gopackx/go-migration/pkg/migrator` with
  `github.com/gopackx/go-migration/pkg/cli/app` and the call `migrator.Run()`
  with `app.Run()`. The CLI wiring moved out of `pkg/migrator` so that the
  library no longer depends on the CLI, the config loader or the YAML parser.
- `migrator.ErrConfigValidation` was removed (the migrator never returned it);
  use `config.ErrConfigValidation`.
- If you build on `pkg/cli/commands` directly: `commands.MigratorRunner` and
  `commands.MigrationStatusInfo` were removed. `CommandContext.Migrator` (and
  `cli.CLIContext.Migrator`) is now a `commands.Migrator`, which
  `*migrator.Migrator` satisfies directly (`Status()` returns
  `[]migrator.MigrationStatus`); no adapter is needed.

### Behaviour changes

- `migrate:install` now takes the migration lock and accepts `--lock-timeout`
  and `--no-lock`. On PostgreSQL/MySQL it therefore needs a pool of at least two
  connections (or `--no-lock`), like the other writing commands.
- The tracking table name is now quoted in every statement (it was previously
  interpolated as-is). On PostgreSQL an unquoted name was folded to lower case,
  so if your `migration_table` / `WithTableName` contains upper-case letters,
  rename the existing table to the exact configured name (or configure the
  lower-case name) before upgrading; otherwise a new, empty tracking table is
  created. A schema-qualified name such as `app.migrations` keeps working: it
  is quoted per part (`"app"."migrations"`, `` `app`.`migrations` ``).
  `migrate:fresh` / `Fresh()` now also drops a schema-qualified tracking table
  (it previously survived, so nothing was re-applied after the drop).

### Custom `Grammar` implementations

If you implement `schema.Grammar` yourself, the interface changed:

- `CompileCreate(bp) ([]string, error)` now returns a slice — the `CREATE TABLE`
  statement followed by any separate `CREATE INDEX` statements, each executed
  individually.
- `CompileDropAllTables(tables []string) []string` receives the table list
  produced by `CompileListTables()` and returns the statements to run, in order,
  on one connection (it was previously `CompileDropAllTables() string`).
- `CompileCreate` was previously `CompileCreate(bp) (string, error)`.
- New methods:
  - `CompileListTables() string` — a query returning one row per table name.
  - `CompileTrackingTable(table string) string` — `CREATE TABLE IF NOT EXISTS`
    for the tracking table (`id`, `migration` unique, `batch`, `created_at`),
    with the possibly schema-qualified table name quoted by
    `schema.QuoteQualified(g, table)`.
  - `QuoteIdentifier(name string) string` — the name quoted as one identifier,
    with the delimiter escaped by doubling it (`"a""b"`, `` `a``b` ``).
    `schema.QuoteQualified` applies it to each dot-separated part of the
    tracking table name, which the tracker uses in every query.
  - `CompileHasTableIn(schema, table string) string` — like `CompileHasTable`,
    but looks in the given schema (PostgreSQL schema, MySQL database, SQLite
    attached database); used by `Tracker.TableExists` for a schema-qualified
    tracking table.
  - `Placeholder(index int) string` — the bind parameter for the 1-based index
    (`$1` on PostgreSQL, `?` on MySQL/SQLite).
  - `CompileTryLock(key string) string` and `CompileUnlock(key string) string` —
    queries that take the lock key as their only bind parameter
    (`Placeholder(1)`) and return one boolean-ish column (1/true = acquired /
    released; 0, false or NULL = not). The lock must be session-level, not
    transaction-level. Return `""` from both if the database needs no
    migration lock (as SQLite does). See
    [Concurrency and locking](#concurrency-and-locking).

The built-in grammars already implement all of these. The tracker now uses the
grammar passed with `WithGrammar` (it falls back to PostgreSQL syntax when no
grammar is given), so always pass `WithGrammar` for MySQL and SQLite.

## Testing

The tests are split in two:

**Unit tests** (the default) need no database. They use the standard `testing`
package, [go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) for SQL
expectations, and property-based tests with
[pgregory.net/rapid](https://pkg.go.dev/pgregory.net/rapid):

```bash
go test ./...        # or: make test
```

**Integration tests** live in `tests/integration` behind the `integration` build
tag and run the migrator and seeder against real databases through the public
API (Up/Rollback/Reset/Refresh/Fresh, hooks, dry-run, transaction opt-out,
schema builder, seeders, read-only status, and concurrent migrators competing
for the migration lock on PostgreSQL/MySQL). SQLite always runs, on a fresh temporary file per
test. PostgreSQL and MySQL run only when their DSN is set, and are skipped
otherwise:

```bash
export GOMIGRATE_TEST_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/gomigrate_test?sslmode=disable"
export GOMIGRATE_TEST_MYSQL_DSN="root:root@tcp(127.0.0.1:3306)/gomigrate_test?parseTime=true"

go test -count=1 -tags integration ./tests/integration/...   # or: make test-integration
```

> **Warning:** the integration tests wipe the PostgreSQL/MySQL databases they
> are pointed at (they call `Fresh()` before every test). Use throwaway
> databases, e.g.:
>
> ```bash
> docker run -d --name gm-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=gomigrate_test -p 5432:5432 postgres:16
> docker run -d --name gm-mysql -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=gomigrate_test -p 3306:3306 mysql:8
> ```

CI (`.github/workflows/ci.yml`) runs `go vet`, the unit tests, and the
integration tests against PostgreSQL 16 and MySQL 8 service containers.

## License

MIT — see [LICENSE](LICENSE).

## Author

Andrian Prasetya — [@andrianprasetya](https://github.com/andrianprasetya)
