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

// SheetSource is the type of sheet origin source.
type SheetSource string

const (
	// SheetFromBytebase is the sheet created by Bytebase. e.g. SQL Editor.
	SheetFromBytebase SheetSource = "BYTEBASE"
	// SheetFromGitLabSelfHost is the sheet synced from self host GitLab.
	SheetFromGitLabSelfHost SheetSource = "GITLAB_SELF_HOST"
	// SheetFromGitHubCom is the sheet synced from github.com.
	SheetFromGitHubCom SheetSource = "GITHUB_COM"
)

func (v SheetSource) String() string {
	switch v {
	case SheetFromBytebase:
		return "BYTEBASE"
	case SheetFromGitLabSelfHost:
		return "GITLAB_SELF_HOST"
	case SheetFromGitHubCom:
		return "GITHUB_COM"
	}
	// Default sheet source is BYTEBASE.
	return "BYTEBASE"
}

// SheetType is the type of sheet.
type SheetType string

const (
	// SheetForSQL is the sheet that used for saving SQL statements.
	SheetForSQL SheetType = "SQL"
)

func (v SheetType) String() string {
	switch v {
	case SheetForSQL:
		return "SQL"
	}
	// Default sheet type is SQL.
	return "SQL"
}

// SheetVCSPayload is the additional data payload of the VCS sheet.
type SheetVCSPayload struct {
	FileName     string `json:"fileName"`
	FilePath     string `json:"filePath"`
	Size         int64  `json:"size"`
	Author       string `json:"author"`
	LastCommitID string `json:"lastCommitId"`
	LastSyncTs   int64  `json:"lastSyncTs"`
}

// SheetRaw is the store model for an Sheet.
// Fields have exactly the same meanings as Sheet.
type SheetRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ProjectID int
	// The DatabaseID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	DatabaseID *int

	// Domain specific fields
	Name       string
	Statement  string
	Visibility SheetVisibility
	Source     SheetSource
	Type       SheetType
	// Payload is in the json string format of SheetVCSPayload.
	Payload string
}

// ToSheet creates an instance of Sheet based on the SheetRaw.
// This is intended to be called when we need to compose an Sheet relationship.
func (raw *SheetRaw) ToSheet() *Sheet {
	return &Sheet{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ProjectID: raw.ProjectID,
		// The DatabaseID is optional.
		// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
		// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:       raw.Name,
		Statement:  raw.Statement,
		Visibility: raw.Visibility,
		Source:     raw.Source,
		Type:       raw.Type,
		Payload:    raw.Payload,
	}
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
	ProjectID int      `jsonapi:"attr,projectId"`
	Project   *Project `jsonapi:"relation,project"`
	// The DatabaseID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	DatabaseID *int      `jsonapi:"attr,databaseId"`
	Database   *Database `jsonapi:"relation,database"`

	// Domain specific fields
	Name       string          `jsonapi:"attr,name"`
	Statement  string          `jsonapi:"attr,statement"`
	Visibility SheetVisibility `jsonapi:"attr,visibility"`
	Source     SheetSource     `jsonapi:"attr,source"`
	Type       SheetType       `jsonapi:"attr,type"`
	Payload    string          `jsonapi:"attr,payload"`
}

// SheetCreate is the API message for creating a sheet.
type SheetCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID  int  `jsonapi:"attr,projectId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name       string          `jsonapi:"attr,name"`
	Statement  string          `jsonapi:"attr,statement"`
	Visibility SheetVisibility `jsonapi:"attr,visibility"`
	Source     SheetSource
	Type       SheetType
	Payload    string `jsonapi:"attr,payload"`
}

// SheetPatch is the API message for patching a sheet.
type SheetPatch struct {
	ID int `jsonapi:"primary,sheetPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Related fields
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name       *string `jsonapi:"attr,name"`
	Statement  *string `jsonapi:"attr,statement"`
	Visibility *string `jsonapi:"attr,visibility"`
	Payload    *string `jsonapi:"attr,payload"`
}

// SheetFind is the API message for finding sheets.
type SheetFind struct {
	// Standard fields
	ID        *int
	RowStatus *RowStatus
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID *int

	// Related fields
	ProjectID *int
	// Query all related sheets with databaseId can be used for database transfer checking.
	DatabaseID *int

	// Domain fields
	Name       *string
	Visibility *SheetVisibility
	Source     *SheetSource
	Type       *SheetType
	Payload    *string
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
	CreateSheet(ctx context.Context, create *SheetCreate) (*SheetRaw, error)
	PatchSheet(ctx context.Context, patch *SheetPatch) (*SheetRaw, error)
	FindSheetList(ctx context.Context, find *SheetFind) ([]*SheetRaw, error)
	FindSheet(ctx context.Context, find *SheetFind) (*SheetRaw, error)
	DeleteSheet(ctx context.Context, delete *SheetDelete) error
}
