package ast

// DatabaseOptionType is the type for database options.
type DatabaseOptionType int

const (
	// DatabaseOptionEncoding is the type for the database encoding option.
	DatabaseOptionEncoding DatabaseOptionType = iota
)

// DatabaseOptionDef is the struct for database option.
type DatabaseOptionDef struct {
	node

	Type  DatabaseOptionType
	Value string
}
