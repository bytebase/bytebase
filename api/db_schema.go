package api

// DBSchema is the API message for database schema.
type DBSchema struct {
	ID int

	// Standard fields

	// Related fields
	DatabaseID int `json:"databaseId"`

	// Domain specific fields
	Metadata string `json:"metadata"`
	RawDump  string `json:"rawDump"`
}

// DBSchemaUpsert is the API message for creating a database schema.
type DBSchemaUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdatorID int

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
