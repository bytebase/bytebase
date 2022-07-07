package ast

// IndexDef is the struct for index definition.
type IndexDef struct {
	node

	Name    string
	Table   *TableDef
	Unique  bool
	KeyList []*IndexKeyDef
}
