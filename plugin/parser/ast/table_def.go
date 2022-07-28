package ast

// TableType is the type for table.
type TableType int

const (
	// TableTypeUnknown is the type for table or view, which means we can't be sure of it.
	TableTypeUnknown TableType = iota
	// TableTypeBaseTable is the type for table.
	TableTypeBaseTable
	// TableTypeView is the type for view.
	TableTypeView
)

// TableDef is the strcut for table.
type TableDef struct {
	node

	// Type is the table type for table: base table or view.
	Type TableType
	// Database is the name of database.
	// It's also called "catalog" in PostgreSQL.
	Database string
	// Schema is a PostgreSQL specific field.
	// See https://www.postgresql.org/docs/current/ddl-schemas.html.
	Schema string
	// Name is the name of the table.
	Name string
}
