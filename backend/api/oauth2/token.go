package oauth2

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
)

type tokenRequest struct {
	GrantType    string `form:"grant_type"`
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	CodeVerifier string `form:"code_verifier"`
	RefreshToken string `form:"refresh_token"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (s *Service) handleToken(c echo.Context) error {
	ctx := c.Request().Context()

	var req tokenRequest
	if err := c.Bind(&req); err != nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "failed to parse request")
	}

	// Authenticate client
	clientID, clientSecret := s.extractClientCredentials(c, &req)
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

	// Verify client credentials based on token_endpoint_auth_method
	// Public clients (token_endpoint_auth_method: none) don't have secrets
	if client.Config.TokenEndpointAuthMethod != "none" {
		if !verifySecret(client.ClientSecretHash, clientSecret) {
			return oauth2Error(c, http.StatusUnauthorized, "invalid_client", "invalid client credentials")
		}
	}

	// Handle grant types
	switch req.GrantType {
	case "authorization_code":
		return s.handleAuthorizationCodeGrant(c, client, &req)
	case "refresh_token":
		return s.handleRefreshTokenGrant(c, client, &req)
	default:
		return oauth2Error(c, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func (*Service) extractClientCredentials(c echo.Context, req *tokenRequest) (clientID, clientSecret string) {
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

func (s *Service) handleAuthorizationCodeGrant(c echo.Context, client *store.OAuth2ClientMessage, req *tokenRequest) error {
	ctx := c.Request().Context()

	// Validate grant type is allowed
	if !slices.Contains(client.Config.GrantTypes, "authorization_code") {
		return oauth2Error(c, http.StatusBadRequest, "unauthorized_client", "client not authorized for authorization_code grant")
	}

	// Validate code
	if req.Code == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "code is required")
	}

	authCode, err := s.store.GetOAuth2AuthorizationCode(ctx, req.Code)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to lookup code")
	}
	if authCode == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "invalid or expired code")
	}

	// Validate code belongs to this client BEFORE deleting
	// This prevents DoS where attacker with stolen code invalidates it for legitimate client
	if authCode.ClientID != client.ClientID {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "code was not issued to this client")
	}

	// Validate code not expired
	if time.Now().After(authCode.ExpiresAt) {
		// Delete expired code
		if err := s.store.DeleteOAuth2AuthorizationCode(ctx, req.Code); err != nil {
			slog.Warn("failed to delete expired OAuth2 authorization code", slog.String("code", req.Code), log.BBError(err))
		}
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "code has expired")
	}

	// Validate redirect_uri matches
	if req.RedirectURI != authCode.Config.RedirectUri {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
	}

	// Validate PKCE
	if req.CodeVerifier == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "code_verifier is required")
	}
	if !verifyPKCE(req.CodeVerifier, authCode.Config.CodeChallenge, authCode.Config.CodeChallengeMethod) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "invalid code_verifier")
	}

	// Delete code after all validations pass (single use)
	if err := s.store.DeleteOAuth2AuthorizationCode(ctx, req.Code); err != nil {
		slog.Warn("failed to delete OAuth2 authorization code after use", slog.String("code", req.Code), log.BBError(err))
	}

	// Get user
	user, err := s.store.GetUserByEmail(ctx, authCode.UserEmail)
	if err != nil || user == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "user not found")
	}

	// Generate tokens
	return s.issueTokens(c, client, user.Email)
}

func (s *Service) handleRefreshTokenGrant(c echo.Context, client *store.OAuth2ClientMessage, req *tokenRequest) error {
	ctx := c.Request().Context()

	// Validate grant type is allowed
	if !slices.Contains(client.Config.GrantTypes, "refresh_token") {
		return oauth2Error(c, http.StatusBadRequest, "unauthorized_client", "client not authorized for refresh_token grant")
	}

	// Validate refresh token
	if req.RefreshToken == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "refresh_token is required")
	}

	tokenHash := auth.HashToken(req.RefreshToken)
	refreshToken, err := s.store.GetOAuth2RefreshToken(ctx, tokenHash)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to lookup refresh token")
	}
	if refreshToken == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "invalid refresh token")
	}

	// Validate token belongs to this client BEFORE deleting
	// This prevents DoS where attacker with stolen token invalidates it for legitimate client
	if refreshToken.ClientID != client.ClientID {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "refresh token was not issued to this client")
	}

	// Validate not expired
	if time.Now().After(refreshToken.ExpiresAt) {
		// Delete expired token
		if err := s.store.DeleteOAuth2RefreshToken(ctx, tokenHash); err != nil {
			slog.Warn("failed to delete expired OAuth2 refresh token", log.BBError(err))
		}
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "refresh token has expired")
	}

	// Delete token after validations pass (single use, will issue new one)
	if err := s.store.DeleteOAuth2RefreshToken(ctx, tokenHash); err != nil {
		slog.Warn("failed to delete OAuth2 refresh token after use", log.BBError(err))
	}

	// Get user
	user, err := s.store.GetUserByEmail(ctx, refreshToken.UserEmail)
	if err != nil || user == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "user not found")
	}

	// Issue new tokens
	return s.issueTokens(c, client, user.Email)
}

func (s *Service) issueTokens(c echo.Context, client *store.OAuth2ClientMessage, userEmail string) error {
	ctx := c.Request().Context()

	// Generate access token (JWT)
	accessToken, err := auth.GenerateOAuth2AccessToken(userEmail, client.ClientID, s.secret, accessTokenExpiry)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate access token")
	}

	now := time.Now()

	// Generate refresh token if allowed
	var refreshTokenStr string
	if slices.Contains(client.Config.GrantTypes, "refresh_token") {
		refreshTokenStr, err = generateRefreshToken()
		if err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate refresh token")
		}

		// Store refresh token
		if _, err := s.store.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: auth.HashToken(refreshTokenStr),
			ClientID:  client.ClientID,
			UserEmail: userEmail,
			ExpiresAt: now.Add(refreshTokenExpiry),
		}); err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to store refresh token")
		}
	}

	// Update client last active
	if err := s.store.UpdateOAuth2ClientLastActiveAt(ctx, client.ClientID); err != nil {
		slog.Warn("failed to update OAuth2 client last active", slog.String("clientID", client.ClientID), log.BBError(err))
	}

	return c.JSON(http.StatusOK, &tokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenExpiry.Seconds()),
		RefreshToken: refreshTokenStr,
	})
}
