//go:build integration

// Package integration_test runs the migrator and seeder against real
// databases through the public API only.
//
// SQLite (modernc.org/sqlite, pure Go) always runs, using a fresh temporary
// database file per test. PostgreSQL and MySQL run when a DSN is provided:
//
//	GOMIGRATE_TEST_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/gomigrate_test?sslmode=disable"
//	GOMIGRATE_TEST_MYSQL_DSN="root:root@tcp(127.0.0.1:3306)/gomigrate_test?parseTime=true"
//
// WARNING: the PostgreSQL and MySQL databases are wiped (Fresh) before every
// test. Never point these variables at a database you care about.
//
//	go test -count=1 -tags integration ./tests/integration/...
package integration_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
	"github.com/gopackx/go-migration/pkg/seeder"
)

const (
	envPostgresDSN = "GOMIGRATE_TEST_POSTGRES_DSN"
	envMySQLDSN    = "GOMIGRATE_TEST_MYSQL_DSN"
)

// backend describes one database engine under test.
type backend struct {
	name    string
	grammar schema.Grammar
	dialect seeder.Dialect

	// transactionalDDL is true when DDL rolls back with the surrounding
	// transaction (PostgreSQL, SQLite). MySQL commits DDL implicitly.
	transactionalDDL bool

	// fkColumnType is a raw SQL column type compatible with the "id" column
	// produced by Blueprint.ID(), for hand-written foreign keys.
	fkColumnType string

	// nonTxUp / nonTxDown are statements used by the DisableTransaction
	// scenario. On PostgreSQL they cannot run inside a transaction block.
	nonTxUp   string
	nonTxDown string

	open func(t *testing.T) *sql.DB

	// openPeer opens an additional, independent connection pool to the same
	// database without wiping it, standing in for a second process. It is nil
	// for SQLite, which needs no migration lock.
	openPeer func(t *testing.T) *sql.DB
}

func backends() []backend {
	return []backend{
		{
			name:             "sqlite",
			grammar:          grammars.NewSQLiteGrammar(),
			dialect:          seeder.DialectSQLite,
			transactionalDDL: true,
			fkColumnType:     "INTEGER",
			nonTxUp:          `CREATE INDEX "idx_users_email_name" ON "users" ("email", "name")`,
			nonTxDown:        `DROP INDEX "idx_users_email_name"`,
			open:             openSQLite,
		},
		{
			name:             "postgres",
			grammar:          grammars.NewPostgresGrammar(),
			dialect:          seeder.DialectPostgres,
			transactionalDDL: true,
			fkColumnType:     "BIGINT",
			nonTxUp:          `CREATE INDEX CONCURRENTLY "idx_users_email_name" ON "users" ("email", "name")`,
			nonTxDown:        `DROP INDEX CONCURRENTLY "idx_users_email_name"`,
			open: func(t *testing.T) *sql.DB {
				return openServer(t, "postgres", envPostgresDSN, grammars.NewPostgresGrammar())
			},
			openPeer: func(t *testing.T) *sql.DB { return openPeer(t, "postgres", envPostgresDSN) },
		},
		{
			name:             "mysql",
			grammar:          grammars.NewMySQLGrammar(),
			dialect:          seeder.DialectMySQL,
			transactionalDDL: false,
			fkColumnType:     "BIGINT UNSIGNED",
			nonTxUp:          "CREATE INDEX `idx_users_email_name` ON `users` (`email`, `name`)",
			nonTxDown:        "DROP INDEX `idx_users_email_name` ON `users`",
			open: func(t *testing.T) *sql.DB {
				return openServer(t, "mysql", envMySQLDSN, grammars.NewMySQLGrammar())
			},
			openPeer: func(t *testing.T) *sql.DB { return openPeer(t, "mysql", envMySQLDSN) },
		},
	}
}

// openSQLite opens a new database file in a per-test temporary directory.
// Foreign keys are enabled on every connection through the DSN, and the pool
// is limited to one connection, the usual configuration for SQLite.
func openSQLite(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "integration.db")
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping sqlite: %v", err)
	}
	return db
}

// openServer connects to a PostgreSQL/MySQL server named by an environment
// variable and wipes it so that each test starts from an empty database.
func openServer(t *testing.T, driverName, env string, g schema.Grammar) *sql.DB {
	t.Helper()
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skipf("%s not set; skipping %s backend", env, driverName)
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatalf("open %s: %v", driverName, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping %s: %v", driverName, err)
	}

	// Fresh with nothing registered drops every table and leaves only an
	// empty tracking table, which is then dropped too.
	if err := migrator.New(db, migrator.WithGrammar(g)).Fresh(); err != nil {
		t.Fatalf("reset %s database: %v", driverName, err)
	}
	if _, err := db.Exec(g.CompileDropIfExists("migrations")); err != nil {
		t.Fatalf("drop tracking table: %v", err)
	}
	return db
}

// openPeer opens a second connection pool to the server named by env,
// without resetting the database.
func openPeer(t *testing.T, driverName, env string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driverName, os.Getenv(env))
	if err != nil {
		t.Fatalf("open peer %s: %v", driverName, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping peer %s: %v", driverName, err)
	}
	return db
}

// forEachBackend runs fn as a subtest for every backend, each with a fresh
// database.
func forEachBackend(t *testing.T, fn func(t *testing.T, b backend, db *sql.DB)) {
	t.Helper()
	for _, b := range backends() {
		b := b
		t.Run(b.name, func(t *testing.T) {
			db := b.open(t)
			fn(t, b, db)
		})
	}
}

// q rewrites "?" placeholders to "$N" for PostgreSQL.
func (b backend) q(query string) string {
	if b.name != "postgres" {
		return query
	}
	var sb strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			fmt.Fprintf(&sb, "$%d", n)
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func (b backend) newMigrator(db *sql.DB, opts ...migrator.Option) *migrator.Migrator {
	return migrator.New(db, append([]migrator.Option{migrator.WithGrammar(b.grammar)}, opts...)...)
}

// --- catalog helpers (independent of the library's own HasTable queries) ---

func tableExists(t *testing.T, b backend, db *sql.DB, table string) bool {
	t.Helper()
	var query string
	switch b.name {
	case "sqlite":
		query = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?"
	case "postgres":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?"
	case "mysql":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
	}
	var n int
	if err := db.QueryRow(b.q(query), table).Scan(&n); err != nil {
		t.Fatalf("tableExists(%s): %v", table, err)
	}
	return n > 0
}

func columnNames(t *testing.T, b backend, db *sql.DB, table string) []string {
	t.Helper()
	var query string
	switch b.name {
	case "sqlite":
		query = "SELECT name FROM pragma_table_info(?)"
	case "postgres":
		query = "SELECT column_name FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ?"
	case "mysql":
		query = "SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ?"
	}
	rows, err := db.Query(b.q(query), table)
	if err != nil {
		t.Fatalf("columnNames(%s): %v", table, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("columnNames(%s) scan: %v", table, err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("columnNames(%s) rows: %v", table, err)
	}
	sort.Strings(cols)
	return cols
}

func hasColumn(t *testing.T, b backend, db *sql.DB, table, column string) bool {
	t.Helper()
	for _, c := range columnNames(t, b, db, table) {
		if c == column {
			return true
		}
	}
	return false
}

func indexExists(t *testing.T, b backend, db *sql.DB, table, index string) bool {
	t.Helper()
	var query string
	switch b.name {
	case "sqlite":
		query = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND name = ?"
	case "postgres":
		query = "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND tablename = ? AND indexname = ?"
	case "mysql":
		query = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?"
	}
	var n int
	if err := db.QueryRow(b.q(query), table, index).Scan(&n); err != nil {
		t.Fatalf("indexExists(%s.%s): %v", table, index, err)
	}
	return n > 0
}

func rowCount(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func mustExec(t *testing.T, b backend, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(b.q(query), args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// trackRow is one row of the migration tracking table.
type trackRow struct {
	Name  string
	Batch int
}

// trackingRows reads the tracking table directly, ordered by name.
func trackingRows(t *testing.T, db *sql.DB, table string) []trackRow {
	t.Helper()
	rows, err := db.Query("SELECT migration, batch FROM " + table + " ORDER BY migration")
	if err != nil {
		t.Fatalf("read tracking table %s: %v", table, err)
	}
	defer rows.Close()
	var out []trackRow
	for rows.Next() {
		var r trackRow
		if err := rows.Scan(&r.Name, &r.Batch); err != nil {
			t.Fatalf("scan tracking row: %v", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("tracking rows: %v", err)
	}
	return out
}

func assertTracking(t *testing.T, db *sql.DB, want ...trackRow) {
	t.Helper()
	assertTrackingTable(t, db, "migrations", want...)
}

func assertTrackingTable(t *testing.T, db *sql.DB, table string, want ...trackRow) {
	t.Helper()
	got := trackingRows(t, db, table)
	if len(got) != len(want) {
		t.Fatalf("tracking table %s: got %v, want %v", table, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tracking table %s: got %v, want %v", table, got, want)
		}
	}
}

func assertColumns(t *testing.T, b backend, db *sql.DB, table string, want ...string) {
	t.Helper()
	sort.Strings(want)
	got := columnNames(t, b, db, table)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columns of %s: got %v, want %v", table, got, want)
	}
}

func must(t *testing.T, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}
