package api

// DBSchema is the API message for database schema.
type DBSchema struct {
	ID int `jsonapi:"primary,dbSchema"`

	// Standard fields

	// Related fields
	DatabaseID int

	// Domain specific fields
	Metadata string `jsonapi:"attr,metadata"`
	RawDump  string `jsonapi:"attr,rawDump"`
}

// DBSchemaUpsert is the API message for creating a database schema.
type DBSchemaUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdatorID int
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Metadata string
	RawDump  string
}

// DBSchemaFind is the API message for finding database schemas.
type DBSchemaFind struct {
	// Related fields
	DatabaseID int

	// Domain specific fields
}

// DBSchemaDelete is the API message for deleting a database schema.
type DBSchemaDelete struct {
	DatabaseID int
}
