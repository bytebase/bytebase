package ast

// DropExtensionStmt is the struct for drop extension statement.
type DropExtensionStmt struct {
	ddl

	NameList []string
	IfExists bool
	Behavior DropBehavior
}
