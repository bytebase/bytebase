package ast

// DropSchemaBehavior is the type for drop schema statement.
type DropSchemaBehavior int

const (
	// DropSchemaBehaviorNone is the default type for drop schema statement.
	DropSchemaBehaviorNone DropSchemaBehavior = iota
	// DropSchemaBehaviorCascade is the type for drop schema statement with cascade.
	DropSchemaBehaviorCascade
	// DropSchemaBehaviorRestrict is the type for drop schema statement with restrict.
	DropSchemaBehaviorRestrict
)

// DropSchemaStmt is the struct for drop schema statement.
type DropSchemaStmt struct {
	ddl

	IfExists   bool
	SchemaList []string
	Behavior   DropSchemaBehavior
}
