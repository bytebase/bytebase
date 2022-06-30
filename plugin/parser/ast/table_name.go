package ast

// TableName is the strcut for table name.
type TableName struct {
	node

	Catalog string
	Schema  string
	Name    string
}
