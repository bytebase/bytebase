// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server and provides HTTP handler integration.
type Server struct {
	mcpServer *mcp.Server
	registry  *Registry
}

// NewServer creates a new MCP server with all Bytebase tools registered.
func NewServer(registry *Registry) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "bytebase",
		Version: "1.0.0",
	}, nil)

	// Register all tools from the registry
	if err := registry.RegisterTools(mcpServer); err != nil {
		return nil, err
	}

	return &Server{
		mcpServer: mcpServer,
		registry:  registry,
	}, nil
}

// Handler returns the HTTP handler for the MCP server.
// It wraps the SDK handler to inject auth context.
func (s *Server) Handler() http.Handler {
	sdkHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			// Extract auth header and store in invoker context
			auth := r.Header.Get("Authorization")
			if auth != "" {
				// Store in request context for invoker to use
				ctx := WithAuthHeader(r.Context(), auth)
				*r = *r.WithContext(ctx)
			}
			return s.mcpServer
		},
		nil,
	)

	return sdkHandler
}
