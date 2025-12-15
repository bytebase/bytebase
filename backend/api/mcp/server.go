// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

// Server is the MCP server for Bytebase.
type Server struct {
	mcpServer    *mcp.Server
	httpHandler  *mcp.StreamableHTTPHandler
	store        *store.Store
	profile      *config.Profile
	secret       string
	openAPIIndex *OpenAPIIndex
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
}

// authMiddleware validates OAuth2 bearer tokens for MCP requests.
func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Extract Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "authorization required")
		}

		// Validate Bearer format
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "authorization header format must be Bearer {token}")
		}
		tokenStr := parts[1]

		// Parse and validate JWT
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Name {
				return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid token signing method")
			}
			if kid, ok := t.Header["kid"].(string); ok && kid == "v1" {
				return []byte(s.secret), nil
			}
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid token key id")
		})
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		if !token.Valid {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		// Validate audience - must be OAuth2 access token
		aud, ok := claims["aud"]
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token: missing audience")
		}

		validAudience := false
		switch v := aud.(type) {
		case string:
			validAudience = v == auth.OAuth2AccessTokenAudience
		case []any:
			for _, a := range v {
				if str, ok := a.(string); ok && str == auth.OAuth2AccessTokenAudience {
					validAudience = true
					break
				}
			}
		}
		if !validAudience {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token: audience mismatch")
		}

		// Extract user email from subject
		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token: missing subject")
		}

		// Store access token in request context for MCP tools to forward auth
		ctx := c.Request().Context()
		ctx = withAccessToken(ctx, tokenStr)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// RegisterRoutes registers the MCP server routes with Echo.
func (s *Server) RegisterRoutes(e *echo.Echo) {
	// MCP Streamable HTTP endpoint with authentication
	e.Any("/mcp", echo.WrapHandler(s.httpHandler), s.authMiddleware)
}
