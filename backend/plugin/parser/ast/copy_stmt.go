package ast

// CopyStmt is the struct for copy statement.
type CopyStmt struct {
	dml

	Table    *TableDef
	FilePath string
}
