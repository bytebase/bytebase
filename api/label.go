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
	Key       string   `jsonapi:"attr,key"`
	ValueList []string `jsonapi:"attr,valueList"`
}

// LabelKeyFind is the find request for label keys.
type LabelKeyFind struct {
}

// LabelKeyPatch is the message to patch a label key.
type LabelKeyPatch struct {
	ID int `jsonapi:"primary,labelKeyPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorID is the ID of the creator.
	UpdaterID int

	// Related fields

	// Domain specific fields
	ValueList []string `jsonapi:"attr,valueList"`
}

// DatabaseLabel is the label associated with a database.
type DatabaseLabel struct {
	ID int `json:"-"`

	// Standard fields
	RowStatus RowStatus  `json:"-"`
	CreatorID int        `json:"-"`
	Creator   *Principal `json:"-"`
	CreatedTs int64      `json:"-"`
	UpdaterID int        `json:"-"`
	Updater   *Principal `json:"-"`
	UpdatedTs int64      `json:"-"`

	// Related fields
	DatabaseID int    `json:"-"`
	Key        string `json:"key"`

	// Domain specific fields
	Value string `json:"value"`
}

// DatabaseLabelFind finds the labels associated with the database.
type DatabaseLabelFind struct {
	// Standard fields
	ID        *int
	RowStatus *RowStatus

	// Related fields
	DatabaseID *int
}

// DatabaseLabelUpsert upserts the label associated with the database.
type DatabaseLabelUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int
	RowStatus RowStatus

	// Related fields
	DatabaseID int
	Key        string

	// Domain specific fields
	Value string
}

// LabelService is the service for labels.
type LabelService interface {
	// FindLabelKeyList finds all available keys for labels.
	FindLabelKeyList(ctx context.Context, find *LabelKeyFind) ([]*LabelKey, error)
	// PatchLabelKey patches a label key.
	PatchLabelKey(ctx context.Context, patch *LabelKeyPatch) (*LabelKey, error)
	// FindDatabaseLabelList finds all database labels matching the conditions, ascending by key.
	FindDatabaseLabelList(ctx context.Context, find *DatabaseLabelFind) ([]*DatabaseLabel, error)
	// SetDatabaseLabelList sets a database's labels to new labels.
	SetDatabaseLabelList(ctx context.Context, labels []*DatabaseLabel, databaseID int, updaterID int) ([]*DatabaseLabel, error)
}
