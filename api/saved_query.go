package api

import (
	"context"
	"encoding/json"
)

// SavedQuery is the API message for a saved_query.
type SavedQuery struct {
	ID int `jsonapi:"primary,saved_query"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name      string `jsonapi:"attr,name"`
	Statement string `jsonapi:"attr,statement"`
}

// SavedQueryCreate is the API message for creating a saved_query.
type SavedQueryCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name      string `jsonapi:"attr,name"`
	Statement string `jsonapi:"attr,statement"`
}

// SavedQueryPatch is the API message for patching a saved_query.
type SavedQueryPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name      *string `jsonapi:"attr,name"`
	Statement *string `jsonapi:"attr,statement"`
}

// SavedQueryFind is the API message for finding saved_queries.
type SavedQueryFind struct {
	ID *int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID *int
}

func (find *SavedQueryFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// SavedQueryDelete is the API message for deleting a saved_query.
type SavedQueryDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

// SavedQueryService is the service for saved_query.
type SavedQueryService interface {
	CreateSavedQuery(ctx context.Context, create *SavedQueryCreate) (*SavedQuery, error)
	PatchSavedQuery(ctx context.Context, patch *SavedQueryPatch) (*SavedQuery, error)
	FindSavedQueryList(ctx context.Context, find *SavedQueryFind) ([]*SavedQuery, error)
	FindSavedQuery(ctx context.Context, find *SavedQueryFind) (*SavedQuery, error)
	DeleteSavedQuery(ctx context.Context, delete *SavedQueryDelete) error
}
