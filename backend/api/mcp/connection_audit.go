package mcp

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// One sentence per way the workspace's MCP policy refuses a connection. Only
// the first is fixed by turning MCP back on; the other two are a stored value
// an admin has to rewrite, and saying "disabled" for either sends them to the
// wrong control.
var mcpConnectionRefusals = map[auth.MCPCeilingVerdict]string{
	auth.MCPCeilingDisabled: "MCP access is disabled for this workspace by policy. " +
		"Ask a workspace admin to raise the MCP ceiling in the workspace settings",
	auth.MCPCeilingUnreadable: "this workspace's stored MCP capability ceiling is not one this build understands, " +
		"so the connection fails closed. Ask a workspace admin to set the MCP ceiling again in the workspace settings",
	auth.MCPCeilingUnserved: "this workspace's stored MCP capability ceiling is not one this build serves, " +
		"so the connection fails closed. Ask a workspace admin to set the MCP ceiling to a supported value in the workspace settings",
}

// refuseByCeiling holds a verified token against the workspace's MCP capability
// ceiling. It returns nil to admit the request, or the HTTP error to refuse it
// with, recording a row first when the refusal is a verdict about the
// workspace. auth.ClassifyMCPCeiling makes that call, so this door and the
// per-request gate cannot disagree about one workspace.
func (s *Server) refuseByCeiling(c *echo.Context, delegated auth.DelegatedMCPCredential) error {
	ctx := c.Request().Context()
	settings, err := s.store.GetMCPSettingsUncached(ctx, delegated.WorkspaceID)
	verdict := auth.ClassifyMCPCeiling(settings.Capability, err)
	if verdict == auth.MCPCeilingServes {
		return nil
	}
	if !verdict.IsPolicy() {
		slog.Error("failed to read the MCP capability ceiling; cannot admit the session", log.BBError(err))
		return echo.NewHTTPError(http.StatusServiceUnavailable, "cannot read the MCP policy; retry shortly")
	}
	if err != nil {
		slog.Warn("the stored MCP capability ceiling cannot be interpreted; refusing the connection",
			slog.String("workspace", delegated.WorkspaceID), log.BBError(err))
	}
	return s.refuseConnection(c, delegated, mcpConnectionRefusals[verdict])
}

// refuseConnection records the refusal and returns the 403 to answer with.
//
// The row is written here because this refusal happens in echo middleware,
// before any RPC: neither connect chain ever sees it, so no interceptor would
// write it.
func (s *Server) refuseConnection(c *echo.Context, delegated auth.DelegatedMCPCredential, reason string) error {
	row := &storepb.AuditLog{
		Parent:   common.FormatWorkspace(delegated.WorkspaceID),
		Method:   common.AuditMethodMCPSessionAuthorize,
		Resource: common.FormatWorkspace(delegated.WorkspaceID),
		Severity: storepb.AuditLog_INFO,
		User:     common.FormatUserEmail(delegated.Principal),
		Status: &spb.Status{
			Code:    int32(codes.PermissionDenied),
			Message: reason,
		},
		RequestMetadata: common.RequestMetadataFromHTTP(c.Request()),
		McpDelegation: &storepb.MCPDelegation{
			Scope:         delegated.Scope,
			Resource:      delegated.Resource,
			ClientId:      delegated.ClientID,
			CorrelationId: delegated.CorrelationID,
		},
	}
	common.RecordOutOfBandAudit(c.Request().Context(), s.store,
		s.profile.RuntimeEnableAuditLogStdout.Load(), delegated.WorkspaceID, row)
	return echo.NewHTTPError(http.StatusForbidden, reason)
}
