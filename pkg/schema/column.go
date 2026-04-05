package schema

// ColumnType represents the type of a database column.
type ColumnType int

const (
	TypeString     ColumnType = iota // VARCHAR
	TypeText                         // TEXT
	TypeInteger                      // INTEGER
	TypeBigInteger                   // BIGINT
	TypeBoolean                      // BOOLEAN
	TypeTimestamp                    // TIMESTAMP
	TypeDate                         // DATE
	TypeDecimal                      // DECIMAL
	TypeFloat                        // FLOAT
	TypeUUID                         // UUID
	TypeJSON                         // JSON
	TypeBinary                       // BINARY/BLOB
	TypeEnum                         // ENUM
	TypeChar                         // CHAR(n)
	TypeLongText                     // LONGTEXT
	TypeMediumText                   // MEDIUMTEXT
	TypeTinyInt                      // TINYINT
	TypeSmallInt                     // SMALLINT
)

// String returns the string representation of a ColumnType.
func (ct ColumnType) String() string {
	switch ct {
	case TypeString:
		return "string"
	case TypeText:
		return "text"
	case TypeInteger:
		return "integer"
	case TypeBigInteger:
		return "bigInteger"
	case TypeBoolean:
		return "boolean"
	case TypeTimestamp:
		return "timestamp"
	case TypeDate:
		return "date"
	case TypeDecimal:
		return "decimal"
	case TypeFloat:
		return "float"
	case TypeUUID:
		return "uuid"
	case TypeJSON:
		return "json"
	case TypeBinary:
		return "binary"
	case TypeEnum:
		return "enum"
	case TypeChar:
		return "char"
	case TypeLongText:
		return "longText"
	case TypeMediumText:
		return "mediumText"
	case TypeTinyInt:
		return "tinyInt"
	case TypeSmallInt:
		return "smallInt"
	default:
		return "unknown"
	}
}

// ColumnDefinition represents a column in a database table.
type ColumnDefinition struct {
	Name            string
	Type            ColumnType
	Length          int
	Precision       int
	Scale           int
	IsNullable      bool
	DefaultValue    any
	IsPrimary       bool
	IsUnique        bool
	IsUnsigned      bool
	IsAutoIncrement bool
	AllowedValues   []string
}

// Nullable marks the column as nullable.
func (cd *ColumnDefinition) Nullable() *ColumnDefinition {
	cd.IsNullable = true
	return cd
}

// Default sets the default value for the column.
func (cd *ColumnDefinition) Default(value any) *ColumnDefinition {
	cd.DefaultValue = value
	return cd
}

// DefaultRaw sets the default value for the column to a raw SQL expression.
// If expression is empty, the default value is not changed.
func (cd *ColumnDefinition) DefaultRaw(expression string) *ColumnDefinition {
	if expression == "" {
		return cd
	}
	cd.DefaultValue = RawExpression{Expression: expression}
	return cd
}

// Primary marks the column as a primary key.
func (cd *ColumnDefinition) Primary() *ColumnDefinition {
	cd.IsPrimary = true
	return cd
}

// Unique marks the column as unique.
func (cd *ColumnDefinition) Unique() *ColumnDefinition {
	cd.IsUnique = true
	return cd
}

// Unsigned marks the column as unsigned.
func (cd *ColumnDefinition) Unsigned() *ColumnDefinition {
	cd.IsUnsigned = true
	return cd
}

// AutoIncrement marks the column as auto-incrementing.
func (cd *ColumnDefinition) AutoIncrement() *ColumnDefinition {
	cd.IsAutoIncrement = true
	return cd
}
