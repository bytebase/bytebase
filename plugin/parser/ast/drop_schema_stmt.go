package ast

// DropSchemaType is the type for drop schema statement.
type DropSchemaType int

const (
	// DropSchemaTypeNone is the default type for drop schema statement.
	DropSchemaTypeNone DropSchemaType = iota
	// DropSchemaTypeCascade is the type for drop schema statement with cascade.
	DropSchemaTypeCascade
	// DropSchemaTypeRestrict is the type for drop schema statement with restrict.
	DropSchemaTypeRestrict
)

// DropSchemaStmt is the struct for drop schema statement.
type DropSchemaStmt struct {
	ddl

	IfExists   bool
	SchemaList []string
	Type       DropSchemaType
}
