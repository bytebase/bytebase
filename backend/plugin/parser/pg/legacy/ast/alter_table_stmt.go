package ast

// AlterTableStmt is the struct for alter table or view statement.
type AlterTableStmt struct {
	ddl

	Table         *TableDef
	AlterItemList []Node
}
