package ast

// DropDatabaseStmt is the struct for drop database statement.
type DropDatabaseStmt struct {
	node

	DatabaseName string
	IfExists     bool
}
