package ast

// DropTableStmt is the struct for drop table statement.
type DropTableStmt struct {
	node

	TableList []*TableDef
}
