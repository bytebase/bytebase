// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EchoInput is the input for the echo tool.
type EchoInput struct {
	// The message to echo back.
	Message string `json:"message"`
}

// EchoOutput is the output for the echo tool.
type EchoOutput struct {
	// The echoed message.
	Echo string `json:"echo"`
}

// Server is the MCP server for Bytebase.
type Server struct {
	mcpServer   *mcp.Server
	httpHandler *mcp.StreamableHTTPHandler
}

// NewServer creates a new MCP server.
func NewServer() *Server {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "bytebase",
		Version: "1.0.0",
	}, nil)

	s := &Server{mcpServer: mcpServer}
	s.registerTools()

	// Create HTTP handler for streamable HTTP transport
	s.httpHandler = mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)

	return s
}

// registerTools registers all MCP tools.
func (s *Server) registerTools() {
	// Echo tool for testing
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "echo",
		Description: "Echo the input message back. Useful for testing the MCP connection.",
	}, s.handleEcho)
}

// handleEcho handles the echo tool call.
func (*Server) handleEcho(_ context.Context, _ *mcp.CallToolRequest, input EchoInput) (*mcp.CallToolResult, EchoOutput, error) {
	output := EchoOutput{Echo: input.Message}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output.Echo},
		},
	}, output, nil
}

// RegisterRoutes registers the MCP server routes with Echo.
func (s *Server) RegisterRoutes(e *echo.Echo) {
	// MCP Streamable HTTP endpoint
	e.Any("/mcp", echo.WrapHandler(s.httpHandler))
}
