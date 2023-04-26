package ast

import "strings"

// Decimal is the struct for decimal.
type Decimal struct {
	numericType

	// See https://www.postgresql.org/docs/14/datatype-numeric.html.
	Precision int
	Scale     int
}

// EquivalentType implements the DataType interface.
func (*Decimal) EquivalentType(tp string) bool {
	tp = strings.ToLower(tp)
	return strings.HasPrefix(tp, "decimal") || strings.HasPrefix(tp, "numeric")
}
