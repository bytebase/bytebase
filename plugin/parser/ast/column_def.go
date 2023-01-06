package ast

// ColumnDef is struct for column definition.
type ColumnDef struct {
	node

	ColumnName     string
	Type           DataType
	Collation      *CollationNameDef
	ConstraintList []*ConstraintDef
}

// CollationNameDef is the struct for collation name.
type CollationNameDef struct {
	node

	Schema string
	Name   string
}
