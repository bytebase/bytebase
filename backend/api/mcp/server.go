// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/component/config"
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

	// Create HTTP handler for streamable HTTP transport
	s.httpHandler = mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)

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

		// Validate audience - accept both user access tokens and OAuth2 access tokens
		aud, ok := claims["aud"]
		if !ok {
			return s.unauthorized(c, "invalid token: missing audience")
		}

		validAudience := false
		switch v := aud.(type) {
		case string:
			validAudience = v == auth.OAuth2AccessTokenAudience || v == auth.AccessTokenAudience
		case []any:
			for _, a := range v {
				if str, ok := a.(string); ok && (str == auth.OAuth2AccessTokenAudience || str == auth.AccessTokenAudience) {
					validAudience = true
					break
				}
			}
		default:
		}
		if !validAudience {
			return s.unauthorized(c, "invalid token: audience mismatch")
		}

		// Extract user email from subject
		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			return s.unauthorized(c, "invalid token: missing subject")
		}

		// Extract workspace ID from token claims.
		workspaceID, ok := claims["workspace_id"].(string)
		if !ok {
			workspaceID = ""
		}

		// Store access token and workspace ID in request context for MCP tools.
		ctx := c.Request().Context()
		ctx = withAccessToken(ctx, tokenStr)
		ctx = withWorkspaceID(ctx, workspaceID)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// RegisterRoutes registers the MCP server routes with Echo.
func (s *Server) RegisterRoutes(e *echo.Echo) {
	// MCP Streamable HTTP endpoint with authentication
	e.Any("/mcp", echo.WrapHandler(s.httpHandler), s.authMiddleware)
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
		return strings.TrimSuffix(s.profile.ExternalURL, "/") + resourceMetadataPath
	}
	workspaceID := ""
	if !s.profile.SaaS {
		if ws, err := s.store.GetWorkspaceID(ctx); err == nil {
			workspaceID = ws
		}
	}
	if externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID); err == nil && externalURL != "" {
		return strings.TrimSuffix(externalURL, "/") + resourceMetadataPath
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
