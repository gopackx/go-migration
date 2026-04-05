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
    bp.Foreign("author_id").References("id").On("users").OnDelete("CASCADE")
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

`.Nullable()`, `.Default(value)`, `.Primary()`, `.Unique()`, `.Unsigned()`, `.AutoIncrement()`

## Migrator API

```go
m := migrator.New(db,
    migrator.WithGrammar(&grammars.PostgresGrammar{}),
    migrator.WithTableName("migrations"),  // default: "migrations"
    migrator.WithLogger(myLogger),
)

m.Register("20260101120000_create_users", &CreateUsersTable{})

m.Up()              // Run all pending migrations
m.Rollback(0)       // Rollback last batch
m.Rollback(3)       // Rollback last 3 migrations
m.Reset()           // Rollback all migrations
m.Refresh()         // Reset + Up
m.Fresh()           // Drop all tables + Up (requires grammar)
m.Status()          // []MigrationStatus
```

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

mgr := database.NewManager()
mgr.RegisterDriver("postgres", &drivers.PostgresDriver{})
mgr.RegisterDriver("mysql", &drivers.MySQLDriver{})

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

The package includes a CLI built with Cobra. Build it from `cmd/migrator/`:

```bash
go build -o migrator ./cmd/migrator
```

Available commands:

| Command | Description |
|---|---|
| `migrate` | Run all pending migrations |
| `migrate:rollback` | Rollback last batch (use `--step N` for N migrations) |
| `migrate:reset` | Rollback all migrations |
| `migrate:refresh` | Reset + migrate up |
| `migrate:fresh` | Drop all tables + migrate up |
| `migrate:status` | Show migration status |
| `migrate:install` | Create the migration tracking table |
| `make:migration` | Generate a migration file (`--create` or `--table` flags) |
| `make:seeder` | Generate a seeder file |
| `db:seed` | Run seeders (`--class` flag for specific seeder) |

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

Supports YAML, JSON, or environment variables:

```yaml
# config.yaml
default: primary
migration_table: migrations
migration_dir: migrations
seeder_dir: seeders
log_level: info
log_output: console

connections:
  primary:
    driver: postgres
    host: localhost
    port: 5432
    database: myapp
    username: user
    password: pass
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
```

## Framework Integration

go-migration works with any Go framework — it only depends on `database/sql`. See the [examples/](examples/) directory:

- [Gin](examples/gin/) — pass `*sql.DB` from your Gin setup
- [Echo](examples/echo/) — pass `*sql.DB` from your Echo setup
- [Fiber](examples/fiber/) — pass `*sql.DB` from your Fiber setup
- [net/http](examples/nethttp/) — standard library, no framework

The pattern is always the same: open a `*sql.DB`, create a `migrator.New(db)`, register migrations, call `m.Up()`.

## Supported Databases

- PostgreSQL
- MySQL
- SQLite

## Testing

```bash
go test ./...
```

The project uses property-based testing with [pgregory.net/rapid](https://pkg.go.dev/pgregory.net/rapid) alongside standard unit tests with [testify](https://github.com/stretchr/testify).

## License

MIT — see [LICENSE](LICENSE).

## Author

Andrian Prasetya — [@andrianprasetya](https://github.com/andrianprasetya)
