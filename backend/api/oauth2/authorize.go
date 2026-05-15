package oauth2

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// sessionClaims is the subset of session JWT claims we need at the OAuth2
// authorize step. workspace_id carries the workspace the user is currently
// acting in; that workspace becomes the one bound to the issued authorization
// code (and ultimately the OAuth2 access token).
type sessionClaims struct {
	jwt.RegisteredClaims
	WorkspaceID string `json:"workspace_id,omitempty"`
}

func (s *Service) handleAuthorizeGet(c *echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters
	responseType := c.QueryParam("response_type")
	clientID := c.QueryParam("client_id")
	redirectURI := c.QueryParam("redirect_uri")
	state := c.QueryParam("state")
	codeChallenge := c.QueryParam("code_challenge")
	codeChallengeMethod := c.QueryParam("code_challenge_method")

	if responseType != "code" {
		return oauth2Error(c, http.StatusBadRequest, "unsupported_response_type", "only 'code' response type is supported")
	}

	if clientID == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "client_id is required")
	}

	client, err := s.store.GetOAuth2Client(ctx, clientID)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to lookup client")
	}
	if client == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client", "client not found")
	}

	if redirectURI == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "redirect_uri is required")
	}
	if !validateRedirectURI(redirectURI, client.Config.RedirectUris) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uri not registered")
	}

	// PKCE is required
	if codeChallenge == "" {
		return oauth2ErrorRedirect(c, redirectURI, state, "invalid_request", "code_challenge is required")
	}
	if codeChallengeMethod != "S256" {
		return oauth2ErrorRedirect(c, redirectURI, state, "invalid_request", "code_challenge_method must be S256")
	}

	// Redirect to frontend consent page.
	// The frontend handles login if needed and binds consent to the user's
	// currently active workspace (see handleAuthorizePost).
	consentURL := fmt.Sprintf("/oauth2/consent?client_id=%s&redirect_uri=%s&state=%s&code_challenge=%s&code_challenge_method=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		url.QueryEscape(codeChallenge),
		url.QueryEscape(codeChallengeMethod),
	)
	return c.Redirect(http.StatusFound, consentURL)
}

func (s *Service) handleAuthorizePost(c *echo.Context) error {
	ctx := c.Request().Context()

	clientID := c.FormValue("client_id")
	redirectURI := c.FormValue("redirect_uri")
	state := c.FormValue("state")
	codeChallenge := c.FormValue("code_challenge")
	codeChallengeMethod := c.FormValue("code_challenge_method")
	action := c.FormValue("action")

	client, err := s.store.GetOAuth2Client(ctx, clientID)
	if err != nil || client == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client", "client not found")
	}

	if !validateRedirectURI(redirectURI, client.Config.RedirectUris) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uri not registered")
	}

	if action == "deny" {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "user denied the request")
	}

	// Get current user from session
	accessToken, err := auth.GetTokenFromHeaders(c.Request().Header)
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", err.Error())
	}
	if accessToken == "" {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "user not authenticated")
	}

	// Parse the session token to get the user and their active workspace.
	// The workspace_id claim is the workspace the user is currently in;
	// that's the workspace OAuth consent is granted for.
	claims := &sessionClaims{}
	_, err = jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errors.Errorf("unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(s.secret), nil
			}
		}
		return nil, errors.Errorf("unexpected access token kid=%v", t.Header["kid"])
	})
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "invalid session")
	}

	if !audienceContains(claims.Audience, auth.AccessTokenAudience) {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "invalid token audience")
	}

	// Resolve the workspace to bind this consent to.
	// On SaaS the session always carries workspace_id; if it's missing we
	// fail closed rather than fall back to GetWorkspaceID(), which would
	// otherwise pick an arbitrary non-deleted workspace and silently bind
	// the token there.
	workspaceID := claims.WorkspaceID
	if workspaceID == "" {
		if s.profile.SaaS {
			return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "session is missing workspace claim")
		}
		// Self-hosted: there's exactly one workspace.
		workspaceID, err = s.store.GetWorkspaceID(ctx)
		if err != nil {
			return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to resolve workspace")
		}
	}
	if workspaceID == "" {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "no workspace in session")
	}

	// Legacy clients registered before the 3.18.2 migration are pinned to a
	// workspace via oauth2_client.workspace. Refuse to mint a token for a
	// different workspace via that client — the user is in a workspace it
	// wasn't authorized for. Post-migration clients have client.Workspace
	// empty (workspace-agnostic) and bind freely.
	if client.Workspace != "" && client.Workspace != workspaceID {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "client is registered to a different workspace; switch workspaces and try again")
	}

	user, err := s.store.GetUserByEmail(ctx, claims.Subject)
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "failed to find user")
	}
	if user == nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "user not found")
	}

	code, err := generateAuthCode()
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to generate code")
	}

	codeConfig := &storepb.OAuth2AuthorizationCodeConfig{
		RedirectUri:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}
	if _, err := s.store.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
		Code:      code,
		ClientID:  clientID,
		UserEmail: user.Email,
		Workspace: workspaceID,
		Config:    codeConfig,
		ExpiresAt: time.Now().Add(authCodeExpiry),
	}); err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to store code")
	}

	if err := s.store.UpdateOAuth2ClientLastActiveAt(ctx, clientID); err != nil {
		slog.Warn("failed to update OAuth2 client last active", slog.String("clientID", clientID), log.BBError(err))
	}

	u, err := url.Parse(redirectURI)
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to parse redirect URI")
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	redirectURL := u.String()

	// Return HTML page that redirects to callback URL.
	// This avoids CSP form-action restrictions.
	return c.HTML(http.StatusOK, buildRedirectHTML(redirectURL))
}

// audienceContains checks if the audience claim contains the given token.
func audienceContains(audience jwt.ClaimStrings, token string) bool {
	return slices.Contains(audience, token)
}
