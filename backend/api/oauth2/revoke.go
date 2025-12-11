package oauth2

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type revokeRequest struct {
	Token         string `form:"token"`
	TokenTypeHint string `form:"token_type_hint"`
	ClientID      string `form:"client_id"`
	ClientSecret  string `form:"client_secret"`
}

func (s *Service) handleRevoke(c echo.Context) error {
	ctx := c.Request().Context()

	var req revokeRequest
	if err := c.Bind(&req); err != nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "failed to parse request")
	}

	// Authenticate client
	clientID, clientSecret := extractRevokeClientCredentials(c, &req)
	if clientID == "" {
		return oauth2Error(c, http.StatusUnauthorized, "invalid_client", "client authentication required")
	}

	client, err := s.store.GetOAuth2Client(ctx, clientID)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to lookup client")
	}
	if client == nil {
		return oauth2Error(c, http.StatusUnauthorized, "invalid_client", "client not found")
	}
	if !verifySecret(client.ClientSecretHash, clientSecret) {
		return oauth2Error(c, http.StatusUnauthorized, "invalid_client", "invalid client credentials")
	}

	// Validate token
	if req.Token == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "token is required")
	}

	// Try to revoke as refresh token
	// RFC 7009 says to return 200 even if token is invalid, so we ignore errors
	tokenHash := hashToken(req.Token)
	_ = s.store.DeleteOAuth2RefreshToken(ctx, tokenHash)

	// Return success (RFC 7009: always return 200)
	return c.NoContent(http.StatusOK)
}

func extractRevokeClientCredentials(c echo.Context, req *revokeRequest) (clientID, clientSecret string) {
	// Try Basic auth first
	authHeader := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				return parts[0], parts[1]
			}
		}
	}
	// Fall back to form params
	return req.ClientID, req.ClientSecret
}
