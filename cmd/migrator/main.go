package main

import (
	"github.com/gopackx/go-migration/pkg/migrator"

	// Database drivers — blank imports register them with database/sql.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	// Blank-import your migrations and seeders packages here so their
	// init() functions call migrator.AutoRegister / seeder.AutoRegister:
	//   _ "myapp/migrations"
	//   _ "myapp/seeders"
)

func main() {
	migrator.Run()
}
