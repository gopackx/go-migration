package migrator

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
)

// The tracking table name is quoted through the grammar in every statement
// the tracker issues, with the dialect's delimiter escaped by doubling.
func TestTracker_QuotesTableNameWithDelimiter(t *testing.T) {
	cases := []struct {
		name    string
		grammar schema.Grammar
		table   string
		quoted  string
		p1, p2  string
	}{
		{"postgres", grammars.NewPostgresGrammar(), `odd"migrations`, `"odd""migrations"`, "$1", "$2"},
		{"sqlite", grammars.NewSQLiteGrammar(), `odd"migrations`, `"odd""migrations"`, "?", "?"},
		{"mysql", grammars.NewMySQLGrammar(), "odd`migrations", "`odd``migrations`", "?", "?"},
		{"postgres qualified", grammars.NewPostgresGrammar(), "app.migrations", `"app"."migrations"`, "$1", "$2"},
		{"sqlite qualified", grammars.NewSQLiteGrammar(), "app.migrations", `"app"."migrations"`, "?", "?"},
		{"mysql qualified", grammars.NewMySQLGrammar(), "app.migrations", "`app`.`migrations`", "?", "?"},
		{"postgres qualified odd", grammars.NewPostgresGrammar(), `o"dd.mig"rations`, `"o""dd"."mig""rations"`, "$1", "$2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			tr := NewTrackerWithGrammar(db, c.table, c.grammar)
			exact := func(q string) string { return "^" + regexp.QuoteMeta(q) + "$" }

			mock.ExpectExec(exact(c.grammar.CompileTrackingTable(c.table))).
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec(exact("INSERT INTO "+c.quoted+" (migration, batch) VALUES ("+c.p1+", "+c.p2+")")).
				WithArgs("20240101000000_x", 1).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectQuery(exact("SELECT migration, batch, created_at FROM " + c.quoted + " ORDER BY migration ASC")).
				WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}))
			mock.ExpectQuery(exact("SELECT migration, batch, created_at FROM " + c.quoted + " WHERE batch = " + c.p1 + " ORDER BY migration ASC")).
				WithArgs(1).
				WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}))
			mock.ExpectQuery(exact("SELECT COALESCE(MAX(batch), 0) FROM " + c.quoted)).
				WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(1))
			mock.ExpectExec(exact("DELETE FROM " + c.quoted + " WHERE migration = " + c.p1)).
				WithArgs("20240101000000_x").WillReturnResult(sqlmock.NewResult(0, 1))

			if err := tr.EnsureTable(); err != nil {
				t.Fatalf("EnsureTable: %v", err)
			}
			if err := tr.Record("20240101000000_x", 1); err != nil {
				t.Fatalf("Record: %v", err)
			}
			if _, err := tr.GetApplied(); err != nil {
				t.Fatalf("GetApplied: %v", err)
			}
			if _, err := tr.GetByBatch(1); err != nil {
				t.Fatalf("GetByBatch: %v", err)
			}
			if _, err := tr.GetLastBatchNumber(); err != nil {
				t.Fatalf("GetLastBatchNumber: %v", err)
			}
			if err := tr.Remove("20240101000000_x"); err != nil {
				t.Fatalf("Remove: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// TableExists passes the raw name as an escaped string literal.
func TestTracker_TableExists_EscapesName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tr := NewTrackerWithGrammar(db, "o'migrations", grammars.NewSQLiteGrammar())
	mock.ExpectQuery("^" + regexp.QuoteMeta("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='o''migrations'") + "$").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	ok, err := tr.TableExists()
	if err != nil || !ok {
		t.Fatalf("TableExists = %v, %v", ok, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// A schema-qualified tracking table is looked up in its own schema.
func TestTracker_TableExists_Qualified(t *testing.T) {
	cases := []struct {
		name    string
		grammar schema.Grammar
		table   string
		want    string
	}{
		{"postgres qualified", grammars.NewPostgresGrammar(), "app.migrations",
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'app' AND table_name = 'migrations'"},
		{"mysql qualified", grammars.NewMySQLGrammar(), "app.migrations",
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'app' AND table_name = 'migrations'"},
		{"sqlite qualified", grammars.NewSQLiteGrammar(), "app.migrations",
			`SELECT COUNT(*) FROM "app".sqlite_master WHERE type='table' AND name='migrations'`},
		{"postgres plain", grammars.NewPostgresGrammar(), "migrations",
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'migrations'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery("^" + regexp.QuoteMeta(c.want) + "$").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			ok, err := NewTrackerWithGrammar(db, c.table, c.grammar).TableExists()
			if err != nil || !ok {
				t.Fatalf("TableExists = %v, %v", ok, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// Fresh drops a schema-qualified tracking table explicitly: PostgreSQL's
// drop-all only recreates the public schema, so the tracking table in "app"
// would otherwise survive and Up would find nothing pending.
func TestFresh_QualifiedTrackingTable_IsDropped(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer keepMockDriverConn(t, db)()

	m := New(db, WithGrammar(grammars.NewPostgresGrammar()), WithTableName("app.migrations"))
	if err := m.Register("20240101000000_create_users", &noopMigration{}); err != nil {
		t.Fatal(err)
	}
	exact := func(q string) string { return "^" + regexp.QuoteMeta(q) + "$" }

	mock.ExpectQuery(exact("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs("go-migration:app.migrations").
		WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(true))
	mock.ExpectQuery(exact("SELECT tablename FROM pg_tables WHERE schemaname = 'public'")).
		WillReturnRows(sqlmock.NewRows([]string{"tablename"}).AddRow("users"))
	mock.ExpectExec(exact("DROP SCHEMA public CASCADE")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(exact("CREATE SCHEMA public")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(exact(`DROP TABLE IF EXISTS "app"."migrations"`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS "app"\."migrations" \(`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(exact(`SELECT migration, batch, created_at FROM "app"."migrations" ORDER BY migration ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"migration", "batch", "created_at"}))
	mock.ExpectQuery(exact(`SELECT COALESCE(MAX(batch), 0) FROM "app"."migrations"`)).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(exact(`INSERT INTO "app"."migrations" (migration, batch) VALUES ($1, $2)`)).
		WithArgs("20240101000000_create_users", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(exact("SELECT pg_advisory_unlock(hashtext($1))")).
		WithArgs("go-migration:app.migrations").
		WillReturnRows(sqlmock.NewRows([]string{"unlock"}).AddRow(true))

	if err := m.Fresh(); err != nil {
		t.Fatalf("Fresh: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
