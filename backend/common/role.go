package common

// ProjectRole is the role in projects.
type ProjectRole string

const (
	// ProjectOwner is the owner of a project.
	ProjectOwner ProjectRole = "OWNER"
	// ProjectDeveloper is the developer of a project.
	ProjectDeveloper ProjectRole = "DEVELOPER"
)
