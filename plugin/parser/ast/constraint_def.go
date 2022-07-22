package ast

// ConstraintType is the type for constraints.
type ConstraintType int

const (
	// ConstraintTypeUndefined is the undefined type.
	ConstraintTypeUndefined ConstraintType = iota
	// ConstraintTypePrimary is the primary key constraint.
	ConstraintTypePrimary
	// ConstraintTypeUnique is the unique constraint.
	ConstraintTypeUnique
	// ConstraintTypeForeign is the foreign key constraint.
	ConstraintTypeForeign
	// ConstraintTypePrimaryUsingIndex is the primary key constraint only for the PostgreSQL table_constraint_using_index.
	// See https://www.postgresql.org/docs/current/sql-altertable.html.
	ConstraintTypePrimaryUsingIndex
	// ConstraintTypeUniqueUsingIndex is the unique constraint only for the PostgreSQL table_constraint_using_index.
	// See https://www.postgresql.org/docs/current/sql-altertable.html.
	ConstraintTypeUniqueUsingIndex
	// ConstraintTypeNotNull is the not null constraint.
	ConstraintTypeNotNull
)

// ConstraintDef is struct for constraint definition.
// For PRIMARY:
//     Name:    It's the PK constraint name.
//     KeyList: It's the name list of the columns in PK.
//
// For UNIQUE
//     Name:    It's the UK constraint name.
//     KeyList: It's the name list of the columns in UK.
//
// For Foreign
//     Name:    It's the FK constraint name.
//     KeyList: It's the name list of the columns in FK.
//     Foreign: It's the reference content for this FK.
type ConstraintDef struct {
	node

	Type ConstraintType
	Name string
	// KeyList is the list for constraint key.
	// Whether it is a column constraint or a table constraint,
	// there is a corresponding key list.
	KeyList []string
	// Foreign is a FOREIGN type specific field.
	Foreign *ForeignDef
	// IndexName is a ConstraintTypePrimaryUsingIndex or ConstraintTypeUniqueUsingIndex type specific field.
	// See https://www.postgresql.org/docs/current/sql-altertable.html.
	IndexName string
}
