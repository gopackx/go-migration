package grammars

import (
	"errors"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
)

func newGrammar() *PostgresGrammar {
	return NewPostgresGrammar()
}

// --- CompileColumnType tests ---

func TestCompileColumnType_String(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "name", Type: schema.TypeString, Length: 100}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "VARCHAR(100)" != result {
		t.Errorf("got %v, want %v", result, "VARCHAR(100)")
	}
}

func TestCompileColumnType_StringDefaultLength(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "name", Type: schema.TypeString}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "VARCHAR(255)" != result {
		t.Errorf("got %v, want %v", result, "VARCHAR(255)")
	}
}

func TestCompileColumnType_Text(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "body", Type: schema.TypeText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestCompileColumnType_Integer(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "age", Type: schema.TypeInteger}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER" != result {
		t.Errorf("got %v, want %v", result, "INTEGER")
	}
}

func TestCompileColumnType_IntegerAutoIncrement(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeInteger, IsAutoIncrement: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "SERIAL" != result {
		t.Errorf("got %v, want %v", result, "SERIAL")
	}
}

func TestCompileColumnType_BigInteger(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "total", Type: schema.TypeBigInteger}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BIGINT" != result {
		t.Errorf("got %v, want %v", result, "BIGINT")
	}
}

func TestCompileColumnType_BigIntegerAutoIncrement(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeBigInteger, IsAutoIncrement: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BIGSERIAL" != result {
		t.Errorf("got %v, want %v", result, "BIGSERIAL")
	}
}

func TestCompileColumnType_Boolean(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "active", Type: schema.TypeBoolean}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BOOLEAN" != result {
		t.Errorf("got %v, want %v", result, "BOOLEAN")
	}
}

func TestCompileColumnType_Timestamp(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "created_at", Type: schema.TypeTimestamp}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TIMESTAMPTZ" != result {
		t.Errorf("got %v, want %v", result, "TIMESTAMPTZ")
	}
}

func TestCompileColumnType_Date(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "birth_date", Type: schema.TypeDate}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DATE" != result {
		t.Errorf("got %v, want %v", result, "DATE")
	}
}

func TestCompileColumnType_Decimal(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "price", Type: schema.TypeDecimal, Precision: 10, Scale: 2}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DECIMAL(10, 2)" != result {
		t.Errorf("got %v, want %v", result, "DECIMAL(10, 2)")
	}
}

func TestCompileColumnType_DecimalDefaults(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "amount", Type: schema.TypeDecimal}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DECIMAL(10, 0)" != result {
		t.Errorf("got %v, want %v", result, "DECIMAL(10, 0)")
	}
}

func TestCompileColumnType_Float(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "value", Type: schema.TypeFloat}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DOUBLE PRECISION" != result {
		t.Errorf("got %v, want %v", result, "DOUBLE PRECISION")
	}
}

func TestCompileColumnType_UUID(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "external_id", Type: schema.TypeUUID}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "UUID" != result {
		t.Errorf("got %v, want %v", result, "UUID")
	}
}

func TestCompileColumnType_JSON(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "metadata", Type: schema.TypeJSON}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "JSONB" != result {
		t.Errorf("got %v, want %v", result, "JSONB")
	}
}

func TestCompileColumnType_Binary(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "data", Type: schema.TypeBinary}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BYTEA" != result {
		t.Errorf("got %v, want %v", result, "BYTEA")
	}
}

func TestCompileColumnType_Unsupported(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "bad", Type: schema.ColumnType(999)}
	_, err := g.CompileColumnType(col)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("expected true")
	}
	if !strings.Contains(err.Error(), `"bad"`) {
		t.Errorf("%q not in %q", `"bad"`, err.Error())
	}
}

// --- CompileCreate tests ---

func TestCompileCreate_SimpleTable(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("name", 255)

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CREATE TABLE "users"`) {
		t.Errorf("%q not in %q", `CREATE TABLE "users"`, sql)
	}
	if !strings.Contains(sql, `"id" BIGSERIAL`) {
		t.Errorf("%q not in %q", `"id" BIGSERIAL`, sql)
	}
	if !strings.Contains(sql, `"name" VARCHAR(255) NOT NULL`) {
		t.Errorf("%q not in %q", `"name" VARCHAR(255) NOT NULL`, sql)
	}
	if !strings.Contains(sql, `PRIMARY KEY ("id")`) {
		t.Errorf("%q not in %q", `PRIMARY KEY ("id")`, sql)
	}
}

func TestCompileCreate_NoColumns(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("empty")

	_, err := g.CompileCreate(bp)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no columns defined") {
		t.Errorf("%q not in %q", "no columns defined", err.Error())
	}
}

func TestCompileCreate_AllColumnTypes(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("all_types")
	bp.String("col_string", 100)
	bp.Text("col_text")
	bp.Integer("col_int")
	bp.BigInteger("col_bigint")
	bp.Boolean("col_bool")
	bp.Timestamp("col_ts")
	bp.Date("col_date")
	bp.Decimal("col_dec", 8, 2)
	bp.Float("col_float")
	bp.UUID("col_uuid")
	bp.JSON("col_json")
	bp.Binary("col_bin")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "VARCHAR(100)") {
		t.Errorf("%q not in %q", "VARCHAR(100)", sql)
	}
	if !strings.Contains(sql, "TEXT") {
		t.Errorf("%q not in %q", "TEXT", sql)
	}
	if !strings.Contains(sql, "INTEGER") {
		t.Errorf("%q not in %q", "INTEGER", sql)
	}
	if !strings.Contains(sql, "BIGINT") {
		t.Errorf("%q not in %q", "BIGINT", sql)
	}
	if !strings.Contains(sql, "BOOLEAN") {
		t.Errorf("%q not in %q", "BOOLEAN", sql)
	}
	if !strings.Contains(sql, "TIMESTAMPTZ") {
		t.Errorf("%q not in %q", "TIMESTAMPTZ", sql)
	}
	if !strings.Contains(sql, "DATE") {
		t.Errorf("%q not in %q", "DATE", sql)
	}
	if !strings.Contains(sql, "DECIMAL(8, 2)") {
		t.Errorf("%q not in %q", "DECIMAL(8, 2)", sql)
	}
	if !strings.Contains(sql, "DOUBLE PRECISION") {
		t.Errorf("%q not in %q", "DOUBLE PRECISION", sql)
	}
	if !strings.Contains(sql, "UUID") {
		t.Errorf("%q not in %q", "UUID", sql)
	}
	if !strings.Contains(sql, "JSONB") {
		t.Errorf("%q not in %q", "JSONB", sql)
	}
	if !strings.Contains(sql, "BYTEA") {
		t.Errorf("%q not in %q", "BYTEA", sql)
	}
}

func TestCompileCreate_NullableColumn(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("posts")
	bp.Text("body").Nullable()

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nullable columns should NOT have NOT NULL
	if strings.Contains(sql, "NOT NULL") {
		t.Errorf("%q should not be in %q", "NOT NULL", sql)
	}
	if !strings.Contains(sql, `"body" TEXT`) {
		t.Errorf("%q not in %q", `"body" TEXT`, sql)
	}
}

func TestCompileCreate_DefaultValue(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("settings")
	bp.Boolean("active").Default(true)
	bp.String("status", 50).Default("pending")
	bp.Integer("count").Default(0)

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "DEFAULT TRUE") {
		t.Errorf("%q not in %q", "DEFAULT TRUE", sql)
	}
	if !strings.Contains(sql, "DEFAULT 'pending'") {
		t.Errorf("%q not in %q", "DEFAULT 'pending'", sql)
	}
	if !strings.Contains(sql, "DEFAULT 0") {
		t.Errorf("%q not in %q", "DEFAULT 0", sql)
	}
}

func TestCompileCreate_UniqueColumn(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("email", 255).Unique()

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "UNIQUE") {
		t.Errorf("%q not in %q", "UNIQUE", sql)
	}
}

func TestCompileCreate_WithIndexes(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("name", 255)
	bp.String("email", 255)
	bp.Index("name")
	bp.UniqueIndex("email")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unique index should be inline as CONSTRAINT
	if !strings.Contains(sql, `CONSTRAINT "uniq_users_email" UNIQUE ("email")`) {
		t.Errorf("%q not in %q", `CONSTRAINT "uniq_users_email" UNIQUE ("email")`, sql)
	}
	// Non-unique index should be a separate CREATE INDEX statement
	if !strings.Contains(sql, `CREATE INDEX "idx_users_name" ON "users" ("name")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "idx_users_name" ON "users" ("name")`, sql)
	}
}

func TestCompileCreate_WithForeignKey(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("posts")
	bp.BigInteger("user_id")
	bp.Foreign("user_id").References("id").On("users").OnDeleteAction("CASCADE").OnUpdateAction("SET NULL")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CONSTRAINT "fk_posts_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE SET NULL`) {
		t.Errorf("%q not in %q", `CONSTRAINT "fk_posts_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE SET NULL`, sql)
	}
}

func TestCompileCreate_FullTable(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("name", 255)
	bp.String("email", 255).Unique()
	bp.Boolean("active").Default(true)
	bp.Timestamps()
	bp.Index("name")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CREATE TABLE "users"`) {
		t.Errorf("%q not in %q", `CREATE TABLE "users"`, sql)
	}
	if !strings.Contains(sql, `"id" BIGSERIAL`) {
		t.Errorf("%q not in %q", `"id" BIGSERIAL`, sql)
	}
	if !strings.Contains(sql, `"name" VARCHAR(255) NOT NULL`) {
		t.Errorf("%q not in %q", `"name" VARCHAR(255) NOT NULL`, sql)
	}
	if !strings.Contains(sql, `"email" VARCHAR(255) NOT NULL UNIQUE`) {
		t.Errorf("%q not in %q", `"email" VARCHAR(255) NOT NULL UNIQUE`, sql)
	}
	if !strings.Contains(sql, `"active" BOOLEAN NOT NULL DEFAULT TRUE`) {
		t.Errorf("%q not in %q", `"active" BOOLEAN NOT NULL DEFAULT TRUE`, sql)
	}
	if !strings.Contains(sql, `"created_at" TIMESTAMPTZ`) {
		t.Errorf("%q not in %q", `"created_at" TIMESTAMPTZ`, sql)
	}
	if !strings.Contains(sql, `"updated_at" TIMESTAMPTZ`) {
		t.Errorf("%q not in %q", `"updated_at" TIMESTAMPTZ`, sql)
	}
	if !strings.Contains(sql, `PRIMARY KEY ("id")`) {
		t.Errorf("%q not in %q", `PRIMARY KEY ("id")`, sql)
	}
	if !strings.Contains(sql, `CREATE INDEX "idx_users_name" ON "users" ("name")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "idx_users_name" ON "users" ("name")`, sql)
	}
}

// --- CompileAlter tests ---

func TestCompileAlter_AddColumn(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("phone", 20).Nullable()

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `ALTER TABLE "users" ADD COLUMN "phone" VARCHAR(20)` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `ALTER TABLE "users" ADD COLUMN "phone" VARCHAR(20)`)
	}
}

func TestCompileAlter_DropColumn(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.DropColumn("age")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `ALTER TABLE "users" DROP COLUMN "age"` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `ALTER TABLE "users" DROP COLUMN "age"`)
	}
}

func TestCompileAlter_RenameColumn(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.RenameColumn("name", "full_name")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `ALTER TABLE "users" RENAME COLUMN "name" TO "full_name"` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `ALTER TABLE "users" RENAME COLUMN "name" TO "full_name"`)
	}
}

func TestCompileAlter_DropIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.DropIndex("idx_users_email")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `DROP INDEX "idx_users_email"` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `DROP INDEX "idx_users_email"`)
	}
}

func TestCompileAlter_DropForeign(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("posts")
	bp.DropForeign("fk_posts_user_id")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `ALTER TABLE "posts" DROP CONSTRAINT "fk_posts_user_id"` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `ALTER TABLE "posts" DROP CONSTRAINT "fk_posts_user_id"`)
	}
}

func TestCompileAlter_AddIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.Index("name")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `CREATE INDEX "idx_users_name" ON "users" ("name")` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `CREATE INDEX "idx_users_name" ON "users" ("name")`)
	}
}

func TestCompileAlter_AddUniqueIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.UniqueIndex("email")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `CREATE UNIQUE INDEX "uniq_users_email" ON "users" ("email")` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `CREATE UNIQUE INDEX "uniq_users_email" ON "users" ("email")`)
	}
}

func TestCompileAlter_AddForeignKey(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("posts")
	bp.Foreign("user_id").References("id").On("users").OnDeleteAction("CASCADE")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if !strings.Contains(stmts[0], `ALTER TABLE "posts" ADD CONSTRAINT "fk_posts_user_id"`) {
		t.Errorf("%q not in %q", `ALTER TABLE "posts" ADD CONSTRAINT "fk_posts_user_id"`, stmts[0])
	}
	if !strings.Contains(stmts[0], `FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE`) {
		t.Errorf("%q not in %q", `FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE`, stmts[0])
	}
}

func TestCompileAlter_MultipleOperations(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("phone", 20).Nullable()
	bp.DropColumn("age")
	bp.RenameColumn("name", "full_name")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 3 {
		t.Fatalf("len = %d, want %d", len(stmts), 3)
	}
	if !strings.Contains(stmts[0], "ADD COLUMN") {
		t.Errorf("%q not in %q", "ADD COLUMN", stmts[0])
	}
	if !strings.Contains(stmts[1], "DROP COLUMN") {
		t.Errorf("%q not in %q", "DROP COLUMN", stmts[1])
	}
	if !strings.Contains(stmts[2], "RENAME COLUMN") {
		t.Errorf("%q not in %q", "RENAME COLUMN", stmts[2])
	}
}

// --- CompileDrop tests ---

func TestCompileDrop(t *testing.T) {
	g := newGrammar()
	if `DROP TABLE "users"` != g.CompileDrop("users") {
		t.Errorf("got %v, want %v", g.CompileDrop("users"), `DROP TABLE "users"`)
	}
}

func TestCompileDropIfExists(t *testing.T) {
	g := newGrammar()
	if `DROP TABLE IF EXISTS "users"` != g.CompileDropIfExists("users") {
		t.Errorf("got %v, want %v", g.CompileDropIfExists("users"), `DROP TABLE IF EXISTS "users"`)
	}
}

// --- CompileRename test ---

func TestCompileRename(t *testing.T) {
	g := newGrammar()
	if `ALTER TABLE "old_table" RENAME TO "new_table"` != g.CompileRename("old_table", "new_table") {
		t.Errorf("got %v, want %v", g.CompileRename("old_table", "new_table"), `ALTER TABLE "old_table" RENAME TO "new_table"`)
	}
}

// --- CompileHasTable test ---

func TestCompileHasTable(t *testing.T) {
	g := newGrammar()
	sql := g.CompileHasTable("users")
	if !strings.Contains(sql, "information_schema.tables") {
		t.Errorf("%q not in %q", "information_schema.tables", sql)
	}
	if !strings.Contains(sql, "table_name = 'users'") {
		t.Errorf("%q not in %q", "table_name = 'users'", sql)
	}
	if !strings.Contains(sql, "table_schema = 'public'") {
		t.Errorf("%q not in %q", "table_schema = 'public'", sql)
	}
}

// --- CompileHasColumn test ---

func TestCompileHasColumn(t *testing.T) {
	g := newGrammar()
	sql := g.CompileHasColumn("users", "email")
	if !strings.Contains(sql, "information_schema.columns") {
		t.Errorf("%q not in %q", "information_schema.columns", sql)
	}
	if !strings.Contains(sql, "table_name = 'users'") {
		t.Errorf("%q not in %q", "table_name = 'users'", sql)
	}
	if !strings.Contains(sql, "column_name = 'email'") {
		t.Errorf("%q not in %q", "column_name = 'email'", sql)
	}
	if !strings.Contains(sql, "table_schema = 'public'") {
		t.Errorf("%q not in %q", "table_schema = 'public'", sql)
	}
}

// --- CompileDropAllTables test ---

func TestCompileDropAllTables(t *testing.T) {
	g := newGrammar()
	sql := g.CompileDropAllTables()
	if !strings.Contains(sql, "DROP SCHEMA public CASCADE") {
		t.Errorf("%q not in %q", "DROP SCHEMA public CASCADE", sql)
	}
	if !strings.Contains(sql, "CREATE SCHEMA public") {
		t.Errorf("%q not in %q", "CREATE SCHEMA public", sql)
	}
}

// --- Grammar interface compliance ---

func TestPostgresGrammar_ImplementsGrammar(t *testing.T) {
	var _ schema.Grammar = (*PostgresGrammar)(nil)
}

// --- Edge cases ---

func TestCompileCreate_CompositeIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("first_name", 100)
	bp.String("last_name", 100)
	bp.Index("first_name", "last_name")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `CREATE INDEX "idx_users_first_name_last_name" ON "users" ("first_name", "last_name")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "idx_users_first_name_last_name" ON "users" ("first_name", "last_name")`, sql)
	}
}

func TestCompileCreate_DefaultStringEscapesSingleQuotes(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("test")
	bp.String("val", 100).Default("it's")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "DEFAULT 'it''s'") {
		t.Errorf("%q not in %q", "DEFAULT 'it''s'", sql)
	}
}

func TestCompileCreate_Timestamps(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("posts")
	bp.String("title", 255)
	bp.Timestamps()

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Timestamps should be nullable (no NOT NULL)
	// Check that created_at and updated_at are present
	if !strings.Contains(sql, `"created_at" TIMESTAMPTZ`) {
		t.Errorf("%q not in %q", `"created_at" TIMESTAMPTZ`, sql)
	}
	if !strings.Contains(sql, `"updated_at" TIMESTAMPTZ`) {
		t.Errorf("%q not in %q", `"updated_at" TIMESTAMPTZ`, sql)
	}
	// They should NOT have NOT NULL since they are nullable
	parts := strings.Split(sql, ",")
	for _, part := range parts {
		if strings.Contains(part, "created_at") || strings.Contains(part, "updated_at") {
			if strings.Contains(part, "NOT NULL") {
				t.Errorf("%q should not be in %q", "NOT NULL", part)
			}
		}
	}
}

// --- New column type compilation tests ---

func TestCompileColumnType_Enum(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum, AllowedValues: []string{"active", "inactive", "pending"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if `VARCHAR(255) CHECK ("status" IN ('active','inactive','pending'))` != result {
		t.Errorf("got %v, want %v", result, `VARCHAR(255) CHECK ("status" IN ('active','inactive','pending'))`)
	}
}

func TestCompileColumnType_EnumSingleValue(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "role", Type: schema.TypeEnum, AllowedValues: []string{"admin"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if `VARCHAR(255) CHECK ("role" IN ('admin'))` != result {
		t.Errorf("got %v, want %v", result, `VARCHAR(255) CHECK ("role" IN ('admin'))`)
	}
}

func TestCompileColumnType_EnumWithQuotes(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "label", Type: schema.TypeEnum, AllowedValues: []string{"it's", "they're"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if `VARCHAR(255) CHECK ("label" IN ('it''s','they''re'))` != result {
		t.Errorf("got %v, want %v", result, `VARCHAR(255) CHECK ("label" IN ('it''s','they''re'))`)
	}
}

func TestCompileColumnType_EnumEmpty(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum, AllowedValues: []string{}}
	_, err := g.CompileColumnType(col)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "enum column requires at least one allowed value") {
		t.Errorf("%q not in %q", "enum column requires at least one allowed value", err.Error())
	}
}

func TestCompileColumnType_EnumNilValues(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum}
	_, err := g.CompileColumnType(col)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "enum column requires at least one allowed value") {
		t.Errorf("%q not in %q", "enum column requires at least one allowed value", err.Error())
	}
}

func TestCompileColumnType_Char(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar, Length: 10}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "CHAR(10)" != result {
		t.Errorf("got %v, want %v", result, "CHAR(10)")
	}
}

func TestCompileColumnType_CharDefaultLength(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "CHAR(255)" != result {
		t.Errorf("got %v, want %v", result, "CHAR(255)")
	}
}

func TestCompileColumnType_CharZeroLength(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar, Length: 0}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "CHAR(255)" != result {
		t.Errorf("got %v, want %v", result, "CHAR(255)")
	}
}

func TestCompileColumnType_LongText(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "content", Type: schema.TypeLongText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestCompileColumnType_MediumText(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "description", Type: schema.TypeMediumText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestCompileColumnType_TinyInt(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "priority", Type: schema.TypeTinyInt}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "SMALLINT" != result {
		t.Errorf("got %v, want %v", result, "SMALLINT")
	}
}

func TestCompileColumnType_SmallInt(t *testing.T) {
	g := newGrammar()
	col := schema.ColumnDefinition{Name: "age", Type: schema.TypeSmallInt}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "SMALLINT" != result {
		t.Errorf("got %v, want %v", result, "SMALLINT")
	}
}

// --- New column types in CompileCreate ---

func TestCompileCreate_WithEnumColumn(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("name", 255)
	bp.Enum("status", []string{"active", "inactive"})

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `VARCHAR(255) CHECK ("status" IN ('active','inactive'))`) {
		t.Errorf("%q not in %q", `VARCHAR(255) CHECK ("status" IN ('active','inactive'))`, sql)
	}
}

func TestCompileCreate_WithCharColumn(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("countries")
	bp.Char("code", 2)

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `"code" CHAR(2)`) {
		t.Errorf("%q not in %q", `"code" CHAR(2)`, sql)
	}
}

func TestCompileCreate_WithNewTextTypes(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("articles")
	bp.LongText("body")
	bp.MediumText("summary")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `"body" TEXT`) {
		t.Errorf("%q not in %q", `"body" TEXT`, sql)
	}
	if !strings.Contains(sql, `"summary" TEXT`) {
		t.Errorf("%q not in %q", `"summary" TEXT`, sql)
	}
}

func TestCompileCreate_WithNewIntTypes(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("metrics")
	bp.TinyInt("priority")
	bp.SmallInt("score")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `"priority" SMALLINT`) {
		t.Errorf("%q not in %q", `"priority" SMALLINT`, sql)
	}
	if !strings.Contains(sql, `"score" SMALLINT`) {
		t.Errorf("%q not in %q", `"score" SMALLINT`, sql)
	}
}

// --- Fulltext and spatial index compilation tests ---

func TestCompileCreate_WithFulltextIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("articles")
	bp.String("title", 255)
	bp.Text("body")
	bp.FulltextIndex("title")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `CREATE INDEX "ft_articles_title" ON "articles" USING GIN (to_tsvector('english', "title"))`) {
		t.Errorf("%q not in %q", `CREATE INDEX "ft_articles_title" ON "articles" USING GIN (to_tsvector('english', "title"))`, sql)
	}
}

func TestCompileCreate_WithSpatialIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("locations")
	bp.String("geom", 255)
	bp.SpatialIndex("geom")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `CREATE INDEX "sp_locations_geom" ON "locations" USING GIST ("geom")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "sp_locations_geom" ON "locations" USING GIST ("geom")`, sql)
	}
}

func TestCompileAlter_AddFulltextIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("articles")
	bp.FulltextIndex("title")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `CREATE INDEX "ft_articles_title" ON "articles" USING GIN (to_tsvector('english', "title"))` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `CREATE INDEX "ft_articles_title" ON "articles" USING GIN (to_tsvector('english', "title"))`)
	}
}

func TestCompileAlter_AddSpatialIndex(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("locations")
	bp.SpatialIndex("geom")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `CREATE INDEX "sp_locations_geom" ON "locations" USING GIST ("geom")` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `CREATE INDEX "sp_locations_geom" ON "locations" USING GIST ("geom")`)
	}
}

func TestCompileCreate_WithFulltextAndRegularIndexes(t *testing.T) {
	g := newGrammar()
	bp := schema.NewBlueprint("posts")
	bp.String("title", 255)
	bp.String("slug", 255)
	bp.Index("slug")
	bp.FulltextIndex("title")

	sql, err := g.CompileCreate(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `CREATE INDEX "idx_posts_slug" ON "posts" ("slug")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "idx_posts_slug" ON "posts" ("slug")`, sql)
	}
	if !strings.Contains(sql, `CREATE INDEX "ft_posts_title" ON "posts" USING GIN (to_tsvector('english', "title"))`) {
		t.Errorf("%q not in %q", `CREATE INDEX "ft_posts_title" ON "posts" USING GIN (to_tsvector('english', "title"))`, sql)
	}
}
