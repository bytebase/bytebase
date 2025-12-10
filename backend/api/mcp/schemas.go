package mcp

import "embed"

// embeddedSchemas contains existing openapi.yaml
//
//go:embed spec/openapi.yaml
var embeddedSchemas embed.FS
