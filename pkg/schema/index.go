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
