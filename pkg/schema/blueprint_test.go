package schema

import (
	"reflect"
	"testing"
)

func TestNewBlueprint(t *testing.T) {
	bp := NewBlueprint("users")
	if "users" != bp.Table() {
		t.Errorf("got %v, want %v", bp.Table(), "users")
	}
	if len(bp.Columns()) != 0 {
		t.Errorf("expected empty, got %v", bp.Columns())
	}
	if len(bp.Indexes()) != 0 {
		t.Errorf("expected empty, got %v", bp.Indexes())
	}
	if len(bp.ForeignKeys()) != 0 {
		t.Errorf("expected empty, got %v", bp.ForeignKeys())
	}
	if len(bp.Commands()) != 0 {
		t.Errorf("expected empty, got %v", bp.Commands())
	}
}

// --- Column type methods ---

func TestBlueprint_ID(t *testing.T) {
	bp := NewBlueprint("users")
	col := bp.ID()

	if "id" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "id")
	}
	if TypeBigInteger != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeBigInteger)
	}
	if !col.IsPrimary {
		t.Errorf("expected true")
	}
	if !col.IsAutoIncrement {
		t.Errorf("expected true")
	}
	if !col.IsUnsigned {
		t.Errorf("expected true")
	}
}

func TestBlueprint_String(t *testing.T) {
	bp := NewBlueprint("users")
	col := bp.String("name", 255)

	if "name" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "name")
	}
	if TypeString != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeString)
	}
	if 255 != col.Length {
		t.Errorf("got %v, want %v", col.Length, 255)
	}
}

func TestBlueprint_Text(t *testing.T) {
	bp := NewBlueprint("posts")
	col := bp.Text("body")

	if "body" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "body")
	}
	if TypeText != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeText)
	}
}

func TestBlueprint_Integer(t *testing.T) {
	bp := NewBlueprint("users")
	col := bp.Integer("age")

	if "age" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "age")
	}
	if TypeInteger != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeInteger)
	}
}

func TestBlueprint_BigInteger(t *testing.T) {
	bp := NewBlueprint("orders")
	col := bp.BigInteger("total")

	if "total" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "total")
	}
	if TypeBigInteger != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeBigInteger)
	}
}

func TestBlueprint_Boolean(t *testing.T) {
	bp := NewBlueprint("users")
	col := bp.Boolean("active")

	if "active" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "active")
	}
	if TypeBoolean != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeBoolean)
	}
}

func TestBlueprint_Timestamp(t *testing.T) {
	bp := NewBlueprint("events")
	col := bp.Timestamp("occurred_at")

	if "occurred_at" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "occurred_at")
	}
	if TypeTimestamp != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeTimestamp)
	}
}

func TestBlueprint_Date(t *testing.T) {
	bp := NewBlueprint("events")
	col := bp.Date("event_date")

	if "event_date" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "event_date")
	}
	if TypeDate != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeDate)
	}
}

func TestBlueprint_Decimal(t *testing.T) {
	bp := NewBlueprint("products")
	col := bp.Decimal("price", 10, 2)

	if "price" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "price")
	}
	if TypeDecimal != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeDecimal)
	}
	if 10 != col.Precision {
		t.Errorf("got %v, want %v", col.Precision, 10)
	}
	if 2 != col.Scale {
		t.Errorf("got %v, want %v", col.Scale, 2)
	}
}

func TestBlueprint_Float(t *testing.T) {
	bp := NewBlueprint("measurements")
	col := bp.Float("value")

	if "value" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "value")
	}
	if TypeFloat != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeFloat)
	}
}

func TestBlueprint_UUID(t *testing.T) {
	bp := NewBlueprint("users")
	col := bp.UUID("external_id")

	if "external_id" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "external_id")
	}
	if TypeUUID != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeUUID)
	}
}

func TestBlueprint_JSON(t *testing.T) {
	bp := NewBlueprint("settings")
	col := bp.JSON("metadata")

	if "metadata" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "metadata")
	}
	if TypeJSON != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeJSON)
	}
}

func TestBlueprint_Binary(t *testing.T) {
	bp := NewBlueprint("files")
	col := bp.Binary("data")

	if "data" != col.Name {
		t.Errorf("got %v, want %v", col.Name, "data")
	}
	if TypeBinary != col.Type {
		t.Errorf("got %v, want %v", col.Type, TypeBinary)
	}
}

// --- Convenience methods ---

func TestBlueprint_Timestamps(t *testing.T) {
	bp := NewBlueprint("users")
	bp.Timestamps()

	cols := bp.Columns()
	if len(cols) != 2 {
		t.Fatalf("len = %d, want %d", len(cols), 2)
	}

	if "created_at" != cols[0].Name {
		t.Errorf("got %v, want %v", cols[0].Name, "created_at")
	}
	if TypeTimestamp != cols[0].Type {
		t.Errorf("got %v, want %v", cols[0].Type, TypeTimestamp)
	}
	if !cols[0].IsNullable {
		t.Errorf("expected true")
	}

	if "updated_at" != cols[1].Name {
		t.Errorf("got %v, want %v", cols[1].Name, "updated_at")
	}
	if TypeTimestamp != cols[1].Type {
		t.Errorf("got %v, want %v", cols[1].Type, TypeTimestamp)
	}
	if !cols[1].IsNullable {
		t.Errorf("expected true")
	}
}

func TestBlueprint_SoftDeletes(t *testing.T) {
	bp := NewBlueprint("users")
	bp.SoftDeletes()

	cols := bp.Columns()
	if len(cols) != 1 {
		t.Fatalf("len = %d, want %d", len(cols), 1)
	}

	if "deleted_at" != cols[0].Name {
		t.Errorf("got %v, want %v", cols[0].Name, "deleted_at")
	}
	if TypeTimestamp != cols[0].Type {
		t.Errorf("got %v, want %v", cols[0].Type, TypeTimestamp)
	}
	if !cols[0].IsNullable {
		t.Errorf("expected true")
	}
}

// --- Chaining modifiers through Blueprint ---

func TestBlueprint_ColumnModifierChaining(t *testing.T) {
	bp := NewBlueprint("users")
	bp.String("email", 255).Nullable().Unique().Default("")

	cols := bp.Columns()
	if len(cols) != 1 {
		t.Fatalf("len = %d, want %d", len(cols), 1)
	}
	if !cols[0].IsNullable {
		t.Errorf("expected true")
	}
	if !cols[0].IsUnique {
		t.Errorf("expected true")
	}
	if "" != cols[0].DefaultValue {
		t.Errorf("got %v, want %v", cols[0].DefaultValue, "")
	}
}

// --- Index methods ---

func TestBlueprint_Index(t *testing.T) {
	bp := NewBlueprint("users")
	idx := bp.Index("email")

	if "idx_users_email" != idx.Name {
		t.Errorf("got %v, want %v", idx.Name, "idx_users_email")
	}
	if !reflect.DeepEqual([]string{"email"}, idx.Columns) {
		t.Errorf("got %v, want %v", idx.Columns, []string{"email"})
	}
	if IndexRegular != idx.Type {
		t.Errorf("got %v, want %v", idx.Type, IndexRegular)
	}
}

func TestBlueprint_Index_Composite(t *testing.T) {
	bp := NewBlueprint("users")
	idx := bp.Index("first_name", "last_name")

	if "idx_users_first_name_last_name" != idx.Name {
		t.Errorf("got %v, want %v", idx.Name, "idx_users_first_name_last_name")
	}
	if !reflect.DeepEqual([]string{"first_name", "last_name"}, idx.Columns) {
		t.Errorf("got %v, want %v", idx.Columns, []string{"first_name", "last_name"})
	}
	if IndexRegular != idx.Type {
		t.Errorf("got %v, want %v", idx.Type, IndexRegular)
	}
}

func TestBlueprint_UniqueIndex(t *testing.T) {
	bp := NewBlueprint("users")
	idx := bp.UniqueIndex("email")

	if "uniq_users_email" != idx.Name {
		t.Errorf("got %v, want %v", idx.Name, "uniq_users_email")
	}
	if !reflect.DeepEqual([]string{"email"}, idx.Columns) {
		t.Errorf("got %v, want %v", idx.Columns, []string{"email"})
	}
	if IndexUnique != idx.Type {
		t.Errorf("got %v, want %v", idx.Type, IndexUnique)
	}
}

func TestBlueprint_Index_Named(t *testing.T) {
	bp := NewBlueprint("users")
	idx := bp.Index("email").Named("idx_users_email_lower")

	if "idx_users_email_lower" != idx.Name {
		t.Errorf("got %v, want %v", idx.Name, "idx_users_email_lower")
	}

	// The custom name is what gets stored on the blueprint, not the default.
	stored := bp.Indexes()
	if len(stored) != 1 {
		t.Fatalf("len = %d, want %d", len(stored), 1)
	}
	if "idx_users_email_lower" != stored[0].Name {
		t.Errorf("got %v, want %v", stored[0].Name, "idx_users_email_lower")
	}
}

func TestBlueprint_Index_Named_EmptyKeepsDefault(t *testing.T) {
	bp := NewBlueprint("users")
	idx := bp.UniqueIndex("email").Named("")

	if "uniq_users_email" != idx.Name {
		t.Errorf("got %v, want %v", idx.Name, "uniq_users_email")
	}
}

// --- Foreign key methods ---

func TestBlueprint_Foreign(t *testing.T) {
	bp := NewBlueprint("posts")
	fk := bp.Foreign("user_id").References("id").On("users").OnDeleteAction("CASCADE").OnUpdateAction("SET NULL")

	if "user_id" != fk.Column {
		t.Errorf("got %v, want %v", fk.Column, "user_id")
	}
	if "id" != fk.RefColumn {
		t.Errorf("got %v, want %v", fk.RefColumn, "id")
	}
	if "users" != fk.RefTable {
		t.Errorf("got %v, want %v", fk.RefTable, "users")
	}
	if "CASCADE" != fk.OnDelete {
		t.Errorf("got %v, want %v", fk.OnDelete, "CASCADE")
	}
	if "SET NULL" != fk.OnUpdate {
		t.Errorf("got %v, want %v", fk.OnUpdate, "SET NULL")
	}
	if "fk_posts_user_id" != fk.Name {
		t.Errorf("got %v, want %v", fk.Name, "fk_posts_user_id")
	}
}

func TestBlueprint_Foreign_DefaultName(t *testing.T) {
	bp := NewBlueprint("comments")
	fk := bp.Foreign("post_id")

	if "fk_comments_post_id" != fk.Name {
		t.Errorf("got %v, want %v", fk.Name, "fk_comments_post_id")
	}
}

// --- Alter methods ---

func TestBlueprint_DropColumn(t *testing.T) {
	bp := NewBlueprint("users")
	bp.DropColumn("age")

	cmds := bp.Commands()
	if len(cmds) != 1 {
		t.Fatalf("len = %d, want %d", len(cmds), 1)
	}
	if CommandDropColumn != cmds[0].Type {
		t.Errorf("got %v, want %v", cmds[0].Type, CommandDropColumn)
	}
	if "age" != cmds[0].Name {
		t.Errorf("got %v, want %v", cmds[0].Name, "age")
	}
}

func TestBlueprint_RenameColumn(t *testing.T) {
	bp := NewBlueprint("users")
	bp.RenameColumn("name", "full_name")

	cmds := bp.Commands()
	if len(cmds) != 1 {
		t.Fatalf("len = %d, want %d", len(cmds), 1)
	}
	if CommandRenameColumn != cmds[0].Type {
		t.Errorf("got %v, want %v", cmds[0].Type, CommandRenameColumn)
	}
	if "name" != cmds[0].Name {
		t.Errorf("got %v, want %v", cmds[0].Name, "name")
	}
	if "full_name" != cmds[0].To {
		t.Errorf("got %v, want %v", cmds[0].To, "full_name")
	}
}

func TestBlueprint_DropIndex(t *testing.T) {
	bp := NewBlueprint("users")
	bp.DropIndex("idx_users_email")

	cmds := bp.Commands()
	if len(cmds) != 1 {
		t.Fatalf("len = %d, want %d", len(cmds), 1)
	}
	if CommandDropIndex != cmds[0].Type {
		t.Errorf("got %v, want %v", cmds[0].Type, CommandDropIndex)
	}
	if "idx_users_email" != cmds[0].Name {
		t.Errorf("got %v, want %v", cmds[0].Name, "idx_users_email")
	}
}

func TestBlueprint_DropForeign(t *testing.T) {
	bp := NewBlueprint("posts")
	bp.DropForeign("fk_posts_user_id")

	cmds := bp.Commands()
	if len(cmds) != 1 {
		t.Fatalf("len = %d, want %d", len(cmds), 1)
	}
	if CommandDropForeign != cmds[0].Type {
		t.Errorf("got %v, want %v", cmds[0].Type, CommandDropForeign)
	}
	if "fk_posts_user_id" != cmds[0].Name {
		t.Errorf("got %v, want %v", cmds[0].Name, "fk_posts_user_id")
	}
}

// --- Accessor methods return copies ---

func TestBlueprint_Columns_ReturnsCopy(t *testing.T) {
	bp := NewBlueprint("users")
	bp.String("name", 100)

	cols1 := bp.Columns()
	cols1[0].Name = "mutated"

	cols2 := bp.Columns()
	if "name" != cols2[0].Name {
		t.Errorf("mutating returned slice should not affect blueprint")
	}
}

// --- Full table definition ---

func TestBlueprint_FullTableDefinition(t *testing.T) {
	bp := NewBlueprint("users")
	bp.ID()
	bp.String("name", 255)
	bp.String("email", 255).Unique()
	bp.Boolean("active").Default(true)
	bp.Timestamps()
	bp.SoftDeletes()
	bp.Index("name")
	bp.Foreign("team_id").References("id").On("teams").OnDeleteAction("CASCADE")

	if 7 != len(bp.Columns()) {
		t.Errorf("got %v, want %v", len(bp.Columns()), 7)
	}
	if 1 != len(bp.Indexes()) {
		t.Errorf("got %v, want %v", len(bp.Indexes()), 1)
	}
	if 1 != len(bp.ForeignKeys()) {
		t.Errorf("got %v, want %v", len(bp.ForeignKeys()), 1)
	}
}
