//go:build ignore

// This example demonstrates using go-migration with Go's standard net/http.
// No third-party web framework is needed — go-migration works identically.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	// Import your database driver
	_ "github.com/lib/pq"

	// Import go-migration packages
	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
)

// CreateProductsTable is a sample migration.
type CreateProductsTable struct{}

func (m *CreateProductsTable) Up(s *schema.Builder) error {
	return s.Create("products", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("name", 255)
		bp.Text("description").Nullable()
		bp.Decimal("price", 10, 2)
		bp.Integer("stock").Default(0)
		bp.Timestamps()
	})
}

func (m *CreateProductsTable) Down(s *schema.Builder) error {
	return s.Drop("products")
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
	//    This is identical to framework-based usage — no adapters needed.
	m := migrator.New(db)
	_ = m.Register("20240401120000_create_products", &CreateProductsTable{})

	// 3. Run pending migrations.
	if err := m.Up(); err != nil {
		log.Fatal("migration failed:", err)
	}
	fmt.Println("Migrations applied successfully")

	// 4. Set up standard net/http handlers.
	http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, name, price FROM products")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type product struct {
			ID    int     `json:"id"`
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		}

		var products []product
		for rows.Next() {
			var p product
			if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			products = append(products, p)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	})

	fmt.Println("net/http server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
