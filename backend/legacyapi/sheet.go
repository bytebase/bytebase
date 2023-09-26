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
