package base

// Role is the type of a role.
type Role string

const (
	WorkspaceAdmin   Role = "workspaceAdmin"
	WorkspaceDBA     Role = "workspaceDBA"
	WorkspaceMember  Role = "workspaceMember"
	ProjectOwner     Role = "projectOwner"
	ProjectDeveloper Role = "projectDeveloper"
	ProjectExporter  Role = "projectExporter"
	ProjectReleaser  Role = "projectReleaser"
	ProjectViewer    Role = "projectViewer"
	SQLEditorUser    Role = "sqlEditorUser"
	UnknownRole      Role = "UNKNOWN"
)

func (r Role) String() string {
	return string(r)
}
