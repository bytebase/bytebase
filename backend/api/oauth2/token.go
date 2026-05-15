package oauth2

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pkg/errors"

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

func (s *Service) handleToken(c *echo.Context) error {
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

func (*Service) extractClientCredentials(c *echo.Context, req *tokenRequest) (clientID, clientSecret string) {
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

func (s *Service) handleAuthorizationCodeGrant(c *echo.Context, client *store.OAuth2ClientMessage, req *tokenRequest) error {
	ctx := c.Request().Context()

	// Validate grant type is allowed
	if !slices.Contains(client.Config.GrantTypes, "authorization_code") {
		return oauth2Error(c, http.StatusBadRequest, "unauthorized_client", "client not authorized for authorization_code grant")
	}

	// Validate code
	if req.Code == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "code is required")
	}

	authCode, err := s.store.GetOAuth2AuthorizationCode(ctx, client.ClientID, req.Code)
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
		if err := s.store.DeleteOAuth2AuthorizationCode(ctx, client.ClientID, req.Code); err != nil {
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
	if err := s.store.DeleteOAuth2AuthorizationCode(ctx, client.ClientID, req.Code); err != nil {
		slog.Warn("failed to delete OAuth2 authorization code after use", slog.String("code", req.Code), log.BBError(err))
	}

	// Get user
	user, err := s.store.GetUserByEmail(ctx, authCode.UserEmail)
	if err != nil || user == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "user not found")
	}

	// Resolve the workspace bound at consent time. Auth codes created before
	// the 3.18.2 migration may have an empty workspace; fall back to the
	// client's legacy workspace, then to the singleton workspace.
	workspaceID, err := s.resolveBoundWorkspace(ctx, authCode.Workspace, client.Workspace, user.Email)
	if err != nil {
		return workspaceResolutionError(c, err)
	}

	return s.issueTokens(c, client, user.Email, workspaceID)
}

func (s *Service) handleRefreshTokenGrant(c *echo.Context, client *store.OAuth2ClientMessage, req *tokenRequest) error {
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
	refreshToken, err := s.store.GetOAuth2RefreshToken(ctx, client.ClientID, tokenHash)
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
		if err := s.store.DeleteOAuth2RefreshToken(ctx, client.ClientID, tokenHash); err != nil {
			slog.Warn("failed to delete expired OAuth2 refresh token", log.BBError(err))
		}
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "refresh token has expired")
	}

	// Delete token after validations pass (single use, will issue new one)
	if err := s.store.DeleteOAuth2RefreshToken(ctx, client.ClientID, tokenHash); err != nil {
		slog.Warn("failed to delete OAuth2 refresh token after use", log.BBError(err))
	}

	// Get user
	user, err := s.store.GetUserByEmail(ctx, refreshToken.UserEmail)
	if err != nil || user == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "user not found")
	}

	// Preserve the workspace binding from the refresh token. Fall back paths
	// mirror the auth-code grant for pre-migration tokens. Membership is
	// re-checked on every refresh so a user removed from the workspace
	// after consent loses access at most one access-token lifetime later
	// rather than waiting out the refresh token's 30-day expiry.
	workspaceID, err := s.resolveBoundWorkspace(ctx, refreshToken.Workspace, client.Workspace, user.Email)
	if err != nil {
		return workspaceResolutionError(c, err)
	}

	return s.issueTokens(c, client, user.Email, workspaceID)
}

// workspaceResolutionError maps the typed errors from resolveBoundWorkspace
// onto RFC 6749 OAuth2 error responses. Membership failure is invalid_grant
// (400); everything else is an internal failure surfaced as server_error
// (500) with the wrapped detail logged server-side, not leaked to the client.
func workspaceResolutionError(c *echo.Context, err error) error {
	if errors.Is(err, errWorkspaceNotMember) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "user is no longer a member of the workspace")
	}
	slog.Error("OAuth2 workspace resolution failed", log.BBError(err))
	return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to resolve workspace")
}

// errWorkspaceNotMember signals that the user has been removed from the
// workspace their OAuth grant was issued for. Mapped to RFC 6749 `invalid_grant`
// at the call site. All other errors from resolveBoundWorkspace are internal
// failures that should produce 500/server_error instead.
var errWorkspaceNotMember = errors.New("user is no longer a member of the consented workspace")

// workspaceResolver is the slice of store methods resolveBoundWorkspace needs.
// Defining it as an interface keeps the helper independently unit-testable.
type workspaceResolver interface {
	GetWorkspaceID(ctx context.Context) (string, error)
	FindWorkspace(ctx context.Context, find *store.FindWorkspaceMessage) (*store.WorkspaceMessage, error)
}

// resolveBoundWorkspace returns the workspace the issued token should bind to,
// applying the legacy fallback chain (issued.Workspace → client.Workspace →
// singleton) and then verifying current IAM membership before returning. The
// membership check is the defense-in-depth guard against issuing a usable
// token to a user who has been removed from the workspace since consent.
//
// On SaaS only: returns errWorkspaceNotMember if the user is not currently a
// member of the resolved workspace. All other errors are internal failures.
func (s *Service) resolveBoundWorkspace(ctx context.Context, issuedWorkspace, clientWorkspace, userEmail string) (string, error) {
	return resolveBoundWorkspace(ctx, s.store, s.profile.SaaS, issuedWorkspace, clientWorkspace, userEmail)
}

func resolveBoundWorkspace(ctx context.Context, resolver workspaceResolver, saas bool, issuedWorkspace, clientWorkspace, userEmail string) (string, error) {
	workspaceID := issuedWorkspace
	if workspaceID == "" {
		workspaceID = clientWorkspace
	}
	if workspaceID == "" {
		singleton, err := resolver.GetWorkspaceID(ctx)
		if err != nil {
			return "", errors.Wrap(err, "failed to resolve workspace")
		}
		workspaceID = singleton
	}
	if workspaceID == "" {
		return "", errors.New("no workspace bound to this grant")
	}

	// Self-hosted: every user belongs to the singleton workspace implicitly,
	// skip the IAM round-trip. SaaS: verify the user is still a member.
	if !saas {
		return workspaceID, nil
	}
	ws, err := resolver.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		WorkspaceID: &workspaceID,
		Email:       userEmail,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to verify workspace membership")
	}
	if ws == nil {
		return "", errWorkspaceNotMember
	}
	return workspaceID, nil
}

// issueTokens issues a new OAuth2 access token (and refresh token, when the
// grant supports it) bound to the given workspace. The workspace is sourced
// from the authorization code or refresh token being exchanged, not from the
// client — clients are workspace-agnostic.
func (s *Service) issueTokens(c *echo.Context, client *store.OAuth2ClientMessage, userEmail, workspaceID string) error {
	ctx := c.Request().Context()

	// Generate access token (JWT) with the workspace_id claim that
	// downstream APIs (gRPC services, MCP middleware) use to scope requests.
	accessToken, err := auth.GenerateOAuth2AccessToken(userEmail, client.ClientID, workspaceID, s.secret, accessTokenExpiry)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", fmt.Sprintf("failed to generate access token with error: %v", err))
	}

	now := time.Now()

	// Generate refresh token if allowed
	var refreshTokenStr string
	if slices.Contains(client.Config.GrantTypes, "refresh_token") {
		refreshTokenStr, err = generateRefreshToken()
		if err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate refresh token")
		}

		// Store refresh token with the workspace binding preserved so a
		// subsequent /token refresh re-issues for the same workspace.
		if _, err := s.store.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: auth.HashToken(refreshTokenStr),
			ClientID:  client.ClientID,
			UserEmail: userEmail,
			Workspace: workspaceID,
			ExpiresAt: now.Add(refreshTokenExpiry),
		}); err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to store refresh token")
		}
	}

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
