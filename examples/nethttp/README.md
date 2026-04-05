# go-migration + Standard net/http Example

This example shows how to use **go-migration** with Go's built-in `net/http` package — no third-party web framework required.

## Key Point

go-migration depends only on `database/sql`. It works identically whether you use Gin, Echo, Fiber, or plain `net/http`. This example proves there is no framework coupling at all.

## Setup

1. Create a new Go project:

```bash
go mod init myapp
go get github.com/gopackx/go-migration
go get github.com/lib/pq
```

2. Copy `main.go` into your project.

3. Start a PostgreSQL database (or adjust the DSN for MySQL/SQLite).

4. Run:

```bash
go run main.go
```

5. Visit `http://localhost:8080/products` to see the JSON response.

## What This Demonstrates

- Opening a standard `*sql.DB` connection
- Creating a `migrator.New(db)` instance with that connection
- Registering and running migrations at application startup
- Using the migrated database directly in `http.HandleFunc` handlers
- go-migration behaves identically to framework-based usage
