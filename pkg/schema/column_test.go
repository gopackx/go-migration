package schema

import (
	"testing"
)

func TestColumnType_String(t *testing.T) {
	tests := []struct {
		ct   ColumnType
		want string
	}{
		{TypeString, "string"},
		{TypeText, "text"},
		{TypeInteger, "integer"},
		{TypeBigInteger, "bigInteger"},
		{TypeBoolean, "boolean"},
		{TypeTimestamp, "timestamp"},
		{TypeDate, "date"},
		{TypeDecimal, "decimal"},
		{TypeFloat, "float"},
		{TypeUUID, "uuid"},
		{TypeJSON, "json"},
		{TypeBinary, "binary"},
		{ColumnType(999), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if tt.want != tt.ct.String() {
				t.Errorf("got %v, want %v", tt.ct.String(), tt.want)
			}
		})
	}
}

func TestColumnDefinition_Chainable(t *testing.T) {
	col := &ColumnDefinition{Name: "email", Type: TypeString}

	result := col.Nullable().Default("test@example.com").Unique()

	if col != result {
		t.Errorf("modifiers should return the same pointer for chaining")
	}
	if !col.IsNullable {
		t.Errorf("expected true")
	}
	if "test@example.com" != col.DefaultValue {
		t.Errorf("got %v, want %v", col.DefaultValue, "test@example.com")
	}
	if !col.IsUnique {
		t.Errorf("expected true")
	}
}

func TestColumnDefinition_AllModifiers(t *testing.T) {
	col := &ColumnDefinition{Name: "id", Type: TypeBigInteger}

	col.Primary().Unsigned().AutoIncrement().Default(0).Nullable()

	if !col.IsPrimary {
		t.Errorf("expected true")
	}
	if !col.IsUnsigned {
		t.Errorf("expected true")
	}
	if !col.IsAutoIncrement {
		t.Errorf("expected true")
	}
	if 0 != col.DefaultValue {
		t.Errorf("got %v, want %v", col.DefaultValue, 0)
	}
	if !col.IsNullable {
		t.Errorf("expected true")
	}
}
