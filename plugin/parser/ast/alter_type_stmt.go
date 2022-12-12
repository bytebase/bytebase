package ast

type AlterTypeStmt struct {
	ddl

	Type          *TypeNameDef
	AlterItemList []Node
}
