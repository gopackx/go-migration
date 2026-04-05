# go-migration + Echo Example

This example shows how to use **go-migration** with the [Echo](https://echo.labstack.com/) web framework.

## Key Point

go-migration works with any `*sql.DB` — it has **zero Echo-specific dependencies or adapters**. You open a database connection the standard way and pass it to `migrator.New()`.

## Setup

1. Create a new Go project and add both dependencies:

```bash
go mod init myapp
go get github.com/gopackx/go-migration
go get github.com/labstack/echo/v4
go get github.com/lib/pq
```

2. Copy `main.go` into your project and uncomment the Echo imports and route setup.

3. Start a PostgreSQL database (or adjust the DSN for MySQL/SQLite).

4. Run:

```bash
go run main.go
```

## What This Demonstrates

- Opening a standard `*sql.DB` connection
- Creating a `migrator.New(db)` instance with that connection
- Registering and running migrations before the Echo server starts
- Echo handlers can then query the migrated tables directly via `db`
