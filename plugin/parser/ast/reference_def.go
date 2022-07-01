package ast

// ReferenceDef is struct for foreign key reference.
type ReferenceDef struct {
	node

	Table      *TableDef
	ColumnList []string
}
