//go:build ignore

// This example demonstrates using go-migration with the Gin web framework.
// It is not compiled as part of the main module — run it from your own project.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	// Import your database driver
	_ "github.com/lib/pq"

	// Import go-migration packages
	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
	// Import Gin from your project's go.mod
	// "github.com/gin-gonic/gin"
)

// CreateUsersTable is a sample migration.
type CreateUsersTable struct{}

func (m *CreateUsersTable) Up(s *schema.Builder) error {
	return s.Create("users", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("name", 255)
		bp.String("email", 255).Unique()
		bp.Timestamps()
	})
}

func (m *CreateUsersTable) Down(s *schema.Builder) error {
	return s.Drop("users")
}

func main() {
	// 1. Open a *sql.DB connection — this is standard database/sql, no framework magic.
	dsn := "host=localhost port=5432 user=postgres password=secret dbname=myapp sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Create a Migrator using the plain *sql.DB.
	//    go-migration has zero Gin-specific dependencies.
	m := migrator.New(db)
	_ = m.Register("20240101120000_create_users", &CreateUsersTable{})

	// 3. Run pending migrations.
	if err := m.Up(); err != nil {
		log.Fatal("migration failed:", err)
	}
	fmt.Println("Migrations applied successfully")

	// 4. Set up Gin as usual — go-migration is already done.
	//
	// r := gin.Default()
	// r.GET("/users", func(c *gin.Context) {
	//     rows, _ := db.Query("SELECT id, name, email FROM users")
	//     defer rows.Close()
	//     // ... scan and return JSON
	//     c.JSON(http.StatusOK, gin.H{"status": "ok"})
	// })
	// r.Run(":8080")

	_ = http.StatusOK // keep import used for the example skeleton
	fmt.Println("Gin server would start on :8080")
}
