package ast

// DropColumnStmt is the struct for drop column statement.
type DropColumnStmt struct {
	node

	Table      *TableDef
	ColumnName string
	IfExists   bool
	Behavior   DropBehavior
}
