# go-migration + Fiber Example

This example shows how to use **go-migration** with the [Fiber](https://gofiber.io/) web framework.

## Key Point

go-migration works with any `*sql.DB` — it has **zero Fiber-specific dependencies or adapters**. You open a database connection the standard way and pass it to `migrator.New()`.

## Setup

1. Create a new Go project and add both dependencies:

```bash
go mod init myapp
go get github.com/gopackx/go-migration
go get github.com/gofiber/fiber/v2
go get github.com/lib/pq
```

2. Copy `main.go` into your project and uncomment the Fiber imports and route setup.

3. Start a PostgreSQL database (or adjust the DSN for MySQL/SQLite).

4. Run:

```bash
go run main.go
```

## What This Demonstrates

- Opening a standard `*sql.DB` connection
- Creating a `migrator.New(db)` instance with that connection
- Registering and running migrations before the Fiber server starts
- Fiber handlers can then query the migrated tables directly via `db`
