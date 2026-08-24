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
		return echo.NewHTTPError(http.StatusServiceUnavailable, verdict.Refusal())
	}
	if err != nil {
		slog.Warn("the stored MCP capability ceiling cannot be interpreted; refusing the connection",
			slog.String("workspace", delegated.WorkspaceID), log.BBError(err))
	}
	return s.refuseConnection(c, delegated, verdict.Refusal())
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
		// The grant this token carries, and no correlation ID. That field is
		// session-scoped, and this refusal is decided before the SDK resolves a
		// session: an initial connection has none, and a mid-session refusal
		// cannot reach the ID the session already writes under. The per-request
		// value minted here would read as a session ID that correlates exactly
		// one row.
		McpDelegation: &storepb.MCPDelegation{
			Scope:    delegated.Scope,
			Resource: delegated.Resource,
			ClientId: delegated.ClientID,
		},
	}
	common.RecordOutOfBandAudit(c.Request().Context(), s.store,
		s.profile.RuntimeEnableAuditLogStdout.Load(), delegated.WorkspaceID, row)
	return echo.NewHTTPError(http.StatusForbidden, reason)
}
