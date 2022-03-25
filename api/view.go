package api

import (
	"context"
	"encoding/json"
)

// ViewRaw is the store model for an View.
// Fields have exactly the same meanings as View.
type ViewRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name       string
	Definition string
	Comment    string
}

// ToView creates an instance of View based on the ViewRaw.
// This is intended to be called when we need to compose an View relationship.
func (raw *ViewRaw) ToView() *View {
	return &View{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:       raw.Name,
		Definition: raw.Definition,
		Comment:    raw.Comment,
	}
}

// View is the API message for a view.
type View struct {
	ID int `jsonapi:"primary,view"`

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
	Name       string `jsonapi:"attr,name"`
	Definition string `jsonapi:"attr,definition"`
	Comment    string `jsonapi:"attr,comment"`
}

// ViewCreate is the API message for creating a view.
type ViewCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name       string
	Definition string
	Comment    string
}

// ViewFind is the API message for finding views.
type ViewFind struct {
	ID *int

	// Related fields
	DatabaseID *int

	// Domain specific fields
	Name *string
}

func (find *ViewFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// ViewDelete is the API message for deleting a view.
type ViewDelete struct {
	// Related fields
	DatabaseID int
}

// ViewService is the service for views.
type ViewService interface {
	CreateView(ctx context.Context, create *ViewCreate) (*ViewRaw, error)
	FindViewList(ctx context.Context, find *ViewFind) ([]*ViewRaw, error)
	FindView(ctx context.Context, find *ViewFind) (*ViewRaw, error)
	DeleteView(ctx context.Context, delete *ViewDelete) error
}
