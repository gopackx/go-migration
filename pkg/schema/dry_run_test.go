package schema_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/schema"
	"github.com/gopackx/go-migration/pkg/schema/grammars"
)

func TestBuilder_HasTable_DryRun_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	builder := schema.NewBuilder(&schema.DryRunExecutor{Writer: &buf}, grammars.NewPostgresGrammar())

	exists, err := builder.HasTable("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Errorf("dry-run HasTable should report the table as absent")
	}
	if !strings.Contains(buf.String(), "information_schema.tables") {
		t.Errorf("expected HasTable query in dry-run output, got %q", buf.String())
	}
}

func TestBuilder_HasColumn_DryRun_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	builder := schema.NewBuilder(&schema.DryRunExecutor{Writer: &buf}, grammars.NewPostgresGrammar())

	exists, err := builder.HasColumn("users", "email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Errorf("dry-run HasColumn should report the column as absent")
	}
	if !strings.Contains(buf.String(), "information_schema.columns") {
		t.Errorf("expected HasColumn query in dry-run output, got %q", buf.String())
	}
}

func TestBuilder_HasTable_DryRunThroughRecordingExecutor(t *testing.T) {
	var buf bytes.Buffer
	rec := schema.NewRecordingExecutor(&schema.DryRunExecutor{Writer: &buf})
	builder := schema.NewBuilder(rec, grammars.NewSQLiteGrammar())

	exists, err := builder.HasTable("users")
	if err != nil || exists {
		t.Fatalf("got (%v, %v), want (false, nil)", exists, err)
	}
	if !strings.Contains(rec.LastSQL, "sqlite_master") {
		t.Errorf("LastSQL = %q", rec.LastSQL)
	}
}

func TestBuilder_Create_ExecutesEachStatement(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	builder := schema.NewBuilder(db, grammars.NewPostgresGrammar())

	mock.ExpectExec(`^CREATE TABLE "users" \([^;]*\)$`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`^CREATE INDEX "idx_users_email" ON "users" \("email"\)$`).WillReturnResult(sqlmock.NewResult(0, 0))

	err = builder.Create("users", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("email", 255)
		bp.Index("email").Named("idx_users_email")
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBuilder_Create_StopsOnStatementError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	builder := schema.NewBuilder(db, grammars.NewSQLiteGrammar())

	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE INDEX").WillReturnError(errors.New("index failed"))

	err = builder.Create("users", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("email", 255)
		bp.Index("email")
		bp.String("name", 255)
		bp.Index("name")
	})
	if err == nil || !strings.Contains(err.Error(), "index failed") {
		t.Errorf("got %v, want index failed", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
