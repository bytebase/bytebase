package api

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
