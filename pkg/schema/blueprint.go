package schema

import "fmt"

// CommandType represents the type of alter command.
type CommandType int

const (
	CommandDropColumn CommandType = iota
	CommandRenameColumn
	CommandDropIndex
	CommandDropForeign
)

// Command represents an alter-table command.
type Command struct {
	Type CommandType
	Name string // column/index/foreign key name to drop, or "from" name for rename
	To   string // "to" name for rename
}

// Blueprint represents a table definition with columns, indexes, foreign keys, and commands.
type Blueprint struct {
	table       string
	columns     []ColumnDefinition
	indexes     []IndexDefinition
	foreignKeys []ForeignKeyDefinition
	commands    []Command
}

// NewBlueprint creates a new Blueprint for the given table.
func NewBlueprint(table string) *Blueprint {
	return &Blueprint{table: table}
}

// Table returns the table name.
func (bp *Blueprint) Table() string {
	return bp.table
}

// Columns returns a copy of the column definitions.
func (bp *Blueprint) Columns() []ColumnDefinition {
	out := make([]ColumnDefinition, len(bp.columns))
	copy(out, bp.columns)
	return out
}

// Indexes returns a copy of the index definitions.
func (bp *Blueprint) Indexes() []IndexDefinition {
	out := make([]IndexDefinition, len(bp.indexes))
	copy(out, bp.indexes)
	return out
}

// ForeignKeys returns a copy of the foreign key definitions.
func (bp *Blueprint) ForeignKeys() []ForeignKeyDefinition {
	out := make([]ForeignKeyDefinition, len(bp.foreignKeys))
	copy(out, bp.foreignKeys)
	return out
}

// Commands returns a copy of the alter commands.
func (bp *Blueprint) Commands() []Command {
	out := make([]Command, len(bp.commands))
	copy(out, bp.commands)
	return out
}

// addColumn adds a column and returns a pointer to it for chaining.
func (bp *Blueprint) addColumn(name string, colType ColumnType) *ColumnDefinition {
	col := ColumnDefinition{
		Name: name,
		Type: colType,
	}
	bp.columns = append(bp.columns, col)
	return &bp.columns[len(bp.columns)-1]
}

// --- Column type methods ---

// ID adds an auto-incrementing big integer primary key column named "id".
func (bp *Blueprint) ID() *ColumnDefinition {
	col := bp.addColumn("id", TypeBigInteger)
	col.IsPrimary = true
	col.IsAutoIncrement = true
	col.IsUnsigned = true
	return col
}

// String adds a VARCHAR column with the given name and length.
func (bp *Blueprint) String(name string, length int) *ColumnDefinition {
	col := bp.addColumn(name, TypeString)
	col.Length = length
	return col
}

// Text adds a TEXT column.
func (bp *Blueprint) Text(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeText)
}

// Integer adds an INTEGER column.
func (bp *Blueprint) Integer(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeInteger)
}

// BigInteger adds a BIGINT column.
func (bp *Blueprint) BigInteger(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeBigInteger)
}

// Boolean adds a BOOLEAN column.
func (bp *Blueprint) Boolean(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeBoolean)
}

// Timestamp adds a TIMESTAMP column.
func (bp *Blueprint) Timestamp(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeTimestamp)
}

// Date adds a DATE column.
func (bp *Blueprint) Date(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeDate)
}

// Decimal adds a DECIMAL column with the given precision and scale.
func (bp *Blueprint) Decimal(name string, precision, scale int) *ColumnDefinition {
	col := bp.addColumn(name, TypeDecimal)
	col.Precision = precision
	col.Scale = scale
	return col
}

// Float adds a FLOAT column.
func (bp *Blueprint) Float(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeFloat)
}

// UUID adds a UUID column.
func (bp *Blueprint) UUID(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeUUID)
}

// JSON adds a JSON column.
func (bp *Blueprint) JSON(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeJSON)
}

// Binary adds a BINARY/BLOB column.
func (bp *Blueprint) Binary(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeBinary)
}

// Enum adds an ENUM column with the given allowed values.
func (bp *Blueprint) Enum(name string, allowed []string) *ColumnDefinition {
	col := bp.addColumn(name, TypeEnum)
	col.AllowedValues = allowed
	return col
}

// Char adds a fixed-length CHAR column with the given length.
func (bp *Blueprint) Char(name string, length int) *ColumnDefinition {
	col := bp.addColumn(name, TypeChar)
	col.Length = length
	return col
}

// LongText adds a LONGTEXT column.
func (bp *Blueprint) LongText(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeLongText)
}

// MediumText adds a MEDIUMTEXT column.
func (bp *Blueprint) MediumText(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeMediumText)
}

// TinyInt adds a TINYINT column.
func (bp *Blueprint) TinyInt(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeTinyInt)
}

// SmallInt adds a SMALLINT column.
func (bp *Blueprint) SmallInt(name string) *ColumnDefinition {
	return bp.addColumn(name, TypeSmallInt)
}

// Timestamps adds created_at and updated_at nullable timestamp columns.
func (bp *Blueprint) Timestamps() {
	bp.Timestamp("created_at").Nullable()
	bp.Timestamp("updated_at").Nullable()
}

// SoftDeletes adds a deleted_at nullable timestamp column.
func (bp *Blueprint) SoftDeletes() {
	bp.Timestamp("deleted_at").Nullable()
}

// --- Index methods ---

// Index adds a composite index on the given columns.
func (bp *Blueprint) Index(columns ...string) *IndexDefinition {
	name := bp.generateIndexName(columns, "idx")
	idx := IndexDefinition{
		Name:    name,
		Columns: columns,
		Type:    IndexRegular,
	}
	bp.indexes = append(bp.indexes, idx)
	return &bp.indexes[len(bp.indexes)-1]
}

// UniqueIndex adds a unique composite index on the given columns.
func (bp *Blueprint) UniqueIndex(columns ...string) *IndexDefinition {
	name := bp.generateIndexName(columns, "uniq")
	idx := IndexDefinition{
		Name:    name,
		Columns: columns,
		Type:    IndexUnique,
	}
	bp.indexes = append(bp.indexes, idx)
	return &bp.indexes[len(bp.indexes)-1]
}

// FulltextIndex adds a fulltext index on the given columns.
func (bp *Blueprint) FulltextIndex(columns ...string) *IndexDefinition {
	name := bp.generateIndexName(columns, "ft")
	idx := IndexDefinition{
		Name:    name,
		Columns: columns,
		Type:    IndexFulltext,
	}
	bp.indexes = append(bp.indexes, idx)
	return &bp.indexes[len(bp.indexes)-1]
}

// SpatialIndex adds a spatial index on the given columns.
func (bp *Blueprint) SpatialIndex(columns ...string) *IndexDefinition {
	name := bp.generateIndexName(columns, "sp")
	idx := IndexDefinition{
		Name:    name,
		Columns: columns,
		Type:    IndexSpatial,
	}
	bp.indexes = append(bp.indexes, idx)
	return &bp.indexes[len(bp.indexes)-1]
}

// generateIndexName creates a conventional index name from table and columns.
func (bp *Blueprint) generateIndexName(columns []string, prefix string) string {
	name := prefix + "_" + bp.table
	for _, col := range columns {
		name += "_" + col
	}
	return name
}

// --- Foreign key methods ---

// Foreign adds a foreign key constraint on the given column.
func (bp *Blueprint) Foreign(column string) *ForeignKeyDefinition {
	fk := ForeignKeyDefinition{
		Column: column,
		Name:   fmt.Sprintf("fk_%s_%s", bp.table, column),
	}
	bp.foreignKeys = append(bp.foreignKeys, fk)
	return &bp.foreignKeys[len(bp.foreignKeys)-1]
}

// --- Alter methods ---

// DropColumn adds a command to drop a column.
func (bp *Blueprint) DropColumn(name string) {
	bp.commands = append(bp.commands, Command{
		Type: CommandDropColumn,
		Name: name,
	})
}

// RenameColumn adds a command to rename a column.
func (bp *Blueprint) RenameColumn(from, to string) {
	bp.commands = append(bp.commands, Command{
		Type: CommandRenameColumn,
		Name: from,
		To:   to,
	})
}

// DropIndex adds a command to drop an index.
func (bp *Blueprint) DropIndex(name string) {
	bp.commands = append(bp.commands, Command{
		Type: CommandDropIndex,
		Name: name,
	})
}

// DropForeign adds a command to drop a foreign key constraint.
func (bp *Blueprint) DropForeign(name string) {
	bp.commands = append(bp.commands, Command{
		Type: CommandDropForeign,
		Name: name,
	})
}
