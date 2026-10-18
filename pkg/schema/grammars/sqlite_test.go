package grammars

import (
	"errors"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
)

func newSQLiteGrammar() *SQLiteGrammar {
	return NewSQLiteGrammar()
}

// --- Grammar interface compliance ---

func TestSQLiteGrammar_ImplementsGrammar(t *testing.T) {
	var _ schema.Grammar = (*SQLiteGrammar)(nil)
}

// --- CompileColumnType tests ---

func TestSQLite_CompileColumnType_String(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "name", Type: schema.TypeString, Length: 100}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_Text(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "body", Type: schema.TypeText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_Integer(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "age", Type: schema.TypeInteger}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER" != result {
		t.Errorf("got %v, want %v", result, "INTEGER")
	}
}

func TestSQLite_CompileColumnType_IntegerAutoIncrementPK(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeInteger, IsAutoIncrement: true, IsPrimary: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER PRIMARY KEY AUTOINCREMENT" != result {
		t.Errorf("got %v, want %v", result, "INTEGER PRIMARY KEY AUTOINCREMENT")
	}
}

func TestSQLite_CompileColumnType_IntegerAutoIncrementNoPK(t *testing.T) {
	g := newSQLiteGrammar()
	// Auto-increment without primary key should just be INTEGER
	col := schema.ColumnDefinition{Name: "seq", Type: schema.TypeInteger, IsAutoIncrement: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER" != result {
		t.Errorf("got %v, want %v", result, "INTEGER")
	}
}

func TestSQLite_CompileColumnType_BigInteger(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "total", Type: schema.TypeBigInteger}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER" != result {
		t.Errorf("got %v, want %v", result, "INTEGER")
	}
}

func TestSQLite_CompileColumnType_BigIntegerAutoIncrementPK(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeBigInteger, IsAutoIncrement: true, IsPrimary: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER PRIMARY KEY AUTOINCREMENT" != result {
		t.Errorf("got %v, want %v", result, "INTEGER PRIMARY KEY AUTOINCREMENT")
	}
}

func TestSQLite_CompileColumnType_Boolean(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "active", Type: schema.TypeBoolean}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER" != result {
		t.Errorf("got %v, want %v", result, "INTEGER")
	}
}

func TestSQLite_CompileColumnType_Timestamp(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "created_at", Type: schema.TypeTimestamp}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_Date(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "birth_date", Type: schema.TypeDate}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_Decimal(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "price", Type: schema.TypeDecimal, Precision: 10, Scale: 2}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "REAL" != result {
		t.Errorf("got %v, want %v", result, "REAL")
	}
}

func TestSQLite_CompileColumnType_Float(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "value", Type: schema.TypeFloat}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "REAL" != result {
		t.Errorf("got %v, want %v", result, "REAL")
	}
}

func TestSQLite_CompileColumnType_UUID(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "external_id", Type: schema.TypeUUID}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_JSON(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "metadata", Type: schema.TypeJSON}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_Binary(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "data", Type: schema.TypeBinary}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BLOB" != result {
		t.Errorf("got %v, want %v", result, "BLOB")
	}
}

func TestSQLite_CompileColumnType_Unsupported(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileCreate_SimpleTable(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("name", 255)

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CREATE TABLE "users"`) {
		t.Errorf("%q not in %q", `CREATE TABLE "users"`, sql)
	}
	if !strings.Contains(sql, `"id" INTEGER PRIMARY KEY AUTOINCREMENT`) {
		t.Errorf("%q not in %q", `"id" INTEGER PRIMARY KEY AUTOINCREMENT`, sql)
	}
	if !strings.Contains(sql, `"name" TEXT NOT NULL`) {
		t.Errorf("%q not in %q", `"name" TEXT NOT NULL`, sql)
	}
	// Auto-increment PK is inline, so no separate PRIMARY KEY clause
	if strings.Contains(sql, `PRIMARY KEY ("id")`) {
		t.Errorf("%q should not be in %q", `PRIMARY KEY ("id")`, sql)
	}
}

func TestSQLite_CompileCreate_NoColumns(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("empty")

	_, err := joinCreate(g.CompileCreate(bp))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no columns defined") {
		t.Errorf("%q not in %q", "no columns defined", err.Error())
	}
}

func TestSQLite_CompileCreate_AllColumnTypes(t *testing.T) {
	g := newSQLiteGrammar()
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

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All text-like types map to TEXT
	if !strings.Contains(sql, `"col_string" TEXT`) {
		t.Errorf("%q not in %q", `"col_string" TEXT`, sql)
	}
	if !strings.Contains(sql, `"col_text" TEXT`) {
		t.Errorf("%q not in %q", `"col_text" TEXT`, sql)
	}
	if !strings.Contains(sql, `"col_int" INTEGER`) {
		t.Errorf("%q not in %q", `"col_int" INTEGER`, sql)
	}
	if !strings.Contains(sql, `"col_bigint" INTEGER`) {
		t.Errorf("%q not in %q", `"col_bigint" INTEGER`, sql)
	}
	if !strings.Contains(sql, `"col_bool" INTEGER`) {
		t.Errorf("%q not in %q", `"col_bool" INTEGER`, sql)
	}
	if !strings.Contains(sql, `"col_ts" TEXT`) {
		t.Errorf("%q not in %q", `"col_ts" TEXT`, sql)
	}
	if !strings.Contains(sql, `"col_date" TEXT`) {
		t.Errorf("%q not in %q", `"col_date" TEXT`, sql)
	}
	if !strings.Contains(sql, `"col_dec" REAL`) {
		t.Errorf("%q not in %q", `"col_dec" REAL`, sql)
	}
	if !strings.Contains(sql, `"col_float" REAL`) {
		t.Errorf("%q not in %q", `"col_float" REAL`, sql)
	}
	if !strings.Contains(sql, `"col_uuid" TEXT`) {
		t.Errorf("%q not in %q", `"col_uuid" TEXT`, sql)
	}
	if !strings.Contains(sql, `"col_json" TEXT`) {
		t.Errorf("%q not in %q", `"col_json" TEXT`, sql)
	}
	if !strings.Contains(sql, `"col_bin" BLOB`) {
		t.Errorf("%q not in %q", `"col_bin" BLOB`, sql)
	}
}

func TestSQLite_CompileCreate_NullableColumn(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("posts")
	bp.Text("body").Nullable()

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(sql, "NOT NULL") {
		t.Errorf("%q should not be in %q", "NOT NULL", sql)
	}
	if !strings.Contains(sql, `"body" TEXT`) {
		t.Errorf("%q not in %q", `"body" TEXT`, sql)
	}
}

func TestSQLite_CompileCreate_DefaultValue(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("settings")
	bp.Boolean("active").Default(true)
	bp.String("status", 50).Default("pending")
	bp.Integer("count").Default(0)

	sql, err := joinCreate(g.CompileCreate(bp))
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

func TestSQLite_CompileCreate_UniqueColumn(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("email", 255).Unique()

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "UNIQUE") {
		t.Errorf("%q not in %q", "UNIQUE", sql)
	}
}

func TestSQLite_CompileCreate_WithIndexes(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("name", 255)
	bp.String("email", 255)
	bp.Index("name")
	bp.UniqueIndex("email")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CONSTRAINT "uniq_users_email" UNIQUE ("email")`) {
		t.Errorf("%q not in %q", `CONSTRAINT "uniq_users_email" UNIQUE ("email")`, sql)
	}
	if !strings.Contains(sql, `CREATE INDEX "idx_users_name" ON "users" ("name")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "idx_users_name" ON "users" ("name")`, sql)
	}
}

func TestSQLite_CompileCreate_WithForeignKey(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("posts")
	bp.BigInteger("user_id")
	bp.Foreign("user_id").References("id").On("users").OnDeleteAction("CASCADE").OnUpdateAction("SET NULL")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CONSTRAINT "fk_posts_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE SET NULL`) {
		t.Errorf("%q not in %q", `CONSTRAINT "fk_posts_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE SET NULL`, sql)
	}
}

func TestSQLite_CompileCreate_FullTable(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("name", 255)
	bp.String("email", 255).Unique()
	bp.Boolean("active").Default(true)
	bp.Timestamps()
	bp.Index("name")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CREATE TABLE "users"`) {
		t.Errorf("%q not in %q", `CREATE TABLE "users"`, sql)
	}
	if !strings.Contains(sql, `"id" INTEGER PRIMARY KEY AUTOINCREMENT`) {
		t.Errorf("%q not in %q", `"id" INTEGER PRIMARY KEY AUTOINCREMENT`, sql)
	}
	if !strings.Contains(sql, `"name" TEXT NOT NULL`) {
		t.Errorf("%q not in %q", `"name" TEXT NOT NULL`, sql)
	}
	if !strings.Contains(sql, `"email" TEXT NOT NULL UNIQUE`) {
		t.Errorf("%q not in %q", `"email" TEXT NOT NULL UNIQUE`, sql)
	}
	if !strings.Contains(sql, `"active" INTEGER NOT NULL DEFAULT TRUE`) {
		t.Errorf("%q not in %q", `"active" INTEGER NOT NULL DEFAULT TRUE`, sql)
	}
	if !strings.Contains(sql, `"created_at" TEXT`) {
		t.Errorf("%q not in %q", `"created_at" TEXT`, sql)
	}
	if !strings.Contains(sql, `"updated_at" TEXT`) {
		t.Errorf("%q not in %q", `"updated_at" TEXT`, sql)
	}
	if !strings.Contains(sql, `CREATE INDEX "idx_users_name" ON "users" ("name")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "idx_users_name" ON "users" ("name")`, sql)
	}
}

func TestSQLite_CompileCreate_CompositeIndex(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("first_name", 100)
	bp.String("last_name", 100)
	bp.Index("first_name", "last_name")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `CREATE INDEX "idx_users_first_name_last_name" ON "users" ("first_name", "last_name")`) {
		t.Errorf("%q not in %q", `CREATE INDEX "idx_users_first_name_last_name" ON "users" ("first_name", "last_name")`, sql)
	}
}

func TestSQLite_CompileCreate_Timestamps(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("posts")
	bp.String("title", 255)
	bp.Timestamps()

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `"created_at" TEXT`) {
		t.Errorf("%q not in %q", `"created_at" TEXT`, sql)
	}
	if !strings.Contains(sql, `"updated_at" TEXT`) {
		t.Errorf("%q not in %q", `"updated_at" TEXT`, sql)
	}
	// Timestamps should be nullable (no NOT NULL)
	parts := strings.Split(sql, ",")
	for _, part := range parts {
		if strings.Contains(part, "created_at") || strings.Contains(part, "updated_at") {
			if strings.Contains(part, "NOT NULL") {
				t.Errorf("%q should not be in %q", "NOT NULL", part)
			}
		}
	}
}

func TestSQLite_CompileCreate_NonPKAutoIncrement(t *testing.T) {
	// Auto-increment without primary key should just be INTEGER, no AUTOINCREMENT
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("counters")
	bp.Integer("seq").AutoIncrement()

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `"seq" INTEGER NOT NULL`) {
		t.Errorf("%q not in %q", `"seq" INTEGER NOT NULL`, sql)
	}
	if strings.Contains(sql, "AUTOINCREMENT") {
		t.Errorf("%q should not be in %q", "AUTOINCREMENT", sql)
	}
}

func TestSQLite_CompileCreate_UnsignedIgnored(t *testing.T) {
	// SQLite ignores UNSIGNED — it should not appear in the output
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("items")
	bp.Integer("quantity").Unsigned()

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(sql, "UNSIGNED") {
		t.Errorf("%q should not be in %q", "UNSIGNED", sql)
	}
	if !strings.Contains(sql, `"quantity" INTEGER NOT NULL`) {
		t.Errorf("%q not in %q", `"quantity" INTEGER NOT NULL`, sql)
	}
}

// --- CompileAlter tests ---

func TestSQLite_CompileAlter_AddColumn(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("phone", 20).Nullable()

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if `ALTER TABLE "users" ADD COLUMN "phone" TEXT` != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], `ALTER TABLE "users" ADD COLUMN "phone" TEXT`)
	}
}

func TestSQLite_CompileAlter_DropColumn(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileAlter_RenameColumn(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileAlter_DropIndex(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileAlter_AddIndex(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileAlter_AddUniqueIndex(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileAlter_AddForeignKey(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileAlter_MultipleOperations(t *testing.T) {
	g := newSQLiteGrammar()
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

func TestSQLite_CompileDrop(t *testing.T) {
	g := newSQLiteGrammar()
	if `DROP TABLE "users"` != g.CompileDrop("users") {
		t.Errorf("got %v, want %v", g.CompileDrop("users"), `DROP TABLE "users"`)
	}
}

func TestSQLite_CompileDropIfExists(t *testing.T) {
	g := newSQLiteGrammar()
	if `DROP TABLE IF EXISTS "users"` != g.CompileDropIfExists("users") {
		t.Errorf("got %v, want %v", g.CompileDropIfExists("users"), `DROP TABLE IF EXISTS "users"`)
	}
}

// --- CompileRename test ---

func TestSQLite_CompileRename(t *testing.T) {
	g := newSQLiteGrammar()
	if `ALTER TABLE "old_table" RENAME TO "new_table"` != g.CompileRename("old_table", "new_table") {
		t.Errorf("got %v, want %v", g.CompileRename("old_table", "new_table"), `ALTER TABLE "old_table" RENAME TO "new_table"`)
	}
}

// --- CompileHasTable test ---

func TestSQLite_CompileHasTable(t *testing.T) {
	g := newSQLiteGrammar()
	sql := g.CompileHasTable("users")
	if !strings.Contains(sql, "sqlite_master") {
		t.Errorf("%q not in %q", "sqlite_master", sql)
	}
	if !strings.Contains(sql, "type='table'") {
		t.Errorf("%q not in %q", "type='table'", sql)
	}
	if !strings.Contains(sql, "name='users'") {
		t.Errorf("%q not in %q", "name='users'", sql)
	}
}

// --- CompileHasColumn test ---

func TestSQLite_CompileHasColumn(t *testing.T) {
	g := newSQLiteGrammar()
	sql := g.CompileHasColumn("users", "email")
	if !strings.Contains(sql, "pragma_table_info('users')") {
		t.Errorf("%q not in %q", "pragma_table_info('users')", sql)
	}
	if !strings.Contains(sql, "name='email'") {
		t.Errorf("%q not in %q", "name='email'", sql)
	}
}

// --- Default string escaping ---

func TestSQLite_CompileCreate_DefaultStringEscapesSingleQuotes(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("test")
	bp.String("val", 100).Default("it's")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "DEFAULT 'it''s'") {
		t.Errorf("%q not in %q", "DEFAULT 'it''s'", sql)
	}
}

// --- New column type compilation tests ---

func TestSQLite_CompileColumnType_Enum(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum, AllowedValues: []string{"active", "inactive", "pending"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if `TEXT CHECK ("status" IN ('active','inactive','pending'))` != result {
		t.Errorf("got %v, want %v", result, `TEXT CHECK ("status" IN ('active','inactive','pending'))`)
	}
}

func TestSQLite_CompileColumnType_EnumSingleValue(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "role", Type: schema.TypeEnum, AllowedValues: []string{"admin"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if `TEXT CHECK ("role" IN ('admin'))` != result {
		t.Errorf("got %v, want %v", result, `TEXT CHECK ("role" IN ('admin'))`)
	}
}

func TestSQLite_CompileColumnType_EnumWithQuotes(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "label", Type: schema.TypeEnum, AllowedValues: []string{"it's", "they're"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if `TEXT CHECK ("label" IN ('it''s','they''re'))` != result {
		t.Errorf("got %v, want %v", result, `TEXT CHECK ("label" IN ('it''s','they''re'))`)
	}
}

func TestSQLite_CompileColumnType_EnumEmpty(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum, AllowedValues: []string{}}
	_, err := g.CompileColumnType(col)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "enum column requires at least one allowed value") {
		t.Errorf("%q not in %q", "enum column requires at least one allowed value", err.Error())
	}
}

func TestSQLite_CompileColumnType_EnumNilValues(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum}
	_, err := g.CompileColumnType(col)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "enum column requires at least one allowed value") {
		t.Errorf("%q not in %q", "enum column requires at least one allowed value", err.Error())
	}
}

func TestSQLite_CompileColumnType_Char(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar, Length: 10}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_CharZeroLength(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar, Length: 0}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_LongText(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "content", Type: schema.TypeLongText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_MediumText(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "description", Type: schema.TypeMediumText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestSQLite_CompileColumnType_TinyInt(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "priority", Type: schema.TypeTinyInt}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER" != result {
		t.Errorf("got %v, want %v", result, "INTEGER")
	}
}

func TestSQLite_CompileColumnType_SmallInt(t *testing.T) {
	g := newSQLiteGrammar()
	col := schema.ColumnDefinition{Name: "age", Type: schema.TypeSmallInt}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INTEGER" != result {
		t.Errorf("got %v, want %v", result, "INTEGER")
	}
}

// --- New column types in CompileCreate ---

func TestSQLite_CompileCreate_WithEnumColumn(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("name", 255)
	bp.Enum("status", []string{"active", "inactive"})

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `TEXT CHECK ("status" IN ('active','inactive'))`) {
		t.Errorf("%q not in %q", `TEXT CHECK ("status" IN ('active','inactive'))`, sql)
	}
}

func TestSQLite_CompileCreate_WithCharColumn(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("countries")
	bp.Char("code", 2)

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `"code" TEXT`) {
		t.Errorf("%q not in %q", `"code" TEXT`, sql)
	}
}

func TestSQLite_CompileCreate_WithNewTextTypes(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("articles")
	bp.LongText("body")
	bp.MediumText("summary")

	sql, err := joinCreate(g.CompileCreate(bp))
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

func TestSQLite_CompileCreate_WithNewIntTypes(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("metrics")
	bp.TinyInt("priority")
	bp.SmallInt("score")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `"priority" INTEGER`) {
		t.Errorf("%q not in %q", `"priority" INTEGER`, sql)
	}
	if !strings.Contains(sql, `"score" INTEGER`) {
		t.Errorf("%q not in %q", `"score" INTEGER`, sql)
	}
}

func TestSQLite_CompileCreate_WithFulltextIndex_ReturnsError(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("articles")
	bp.String("title", 255)
	bp.FulltextIndex("title")

	_, err := joinCreate(g.CompileCreate(bp))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fulltext indexes are not supported by SQLite") {
		t.Errorf("%q not in %q", "fulltext indexes are not supported by SQLite", err.Error())
	}
}

func TestSQLite_CompileCreate_WithSpatialIndex_ReturnsError(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("locations")
	bp.String("coords", 255)
	bp.SpatialIndex("coords")

	_, err := joinCreate(g.CompileCreate(bp))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "spatial indexes are not supported by SQLite") {
		t.Errorf("%q not in %q", "spatial indexes are not supported by SQLite", err.Error())
	}
}

func TestSQLite_CompileAlter_WithFulltextIndex_ReturnsError(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("articles")
	bp.FulltextIndex("title")

	_, err := g.CompileAlter(bp)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fulltext indexes are not supported by SQLite") {
		t.Errorf("%q not in %q", "fulltext indexes are not supported by SQLite", err.Error())
	}
}

func TestSQLite_CompileAlter_WithSpatialIndex_ReturnsError(t *testing.T) {
	g := newSQLiteGrammar()
	bp := schema.NewBlueprint("locations")
	bp.SpatialIndex("coords")

	_, err := g.CompileAlter(bp)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "spatial indexes are not supported by SQLite") {
		t.Errorf("%q not in %q", "spatial indexes are not supported by SQLite", err.Error())
	}
}
