package lsp

import "github.com/sourcegraph/go-lsp"

type CommandName string

const (
	CommandNameSetMetadata CommandName = "setMetadata"
)

// SetMetadataCommandParams are the parameters to the "setMetadata" command.
type SetMetadataCommandParams struct {
	lsp.ExecuteCommandParams
	Arguments []SetMetadataCommandArguments `json:"arguments,omitempty"`
}

// SetMetadataCommandArguments are the arguments to the "setMetadata" command.
type SetMetadataCommandArguments struct {
	// The InstanceID is the instance ID to set metadata for.
	// Format: instances/{instance}
	InstanceID string `json:"instanceId,omitempty"`
	// The DatabaseName is the connection database name.
	// For PostgreSQL, it's required.
	// For other database engines, it's optional.
	DatabaseName string `json:"databaseName,omitempty"`
}
