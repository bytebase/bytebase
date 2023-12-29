package api

// MemberStatus is the status of an member.
type MemberStatus string

const (
	// Unknown is the member status for UNKNOWN.
	Unknown MemberStatus = "UNKNOWN"
	// Invited is the member status for INVITED.
	Invited MemberStatus = "INVITED"
	// Active is the member status for ACTIVE.
	Active MemberStatus = "ACTIVE"
)

// Role is the type of a role.
type Role string

const (
	WorkspaceAdmin   Role = "workspaceAdmin"
	WorkspaceDBA     Role = "workspaceDBA"
	WorkspaceMember  Role = "workspaceMember"
	ProjectOwner     Role = "projectOwner"
	ProjectDeveloper Role = "projectDeveloper"
	ProjectQuerier   Role = "projectQuerier"
	ProjectExporter  Role = "projectExporter"
	ProjectReleaser  Role = "projectReleaser"
	ProjectViewer    Role = "projectViewer"
	UnknownRole      Role = "UNKNOWN"
)

func (r Role) String() string {
	return string(r)
}
