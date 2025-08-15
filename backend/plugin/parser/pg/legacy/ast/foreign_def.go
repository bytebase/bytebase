package ast

// ForeignMatchType is the match type for referencing columns and referenced columns.
type ForeignMatchType int

const (
	// ForeignMatchTypeSimple is the match type which allows any of the foreign key columns to be null.
	ForeignMatchTypeSimple ForeignMatchType = iota
	// ForeignMatchTypeFull is the match type which doesn't allow one column of a multi-column foreign key to be null unless all foreign key columns are null.
	ForeignMatchTypeFull
	// ForeignMatchTypePartial is not yet implemented.
	ForeignMatchTypePartial
)

// ForeignDef is struct for foreign key reference.
type ForeignDef struct {
	node

	Table      *TableDef
	ColumnList []string
	MatchType  ForeignMatchType
	OnDelete   *ReferentialActionDef
	OnUpdate   *ReferentialActionDef
}
