package oauth2

import (
	"net/http"
	"net/url"
	"slices"

	"github.com/labstack/echo/v4"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type clientRegistrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type clientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

func (s *Service) handleRegister(c echo.Context) error {
	ctx := c.Request().Context()

	var req clientRegistrationRequest
	if err := c.Bind(&req); err != nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "failed to parse request body")
	}

	// Validate client_name
	if req.ClientName == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "client_name is required")
	}

	// Validate redirect_uris
	if len(req.RedirectURIs) == 0 {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "redirect_uris is required")
	}
	for _, uri := range req.RedirectURIs {
		parsed, err := url.Parse(uri)
		if err != nil {
			return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "invalid redirect URI format")
		}
		// Require HTTPS except for localhost
		if parsed.Scheme != "https" && !isLocalhostURI(uri) {
			return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "redirect URI must use HTTPS (except localhost)")
		}
	}

	// Validate grant_types (default to authorization_code)
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}
	allowedGrantTypes := []string{"authorization_code", "refresh_token"}
	for _, gt := range req.GrantTypes {
		if !slices.Contains(allowedGrantTypes, gt) {
			return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "unsupported grant_type: "+gt)
		}
	}

	// Validate token_endpoint_auth_method (default to client_secret_basic)
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "client_secret_basic"
	}
	allowedAuthMethods := []string{"client_secret_basic", "client_secret_post"}
	if !slices.Contains(allowedAuthMethods, req.TokenEndpointAuthMethod) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client_metadata", "unsupported token_endpoint_auth_method")
	}

	// Generate credentials
	clientID, err := generateClientID()
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate client ID")
	}
	clientSecret, err := generateClientSecret()
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate client secret")
	}
	secretHash, err := hashSecret(clientSecret)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to hash client secret")
	}

	// Store client
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

	return c.JSON(http.StatusCreated, &clientRegistrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
	})
}
