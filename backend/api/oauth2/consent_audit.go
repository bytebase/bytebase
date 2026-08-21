package oauth2

import (
	"context"
	"html"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// One sentence per way the workspace's MCP policy refuses a consent: the
// connection gate's three, said about the authorization instead of the session.
// A user who meets both boundaries must hear one story about their workspace.
var consentRefusals = map[auth.MCPCeilingVerdict]string{
	auth.MCPCeilingDisabled: "A workspace admin has turned MCP access off for this workspace, " +
		"so no client can be authorized. Ask a workspace admin to raise the MCP ceiling in the workspace settings.",
	auth.MCPCeilingUnreadable: "This workspace's stored MCP capability ceiling is not one this build understands, " +
		"so authorization fails closed. Ask a workspace admin to set the MCP ceiling again in the workspace settings.",
	auth.MCPCeilingUnserved: "This workspace's stored MCP capability ceiling is not one this build serves, " +
		"so authorization fails closed. Ask a workspace admin to set the MCP ceiling to a supported value in the workspace settings.",
}

// consentAttempt is the consent a ceiling check may refuse: who is consenting,
// which client asked, what was asked for, and where the client waits.
type consentAttempt struct {
	user        consentingUser
	clientID    string
	params      grantParams
	redirectURI string
	state       string
}

// mcpCeilingReader is the whole of what the ceiling check needs from the store.
// *store.Store satisfies it; a test supplies its own, which is what lets the
// outage arm be exercised — a real store cannot be made to fail a single read.
type mcpCeilingReader interface {
	GetMCPSettingsUncached(ctx context.Context, workspace string) (store.MCPSettings, error)
}

// refuseConsentByCeiling holds a consent about to be granted against the
// workspace's MCP capability ceiling. The bool reports whether it has already
// written the response, which is what the caller must branch on: echo's c.HTML
// returns a nil error when it succeeds, so a nil error cannot mean the consent
// may proceed. Reading it that way issues an authorization code underneath the
// refusal page.
//
// It runs after the grant parameters are validated, not before, so the audit
// row records the resource and scope that were asked for and a malformed
// request still gets the error about itself.
//
// auth.ClassifyMCPCeiling decides, so this door and the /mcp gate cannot
// disagree about one workspace — and a token minted under a ceiling that serves
// nothing would be a credential for a door that will refuse it.
func (s *Service) refuseConsentByCeiling(c *echo.Context, attempt consentAttempt) (bool, error) {
	ctx := c.Request().Context()
	settings, err := s.mcpCeiling.GetMCPSettingsUncached(ctx, attempt.user.workspaceID)
	verdict := auth.ClassifyMCPCeiling(settings.Capability, err)
	if verdict == auth.MCPCeilingServes {
		return false, nil
	}
	if !verdict.IsPolicy() {
		// An outage, not a verdict: telling the user their admin disabled MCP
		// during a database blip sends them to an admin with nothing to fix.
		slog.Error("failed to read the MCP capability ceiling; cannot grant the consent", log.BBError(err))
		return true, oauth2ErrorRedirect(c, attempt.redirectURI, attempt.state, "temporarily_unavailable",
			"cannot read the MCP policy; retry shortly")
	}
	if err != nil {
		slog.Warn("the stored MCP capability ceiling cannot be interpreted; refusing the consent",
			slog.String("workspace", attempt.user.workspaceID), log.BBError(err))
	}
	return true, s.refuseConsent(c, attempt, consentRefusals[verdict])
}

// refuseConsent records the refusal and renders the page the user sees.
//
// A page rather than the usual error redirect: the redirect hands the failure
// to the MCP client, and no client knows what to ask an admin for. Nothing was
// connected, so there is nothing to resume; the link back is there for a user
// who wants their client told.
//
// The row is written here for the same reason the connection gate writes its
// own: this route is echo, so no interceptor sees it.
func (s *Service) refuseConsent(c *echo.Context, attempt consentAttempt, reason string) error {
	row := &storepb.AuditLog{
		Parent:   common.FormatWorkspace(attempt.user.workspaceID),
		Method:   common.AuditMethodMCPConsentApprove,
		Resource: common.FormatWorkspace(attempt.user.workspaceID),
		Severity: storepb.AuditLog_INFO,
		User:     common.FormatUserEmail(attempt.user.email),
		Status: &spb.Status{
			Code:    int32(codes.PermissionDenied),
			Message: reason,
		},
		RequestMetadata: common.RequestMetadataFromHTTP(c.Request()),
		// The MCP provenance this flow has: the client asking, and the
		// resource and scope it asked for. No correlation ID — that is minted
		// at /mcp, and this attempt never got there.
		McpDelegation: &storepb.MCPDelegation{
			Scope:    attempt.params.scope,
			Resource: attempt.params.resource,
			ClientId: attempt.clientID,
		},
	}
	common.RecordOutOfBandAudit(c.Request().Context(), s.store,
		s.profile.RuntimeEnableAuditLogStdout.Load(), attempt.user.workspaceID, row)
	return c.HTML(http.StatusForbidden, consentRefusedHTML(reason, attempt.redirectURI, attempt.state))
}

// consentRefusedHTML renders the refusal. Inline styles only: the global CSP
// allows inline style and blocks inline script, so the page carries no
// behavior.
func consentRefusedHTML(reason, redirectURI, state string) string {
	page := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>MCP access is turned off</title>
</head>
<body style="font-family: system-ui, sans-serif; max-width: 32rem; margin: 4rem auto; padding: 0 1rem; line-height: 1.5;">
<h1 style="font-size: 1.25rem;">MCP access is turned off</h1>
<p>` + html.EscapeString(reason) + `</p>
<p>Nothing was connected.</p>`
	if back, err := oauth2ErrorRedirectURL(redirectURI, state, "access_denied",
		"the workspace MCP policy refused this authorization"); err == nil {
		page += `
<p><a href="` + html.EscapeString(back) + `">Return to the application</a></p>`
	}
	return page + `
</body>
</html>`
}
