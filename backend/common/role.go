package common

// ProjectRole is the role in projects.
type ProjectRole string

const (
	// ProjectOwner is the owner of a project.
	ProjectOwner ProjectRole = "OWNER"
	// ProjectDeveloper is the developer of a project.
	ProjectDeveloper ProjectRole = "DEVELOPER"
	// ProjectExporter is the exporter of a project.
	ProjectExporter ProjectRole = "EXPORTER"
	// ProjectQuerier is the querier of a project.
	ProjectQuerier ProjectRole = "QUERIER"
	// Releaser is the RELEASER role.
	Releaser ProjectRole = "RELEASER"
	// ProjectViewer is the viewer of a project.
	ProjectViewer ProjectRole = "DATABASE_VIEWER"
)
