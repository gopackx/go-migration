package schema

// IndexType represents the type of a database index.
type IndexType int

const (
	IndexRegular  IndexType = iota // Regular (non-unique) index
	IndexUnique                    // Unique index
	IndexFulltext                  // Fulltext index
	IndexSpatial                   // Spatial index
)

// IndexDefinition represents an index on a database table.
type IndexDefinition struct {
	Name    string
	Columns []string
	Type    IndexType
}

// Named overrides the auto-generated index name with a custom one.
// It is chainable on the value returned by Blueprint.Index,
// UniqueIndex, FulltextIndex, and SpatialIndex. An empty name is
// ignored so the auto-generated name is preserved.
//
//	bp.Index("email").Named("idx_users_email_lower")
//
// The custom name is what you pass to Blueprint.DropIndex when removing it.
func (idx *IndexDefinition) Named(name string) *IndexDefinition {
	if name != "" {
		idx.Name = name
	}
	return idx
}
