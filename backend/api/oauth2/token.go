package oauth2

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
	// Scope echoes the scope the grant was consented for (RFC 6749 §5.1), so a
	// client can see what it actually holds rather than what it asked for.
	Scope string `json:"scope,omitempty"`
}

// issuedGrant is the consented state one access/refresh token pair is minted
// from. The resource and scope travel unchanged from the authorization code
// through every refresh — a refresh re-issues a grant, it never widens one.
type issuedGrant struct {
	userEmail   string
	workspaceID string
	resource    string
	scope       string
}

func (s *Service) handleToken(c *echo.Context) error {
	ctx := c.Request().Context()

	var req tokenRequest
	if err := c.Bind(&req); err != nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "failed to parse request")
	}
	// Bind keeps only the first value of a repeated parameter; the raw values are
	// what the resource/scope checks need in order to reject repeats outright.
	formValues, err := c.FormValues()
	if err != nil {
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
		return s.handleAuthorizationCodeGrant(c, client, &req, formValues)
	case "refresh_token":
		return s.handleRefreshTokenGrant(c, client, &req, formValues)
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

func (s *Service) handleAuthorizationCodeGrant(c *echo.Context, client *store.OAuth2ClientMessage, req *tokenRequest, formValues url.Values) error {
	ctx := c.Request().Context()

	// Validate grant type is allowed
	if !slices.Contains(client.Config.GrantTypes, "authorization_code") {
		return oauth2Error(c, http.StatusBadRequest, "unauthorized_client", "client not authorized for authorization_code grant")
	}

	authCode, failure := s.validateAuthorizationCode(ctx, client, req, formValues)
	if failure != nil {
		return tokenFailure(c, failure)
	}

	// Consume the code after all validations pass. This atomic delete is the
	// single-use gate: PKCE is verified above so a failed verifier never burns
	// the code, and concurrent redemptions race here so only the caller that
	// actually claims the row proceeds to issue tokens. A consume failure aborts
	// issuance rather than warning and continuing (the prior behavior left the
	// code replayable on a transient error).
	consumed, err := s.store.ConsumeOAuth2AuthorizationCode(ctx, client.ClientID, req.Code)
	if err != nil {
		slog.Error("failed to consume OAuth2 authorization code", slog.String("code", req.Code), log.BBError(err))
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to consume authorization code")
	}
	if !consumed {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "invalid or expired code")
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

	return s.issueTokens(c, client, issuedGrant{
		userEmail:   user.Email,
		workspaceID: workspaceID,
		resource:    authCode.Config.Resource,
		scope:       authCode.Config.Scope,
	})
}

// validateAuthorizationCode looks up the code and runs every check that must
// pass before it may be consumed: client binding, expiry, redirect_uri, PKCE,
// and the consented resource/scope. Returning a failure leaves the code intact
// so the client can correct its request and retry.
func (s *Service) validateAuthorizationCode(ctx context.Context, client *store.OAuth2ClientMessage, req *tokenRequest, formValues url.Values) (*store.OAuth2AuthorizationCodeMessage, *oauth2Failure) {
	if req.Code == "" {
		return nil, &oauth2Failure{code: "invalid_request", description: "code is required"}
	}

	authCode, err := s.store.GetOAuth2AuthorizationCode(ctx, client.ClientID, req.Code)
	if err != nil {
		return nil, &oauth2Failure{code: "server_error", description: "failed to lookup code"}
	}
	if authCode == nil {
		return nil, &oauth2Failure{code: "invalid_grant", description: "invalid or expired code"}
	}

	// Validate code belongs to this client BEFORE deleting
	// This prevents DoS where attacker with stolen code invalidates it for legitimate client
	if authCode.ClientID != client.ClientID {
		return nil, &oauth2Failure{code: "invalid_grant", description: "code was not issued to this client"}
	}

	// Validate code not expired
	if time.Now().After(authCode.ExpiresAt) {
		// Delete expired code
		if err := s.store.DeleteOAuth2AuthorizationCode(ctx, client.ClientID, req.Code); err != nil {
			slog.Warn("failed to delete expired OAuth2 authorization code", slog.String("code", req.Code), log.BBError(err))
		}
		return nil, &oauth2Failure{code: "invalid_grant", description: "code has expired"}
	}

	// Validate redirect_uri matches
	if req.RedirectURI != authCode.Config.RedirectUri {
		return nil, &oauth2Failure{code: "invalid_grant", description: "redirect_uri mismatch"}
	}

	// Validate PKCE
	if req.CodeVerifier == "" {
		return nil, &oauth2Failure{code: "invalid_request", description: "code_verifier is required"}
	}
	if !verifyPKCE(req.CodeVerifier, authCode.Config.CodeChallenge, authCode.Config.CodeChallengeMethod) {
		return nil, &oauth2Failure{code: "invalid_grant", description: "invalid code_verifier"}
	}

	if failure := checkConsentedResource(formValues, authCode.Config.Resource); failure != nil {
		return nil, failure
	}
	if failure := checkConsentedScope(formValues, authCode.Config.Scope); failure != nil {
		return nil, failure
	}
	return authCode, nil
}

// tokenFailure renders a helper's refusal as an RFC 6749 token-endpoint error.
// server_error is the only 500-class code these helpers produce; every other
// code is a client error.
func tokenFailure(c *echo.Context, failure *oauth2Failure) error {
	status := http.StatusBadRequest
	if failure.code == "server_error" {
		status = http.StatusInternalServerError
	}
	return oauth2Error(c, status, failure.code, failure.description)
}

func (s *Service) handleRefreshTokenGrant(c *echo.Context, client *store.OAuth2ClientMessage, req *tokenRequest, formValues url.Values) error {
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

	// A refresh re-issues the grant as consented: the stored resource and scope
	// are carried forward verbatim, and a request naming different ones is
	// rejected rather than honored. Checked before the token is consumed so a
	// rejected refresh leaves the client able to retry correctly.
	if failure := checkConsentedResource(formValues, refreshToken.Resource); failure != nil {
		return tokenFailure(c, failure)
	}
	if failure := checkConsentedScope(formValues, refreshToken.Scope); failure != nil {
		return tokenFailure(c, failure)
	}

	// Consume the refresh token after validations pass. This atomic delete is
	// the single-use rotation gate: concurrent refreshes race here so only the
	// caller that actually claims the row issues a new pair. A consume failure
	// aborts issuance rather than warning and continuing (the prior behavior
	// left the token replayable on a transient error).
	consumed, err := s.store.ConsumeOAuth2RefreshToken(ctx, client.ClientID, tokenHash)
	if err != nil {
		slog.Error("failed to consume OAuth2 refresh token", log.BBError(err))
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to consume refresh token")
	}
	if !consumed {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "invalid refresh token")
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

	return s.issueTokens(c, client, issuedGrant{
		userEmail:   user.Email,
		workspaceID: workspaceID,
		resource:    refreshToken.Resource,
		scope:       refreshToken.Scope,
	})
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
// grant supports it) for the given consented grant. Everything in the grant is
// sourced from the authorization code or refresh token being exchanged, not from
// the client — clients are workspace-agnostic and cannot pick their own
// resource or scope here.
func (s *Service) issueTokens(c *echo.Context, client *store.OAuth2ClientMessage, grant issuedGrant) error {
	ctx := c.Request().Context()

	// Generate access token (JWT) with the workspace_id claim that
	// downstream APIs (gRPC services, MCP middleware) use to scope requests.
	accessToken, err := auth.GenerateOAuth2AccessToken(grant.userEmail, client.ClientID, grant.workspaceID, s.secret, accessTokenExpiry)
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

		// Store the refresh token with the whole grant preserved so a subsequent
		// /token refresh re-issues for the same workspace, resource, and scope.
		if _, err := s.store.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: auth.HashToken(refreshTokenStr),
			ClientID:  client.ClientID,
			UserEmail: grant.userEmail,
			Workspace: grant.workspaceID,
			Resource:  grant.resource,
			Scope:     grant.scope,
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
		Scope:        grant.scope,
	})
}
