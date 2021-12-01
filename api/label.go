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

// LabelService is the service for labels.
type LabelService interface {
	// FindLabelKeyList finds all available keys for labels.
	FindLabelKeyList(ctx context.Context, find *LabelKeyFind) ([]*LabelKey, error)
}
