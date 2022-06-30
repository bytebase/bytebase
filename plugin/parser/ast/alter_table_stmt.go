package ast

// AlterTableStmt is the struct for alter table statement.
type AlterTableStmt struct {
	node

	Table         *TableDef
	AlterItemList []Node
}
