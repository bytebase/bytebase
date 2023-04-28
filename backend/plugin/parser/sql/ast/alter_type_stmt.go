package ast

// AlterTypeStmt is the struct for ALTER TYPE statements.
type AlterTypeStmt struct {
	ddl

	Type          *TypeNameDef
	AlterItemList []Node
}
