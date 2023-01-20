package ast

// DropBehavior is the type for drop object statements.
type DropBehavior int

const (
	// DropBehaviorNone is the default type for drop object statements.
	DropBehaviorNone DropBehavior = iota
	// DropBehaviorCascade is the type for drop object statements with cascade.
	DropBehaviorCascade
	// DropBehaviorRestrict is the type for drop object statement with restrict.
	DropBehaviorRestrict
)

// DropSchemaStmt is the struct for drop schema statement.
type DropSchemaStmt struct {
	ddl

	IfExists   bool
	SchemaList []string
	Behavior   DropBehavior
}
