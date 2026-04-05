//go:build ignore

// This example demonstrates using go-migration with the Fiber web framework.
// It is not compiled as part of the main module — run it from your own project.
package main

import (
	"database/sql"
	"fmt"
	"log"

	// Import your database driver
	_ "github.com/lib/pq"

	// Import go-migration packages
	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
	// Import Fiber from your project's go.mod
	// "github.com/gofiber/fiber/v2"
)

// CreateOrdersTable is a sample migration.
type CreateOrdersTable struct{}

func (m *CreateOrdersTable) Up(s *schema.Builder) error {
	return s.Create("orders", func(bp *schema.Blueprint) {
		bp.ID()
		bp.BigInteger("user_id").Unsigned()
		bp.Decimal("total", 10, 2)
		bp.String("status", 50).Default("pending")
		bp.Timestamps()
	})
}

func (m *CreateOrdersTable) Down(s *schema.Builder) error {
	return s.Drop("orders")
}

func main() {
	// 1. Open a *sql.DB connection — standard database/sql.
	dsn := "host=localhost port=5432 user=postgres password=secret dbname=myapp sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Create a Migrator using the plain *sql.DB.
	//    go-migration has zero Fiber-specific dependencies.
	m := migrator.New(db)
	_ = m.Register("20240301120000_create_orders", &CreateOrdersTable{})

	// 3. Run pending migrations.
	if err := m.Up(); err != nil {
		log.Fatal("migration failed:", err)
	}
	fmt.Println("Migrations applied successfully")

	// 4. Set up Fiber as usual — go-migration is already done.
	//
	// app := fiber.New()
	// app.Get("/orders", func(c *fiber.Ctx) error {
	//     rows, _ := db.Query("SELECT id, total, status FROM orders")
	//     defer rows.Close()
	//     // ... scan and return JSON
	//     return c.JSON(map[string]string{"status": "ok"})
	// })
	// app.Listen(":8080")

	fmt.Println("Fiber server would start on :8080")
}
