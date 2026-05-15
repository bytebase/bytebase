package oauth2

import (
	"net/http"
	"slices"

	"github.com/labstack/echo/v5"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Bounds on Dynamic Client Registration input. These are loose enough that no
// realistic MCP client trips them, but tight enough that degenerate input
// (megabyte-sized client_name fields, hundreds of redirect URIs) is rejected
// before any DB or bcrypt work happens — the endpoint is unauthenticated and
// publicly reachable on SaaS.
const (
	maxClientNameLen  = 200
	maxRedirectURIs   = 5
	maxRedirectURILen = 2048
)

type clientRegistrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type clientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// handleRegister implements RFC 7591 Dynamic Client Registration. The endpoint
// is unauthenticated — clients are workspace-agnostic and get bound to a
// workspace when the user grants consent at /authorize. This matches the
// pattern used by Linear, Atlassian, Notion, and Cloudflare MCP servers.
func (s *Service) handleRegister(c *echo.Context) error {
	ctx := c.Request().Context()

	var req clientRegistrationRequest
	if err := c.Bind(&req); err != nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "failed to parse request body")
	}

	if req.ClientName == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "client_name is required")
	}
	if len(req.ClientName) > maxClientNameLen {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "client_name is too long")
	}

	if len(req.RedirectURIs) == 0 {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "redirect_uris is required")
	}
	if len(req.RedirectURIs) > maxRedirectURIs {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "too many redirect_uris")
	}
	for _, uri := range req.RedirectURIs {
		if len(uri) > maxRedirectURILen {
			return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "redirect URI is too long")
		}
		if !isAllowedDynamicClientRedirectURI(uri) {
			return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "redirect URI must be a localhost URL or a whitelisted app scheme (cursor://, vscode://, vscode-insiders://, jetbrains://gateway/...)")
		}
	}

	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}
	allowedGrantTypes := []string{"authorization_code", "refresh_token"}
	for _, gt := range req.GrantTypes {
		if !slices.Contains(allowedGrantTypes, gt) {
			return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "unsupported grant_type: "+gt)
		}
	}

	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "none"
	}
	allowedAuthMethods := []string{"none"}
	if !slices.Contains(allowedAuthMethods, req.TokenEndpointAuthMethod) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "unsupported token_endpoint_auth_method")
	}

	clientID, err := generateClientID()
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate client ID")
	}

	// Public clients (token_endpoint_auth_method=none) do not authenticate
	// at the token endpoint and never receive a usable secret, so skip the
	// (expensive) bcrypt round entirely. The token.go grant path already
	// gates secret verification on Config.TokenEndpointAuthMethod != "none".
	var clientSecret, secretHash string
	if req.TokenEndpointAuthMethod != "none" {
		clientSecret, err = generateClientSecret()
		if err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate client secret")
		}
		secretHash, err = hashSecret(clientSecret)
		if err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to hash client secret")
		}
	}

	config := &storepb.OAuth2ClientConfig{
		ClientName:              req.ClientName,
		RedirectUris:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
	}
	if _, err := s.store.CreateOAuth2Client(ctx, &store.OAuth2ClientMessage{
		ClientID:         clientID,
		ClientSecretHash: secretHash,
		Config:           config,
	}); err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to create client")
	}

	resp := &clientRegistrationResponse{
		ClientID:                clientID,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
	}
	// Only include client_secret for confidential clients.
	if req.TokenEndpointAuthMethod != "none" {
		resp.ClientSecret = clientSecret
	}
	return c.JSON(http.StatusCreated, resp)
}
