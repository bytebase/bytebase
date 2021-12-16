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
	Key    string   `jsonapi:"attr,key"`
	Values []string `jsonapi:"attr,values"`
}

// LabelKeyFind is the find request for label keys.
type LabelKeyFind struct {
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
	// FindLabelKeys finds all available keys for labels.
	FindLabelKeys(ctx context.Context, find *LabelKeyFind) ([]*LabelKey, error)
	// FindDatabaseLabels finds all database labels matching the conditions, ascending by key.
	FindDatabaseLabels(ctx context.Context, find *DatabaseLabelFind) ([]*DatabaseLabel, error)
	// SetDatabaseLabels sets a database's labels to new labels.
	SetDatabaseLabels(ctx context.Context, labels []*DatabaseLabel, databaseID int, updaterID int) ([]*DatabaseLabel, error)
}
