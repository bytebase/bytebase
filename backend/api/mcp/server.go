// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
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
	httpHandler  *mcp.StreamableHTTPHandler
	store        *store.Store
	profile      *config.Profile
	secret       string
	openAPIIndex *OpenAPIIndex

	revokedAccessTokens sync.Map // map[string]struct{}

	// planCheckPollBudgetOverride lets tests shorten the plan-check poll budget.
	// Zero means use the default (planCheckPollBudget).
	planCheckPollBudgetOverride time.Duration
}

// NewServer creates a new MCP server.
func NewServer(store *store.Store, profile *config.Profile, secret string) (*Server, error) {
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
		mcpServer:    mcpServer,
		store:        store,
		profile:      profile,
		secret:       secret,
		openAPIIndex: openAPIIndex,
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
	s.httpHandler = mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{DisableLocalhostProtection: true})

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

		// Extract user email from subject
		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			return s.unauthorized(c, "invalid token: missing subject")
		}
		clientID, ok := claims["client_id"].(string)
		if !ok {
			clientID = ""
		}

		// Extract workspace ID from token claims. Read before the audience
		// check because resolving the expected audience needs the workspace to
		// look up the trusted external URL.
		workspaceID, ok := claims["workspace_id"].(string)
		if !ok {
			workspaceID = ""
		}

		aud, ok := claims["aud"]
		if !ok {
			return s.unauthorized(c, "invalid token: missing audience")
		}
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
		// tool. DISABLED — and, until per-tool enforcement lands in a later
		// phase, the not-yet-enforceable READ_ONLY ceiling — reject the
		// connection outright. Read live so an admin change takes effect on the
		// next request without re-issuing tokens.
		if !mcpConnectionAllowed(s.mcpCapability(c.Request().Context(), workspaceID)) {
			return echo.NewHTTPError(http.StatusForbidden, "MCP access is disabled for this workspace by policy")
		}

		// Store access token and workspace ID in request context for MCP tools.
		ctx := c.Request().Context()
		ctx = withAccessToken(ctx, tokenStr)
		ctx = withUserEmail(ctx, sub)
		ctx = withOAuth2ClientID(ctx, clientID)
		ctx = withWorkspaceID(ctx, workspaceID)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// mcpResourcePath is the path of the MCP endpoint. Appended to the trusted
// external URL it forms the canonical MCP resource URI — the audience every
// OAuth2 access token is minted with since P1a PR 3 (mirrors the constant of
// the same name in the oauth2 package, which stores that URI on the grant).
const mcpResourcePath = "/mcp"

// RegisterRoutes registers the MCP server routes with Echo.
func (s *Server) RegisterRoutes(e *echo.Echo) {
	// MCP Streamable HTTP endpoint with authentication
	e.Any(mcpResourcePath, echo.WrapHandler(s.httpHandler), s.authMiddleware)
}

// rejectPlainUserTokenAtMCP is the flip that retires plain web-session tokens
// (bb.user.access) at /mcp. Deliberately inactive: unlike bb.oauth2.access,
// this audience is minted by every web login, so no migration window can
// retire it — and the spec gates the hard cut on a scoped service-account /
// workload-identity flow existing first (spec §"Deferred product decisions";
// proposal §6.5 names the service-account token proxy as the dependent).
// Flipped in PR 5/6 once that credential ships.
const rejectPlainUserTokenAtMCP = false

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
// admitted until rejectPlainUserTokenAtMCP flips.
//
// If no trusted external URL is configured, there is no expected audience and
// resource-bound tokens fail closed; after the window such a deployment
// rejects everything at /mcp until the external URL is set, matching the PR 2
// rule that MCP OAuth requires a configured external URL. A transient failure
// reading the config is different: it returns an error rather than a false
// verdict, so the caller reports a server problem instead of blaming the token.
func (s *Server) audienceAllowed(ctx context.Context, aud any, workspaceID string) (bool, error) {
	expected, resolveErr := s.expectedMCPAudience(ctx, workspaceID)
	return decideAudience(aud, expected, resolveErr, rejectPlainUserTokenAtMCP)
}

// decideAudience is the pure decision behind audienceAllowed, split out so the
// resolver-failure branches and the rejectPlainUserToken flip are
// unit-testable without a failing store or a const edit.
func decideAudience(aud any, expected string, resolveErr error, rejectPlainUserToken bool) (bool, error) {
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
	if !rejectPlainUserToken && audienceMatches(aud, auth.AccessTokenAudience) {
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
	if workspaceID == "" && !s.profile.SaaS {
		if id, err := s.store.GetWorkspaceID(ctx); err == nil {
			workspaceID = id
		}
	}
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		return "", err
	}
	return externalURL + mcpResourcePath, nil
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
	resourceMetadataURL := s.buildResourceMetadataURL(c)
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
// resolved workspace capability ceiling. This phase enforces the connection-level
// gate only: DISABLED is rejected, and the not-yet-enforceable
// READ_ONLY ceiling is also rejected — a ceiling the server cannot yet
// apply per-tool must fail closed rather than silently grant read-write. A
// later phase turns it into allow-with-clamp. Unknown stored values (e.g. a
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

// mcpCapability resolves the workspace's effective MCP capability ceiling. An
// unset ceiling resolves to READ_WRITE so workspaces that never configured
// one keep working; a genuine lookup error fails closed to DISABLED so a
// policy that cannot be read never silently permits MCP. A missing setting row
// is treated as unset, not an error.
//
// The read deliberately bypasses the store's setting cache: the cache has no
// TTL and only in-process writes refresh it, so a profile cached as unset
// would keep admitting MCP indefinitely after the ceiling is flipped by an
// out-of-band admin path (direct SQL, another process). A kill switch must
// observe the stored truth on the next request.
func (s *Server) mcpCapability(ctx context.Context, workspaceID string) storepb.WorkspaceProfileSetting_MCPCapability {
	if s.store == nil {
		return storepb.WorkspaceProfileSetting_READ_WRITE
	}
	if workspaceID == "" && !s.profile.SaaS {
		if id, err := s.store.GetWorkspaceID(ctx); err == nil {
			workspaceID = id
		}
	}
	setting, err := s.store.GetSettingUncached(ctx, workspaceID, storepb.SettingName_WORKSPACE_PROFILE)
	if err != nil {
		slog.Warn("failed to read MCP capability policy; failing closed",
			slog.String("workspace", workspaceID), log.BBError(err))
		return storepb.WorkspaceProfileSetting_DISABLED
	}
	if setting == nil {
		return storepb.WorkspaceProfileSetting_READ_WRITE
	}
	profile, ok := setting.Value.(*storepb.WorkspaceProfileSetting)
	if !ok {
		return storepb.WorkspaceProfileSetting_READ_WRITE
	}
	if capability := profile.GetMcpCapability(); capability != storepb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED {
		return capability
	}
	return storepb.WorkspaceProfileSetting_READ_WRITE
}

// buildResourceMetadataURL returns the absolute URL of the protected resource
// metadata document for the /mcp endpoint. The `/mcp` path suffix matters:
// RFC 9728 §3.3 requires the document's `resource` field to match the resource
// the client is accessing, and the path-suffixed well-known URL is how the
// metadata handler in the oauth2 package knows to publish `resource=<host>/mcp`.
//
// The configured effective external URL is preferred over request-derived
// host/proto so that proxied deployments (where the inbound Host can differ
// from the public endpoint) emit the correct public URL to MCP clients.
// Request-derived values are the last-resort fallback only.
//
// The --external-url CLI flag (profile.ExternalURL) short-circuits the lookup;
// otherwise on self-hosted we resolve the singleton workspace ID first so the
// DB-backed workspace_profile.external_url setting in GetEffectiveExternalURL
// can be found. On SaaS there is no singleton — the CLI flag is required.
func (s *Server) buildResourceMetadataURL(c *echo.Context) string {
	const resourceMetadataPath = "/.well-known/oauth-protected-resource/mcp"

	ctx := c.Request().Context()
	if s.profile.ExternalURL != "" {
		return s.profile.ExternalURL + resourceMetadataPath
	}
	workspaceID := ""
	if !s.profile.SaaS {
		if ws, err := s.store.GetWorkspaceID(ctx); err == nil {
			workspaceID = ws
		}
	}
	if externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID); err == nil {
		return externalURL + resourceMetadataPath
	}

	req := c.Request()
	scheme := "https"
	if req.TLS == nil {
		scheme = "http"
	}
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, resourceMetadataPath)
}
