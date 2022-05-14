package api

import (
	"encoding/json"
)

// Extension is the API message for an extension.
type Extension struct {
	ID int `jsonapi:"primary,extension"`

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

// ExtensionCreate is the API message for creating an extension.
type ExtensionCreate struct {
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

// ExtensionFind is the API message for finding extensions.
type ExtensionFind struct {
	ID *int

	// Related fields
	DatabaseID *int

	// Domain specific fields
}

func (find *ExtensionFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// ExtensionDelete is the API message for deleting an extension.
type ExtensionDelete struct {
	// Related fields
	DatabaseID int
}
