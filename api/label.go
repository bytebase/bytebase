package api

import (
	"context"
	"fmt"
)

const (
	// EnvironmentKeyName is the reserved key for environment.
	EnvironmentKeyName string = "bb.environment"

	// DatabaseLabelSizeMax is the maximum size of database labels.
	DatabaseLabelSizeMax = 4
	labelLengthMax       = 63

	// LocationLabelKey is the label key for location
	LocationLabelKey = "bb.location"
	// TenantLabelKey is the label key for tenant
	TenantLabelKey = "bb.tenant"
)

// LabelKeyRaw is the store model for an LabelKey.
// Fields have exactly the same meanings as LabelKey.
type LabelKeyRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	// bb.environment is a reserved key and identically mapped from environments. It has zero ID and its values are immutable.
	Key       string
	ValueList []string
}

// ToLabelKey creates an instance of LabelKey based on the LabelKeyRaw.
// This is intended to be called when we need to compose an LabelKey relationship.
func (raw *LabelKeyRaw) ToLabelKey() *LabelKey {
	labelKey := LabelKey{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		// bb.environment is a reserved key and identically mapped from environments. It has zero ID and its values are immutable.
		Key: raw.Key,
	}
	labelKey.ValueList = append(labelKey.ValueList, raw.ValueList...)
	return &labelKey
}

// LabelKey is the available key for labels.
type LabelKey struct {
	ID int `jsonapi:"primary,labelKey"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	// bb.environment is a reserved key and identically mapped from environments. It has zero ID and its values are immutable.
	Key       string   `jsonapi:"attr,key"`
	ValueList []string `jsonapi:"attr,valueList"`
}

// LabelKeyFind is the find request for label keys.
type LabelKeyFind struct {
	// RowStatus is the row status filter.
	RowStatus *RowStatus
}

// LabelKeyPatch is the message to patch a label key.
type LabelKeyPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorID is the ID of the creator.
	UpdaterID int

	// Related fields

	// Domain specific fields
	ValueList []string `jsonapi:"attr,valueList"`
}

// Validate validates the sanity of patch values.
func (patch *LabelKeyPatch) Validate() error {
	for _, v := range patch.ValueList {
		if len(v) <= 0 || len(v) > labelLengthMax {
			return fmt.Errorf("label value has a maximum length of %v characters and cannot be empty", labelLengthMax)
		}
	}
	return nil
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
	FindLabelKeyList(ctx context.Context, find *LabelKeyFind) ([]*LabelKeyRaw, error)
	// PatchLabelKey patches a label key.
	PatchLabelKey(ctx context.Context, patch *LabelKeyPatch) (*LabelKeyRaw, error)
	// FindDatabaseLabelList finds all database labels matching the conditions, ascending by key.
	FindDatabaseLabelList(ctx context.Context, find *DatabaseLabelFind) ([]*DatabaseLabel, error)
	// SetDatabaseLabelList sets a database's labels to new labels.
	SetDatabaseLabelList(ctx context.Context, labels []*DatabaseLabel, databaseID int, updaterID int) ([]*DatabaseLabel, error)
}
