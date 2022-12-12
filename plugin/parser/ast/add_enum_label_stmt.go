package ast

// PositionType is the type of new value position.
type PositionType int

const (
	// PositionTypeEnd is the type of new value at the end of the list values. It's default value.
	PositionTypeEnd PositionType = iota
	// PositionTypeAfter is the type of after neighbor value.
	PositionTypeAfter
	// PositionTypeBefore is the type of before neighbor value.
	PositionTypeBefore
)

// AddEnumValueStmt is the struct of add enum value statements.
type AddEnumValueStmt struct {
	node

	EnumType      *TypeNameDef
	NewLabel      string
	Position      PositionType
	NeighborLabel string
}
