package schema

import "strings"

// Qualified names are used for the migration tracking table only: there a
// dot separates a schema (PostgreSQL schema, MySQL database, SQLite attached
// database) from the table name, so WithTableName("app.migrations") keeps the
// tracking table in schema "app". Names passed to the schema Builder (Create,
// Alter, Drop, HasTable, ...) are always quoted as a single identifier.

// qualifiedParts splits name on "." and drops empty parts.
func qualifiedParts(name string) []string {
	parts := strings.Split(name, ".")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// QuoteQualified quotes a possibly schema-qualified name: it is split on ".",
// each non-empty part is quoted with g.QuoteIdentifier, and the parts are
// joined with ".". "app.migrations" becomes "app"."migrations" on PostgreSQL
// and SQLite and `app`.`migrations` on MySQL; a name without a dot is quoted
// exactly as QuoteIdentifier would. A part cannot itself contain a dot.
func QuoteQualified(g Grammar, name string) string {
	parts := qualifiedParts(name)
	if len(parts) == 0 {
		return g.QuoteIdentifier(name)
	}
	for i, p := range parts {
		parts[i] = g.QuoteIdentifier(p)
	}
	return strings.Join(parts, ".")
}

// SplitQualified splits a possibly schema-qualified name into its schema and
// table parts using the same rules as QuoteQualified: the table is the last
// non-empty part and the schema the one before it ("" when the name has no
// schema). For "app.migrations" it returns ("app", "migrations"); for
// "migrations" it returns ("", "migrations").
func SplitQualified(name string) (schemaName, table string) {
	parts := qualifiedParts(name)
	switch len(parts) {
	case 0:
		return "", name
	case 1:
		return "", parts[0]
	default:
		return parts[len(parts)-2], parts[len(parts)-1]
	}
}
