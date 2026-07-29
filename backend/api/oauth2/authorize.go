package oauth2

import (
	"context"
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

	// Validate the resource/scope the client is asking for before showing a
	// consent screen for something we would refuse to issue.
	params, failure := s.parseGrantParams(ctx, c.QueryParams())
	if failure != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, failure.code, failure.description)
	}

	// Redirect to frontend consent page.
	// The frontend handles login if needed and binds consent to the user's
	// currently active workspace (see handleAuthorizePost). The canonical
	// resource is forwarded (not the raw value) so the consent POST persists
	// exactly what was validated here.
	consentURL := fmt.Sprintf("/oauth2/consent?client_id=%s&redirect_uri=%s&state=%s&code_challenge=%s&code_challenge_method=%s&resource=%s&scope=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		url.QueryEscape(codeChallenge),
		url.QueryEscape(codeChallengeMethod),
		url.QueryEscape(params.resource),
		url.QueryEscape(params.scope),
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

	// Re-validate resource/scope here rather than trusting what the consent page
	// posted back: this endpoint is reachable directly, so the GET's validation
	// is not a guarantee about the POST.
	formValues, err := c.FormValues()
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "invalid_request", "failed to parse request")
	}
	params, failure := s.parseGrantParams(ctx, formValues)
	if failure != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, failure.code, failure.description)
	}

	consenting, failure := s.resolveConsentingUser(c, client)
	if failure != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, failure.code, failure.description)
	}

	code, err := generateAuthCode()
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to generate code")
	}

	codeConfig := &storepb.OAuth2AuthorizationCodeConfig{
		RedirectUri:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Resource:            params.resource,
		Scope:               params.scope,
	}
	if _, err := s.store.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
		Code:      code,
		ClientID:  clientID,
		UserEmail: consenting.email,
		Workspace: consenting.workspaceID,
		Config:    codeConfig,
		ExpiresAt: time.Now().Add(authCodeExpiry),
	}); err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to store code")
	}

	if err := s.store.UpdateOAuth2ClientLastActiveAt(ctx, clientID); err != nil {
		slog.Warn("failed to update OAuth2 client last active", slog.String("clientID", clientID), log.BBError(err))
	}

	redirectURL, err := buildCodeRedirectURL(redirectURI, code, state)
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to parse redirect URI")
	}

	// Return HTML page that redirects to callback URL.
	// This avoids CSP form-action restrictions.
	return c.HTML(http.StatusOK, buildRedirectHTML(redirectURL))
}

// consentingUser is the authenticated user behind a consent POST and the
// workspace their grant binds to.
type consentingUser struct {
	email       string
	workspaceID string
}

// resolveConsentingUser authenticates the browser session that submitted the
// consent form and resolves which workspace the grant is for.
func (s *Service) resolveConsentingUser(c *echo.Context, client *store.OAuth2ClientMessage) (consentingUser, *oauth2Failure) {
	ctx := c.Request().Context()

	accessToken, err := auth.GetTokenFromHeaders(c.Request().Header)
	if err != nil {
		return consentingUser{}, &oauth2Failure{code: "access_denied", description: err.Error()}
	}
	if accessToken == "" {
		return consentingUser{}, &oauth2Failure{code: "access_denied", description: "user not authenticated"}
	}
	claims, failure := s.parseSessionClaims(accessToken)
	if failure != nil {
		return consentingUser{}, failure
	}
	workspaceID, failure := s.consentWorkspace(ctx, claims, client)
	if failure != nil {
		return consentingUser{}, failure
	}

	user, err := s.store.GetUserByEmail(ctx, claims.Subject)
	if err != nil {
		return consentingUser{}, &oauth2Failure{code: "access_denied", description: "failed to find user"}
	}
	if user == nil {
		return consentingUser{}, &oauth2Failure{code: "access_denied", description: "user not found"}
	}
	return consentingUser{email: user.Email, workspaceID: workspaceID}, nil
}

// parseSessionClaims verifies the session JWT behind a consent POST and returns
// its claims. The workspace_id claim is the workspace the user is currently in;
// that's the workspace OAuth consent is granted for.
func (s *Service) parseSessionClaims(accessToken string) (*sessionClaims, *oauth2Failure) {
	claims := &sessionClaims{}
	if _, err := jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errors.Errorf("unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(s.secret), nil
			}
		}
		return nil, errors.Errorf("unexpected access token kid=%v", t.Header["kid"])
	}); err != nil {
		return nil, &oauth2Failure{code: "access_denied", description: "invalid session"}
	}
	if !audienceContains(claims.Audience, auth.AccessTokenAudience) {
		return nil, &oauth2Failure{code: "access_denied", description: "invalid token audience"}
	}
	return claims, nil
}

// consentWorkspace resolves the workspace to bind a consent to.
//
// On SaaS the session always carries workspace_id; if it's missing we fail closed
// rather than fall back to GetWorkspaceID(), which would otherwise pick an
// arbitrary non-deleted workspace and silently bind the token there.
func (s *Service) consentWorkspace(ctx context.Context, claims *sessionClaims, client *store.OAuth2ClientMessage) (string, *oauth2Failure) {
	workspaceID := claims.WorkspaceID
	if workspaceID == "" {
		if s.profile.SaaS {
			return "", &oauth2Failure{code: "access_denied", description: "session is missing workspace claim"}
		}
		// Self-hosted: there's exactly one workspace.
		singleton, err := s.store.GetWorkspaceID(ctx)
		if err != nil {
			return "", &oauth2Failure{code: "server_error", description: "failed to resolve workspace"}
		}
		workspaceID = singleton
	}
	if workspaceID == "" {
		return "", &oauth2Failure{code: "access_denied", description: "no workspace in session"}
	}

	// Legacy clients registered before the 3.18.2 migration are pinned to a
	// workspace via oauth2_client.workspace. Refuse to mint a token for a
	// different workspace via that client — the user is in a workspace it
	// wasn't authorized for. Post-migration clients have client.Workspace
	// empty (workspace-agnostic) and bind freely.
	if client.Workspace != "" && client.Workspace != workspaceID {
		return "", &oauth2Failure{code: "access_denied", description: "client is registered to a different workspace; switch workspaces and try again"}
	}
	return workspaceID, nil
}

// buildCodeRedirectURL appends the issued code (and state, when present) to the
// client's registered redirect URI.
func buildCodeRedirectURL(redirectURI, code, state string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// audienceContains checks if the audience claim contains the given token.
func audienceContains(audience jwt.ClaimStrings, token string) bool {
	return slices.Contains(audience, token)
}
