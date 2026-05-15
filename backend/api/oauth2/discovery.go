package oauth2

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/utils"
)

type authorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// protectedResourceMetadata is per RFC 9728 OAuth 2.0 Protected Resource Metadata.
type protectedResourceMetadata struct {
	Resource                    string   `json:"resource"`
	AuthorizationServers        []string `json:"authorization_servers"`
	BearerMethodsSupported      []string `json:"bearer_methods_supported,omitempty"`
	ResourceSigningAlgSupported []string `json:"resource_signing_alg_values_supported,omitempty"`
	ResourceDocumentation       string   `json:"resource_documentation,omitempty"`
}

// getBaseURL returns the base URL to use for OAuth2 endpoints.
// It uses externalURL from profile/setting if configured, otherwise derives from the request.
func (s *Service) getBaseURL(c *echo.Context) string {
	ctx := c.Request().Context()

	// The --external-url CLI flag (profile.ExternalURL) short-circuits the
	// lookup. Otherwise on self-hosted we resolve the singleton workspace ID
	// first so GetEffectiveExternalURL can find the DB-backed
	// workspace_profile.external_url setting. On SaaS there is no singleton
	// — the CLI flag is required.
	if s.profile.ExternalURL != "" {
		return strings.TrimSuffix(s.profile.ExternalURL, "/")
	}
	workspaceID := ""
	if !s.profile.SaaS {
		if ws, err := s.store.GetWorkspaceID(ctx); err == nil {
			workspaceID = ws
		}
	}
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		slog.Warn("failed to get external url for OAuth2", log.BBError(err))
	}
	if externalURL != "" {
		return strings.TrimSuffix(externalURL, "/")
	}

	// Derive from request as fallback
	req := c.Request()
	scheme := "https"
	if req.TLS == nil {
		scheme = "http"
	}
	// Check X-Forwarded-Proto header for reverse proxy setups
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s", scheme, req.Host)
}

func (s *Service) handleDiscovery(c *echo.Context) error {
	baseURL := s.getBaseURL(c)
	oauthBase := fmt.Sprintf("%s/api/oauth2", baseURL)
	metadata := &authorizationServerMetadata{
		Issuer:                            baseURL,
		AuthorizationEndpoint:             fmt.Sprintf("%s/authorize", oauthBase),
		TokenEndpoint:                     fmt.Sprintf("%s/token", oauthBase),
		RegistrationEndpoint:              fmt.Sprintf("%s/register", oauthBase),
		RevocationEndpoint:                fmt.Sprintf("%s/revoke", oauthBase),
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
	}
	return c.JSON(http.StatusOK, metadata)
}

// handleProtectedResourceMetadata returns RFC 9728 protected resource metadata.
// This tells clients which authorization server protects this resource.
//
// RFC 9728 §3.3 requires the `resource` value to match the resource the client
// is accessing. We support both:
//   - GET /.well-known/oauth-protected-resource         → resource = baseURL
//   - GET /.well-known/oauth-protected-resource/<path>  → resource = baseURL + /<path>
//
// The path-suffixed form lets the `/mcp` endpoint advertise metadata that
// strict clients validate against the resource URL they actually requested.
func (s *Service) handleProtectedResourceMetadata(c *echo.Context) error {
	baseURL := s.getBaseURL(c)

	const wellKnownPrefix = "/.well-known/oauth-protected-resource"
	resource := baseURL
	if subPath := strings.TrimPrefix(c.Request().URL.Path, wellKnownPrefix); subPath != "" && subPath != "/" {
		// Ensure leading slash and trim any trailing slash.
		if !strings.HasPrefix(subPath, "/") {
			subPath = "/" + subPath
		}
		resource = baseURL + strings.TrimRight(subPath, "/")
	}

	metadata := &protectedResourceMetadata{
		Resource:               resource,
		AuthorizationServers:   []string{baseURL},
		BearerMethodsSupported: []string{"header"},
	}
	return c.JSON(http.StatusOK, metadata)
}
