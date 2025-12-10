package mcp

import "embed"

// embeddedSchemas contains all JSON schema files for MCP tool definitions.
// These are generated from proto files via buf generate.
//
//go:embed schemas/*.jsonschema.bundle.json
var embeddedSchemas embed.FS
