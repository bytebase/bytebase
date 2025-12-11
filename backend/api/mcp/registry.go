package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Registry holds MCP tool definitions.
type Registry struct {
	invoker *Invoker
}

// NewRegistry creates a new tool registry.
func NewRegistry(invoker *Invoker) (*Registry, error) {
	return &Registry{
		invoker: invoker,
	}, nil
}

// RegisterTools registers MCP tools.
// Currently registers a dummy tool for testing the MCP server setup.
func (r *Registry) RegisterTools(server *mcp.Server) error {
	// Register a simple echo tool for testing
	server.AddTool(&mcp.Tool{
		Name:        "echo",
		Description: "Echo back the input message. This is a test tool for verifying MCP server connectivity.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string",
					"description": "The message to echo back"
				}
			},
			"required": ["message"]
		}`),
	}, r.echoHandler)

	return nil
}

// echoHandler handles the echo tool invocation.
func (*Registry) echoHandler(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments from raw JSON
	var args struct {
		Message string `json:"message"`
	}
	if req.Params.Arguments != nil {
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, err
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: args.Message,
			},
		},
	}, nil
}
