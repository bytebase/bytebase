package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

// ReauthorizeInput is the input for the reauthorize tool.
type ReauthorizeInput struct{}

const reauthorizeDescription = `Log out the current MCP OAuth connection so the client can run OAuth again.

Use this when you need to change the connected Bytebase account or SaaS workspace. After this tool succeeds, retry the MCP request or reconnect; the server will return the normal OAuth challenge.`

func (s *Server) registerReauthorizeTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "reauthorize",
		Description: reauthorizeDescription,
	}, s.handleReauthorize)
}

func (s *Server) handleReauthorize(ctx context.Context, _ *mcp.CallToolRequest, _ ReauthorizeInput) (*mcp.CallToolResult, any, error) {
	accessToken := getAccessToken(ctx)
	userEmail := getUserEmail(ctx)
	clientID := getOAuth2ClientID(ctx)
	if accessToken == "" || userEmail == "" || clientID == "" {
		return formatToolError(&toolError{
			Code:    "NOT_OAUTH_SESSION",
			Message: "reauthorize requires an MCP OAuth access token",
		}), nil, nil
	}
	if err := s.store.DeleteOAuth2RefreshTokensByUserAndClient(ctx, userEmail, clientID); err != nil {
		return formatToolError(errors.Wrap(err, "failed to revoke OAuth refresh tokens")), nil, nil
	}
	// This is the bearer on the request being served, not the one the session
	// was opened with (liveRequestMetadata overlays it), so a caller that
	// refreshed mid-session revokes the token it actually holds. Tokens minted
	// earlier for the same grant are not listed here; they expire on their own,
	// and the refresh grants deleted above stop any more being issued.
	//
	// MCP sessions are currently process-local and already require requests to
	// stay on the same replica. This marker follows that constraint. If MCP gains
	// multi-replica session support, move the reauthorization state to shared storage.
	s.revokedAccessTokens.Store(accessToken, struct{}{})

	workspaceID := getWorkspaceID(ctx)
	message := "OAuth refresh grant revoked. Retry or reconnect this MCP server to run OAuth again."
	if workspaceID != "" {
		message = fmt.Sprintf("%s The previous grant was for workspace %q.", message, workspaceID)
	}
	output := map[string]any{
		"reauthorizationRequired": true,
		"workspace":               workspaceID,
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}, output, nil
}
