package ast

// CreateTableStmt is the struct for create table stmt.
type CreateTableStmt struct {
	ddl

	IfNotExists    bool
	Name           *TableDef
	ColumnList     []*ColumnDef
	ConstraintList []*ConstraintDef
	PartitionDef   *PartitionDef
}
