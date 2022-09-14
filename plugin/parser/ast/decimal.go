package ast

// Decimal is the struct for decimal.
type Decimal struct {
	numericType

	// See https://www.postgresql.org/docs/14/datatype-numeric.html.
	Precision int
	Scale     int
}
