package ast

// ForeignDef is struct for foreign key reference.
type ForeignDef struct {
	node

	Table      *TableDef
	ColumnList []string
}
