package oauth2

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

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
func (s *Service) getBaseURL(c echo.Context) string {
	ctx := c.Request().Context()

	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile)
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

func (s *Service) handleDiscovery(c echo.Context) error {
	baseURL := s.getBaseURL(c)
	metadata := &authorizationServerMetadata{
		Issuer:                            baseURL,
		AuthorizationEndpoint:             fmt.Sprintf("%s/api/oauth2/authorize", baseURL),
		TokenEndpoint:                     fmt.Sprintf("%s/api/oauth2/token", baseURL),
		RegistrationEndpoint:              fmt.Sprintf("%s/api/oauth2/register", baseURL),
		RevocationEndpoint:                fmt.Sprintf("%s/api/oauth2/revoke", baseURL),
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
	}
	return c.JSON(http.StatusOK, metadata)
}

// handleProtectedResourceMetadata returns RFC 9728 protected resource metadata.
// This tells clients which authorization server protects this resource.
func (s *Service) handleProtectedResourceMetadata(c echo.Context) error {
	baseURL := s.getBaseURL(c)
	metadata := &protectedResourceMetadata{
		Resource:               baseURL,
		AuthorizationServers:   []string{baseURL},
		BearerMethodsSupported: []string{"header"},
	}
	return c.JSON(http.StatusOK, metadata)
}
