package grammars

import (
	"errors"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/schema"
)

func newMySQLGrammar() *MySQLGrammar {
	return NewMySQLGrammar()
}

// --- CompileColumnType tests ---

func TestMySQL_CompileColumnType_String(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "name", Type: schema.TypeString, Length: 100}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "VARCHAR(100)" != result {
		t.Errorf("got %v, want %v", result, "VARCHAR(100)")
	}
}

func TestMySQL_CompileColumnType_StringDefaultLength(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "name", Type: schema.TypeString}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "VARCHAR(255)" != result {
		t.Errorf("got %v, want %v", result, "VARCHAR(255)")
	}
}

func TestMySQL_CompileColumnType_Text(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "body", Type: schema.TypeText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TEXT" != result {
		t.Errorf("got %v, want %v", result, "TEXT")
	}
}

func TestMySQL_CompileColumnType_Integer(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "age", Type: schema.TypeInteger}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INT" != result {
		t.Errorf("got %v, want %v", result, "INT")
	}
}

func TestMySQL_CompileColumnType_IntegerUnsigned(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "count", Type: schema.TypeInteger, IsUnsigned: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INT UNSIGNED" != result {
		t.Errorf("got %v, want %v", result, "INT UNSIGNED")
	}
}

func TestMySQL_CompileColumnType_IntegerAutoIncrement(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeInteger, IsAutoIncrement: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INT AUTO_INCREMENT" != result {
		t.Errorf("got %v, want %v", result, "INT AUTO_INCREMENT")
	}
}

func TestMySQL_CompileColumnType_IntegerUnsignedAutoIncrement(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeInteger, IsUnsigned: true, IsAutoIncrement: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "INT UNSIGNED AUTO_INCREMENT" != result {
		t.Errorf("got %v, want %v", result, "INT UNSIGNED AUTO_INCREMENT")
	}
}

func TestMySQL_CompileColumnType_BigInteger(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "total", Type: schema.TypeBigInteger}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BIGINT" != result {
		t.Errorf("got %v, want %v", result, "BIGINT")
	}
}

func TestMySQL_CompileColumnType_BigIntegerUnsigned(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "total", Type: schema.TypeBigInteger, IsUnsigned: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BIGINT UNSIGNED" != result {
		t.Errorf("got %v, want %v", result, "BIGINT UNSIGNED")
	}
}

func TestMySQL_CompileColumnType_BigIntegerAutoIncrement(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeBigInteger, IsAutoIncrement: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BIGINT AUTO_INCREMENT" != result {
		t.Errorf("got %v, want %v", result, "BIGINT AUTO_INCREMENT")
	}
}

func TestMySQL_CompileColumnType_BigIntegerUnsignedAutoIncrement(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "id", Type: schema.TypeBigInteger, IsUnsigned: true, IsAutoIncrement: true}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BIGINT UNSIGNED AUTO_INCREMENT" != result {
		t.Errorf("got %v, want %v", result, "BIGINT UNSIGNED AUTO_INCREMENT")
	}
}

func TestMySQL_CompileColumnType_Boolean(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "active", Type: schema.TypeBoolean}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TINYINT(1)" != result {
		t.Errorf("got %v, want %v", result, "TINYINT(1)")
	}
}

func TestMySQL_CompileColumnType_Timestamp(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "created_at", Type: schema.TypeTimestamp}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TIMESTAMP" != result {
		t.Errorf("got %v, want %v", result, "TIMESTAMP")
	}
}

func TestMySQL_CompileColumnType_Date(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "birth_date", Type: schema.TypeDate}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DATE" != result {
		t.Errorf("got %v, want %v", result, "DATE")
	}
}

func TestMySQL_CompileColumnType_Decimal(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "price", Type: schema.TypeDecimal, Precision: 10, Scale: 2}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DECIMAL(10, 2)" != result {
		t.Errorf("got %v, want %v", result, "DECIMAL(10, 2)")
	}
}

func TestMySQL_CompileColumnType_DecimalDefaults(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "amount", Type: schema.TypeDecimal}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DECIMAL(10, 0)" != result {
		t.Errorf("got %v, want %v", result, "DECIMAL(10, 0)")
	}
}

func TestMySQL_CompileColumnType_Float(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "value", Type: schema.TypeFloat}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "DOUBLE" != result {
		t.Errorf("got %v, want %v", result, "DOUBLE")
	}
}

func TestMySQL_CompileColumnType_UUID(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "external_id", Type: schema.TypeUUID}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "CHAR(36)" != result {
		t.Errorf("got %v, want %v", result, "CHAR(36)")
	}
}

func TestMySQL_CompileColumnType_JSON(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "metadata", Type: schema.TypeJSON}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "JSON" != result {
		t.Errorf("got %v, want %v", result, "JSON")
	}
}

func TestMySQL_CompileColumnType_Binary(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "data", Type: schema.TypeBinary}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "BLOB" != result {
		t.Errorf("got %v, want %v", result, "BLOB")
	}
}

func TestMySQL_CompileColumnType_Unsupported(t *testing.T) {
	g := newMySQLGrammar()
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

func TestMySQL_CompileCreate_SimpleTable(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.ID()
	bp.String("name", 255)

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE `users`") {
		t.Errorf("%q not in %q", "CREATE TABLE `users`", sql)
	}
	if !strings.Contains(sql, "`id` BIGINT UNSIGNED AUTO_INCREMENT NOT NULL") {
		t.Errorf("%q not in %q", "`id` BIGINT UNSIGNED AUTO_INCREMENT NOT NULL", sql)
	}
	if !strings.Contains(sql, "`name` VARCHAR(255) NOT NULL") {
		t.Errorf("%q not in %q", "`name` VARCHAR(255) NOT NULL", sql)
	}
	if !strings.Contains(sql, "PRIMARY KEY (`id`)") {
		t.Errorf("%q not in %q", "PRIMARY KEY (`id`)", sql)
	}
}

func TestMySQL_CompileCreate_NoColumns(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("empty")

	_, err := joinCreate(g.CompileCreate(bp))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no columns defined") {
		t.Errorf("%q not in %q", "no columns defined", err.Error())
	}
}

func TestMySQL_CompileCreate_AllColumnTypes(t *testing.T) {
	g := newMySQLGrammar()
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

	if !strings.Contains(sql, "VARCHAR(100)") {
		t.Errorf("%q not in %q", "VARCHAR(100)", sql)
	}
	if !strings.Contains(sql, "TEXT") {
		t.Errorf("%q not in %q", "TEXT", sql)
	}
	if !strings.Contains(sql, "INT") {
		t.Errorf("%q not in %q", "INT", sql)
	}
	if !strings.Contains(sql, "BIGINT") {
		t.Errorf("%q not in %q", "BIGINT", sql)
	}
	if !strings.Contains(sql, "TINYINT(1)") {
		t.Errorf("%q not in %q", "TINYINT(1)", sql)
	}
	if !strings.Contains(sql, "TIMESTAMP") {
		t.Errorf("%q not in %q", "TIMESTAMP", sql)
	}
	if !strings.Contains(sql, "DATE") {
		t.Errorf("%q not in %q", "DATE", sql)
	}
	if !strings.Contains(sql, "DECIMAL(8, 2)") {
		t.Errorf("%q not in %q", "DECIMAL(8, 2)", sql)
	}
	if !strings.Contains(sql, "DOUBLE") {
		t.Errorf("%q not in %q", "DOUBLE", sql)
	}
	if !strings.Contains(sql, "CHAR(36)") {
		t.Errorf("%q not in %q", "CHAR(36)", sql)
	}
	if !strings.Contains(sql, "JSON") {
		t.Errorf("%q not in %q", "JSON", sql)
	}
	if !strings.Contains(sql, "BLOB") {
		t.Errorf("%q not in %q", "BLOB", sql)
	}
}

func TestMySQL_CompileCreate_NullableColumn(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("posts")
	bp.Text("body").Nullable()

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(sql, "NOT NULL") {
		t.Errorf("%q should not be in %q", "NOT NULL", sql)
	}
	if !strings.Contains(sql, "`body` TEXT") {
		t.Errorf("%q not in %q", "`body` TEXT", sql)
	}
}

func TestMySQL_CompileCreate_DefaultValue(t *testing.T) {
	g := newMySQLGrammar()
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

func TestMySQL_CompileCreate_UniqueColumn(t *testing.T) {
	g := newMySQLGrammar()
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

func TestMySQL_CompileCreate_WithIndexes(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("name", 255)
	bp.String("email", 255)
	bp.Index("name")
	bp.UniqueIndex("email")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "UNIQUE KEY `uniq_users_email` (`email`)") {
		t.Errorf("%q not in %q", "UNIQUE KEY `uniq_users_email` (`email`)", sql)
	}
	if !strings.Contains(sql, "KEY `idx_users_name` (`name`)") {
		t.Errorf("%q not in %q", "KEY `idx_users_name` (`name`)", sql)
	}
}

func TestMySQL_CompileCreate_WithForeignKey(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("posts")
	bp.BigInteger("user_id")
	bp.Foreign("user_id").References("id").On("users").OnDeleteAction("CASCADE").OnUpdateAction("SET NULL")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "CONSTRAINT `fk_posts_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE SET NULL") {
		t.Errorf("%q not in %q", "CONSTRAINT `fk_posts_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE SET NULL", sql)
	}
}

func TestMySQL_CompileCreate_FullTable(t *testing.T) {
	g := newMySQLGrammar()
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

	if !strings.Contains(sql, "CREATE TABLE `users`") {
		t.Errorf("%q not in %q", "CREATE TABLE `users`", sql)
	}
	if !strings.Contains(sql, "`id` BIGINT UNSIGNED AUTO_INCREMENT NOT NULL") {
		t.Errorf("%q not in %q", "`id` BIGINT UNSIGNED AUTO_INCREMENT NOT NULL", sql)
	}
	if !strings.Contains(sql, "`name` VARCHAR(255) NOT NULL") {
		t.Errorf("%q not in %q", "`name` VARCHAR(255) NOT NULL", sql)
	}
	if !strings.Contains(sql, "`email` VARCHAR(255) NOT NULL UNIQUE") {
		t.Errorf("%q not in %q", "`email` VARCHAR(255) NOT NULL UNIQUE", sql)
	}
	if !strings.Contains(sql, "`active` TINYINT(1) NOT NULL DEFAULT TRUE") {
		t.Errorf("%q not in %q", "`active` TINYINT(1) NOT NULL DEFAULT TRUE", sql)
	}
	if !strings.Contains(sql, "`created_at` TIMESTAMP") {
		t.Errorf("%q not in %q", "`created_at` TIMESTAMP", sql)
	}
	if !strings.Contains(sql, "`updated_at` TIMESTAMP") {
		t.Errorf("%q not in %q", "`updated_at` TIMESTAMP", sql)
	}
	if !strings.Contains(sql, "PRIMARY KEY (`id`)") {
		t.Errorf("%q not in %q", "PRIMARY KEY (`id`)", sql)
	}
	if !strings.Contains(sql, "KEY `idx_users_name` (`name`)") {
		t.Errorf("%q not in %q", "KEY `idx_users_name` (`name`)", sql)
	}
}

// --- CompileAlter tests ---

func TestMySQL_CompileAlter_AddColumn(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("phone", 20).Nullable()

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "ALTER TABLE `users` ADD COLUMN `phone` VARCHAR(20)" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "ALTER TABLE `users` ADD COLUMN `phone` VARCHAR(20)")
	}
}

func TestMySQL_CompileAlter_DropColumn(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.DropColumn("age")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "ALTER TABLE `users` DROP COLUMN `age`" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "ALTER TABLE `users` DROP COLUMN `age`")
	}
}

func TestMySQL_CompileAlter_RenameColumn(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.RenameColumn("name", "full_name")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "ALTER TABLE `users` RENAME COLUMN `name` TO `full_name`" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "ALTER TABLE `users` RENAME COLUMN `name` TO `full_name`")
	}
}

func TestMySQL_CompileAlter_DropIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.DropIndex("idx_users_email")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "ALTER TABLE `users` DROP INDEX `idx_users_email`" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "ALTER TABLE `users` DROP INDEX `idx_users_email`")
	}
}

func TestMySQL_CompileAlter_DropForeign(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("posts")
	bp.DropForeign("fk_posts_user_id")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "ALTER TABLE `posts` DROP FOREIGN KEY `fk_posts_user_id`" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "ALTER TABLE `posts` DROP FOREIGN KEY `fk_posts_user_id`")
	}
}

func TestMySQL_CompileAlter_AddIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.Index("name")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "CREATE INDEX `idx_users_name` ON `users` (`name`)" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "CREATE INDEX `idx_users_name` ON `users` (`name`)")
	}
}

func TestMySQL_CompileAlter_AddUniqueIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.UniqueIndex("email")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "CREATE UNIQUE INDEX `uniq_users_email` ON `users` (`email`)" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "CREATE UNIQUE INDEX `uniq_users_email` ON `users` (`email`)")
	}
}

func TestMySQL_CompileAlter_AddForeignKey(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("posts")
	bp.Foreign("user_id").References("id").On("users").OnDeleteAction("CASCADE")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if !strings.Contains(stmts[0], "ALTER TABLE `posts` ADD CONSTRAINT `fk_posts_user_id`") {
		t.Errorf("%q not in %q", "ALTER TABLE `posts` ADD CONSTRAINT `fk_posts_user_id`", stmts[0])
	}
	if !strings.Contains(stmts[0], "FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE") {
		t.Errorf("%q not in %q", "FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE", stmts[0])
	}
}

func TestMySQL_CompileAlter_MultipleOperations(t *testing.T) {
	g := newMySQLGrammar()
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

func TestMySQL_CompileDrop(t *testing.T) {
	g := newMySQLGrammar()
	if "DROP TABLE `users`" != g.CompileDrop("users") {
		t.Errorf("got %v, want %v", g.CompileDrop("users"), "DROP TABLE `users`")
	}
}

func TestMySQL_CompileDropIfExists(t *testing.T) {
	g := newMySQLGrammar()
	if "DROP TABLE IF EXISTS `users`" != g.CompileDropIfExists("users") {
		t.Errorf("got %v, want %v", g.CompileDropIfExists("users"), "DROP TABLE IF EXISTS `users`")
	}
}

// --- CompileRename test ---

func TestMySQL_CompileRename(t *testing.T) {
	g := newMySQLGrammar()
	if "RENAME TABLE `old_table` TO `new_table`" != g.CompileRename("old_table", "new_table") {
		t.Errorf("got %v, want %v", g.CompileRename("old_table", "new_table"), "RENAME TABLE `old_table` TO `new_table`")
	}
}

// --- CompileHasTable test ---

func TestMySQL_CompileHasTable(t *testing.T) {
	g := newMySQLGrammar()
	sql := g.CompileHasTable("users")
	if !strings.Contains(sql, "information_schema.tables") {
		t.Errorf("%q not in %q", "information_schema.tables", sql)
	}
	if !strings.Contains(sql, "table_name = 'users'") {
		t.Errorf("%q not in %q", "table_name = 'users'", sql)
	}
	if !strings.Contains(sql, "table_schema = DATABASE()") {
		t.Errorf("%q not in %q", "table_schema = DATABASE()", sql)
	}
}

// --- CompileHasColumn test ---

func TestMySQL_CompileHasColumn(t *testing.T) {
	g := newMySQLGrammar()
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
	if !strings.Contains(sql, "table_schema = DATABASE()") {
		t.Errorf("%q not in %q", "table_schema = DATABASE()", sql)
	}
}

// --- Grammar interface compliance ---

func TestMySQLGrammar_ImplementsGrammar(t *testing.T) {
	var _ schema.Grammar = (*MySQLGrammar)(nil)
}

// --- Edge cases ---

func TestMySQL_CompileCreate_CompositeIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("first_name", 100)
	bp.String("last_name", 100)
	bp.Index("first_name", "last_name")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "KEY `idx_users_first_name_last_name` (`first_name`, `last_name`)") {
		t.Errorf("%q not in %q", "KEY `idx_users_first_name_last_name` (`first_name`, `last_name`)", sql)
	}
}

func TestMySQL_CompileCreate_DefaultStringEscapesSingleQuotes(t *testing.T) {
	g := newMySQLGrammar()
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

func TestMySQL_CompileCreate_Timestamps(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("posts")
	bp.String("title", 255)
	bp.Timestamps()

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "`created_at` TIMESTAMP") {
		t.Errorf("%q not in %q", "`created_at` TIMESTAMP", sql)
	}
	if !strings.Contains(sql, "`updated_at` TIMESTAMP") {
		t.Errorf("%q not in %q", "`updated_at` TIMESTAMP", sql)
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

// --- New column type compilation tests ---

func TestMySQL_CompileColumnType_Enum(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum, AllowedValues: []string{"active", "inactive", "pending"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "ENUM('active','inactive','pending')" != result {
		t.Errorf("got %v, want %v", result, "ENUM('active','inactive','pending')")
	}
}

func TestMySQL_CompileColumnType_EnumSingleValue(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "role", Type: schema.TypeEnum, AllowedValues: []string{"admin"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "ENUM('admin')" != result {
		t.Errorf("got %v, want %v", result, "ENUM('admin')")
	}
}

func TestMySQL_CompileColumnType_EnumWithQuotes(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "label", Type: schema.TypeEnum, AllowedValues: []string{"it's", "they're"}}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "ENUM('it''s','they''re')" != result {
		t.Errorf("got %v, want %v", result, "ENUM('it''s','they''re')")
	}
}

func TestMySQL_CompileColumnType_EnumEmpty(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum, AllowedValues: []string{}}
	_, err := g.CompileColumnType(col)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "enum column requires at least one allowed value") {
		t.Errorf("%q not in %q", "enum column requires at least one allowed value", err.Error())
	}
}

func TestMySQL_CompileColumnType_EnumNilValues(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "status", Type: schema.TypeEnum}
	_, err := g.CompileColumnType(col)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "enum column requires at least one allowed value") {
		t.Errorf("%q not in %q", "enum column requires at least one allowed value", err.Error())
	}
}

func TestMySQL_CompileColumnType_Char(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar, Length: 10}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "CHAR(10)" != result {
		t.Errorf("got %v, want %v", result, "CHAR(10)")
	}
}

func TestMySQL_CompileColumnType_CharDefaultLength(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "CHAR(255)" != result {
		t.Errorf("got %v, want %v", result, "CHAR(255)")
	}
}

func TestMySQL_CompileColumnType_CharZeroLength(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "code", Type: schema.TypeChar, Length: 0}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "CHAR(255)" != result {
		t.Errorf("got %v, want %v", result, "CHAR(255)")
	}
}

func TestMySQL_CompileColumnType_LongText(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "content", Type: schema.TypeLongText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "LONGTEXT" != result {
		t.Errorf("got %v, want %v", result, "LONGTEXT")
	}
}

func TestMySQL_CompileColumnType_MediumText(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "description", Type: schema.TypeMediumText}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "MEDIUMTEXT" != result {
		t.Errorf("got %v, want %v", result, "MEDIUMTEXT")
	}
}

func TestMySQL_CompileColumnType_TinyInt(t *testing.T) {
	g := newMySQLGrammar()
	col := schema.ColumnDefinition{Name: "priority", Type: schema.TypeTinyInt}
	result, err := g.CompileColumnType(col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "TINYINT" != result {
		t.Errorf("got %v, want %v", result, "TINYINT")
	}
}

func TestMySQL_CompileColumnType_SmallInt(t *testing.T) {
	g := newMySQLGrammar()
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

func TestMySQL_CompileCreate_WithEnumColumn(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("users")
	bp.String("name", 255)
	bp.Enum("status", []string{"active", "inactive"})

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "ENUM('active','inactive')") {
		t.Errorf("%q not in %q", "ENUM('active','inactive')", sql)
	}
}

func TestMySQL_CompileCreate_WithCharColumn(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("countries")
	bp.Char("code", 2)

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "`code` CHAR(2)") {
		t.Errorf("%q not in %q", "`code` CHAR(2)", sql)
	}
}

func TestMySQL_CompileCreate_WithNewTextTypes(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("articles")
	bp.LongText("body")
	bp.MediumText("summary")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "LONGTEXT") {
		t.Errorf("%q not in %q", "LONGTEXT", sql)
	}
	if !strings.Contains(sql, "MEDIUMTEXT") {
		t.Errorf("%q not in %q", "MEDIUMTEXT", sql)
	}
}

func TestMySQL_CompileCreate_WithNewIntTypes(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("metrics")
	bp.TinyInt("priority")
	bp.SmallInt("score")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "`priority` TINYINT") {
		t.Errorf("%q not in %q", "`priority` TINYINT", sql)
	}
	if !strings.Contains(sql, "`score` SMALLINT") {
		t.Errorf("%q not in %q", "`score` SMALLINT", sql)
	}
}

// --- Fulltext and Spatial index tests ---

func TestMySQL_CompileCreate_WithFulltextIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("articles")
	bp.String("title", 255)
	bp.FulltextIndex("title")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "FULLTEXT KEY `ft_articles_title` (`title`)") {
		t.Errorf("%q not in %q", "FULLTEXT KEY `ft_articles_title` (`title`)", sql)
	}
}

func TestMySQL_CompileCreate_WithSpatialIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("locations")
	bp.String("coords", 255)
	bp.SpatialIndex("coords")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "SPATIAL KEY `sp_locations_coords` (`coords`)") {
		t.Errorf("%q not in %q", "SPATIAL KEY `sp_locations_coords` (`coords`)", sql)
	}
}

func TestMySQL_CompileAlter_AddFulltextIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("articles")
	bp.FulltextIndex("title")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "CREATE FULLTEXT INDEX `ft_articles_title` ON `articles` (`title`)" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "CREATE FULLTEXT INDEX `ft_articles_title` ON `articles` (`title`)")
	}
}

func TestMySQL_CompileAlter_AddSpatialIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("locations")
	bp.SpatialIndex("coords")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "CREATE SPATIAL INDEX `sp_locations_coords` ON `locations` (`coords`)" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "CREATE SPATIAL INDEX `sp_locations_coords` ON `locations` (`coords`)")
	}
}

func TestMySQL_CompileCreate_WithMultiColumnFulltextIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("articles")
	bp.String("title", 255)
	bp.Text("body")
	bp.FulltextIndex("title", "body")

	sql, err := joinCreate(g.CompileCreate(bp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "FULLTEXT KEY `ft_articles_title_body` (`title`, `body`)") {
		t.Errorf("%q not in %q", "FULLTEXT KEY `ft_articles_title_body` (`title`, `body`)", sql)
	}
}

func TestMySQL_CompileAlter_AddMultiColumnSpatialIndex(t *testing.T) {
	g := newMySQLGrammar()
	bp := schema.NewBlueprint("locations")
	bp.SpatialIndex("lat", "lng")

	stmts, err := g.CompileAlter(bp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want %d", len(stmts), 1)
	}
	if "CREATE SPATIAL INDEX `sp_locations_lat_lng` ON `locations` (`lat`, `lng`)" != stmts[0] {
		t.Errorf("got %v, want %v", stmts[0], "CREATE SPATIAL INDEX `sp_locations_lat_lng` ON `locations` (`lat`, `lng`)")
	}
}
