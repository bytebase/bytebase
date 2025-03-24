package lsp

import (
	lsp "github.com/bytebase/lsp-protocol"
)

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
	// The Schema is the connection schema name.
	// Mainly using to set the search path or current schema.
	Schema string `json:"schema,omitempty"`
	// The scene is the scene for completion.
	// Available scenes: "query", "all".
	// If not provided, it defaults to "all".
	// If the scene is "query", it will only return completion items for query statements,
	// 	in other words, SELECT statements only.
	// If the scene is "all", it will return completion items for all statements.
	Scene string `json:"scene,omitempty"`
}
