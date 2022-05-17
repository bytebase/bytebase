package api

import (
	"encoding/json"
)

// DBExtension is the API message for a database extension.
type DBExtension struct {
	ID int `jsonapi:"primary,dbExtension"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseID int
	Database   *Database `jsonapi:"relation,database"`

	// Domain specific fields
	Name        string `jsonapi:"attr,name"`
	Version     string `jsonapi:"attr,version"`
	Schema      string `jsonapi:"attr,schema"`
	Description string `jsonapi:"attr,description"`
}

// DBExtensionCreate is the API message for creating a database extension.
type DBExtensionCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name        string
	Version     string
	Schema      string
	Description string
}

// DBExtensionFind is the API message for finding extensions.
type DBExtensionFind struct {
	ID *int

	// Related fields
	DatabaseID *int

	// Domain specific fields
}

func (find *DBExtensionFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// DBExtensionDelete is the API message for deleting a database extension.
type DBExtensionDelete struct {
	// Related fields
	DatabaseID int
}
