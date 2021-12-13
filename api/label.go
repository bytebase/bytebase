package api

import (
	"context"
)

// LabelKey is the available key for labels.
type LabelKey struct {
	ID int `jsonapi:"primary,labelKey"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Key string `jsonapi:"attr,key"`
}

// LabelKeyFind is the find request for label keys.
type LabelKeyFind struct {
}

// DatabaseLabel is the label associated with a database.
type DatabaseLabel struct {
	ID int

	// Standard fields
	CreatorID int
	Creator   *Principal
	CreatedTs int64
	UpdaterID int
	Updater   *Principal
	UpdatedTs int64

	// Related fields
	DatabaseID int
	Key        string

	// Domain specific fields
	Value string
}

type DatabaseLabelFind struct {
	// Standard fields
	ID        *int
	RowStatus *RowStatus

	// Related fields
	DatabaseID *int
}

type DatabaseLabelCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	DatabaseID int
	Key        string

	// Domain specific fields
	Value string
}

type DatabaseLabelPatch struct {
	ID int
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Value string
}

type DatabaseLabelArchive struct {
	ID int
}

// LabelService is the service for labels.
type LabelService interface {
	// FindLabelKeyList finds all available keys for labels.
	FindLabelKeyList(ctx context.Context, find *LabelKeyFind) ([]*LabelKey, error)
	// FindDatabaseLabelList finds all database labels matching the conditions, ascending by key.
	FindDatabaseLabelList(ctx context.Context, find *DatabaseLabelFind) ([]*DatabaseLabel, error)
	// CreateDatabaseLabel creates a database label.
	CreateDatabaseLabel(ctx context.Context, create *DatabaseLabelCreate) (*DatabaseLabel, error)
	// PatchDatabaseLabel updates the value of the label, given ID.
	PatchDatabaseLabel(ctx context.Context, patch *DatabaseLabelPatch) (*DatabaseLabel, error)
	// ArchiveDatabaseLabel archives a database label by ID.
	ArchiveDatabaseLabel(ctx context.Context, archive *DatabaseLabelArchive) error
}
