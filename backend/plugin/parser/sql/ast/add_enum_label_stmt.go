package ast

// PositionType is the type of new label position.
type PositionType int

const (
	// PositionTypeEnd is the type of new label at the end of the list labels. It's default label.
	PositionTypeEnd PositionType = iota
	// PositionTypeAfter is the type of after neighbor label.
	PositionTypeAfter
	// PositionTypeBefore is the type of before neighbor label.
	PositionTypeBefore
)

// AddEnumLabelStmt is the struct of add enum label statements.
type AddEnumLabelStmt struct {
	node

	EnumType      *TypeNameDef
	NewLabel      string
	Position      PositionType
	NeighborLabel string
}
