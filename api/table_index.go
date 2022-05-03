package api

import (
	"encoding/json"
)

// Index is the API message for an index.
type Index struct {
	ID int `jsonapi:"primary,index"`

	// Standard fields
	CreatorID int
	CreatedTs int64 `json:"createdTs"`
	UpdaterID int
	UpdatedTs int64 `json:"updatedTs"`

	// Related fields
	DatabaseID int
	TableID    int

	// Domain specific fields
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Position   int    `json:"position"`
	Type       string `json:"type"`
	Unique     bool   `json:"unique"`
	Visible    bool   `json:"visible"`
	Comment    string `json:"comment"`
}

// IndexCreate is the API message for creating an index.
type IndexCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	DatabaseID int
	TableID    int

	// Domain specific fields
	Name       string
	Expression string
	Position   int
	Type       string
	Unique     bool
	Visible    bool
	Comment    string
}

// IndexFind is the API message for finding indices.
type IndexFind struct {
	ID *int

	// Related fields
	DatabaseID *int
	TableID    *int

	// Domain specific fields
	Name       *string
	Expression *string
}

func (find *IndexFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}
