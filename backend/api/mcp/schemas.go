package mcp

import "embed"

// embeddedSchemas contains all JSON schema files for MCP tool definitions.
// These are generated from proto files via buf generate.
//
//go:embed schemas/*.jsonschema.json
var embeddedSchemas embed.FS
