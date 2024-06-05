package ast

type CreateViewStmt struct {
	ddl

	Name    *TableDef
	Aliases []string
	Select  *SelectStmt
	Replace bool
}
