// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	bbauth "github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/store"
)

// Server wraps the MCP server and provides HTTP handler integration.
type Server struct {
	mcpServer       *mcp.Server
	registry        *Registry
	store           *store.Store
	profile         *config.Profile
	stateCfg        *state.State
	authInterceptor *bbauth.APIAuthInterceptor
}

// NewServer creates a new MCP server with all Bytebase tools registered.
func NewServer(registry *Registry, s *store.Store, profile *config.Profile, stateCfg *state.State, authInterceptor *bbauth.APIAuthInterceptor) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "bytebase",
		Version: "1.0.0",
	}, nil)

	// Register all tools from the registry
	if err := registry.RegisterTools(mcpServer); err != nil {
		return nil, err
	}

	return &Server{
		mcpServer:       mcpServer,
		registry:        registry,
		store:           s,
		profile:         profile,
		stateCfg:        stateCfg,
		authInterceptor: authInterceptor,
	}, nil
}

// tokenVerifier creates a token verifier function for the MCP auth middleware.
// It validates Bytebase JWT tokens using the auth interceptor.
func (s *Server) tokenVerifier() auth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		// Validate Bytebase JWT token
		user, expiry, err := s.authInterceptor.AuthenticateToken(ctx, token)
		if err != nil {
			return nil, auth.ErrInvalidToken
		}

		return &auth.TokenInfo{
			Scopes:     []string{"mcp:tools"},
			Expiration: expiry,
			Extra: map[string]any{
				"user_id": user.ID,
				"email":   user.Email,
			},
		}, nil
	}
}

// Handler returns the HTTP handler for the MCP server.
// It wraps the SDK handler to inject auth context and applies OAuth middleware.
func (s *Server) Handler(resourceMetadataURL string) http.Handler {
	sdkHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			// Extract auth header and store in invoker context
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// Store in request context for invoker to use
				ctx := WithAuthHeader(r.Context(), authHeader)
				*r = *r.WithContext(ctx)
			}
			return s.mcpServer
		},
		nil,
	)

	// Wrap with OAuth authentication middleware
	return auth.RequireBearerToken(s.tokenVerifier(), &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: resourceMetadataURL,
		Scopes:              []string{"mcp:tools"},
	})(sdkHandler)
}
