package api

import (
	"context"
	"encoding/json"
)

// SheetVisibility is the visibility of a sheet.
type SheetVisibility string

const (
	// PrivateSheet is the sheet visibility for PRIVATE. Only sheet OWNER can read/write.
	PrivateSheet SheetVisibility = "PRIVATE"
	// ProjectSheet is the sheet visibility for PROJECT. Both sheet OWNER and project OWNER can read/write, and project DEVELOPER can read.
	ProjectSheet SheetVisibility = "PROJECT"
	// PublicSheet is the sheet visibility for PUBLIC. Sheet OWNER can read/write, and all others can read.
	PublicSheet SheetVisibility = "PUBLIC"
)

func (v SheetVisibility) String() string {
	switch v {
	case PrivateSheet:
		return "PRIVATE"
	case ProjectSheet:
		return "PROJECT"
	case PublicSheet:
		return "PUBLIC"
	}
	return ""
}

// Sheet is the API message for a sheet.
type Sheet struct {
	ID int `jsonapi:"primary,sheet"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	InstanceID int  `jsonapi:"attr,instanceId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name       string          `jsonapi:"attr,name"`
	Statement  string          `jsonapi:"attr,statement"`
	Visibility SheetVisibility `jsonapi:"attr,visibility"`
}

// SheetCreate is the API message for creating a sheet.
type SheetCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	InstanceID int  `jsonapi:"attr,instanceId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name       string          `jsonapi:"attr,name"`
	Statement  string          `jsonapi:"attr,statement"`
	Visibility SheetVisibility `jsonapi:"attr,visibility"`
}

// SheetPatch is the API message for patching a sheet.
type SheetPatch struct {
	ID int `jsonapi:"primary,sheetPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Related fields
	InstanceID int  `jsonapi:"attr,instanceId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name       *string          `jsonapi:"attr,name"`
	Statement  *string          `jsonapi:"attr,statement"`
	Visibility *string          `jsonapi:"attr,visibility"`
}

// SheetFind is the API message for finding sheets.
type SheetFind struct {
	// Standard fields
	ID        *int
	RowStatus *RowStatus
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID *int

	// Related fields
	InstanceID *int
	DatabaseID *int

	// Domain fields
	Visibility *SheetVisibility

	// Only applicable if InstanceID is specified, if true, then we only return instanceId (database_id is NULL)
	InstanceOnly bool
}

func (find *SheetFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// SheetDelete is the API message for deleting a sheet.
type SheetDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

// SheetService is the service for sheet.
type SheetService interface {
	CreateSheet(ctx context.Context, create *SheetCreate) (*Sheet, error)
	PatchSheet(ctx context.Context, patch *SheetPatch) (*Sheet, error)
	FindSheetList(ctx context.Context, find *SheetFind) ([]*Sheet, error)
	FindSheet(ctx context.Context, find *SheetFind) (*Sheet, error)
	DeleteSheet(ctx context.Context, delete *SheetDelete) error
}
