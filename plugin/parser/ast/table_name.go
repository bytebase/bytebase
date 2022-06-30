package ast

// TableDef is the strcut for table.
type TableDef struct {
	node

	Catalog string
	Schema  string
	Name    string
}
