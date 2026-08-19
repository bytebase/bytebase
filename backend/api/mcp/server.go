// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// Server is the MCP server for Bytebase.
type Server struct {
	mcpServer    *mcp.Server
	httpHandler  http.Handler
	store        serverStore
	profile      *config.Profile
	secret       string
	openAPIIndex *OpenAPIIndex

	// internalClient carries tool API calls to the internal handler chain over
	// the private in-memory transport, authenticated by the delegated
	// credential minted in authMiddleware.
	internalClient *http.Client

	revokedAccessTokens sync.Map // map[string]struct{}

	// planCheckPollBudgetOverride lets tests shorten the plan-check poll budget.
	// Zero means use the default (planCheckPollBudget).
	planCheckPollBudgetOverride time.Duration
}

type serverStore interface {
	GetWorkspaceID(context.Context) (string, error)
	GetWorkspaceProfileSetting(context.Context, string) (*storepb.WorkspaceProfileSetting, error)
	GetMCPCapabilityUncached(context.Context, string) (storepb.WorkspaceProfileSetting_MCPCapability, error)
	DeleteOAuth2RefreshTokensByUserAndClient(context.Context, string, string) error
}

// NewServer creates a new MCP server. internalAPI is the internal API handler
// chain tool calls dispatch to in memory; it is never bound to a listener.
// It takes the concrete store; tests reach newServerWithStore with a fake.
func NewServer(stores *store.Store, profile *config.Profile, secret string, internalAPI http.Handler) (*Server, error) {
	return newServerWithStore(stores, profile, secret, internalAPI)
}

func newServerWithStore(stores serverStore, profile *config.Profile, secret string, internalAPI http.Handler) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "bytebase",
		Version: profile.Version,
	}, nil)

	// Load OpenAPI index for API discovery and execution (embedded)
	openAPIIndex, err := NewOpenAPIIndex()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load OpenAPI spec")
	}

	s := &Server{
		mcpServer:      mcpServer,
		store:          stores,
		profile:        profile,
		secret:         secret,
		openAPIIndex:   openAPIIndex,
		internalClient: newInternalAPIClient(internalAPI),
	}
	s.registerTools()

	// Create HTTP handler for streamable HTTP transport.
	//
	// DisableLocalhostProtection turns off the SDK's DNS-rebinding check
	// (auto-enabled since go-sdk v1.4.0). That check rejects requests that
	// arrive over a loopback connection while carrying a non-loopback Host
	// header. Behind a same-host reverse proxy (proxy_pass http://127.0.0.1),
	// Bytebase accepts a loopback connection while the proxy preserves the
	// public Host, so the check fires on legitimate traffic and returns
	// "403 Forbidden: invalid Host header" (BYT-9693). It is safe to disable:
	// /mcp is gated by mandatory OAuth bearer-token auth (authMiddleware), so
	// the token — not network position — is the security boundary, and the
	// rebinding threat targets unauthenticated, browser-reached localhost
	// servers, which Bytebase is not.
	streamable := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{DisableLocalhostProtection: true})

	// Refresh per-request metadata that would otherwise be frozen at session
	// start. The SDK runs tool handlers on the initialize request's context, but
	// hands each JSON-RPC request's own HTTP headers to receiving middleware, so
	// this is where a long-lived session picks up where its current request came
	// from. Identity stays pinned to the session (see below), which is what
	// makes trusting the live address safe: it cannot belong to another
	// principal.
	mcpServer.AddReceivingMiddleware(liveRequestMetadata)

	// Pin each session to the identity it was opened with. The SDK captures
	// TokenInfo.UserID when a session is created and rejects later requests
	// carrying a different one, but only when a TokenInfo reaches it — and the
	// context key is settable solely through this middleware. Without it the
	// check is inert, and since tool handlers run on the initialize request's
	// context, a substituted-but-valid bearer would be admitted while the tool
	// executed under the session's original identity.
	s.httpHandler = mcpauth.RequireBearerToken(s.verifySessionBinding, nil)(streamable)

	return s, nil
}

// registerTools registers all MCP tools.
func (s *Server) registerTools() {
	s.registerSearchTool()
	s.registerCallTool()
	s.registerSkillTool()
	s.registerQueryTool()
	s.registerSchemaTool()
	s.registerChangeTool()
	s.registerReauthorizeTool()
}

// authMiddleware validates OAuth2 bearer tokens for MCP requests.
//
// On 401, it emits an RFC 9728 / MCP-authorization-spec compliant
// WWW-Authenticate header pointing at the protected-resource-metadata URL.
// MCP clients (Claude Code, Cursor, etc.) use this header to bootstrap the
// OAuth flow without out-of-band configuration.
func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// Extract Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return s.unauthorized(c, "authorization required")
		}

		// Validate Bearer format
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return s.unauthorized(c, "authorization header format must be Bearer {token}")
		}
		tokenStr := parts[1]
		if _, revoked := s.revokedAccessTokens.Load(tokenStr); revoked {
			return s.unauthorized(c, "token revoked")
		}

		// Parse and validate JWT
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Name {
				return nil, errors.New("invalid token signing method")
			}
			if kid, ok := t.Header["kid"].(string); ok && kid == "v1" {
				return []byte(s.secret), nil
			}
			return nil, errors.New("invalid token key id")
		})
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				return s.unauthorized(c, "token expired")
			}
			return s.unauthorized(c, "invalid token")
		}

		if !token.Valid {
			return s.unauthorized(c, "invalid token")
		}

		identity, errMsg := extractTokenIdentity(claims)
		if errMsg != "" {
			return s.unauthorized(c, errMsg)
		}
		sub, clientID, workspaceID, aud := identity.sub, identity.clientID, identity.workspaceID, identity.aud
		allowed, err := s.audienceAllowed(c.Request().Context(), aud, workspaceID)
		if err != nil {
			// Infra failure reading the trusted config, not a verdict on the
			// token: a 401 here would tell a compliant client to discard a
			// perfectly good token mid-outage. 503 says retry instead.
			slog.Error("failed to resolve the MCP resource audience; cannot verify the token", log.BBError(err))
			return echo.NewHTTPError(http.StatusServiceUnavailable, "cannot verify token audience; retry shortly")
		}
		if !allowed {
			return s.unauthorized(c, "invalid token: audience mismatch")
		}

		// Enforce the workspace MCP capability ceiling before dispatching to any
		// tool. DISABLED rejects the connection outright, and so does READ_ONLY:
		// the per-method gate on the internal chain now serves a read-only
		// ceiling correctly, but the query tool still reaches SQLService/Query,
		// which authorizes writes per statement. Admitting a read-only session
		// before that clamp exists would be a read-only session that can write.
		// Read live so an admin change takes effect on the next request without
		// re-issuing tokens.
		if !mcpConnectionAllowed(s.mcpCapability(c.Request().Context(), workspaceID)) {
			return echo.NewHTTPError(http.StatusForbidden, "MCP access is disabled for this workspace by policy")
		}

		// Establish the delegated identity that carries this request's principal
		// and grant state onto the private in-memory transport. The inbound
		// bearer stops at this boundary: internal API requests mint their own
		// credential from this identity and never see the bearer. Grant state is
		// copied verbatim from the inbound token — empty for legacy sessions
		// (common.DelegatedGrant documents the empty-state semantics; P1b
		// resolves them). Identity only, no roles: downstream authorization
		// re-resolves live exactly as for a public request.
		delegated := auth.DelegatedMCPCredential{
			Principal:     sub,
			WorkspaceID:   workspaceID,
			ClientID:      clientID,
			CorrelationID: uuid.NewString(),
			Scope:         grantScope(claims),
			Resource:      grantResource(claims, aud),
		}

		// Store access token and workspace ID in request context for MCP tools.
		ctx := c.Request().Context()
		ctx = withAccessToken(ctx, tokenStr)
		ctx = withUserEmail(ctx, sub)
		ctx = withOAuth2ClientID(ctx, clientID)
		ctx = withWorkspaceID(ctx, workspaceID)
		ctx = withDelegatedIdentity(ctx, delegated)
		// Normalize the resolved address onto the request so the per-request
		// path sees it too: receiving middleware gets each request's headers,
		// but never its peer address.
		resolvedIP := callerIP(c.Request())
		c.Request().Header.Set(headerRealIP, resolvedIP)
		ctx = withCallerIP(ctx, resolvedIP)
		ctx = withSessionBinding(ctx, sessionBinding{
			fingerprint: sessionFingerprint(delegated),
			expiry:      bearerExpiry(claims),
		})
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// mcpResourcePath is the path of the MCP endpoint. Appended to the trusted
// external URL it forms the canonical MCP resource URI — the audience every
// OAuth2 access token is minted with since P1a PR 3 (mirrors the constant of
// the same name in the oauth2 package, which stores that URI on the grant).
const mcpResourcePath = "/mcp"

// Caller-IP headers, in the precedence the audit interceptor reads them.
const (
	headerRealIP       = "X-Real-IP"
	headerForwardedFor = "X-Forwarded-For"
)

// RegisterRoutes registers the MCP server routes with Echo.
func (s *Server) RegisterRoutes(e *echo.Echo) {
	// MCP Streamable HTTP endpoint with authentication
	e.Any(mcpResourcePath, echo.WrapHandler(s.httpHandler), s.authMiddleware)
}

// audienceAllowed reports whether a bearer token's audience admits it to /mcp.
//
// The durable rule is a single audience: the canonical MCP resource URI derived
// from the trusted live config (never from request headers). Tokens are minted
// from the grant's stored resource, so rotating the external URL makes them
// mismatch here — a clean 401 at use time that drives the client to
// re-authorize, by design (grants must not silently rebind to a URL the user
// never consented to).
//
// Legacy bb.oauth2.access tokens are admitted while their own exp claim keeps
// them alive — deliberately with no clock gate. This release never mints that
// audience, but replicas running the previous release keep minting it until
// they leave service (the grant gates only constrain new-code replicas), so
// any process-local window races a rolling deploy: it can close before the
// last old-replica mint expires, 401-ing valid tokens depending on which
// replica serves the request. Keying on token expiry instead gives the exact
// bound: legacy acceptance drains no later than one access-token lifetime
// after the last legacy-capable replica leaves service. Remove the acceptance
// entirely in a later release, once no supported mixed-version deployment can
// still mint the audience.
//
// Plain bb.user.access session tokens pasted into MCP clients manually are
// admitted for now — see the acceptance branch in decideAudience for the
// retirement gating.
//
// If no trusted external URL is configured, there is no expected audience and
// resource-bound tokens fail closed; after the window such a deployment
// rejects everything at /mcp until the external URL is set, matching the PR 2
// rule that MCP OAuth requires a configured external URL. A transient failure
// reading the config is different: it returns an error rather than a false
// verdict, so the caller reports a server problem instead of blaming the token.
func (s *Server) audienceAllowed(ctx context.Context, aud any, workspaceID string) (bool, error) {
	expected, resolveErr := s.expectedMCPAudience(ctx, workspaceID)
	return decideAudience(aud, expected, resolveErr)
}

// decideAudience is the pure decision behind audienceAllowed, split out so the
// resolver-failure branches are unit-testable without a failing store.
func decideAudience(aud any, expected string, resolveErr error) (bool, error) {
	switch {
	case resolveErr == nil:
		if audienceMatches(aud, expected) {
			return true, nil
		}
	case connect.CodeOf(resolveErr) == connect.CodeFailedPrecondition:
		// Deliberately unconfigured (GetEffectiveExternalURL's "isn't setup
		// yet"): no expected audience exists, so fall through to the legacy
		// audiences rather than erroring.
		slog.Warn("no external URL is configured; only legacy-audience tokens can authenticate at /mcp",
			log.BBError(resolveErr))
	default:
		// Infra failure reading the trusted config: not a verdict on the token.
		return false, resolveErr
	}
	// Plain web-session tokens stay accepted at /mcp for now: bb.user.access is
	// minted by every web login, so no migration window can retire it, and the
	// spec gates the hard cut on a scoped service-account / workload-identity
	// flow existing first (spec §"Deferred product decisions"; proposal §6.5
	// names the service-account token proxy as the dependent). PR 5/6 replaces
	// this acceptance with a rejection once that credential ships, bringing its
	// own tests.
	if audienceMatches(aud, auth.AccessTokenAudience) {
		return true, nil
	}
	// Unexpired legacy tokens only: the JWT exp check upstream has already
	// rejected the rest, and this release mints none, so this branch drains on
	// its own (see audienceAllowed).
	return audienceMatches(aud, auth.OAuth2AccessTokenAudience), nil
}

// expectedMCPAudience resolves the audience /mcp accepts: the trusted external
// URL (flag or workspace setting, per utils.GetEffectiveExternalURL) plus the
// /mcp path. Request-derived values must never feed this — a Host header is
// attacker-controlled and would let anyone mint a matching binding.
func (s *Server) expectedMCPAudience(ctx context.Context, workspaceID string) (string, error) {
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		return "", err
	}
	return externalURL + mcpResourcePath, nil
}

// sessionFingerprint is the identity a /mcp session is pinned to for its whole
// life. Two bearers may drive the same session only if they agree on every
// field: principal, workspace, OAuth client, and the grant's stored resource
// and scope.
//
// Substituting any of them is a different session: another user's token would
// otherwise execute as the session's principal, a token for another workspace
// would sidestep the kill switch on the session's own workspace, and a
// re-consented grant would keep riding the state consented earlier. The
// correlation ID is deliberately excluded — it is per request, not per session.
//
// The separator cannot appear in any component, so distinct identities cannot
// collide by concatenation.
func sessionFingerprint(identity auth.DelegatedMCPCredential) string {
	return strings.Join([]string{
		identity.Principal,
		identity.WorkspaceID,
		identity.ClientID,
		identity.Resource,
		identity.Scope,
	}, "\x00")
}

// liveRequestMetadata overlays the current request's caller IP and bearer onto
// the context the tool handler runs with. Tool handlers otherwise see only what
// the session was opened with, which goes stale in two ways that matter:
//
//   - the caller IP, so a client that changed network mid-session would have
//     every later action audited against its first address; and
//   - the bearer, so reauthorize would revoke the token the caller already
//     replaced by refreshing, leaving the one in its hand working until expiry.
//
// The headers come from the live JSON-RPC request. authMiddleware has already
// normalized the peer address into X-Real-IP, so a direct connection resolves
// here too, and it has already validated the bearer, so this only carries a
// value that was accepted moments ago. If a request arrives without them, the
// session's values stand.
func liveRequestMetadata(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if extra := req.GetExtra(); extra != nil && extra.Header != nil {
			if ip := headerCallerIP(extra.Header); ip != "" {
				ctx = withCallerIP(ctx, ip)
			}
			if token, err := auth.GetTokenFromHeaders(extra.Header); err == nil && token != "" {
				ctx = withAccessToken(ctx, token)
			}
		}
		return next(ctx, method, req)
	}
}

// headerCallerIP reads the caller IP from request headers, in the order the
// audit interceptor applies: the proxy-set single IP first, then the standard
// forwarding chain.
func headerCallerIP(header http.Header) string {
	if ip := header.Get(headerRealIP); ip != "" {
		return ip
	}
	return header.Get(headerForwardedFor)
}

// callerIP resolves who made this /mcp request: the forwarding headers if
// present, otherwise the peer address, whose port is dropped so the value reads
// as an IP either way. Same precedence the audit interceptor applies to a
// request that reaches the v1 API directly.
//
// The forwarding headers are client-controllable, exactly as they are on that
// direct path. Reading them preserves the existing trust model rather than
// introducing one.
func callerIP(r *http.Request) string {
	if ip := headerCallerIP(r.Header); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// bearerExpiry reads the inbound token's expiry. The JWT parse upstream has
// already rejected expired and malformed tokens, so this only re-surfaces the
// value for the SDK, which requires a non-zero future expiration.
func bearerExpiry(claims jwt.MapClaims) time.Time {
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return time.Time{}
	}
	return exp.Time
}

// verifySessionBinding hands the SDK the identity behind the current request so
// it can enforce session ownership. The bearer itself was already validated by
// authMiddleware, which runs first and puts the resolved identity on the
// request context; this only translates that into the SDK's shape.
func (*Server) verifySessionBinding(_ context.Context, _ string, req *http.Request) (*mcpauth.TokenInfo, error) {
	binding, ok := getSessionBinding(req.Context())
	if !ok {
		// Unreachable through the registered route: authMiddleware establishes
		// the binding before this runs. Fail closed rather than leave a session
		// unpinned if that ever stops being true.
		return nil, mcpauth.ErrInvalidToken
	}
	return &mcpauth.TokenInfo{
		UserID:     binding.fingerprint,
		Expiration: binding.expiry,
	}, nil
}

// tokenIdentity is the identity material authMiddleware works with after a
// token verifies: subject, workspace, audience, and an optional client.
// The workspace is extracted before the audience check because resolving the
// expected audience needs it to look up the trusted external URL.
type tokenIdentity struct {
	sub         string
	clientID    string
	workspaceID string
	aud         any
}

// extractTokenIdentity pulls the identity claims out of a verified token. A
// missing subject or audience is a token defect (non-empty error message);
// client_id is optional; workspace_id is required for every access token.
func extractTokenIdentity(claims jwt.MapClaims) (*tokenIdentity, string) {
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, "invalid token: missing subject"
	}
	aud, ok := claims["aud"]
	if !ok {
		return nil, "invalid token: missing audience"
	}
	identity := &tokenIdentity{sub: sub, aud: aud}
	if clientID, ok := claims["client_id"].(string); ok {
		identity.clientID = clientID
	}
	workspaceID, ok := claims["workspace_id"].(string)
	if !ok || workspaceID == "" {
		return nil, "invalid token: missing workspace"
	}
	identity.workspaceID = workspaceID
	return identity, ""
}

// grantScope extracts the grant's stored scope from the inbound token's claims.
// The scope claim is minted onto OAuth2 access tokens verbatim from the grant;
// legacy tokens (plain sessions, pre-scope OAuth2) have none — empty grant
// state, by design.
func grantScope(claims jwt.MapClaims) string {
	if scope, ok := claims["scope"].(string); ok {
		return scope
	}
	return ""
}

// grantResource extracts the grant's stored resource from the inbound token.
// For MCP OAuth2 tokens the resource-bound audience IS the stored resource
// (PR 3 mints it from the grant verbatim). The legacy fixed audiences are not
// resources, so legacy tokens yield empty grant state.
//
// Resource is therefore what distinguishes a grant-backed session with an
// empty scope (a scope-less consent, or a PR-3-era token predating the scope
// claim) from a genuinely pre-grant one, which carries neither.
// common.DelegatedGrant — where PR 5 carries both values verbatim — documents
// why the two must never collapse.
func grantResource(claims jwt.MapClaims, aud any) string {
	tokenUse, ok := claims["token_use"].(string)
	if !ok || tokenUse != auth.TokenUseMCP {
		return ""
	}
	if audienceMatches(aud, auth.OAuth2AccessTokenAudience) || audienceMatches(aud, auth.AccessTokenAudience) {
		return ""
	}
	switch v := aud.(type) {
	case string:
		return v
	case []any:
		for _, a := range v {
			if str, ok := a.(string); ok {
				return str
			}
		}
	default:
	}
	return ""
}

// audienceMatches reports whether a JWT aud claim (string or array form)
// contains exactly want.
func audienceMatches(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, a := range v {
			if str, ok := a.(string); ok && str == want {
				return true
			}
		}
	default:
	}
	return false
}

// unauthorized writes a 401 with an RFC 9728 / MCP-authorization-spec
// WWW-Authenticate header so compliant MCP clients can auto-discover the
// authorization server. The header references the host-global protected
// resource metadata endpoint (served by the oauth2 package).
func (s *Server) unauthorized(c *echo.Context, errDescription string) error {
	resourceMetadataURL, err := s.buildResourceMetadataURL(c)
	if err != nil {
		slog.Error("failed to resolve external URL for MCP discovery", log.BBError(err))
		return echo.NewHTTPError(http.StatusServiceUnavailable, "OAuth2 discovery is unavailable").Wrap(err)
	}
	c.Response().Header().Set(
		"WWW-Authenticate",
		fmt.Sprintf(
			`Bearer realm="OAuth", resource_metadata=%q, error="invalid_token", error_description=%q`,
			resourceMetadataURL, errDescription,
		),
	)
	return echo.NewHTTPError(http.StatusUnauthorized, errDescription)
}

// mcpConnectionAllowed reports whether an MCP connection may proceed under the
// resolved workspace capability ceiling. DISABLED is rejected, and so is
// READ_ONLY: per-method enforcement exists from P1b 1b-2, but the SQL statement
// clamp does not, so a read-only session admitted here could still write
// through the query tool. 1b-3 lands the clamp and flips this to
// allow-with-clamp; nothing before it may. Unknown stored values (e.g. the
// reserved number) hit the default arm and fail closed too.
func mcpConnectionAllowed(capability storepb.WorkspaceProfileSetting_MCPCapability) bool {
	switch capability {
	case storepb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED,
		storepb.WorkspaceProfileSetting_READ_WRITE:
		return true
	default:
		return false
	}
}

// mcpCapability resolves the workspace's effective MCP capability ceiling for
// the connection gate. Anything the store cannot make sense of — a failed
// read, a stored value this build does not know — fails closed to DISABLED, so
// a policy that cannot be read never silently permits MCP.
//
// The resolution itself lives in the store, uncached, and the request gate on
// the internal chain reads it from there too: the connection and the per-method
// gate must never disagree about what a workspace's ceiling is. Bypassing the
// setting cache is what makes the kill switch a kill switch — the cache has no
// TTL and only in-process writes refresh it, so a profile cached as unset would
// keep admitting MCP after the ceiling was flipped out of band.
func (s *Server) mcpCapability(ctx context.Context, workspaceID string) storepb.WorkspaceProfileSetting_MCPCapability {
	capability, err := s.store.GetMCPCapabilityUncached(ctx, workspaceID)
	if err != nil {
		slog.Warn("failed to read the MCP capability ceiling; failing closed",
			slog.String("workspace", workspaceID), log.BBError(err))
		return storepb.WorkspaceProfileSetting_DISABLED
	}
	return capability
}

// buildResourceMetadataURL returns the absolute URL of the protected resource
// metadata document for the /mcp endpoint. The `/mcp` path suffix matters:
// RFC 9728 §3.3 requires the document's `resource` field to match the resource
// the client is accessing, and the path-suffixed well-known URL is how the
// metadata handler in the oauth2 package knows to publish `resource=<host>/mcp`.
func (s *Server) buildResourceMetadataURL(c *echo.Context) (string, error) {
	const resourceMetadataPath = "/.well-known/oauth-protected-resource/mcp"
	externalURL, err := utils.GetDiscoveryExternalURL(c.Request().Context(), s.store, s.profile)
	if err != nil {
		return "", err
	}
	return externalURL + resourceMetadataPath, nil
}
