//go:build ignore

// This example demonstrates using go-migration with the Echo web framework.
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
	// Import Echo from your project's go.mod
	// "github.com/labstack/echo/v4"
)

// CreatePostsTable is a sample migration.
type CreatePostsTable struct{}

func (m *CreatePostsTable) Up(s *schema.Builder) error {
	return s.Create("posts", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("title", 255)
		bp.Text("body")
		bp.BigInteger("author_id").Unsigned()
		bp.Timestamps()
	})
}

func (m *CreatePostsTable) Down(s *schema.Builder) error {
	return s.Drop("posts")
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
	//    go-migration has zero Echo-specific dependencies.
	m := migrator.New(db)
	_ = m.Register("20240201120000_create_posts", &CreatePostsTable{})

	// 3. Run pending migrations.
	if err := m.Up(); err != nil {
		log.Fatal("migration failed:", err)
	}
	fmt.Println("Migrations applied successfully")

	// 4. Set up Echo as usual — go-migration is already done.
	//
	// e := echo.New()
	// e.GET("/posts", func(c echo.Context) error {
	//     rows, _ := db.Query("SELECT id, title FROM posts")
	//     defer rows.Close()
	//     // ... scan and return JSON
	//     return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	// })
	// e.Start(":8080")

	_ = http.StatusOK // keep import used for the example skeleton
	fmt.Println("Echo server would start on :8080")
}
