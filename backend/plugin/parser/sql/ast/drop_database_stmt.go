package ast

// DropDatabaseStmt is the struct for drop database statement.
type DropDatabaseStmt struct {
	ddl

	DatabaseName string
	IfExists     bool
}
