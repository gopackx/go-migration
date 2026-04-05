package schema

// RawExpression membungkus string ekspresi SQL mentah.
// Ketika digunakan sebagai DefaultValue pada ColumnDefinition,
// Grammar akan menyisipkan ekspresi langsung tanpa tanda kutip.
type RawExpression struct {
	Expression string
}

// Raw membuat RawExpression baru.
func Raw(expression string) RawExpression {
	return RawExpression{Expression: expression}
}
