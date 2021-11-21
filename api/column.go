package api

import (
	"context"
	"encoding/json"
)

type Column struct {
	ID int `jsonapi:"primary,column"`

	// Standard fields
	CreatorID int
	CreatedTs int64 `json:"createdTs"`
	UpdaterID int
	UpdatedTs int64 `json:"updatedTs"`

	// Related fields
	DatabaseID int
	TableID    int

	// Domain specific fields
	Name         string  `json:"name"`
	Position     int     `json:"position"`
	Default      *string `json:"default"`
	Nullable     bool    `json:"nullable"`
	Type         string  `json:"type"`
	CharacterSet string  `json:"characterSet"`
	Collation    string  `json:"collation"`
	Comment      string  `json:"comment"`
}

type ColumnCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	DatabaseID int
	TableID    int

	// Domain specific fields
	Name         string
	Position     int
	Default      *string
	Nullable     bool
	Type         string
	CharacterSet string
	Collation    string
	Comment      string
}

type ColumnFind struct {
	ID *int

	// Related fields
	DatabaseID *int
	TableID    *int

	// Domain specific fields
	Name *string
}

func (find *ColumnFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type ColumnPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int
}

type ColumnService interface {
	CreateColumn(ctx context.Context, create *ColumnCreate) (*Column, error)
	FindColumnList(ctx context.Context, find *ColumnFind) ([]*Column, error)
	FindColumn(ctx context.Context, find *ColumnFind) (*Column, error)
	PatchColumn(ctx context.Context, patch *ColumnPatch) (*Column, error)
}
