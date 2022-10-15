package ast

// CreateDatabaseStmt is the struct for create database stmt.
type CreateDatabaseStmt struct {
	ddl

	Name       string
	OptionList []*DatabaseOptionDef
}
