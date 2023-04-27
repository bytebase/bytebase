package api

import (
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

// SheetSource is the type of sheet origin source.
type SheetSource string

const (
	// SheetFromBytebase is the sheet created by Bytebase. e.g. SQL Editor.
	SheetFromBytebase SheetSource = "BYTEBASE"
	// SheetFromBytebaseArtifact is the artifact sheet.
	SheetFromBytebaseArtifact SheetSource = "BYTEBASE_ARTIFACT"
	// SheetFromGitLab is the sheet synced from GitLab (for both GitLab.com and self-hosted GitLab).
	SheetFromGitLab SheetSource = "GITLAB"
	// SheetFromGitHub is the sheet synced from GitHub (for both GitHub.com and GitHub Enterprise).
	SheetFromGitHub SheetSource = "GITHUB"
	// SheetFromBitbucket is the sheet synced from Bitbucket.
	SheetFromBitbucket SheetSource = "BITBUCKET"
)

// SheetType is the type of sheet.
type SheetType string

const (
	// SheetForSQL is the sheet that used for saving SQL statements.
	SheetForSQL SheetType = "SQL"
)

// SheetVCSPayload is the additional data payload of the VCS sheet.
// The sheet source should be one of SheetFromGitLab and SheetFromGitHub.
type SheetVCSPayload struct {
	FileName     string `json:"fileName"`
	FilePath     string `json:"filePath"`
	Size         int64  `json:"size"`
	Author       string `json:"author"`
	LastCommitID string `json:"lastCommitId"`
	LastSyncTs   int64  `json:"lastSyncTs"`
}

// Sheet is the API message for a sheet.
type Sheet struct {
	ID int `jsonapi:"primary,sheet"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
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
	Starred    bool            `jsonapi:"attr,starred"`
	Pinned     bool            `jsonapi:"attr,pinned"`

	// Size is the size of statement in bytes.
	Size int64 `jsonapi:"attr,size"`
}

// SheetCreate is the API message for creating a sheet.
type SheetCreate struct {
	// Standard fields
	CreatorID int

	// Related fields
	ProjectID  int  `jsonapi:"attr,projectId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name       string          `jsonapi:"attr,name"`
	Statement  string          `jsonapi:"attr,statement"`
	Visibility SheetVisibility `jsonapi:"attr,visibility"`
	Source     SheetSource     `jsonapi:"attr,source"`
	Type       SheetType
	Payload    string `jsonapi:"attr,payload"`
}

// SheetPatch is the API message for patching a sheet.
type SheetPatch struct {
	ID int `jsonapi:"primary,sheetPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	UpdaterID int

	// Related fields
	ProjectID  *int `jsonapi:"attr,projectId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name       *string `jsonapi:"attr,name"`
	Statement  *string `jsonapi:"attr,statement"`
	Visibility *string `jsonapi:"attr,visibility"`
	Payload    *string `jsonapi:"attr,payload"`
}

// SheetFind is the API message for finding sheets.
type SheetFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus
	// Used to find the creator's sheet list.
	// When finding shared PROJECT/PUBLIC sheets, this value should be empty.
	CreatorID *int
	// LoadFull is used if we want to load the full sheet.
	LoadFull bool

	// Related fields
	ProjectID  *int
	DatabaseID *int

	// Domain fields
	Name       *string
	Visibility *SheetVisibility
	Source     *SheetSource
	Type       *SheetType
	Payload    *string
	// Used to find starred/pinned sheet list, could be PRIVATE/PROJECT/PUBLIC sheet.
	// For now, we only need the starred sheets.
	OrganizerPrincipalID *int
	// Used to find a sheet list from projects containing PrincipalID as an active member.
	// When finding a shared PROJECT/PUBLIC sheets, this value should be present.
	PrincipalID *int
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
	DeleterID int
}
