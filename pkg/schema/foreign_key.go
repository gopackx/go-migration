package schema

// ForeignKeyDefinition represents a foreign key constraint on a database table.
type ForeignKeyDefinition struct {
	Column    string
	RefTable  string
	RefColumn string
	OnDelete  string
	OnUpdate  string
	Name      string
}

// References sets the referenced column.
func (fk *ForeignKeyDefinition) References(column string) *ForeignKeyDefinition {
	fk.RefColumn = column
	return fk
}

// On sets the referenced table.
func (fk *ForeignKeyDefinition) On(table string) *ForeignKeyDefinition {
	fk.RefTable = table
	return fk
}

// OnDeleteAction sets the ON DELETE action.
func (fk *ForeignKeyDefinition) OnDeleteAction(action string) *ForeignKeyDefinition {
	fk.OnDelete = action
	return fk
}

// OnUpdateAction sets the ON UPDATE action.
func (fk *ForeignKeyDefinition) OnUpdateAction(action string) *ForeignKeyDefinition {
	fk.OnUpdate = action
	return fk
}
