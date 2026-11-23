package base

import "fmt"

// SchemaResource is the resource of the schema.
type SchemaResource struct {
	Database string
	Schema   string
	Table    string

	// LinkedServer is the remote reference as the statement wrote it: an MSSQL linked server or
	// an Oracle database link. Empty for a table on the connected server.
	LinkedServer string
}

// String implements fmt.Stringer interface.
func (r SchemaResource) String() string {
	return fmt.Sprintf("%s.%s.%s", r.Database, r.Schema, r.Table)
}
