package ast

// TableDef is the strcut for table.
type TableDef struct {
	node

	// Database is the name of database.
	// It's also called "catalog" in PostgreSQL.
	Database string
	// Schema is a PostgreSQL specific field.
	// See https://www.postgresql.org/docs/current/ddl-schemas.html.
	Schema string
	// Name is the name of the table.
	Name string
}
