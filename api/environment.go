package api

import (
	"encoding/json"
)

// Environment is the API message for an environment.
type Environment struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Order int    `jsonapi:"attr,order"`
}

// EnvironmentCreate is the API message for creating an environment.
type EnvironmentCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

// EnvironmentFind is the API message for finding environments.
type EnvironmentFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus
}

func (find *EnvironmentFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// EnvironmentPatch is the API message for patching an environment.
type EnvironmentPatch struct {
	ID int `jsonapi:"primary,environmentPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name  *string `jsonapi:"attr,name"`
	Order *int    `jsonapi:"attr,order"`
}

// EnvironmentDelete is the API message for deleting an environment.
type EnvironmentDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}
