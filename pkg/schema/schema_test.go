package schema_test

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
	"strings"
)

func TestNewBuilder(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)
	if builder == nil {
		t.Errorf("expected non-nil")
	}
}

func TestBuilder_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Create("users", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("name", 255)
		bp.String("email", 255).Unique()
		bp.Timestamps()
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_Create_CompileError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	// Empty blueprint should cause a compile error (no columns defined)
	err = builder.Create("empty_table", func(bp *schema.Blueprint) {
		// no columns
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestBuilder_Create_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("CREATE TABLE").WillReturnError(fmt.Errorf("connection lost"))

	err = builder.Create("users", func(bp *schema.Blueprint) {
		bp.ID()
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "connection lost") {
		t.Errorf("%q not in %q", "connection lost", err.Error())
	}
}

func TestBuilder_Alter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("ALTER TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Alter("users", func(bp *schema.Blueprint) {
		bp.String("phone", 20).Nullable()
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_Alter_MultipleStatements(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	// Expect two ALTER TABLE statements: one for adding a column, one for dropping
	mock.ExpectExec("ALTER TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ALTER TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Alter("users", func(bp *schema.Blueprint) {
		bp.String("phone", 20).Nullable()
		bp.DropColumn("legacy_field")
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_Alter_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("ALTER TABLE").WillReturnError(fmt.Errorf("table not found"))

	err = builder.Alter("nonexistent", func(bp *schema.Blueprint) {
		bp.String("col", 100)
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "table not found") {
		t.Errorf("%q not in %q", "table not found", err.Error())
	}
}

func TestBuilder_Drop(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("DROP TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Drop("users")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_Drop_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("DROP TABLE").WillReturnError(fmt.Errorf("permission denied"))

	err = builder.Drop("users")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("%q not in %q", "permission denied", err.Error())
	}
}

func TestBuilder_DropIfExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("DROP TABLE IF EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.DropIfExists("users")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_Rename(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("ALTER TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Rename("old_table", "new_table")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_HasTable_True(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	exists, err := builder.HasTable("users")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("expected true")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_HasTable_False(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	exists, err := builder.HasTable("nonexistent")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exists {
		t.Errorf("expected false")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_HasTable_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(fmt.Errorf("query failed"))

	exists, err := builder.HasTable("users")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if exists {
		t.Errorf("expected false")
	}
}

func TestBuilder_HasColumn_True(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	exists, err := builder.HasColumn("users", "email")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("expected true")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_HasColumn_False(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	exists, err := builder.HasColumn("users", "nonexistent")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exists {
		t.Errorf("expected false")
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_HasColumn_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(fmt.Errorf("query failed"))

	exists, err := builder.HasColumn("users", "email")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if exists {
		t.Errorf("expected false")
	}
}

func TestBuilder_WithTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewPostgresGrammar()

	// Begin a transaction and use it as the executor
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	builder := schema.NewBuilder(tx, grammar)

	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Create("posts", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("title", 255)
		bp.Text("body").Nullable()
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	mock.ExpectCommit()
	err = tx.Commit()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_WithMySQLGrammar(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewMySQLGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Create("products", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("name", 100)
		bp.Decimal("price", 10, 2)
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}

func TestBuilder_WithSQLiteGrammar(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	grammar := grammars.NewSQLiteGrammar()
	builder := schema.NewBuilder(db, grammar)

	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Create("settings", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("key", 100).Unique()
		bp.Text("value")
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Errorf("unexpected error: %v", mock.ExpectationsWereMet())
	}
}
