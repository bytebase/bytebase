package api

// Workspace is the API message for workspace.
type Workspace struct {
	ID string

	// Version is the bytebase's version
	Version string
}
