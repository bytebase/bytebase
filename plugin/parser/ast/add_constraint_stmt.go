package ast

// AddConstraintStmt is the struct for add constraint statement.
type AddConstraintStmt struct {
	node

	Table      *TableDef
	Constraint *ConstraintDef
}
