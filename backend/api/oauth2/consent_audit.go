package oauth2

import (
	"context"
	"html"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TODO(ed): move to MCPCeilingVerdict.xx() func
// One heading per way the workspace's MCP policy refuses a consent. Only the
// heading is local: it is page copy, this is its one consumer, and the three
// states have different fixes, so a heading naming the wrong one is the part a
// hurried reader acts on. The sentence under it is the verdict's own
// (auth.MCPCeilingVerdict.Refusal), shared with every other door.
var consentHeadings = map[auth.MCPCeilingVerdict]string{
	auth.MCPCeilingDisabled:   "MCP access is turned off",
	auth.MCPCeilingUnreadable: "This workspace's MCP setting cannot be read",
	auth.MCPCeilingUnserved:   "This workspace's MCP setting is not one this version supports",
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
	GetMCPSettingsUncached(ctx context.Context, workspace string) (*storepb.MCPSetting, error)
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
	verdict := auth.ClassifyMCPCeiling(settings, err)
	if verdict == auth.MCPCeilingServes {
		return false, nil
	}
	if !verdict.IsPolicy() {
		// An outage, not a verdict: telling the user their admin disabled MCP
		// during a database blip sends them to an admin with nothing to fix.
		slog.Error("failed to read the MCP capability ceiling; cannot grant the consent", log.BBError(err))
		return true, oauth2ErrorRedirect(c, attempt.redirectURI, attempt.state, "temporarily_unavailable",
			verdict.Refusal())
	}
	if err != nil {
		slog.Warn("the stored MCP capability ceiling cannot be interpreted; refusing the consent",
			slog.String("workspace", attempt.user.workspaceID), log.BBError(err))
	}
	return true, s.refuseConsent(c, attempt, verdict)
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
func (s *Service) refuseConsent(c *echo.Context, attempt consentAttempt, verdict auth.MCPCeilingVerdict) error {
	row := &storepb.AuditLog{
		Parent:   common.FormatWorkspace(attempt.user.workspaceID),
		Method:   common.AuditMethodMCPConsentApprove,
		Resource: common.FormatWorkspace(attempt.user.workspaceID),
		Severity: storepb.AuditLog_INFO,
		User:     common.FormatUserEmail(attempt.user.email),
		Status: &spb.Status{
			Code:    int32(codes.PermissionDenied),
			Message: verdict.Refusal(),
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
	return c.HTML(http.StatusForbidden, consentRefusedHTML(verdict, attempt.redirectURI, attempt.state))
}

// consentRefusedHTML renders the refusal. Inline styles only: the global CSP
// allows inline style and blocks inline script, so the page carries no
// behavior.
func consentRefusedHTML(verdict auth.MCPCeilingVerdict, redirectURI, state string) string {
	heading := html.EscapeString(consentHeadings[verdict])
	page := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + heading + `</title>
</head>
<body style="font-family: system-ui, sans-serif; max-width: 32rem; margin: 4rem auto; padding: 0 1rem; line-height: 1.5;">
<h1 style="font-size: 1.25rem;">` + heading + `</h1>
<p>` + html.EscapeString(asSentence(verdict.Refusal())) + `</p>
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

// asSentence renders a shared refusal as page prose. The shared form is
// lowercase and unterminated so every other door can compose it into a larger
// error; this is the one door that shows it on its own.
func asSentence(refusal string) string {
	if refusal == "" {
		return ""
	}
	return strings.ToUpper(refusal[:1]) + refusal[1:] + "."
}
