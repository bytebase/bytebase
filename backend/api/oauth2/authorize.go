package oauth2

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Service) handleAuthorizeGet(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters
	responseType := c.QueryParam("response_type")
	clientID := c.QueryParam("client_id")
	redirectURI := c.QueryParam("redirect_uri")
	state := c.QueryParam("state")
	codeChallenge := c.QueryParam("code_challenge")
	codeChallengeMethod := c.QueryParam("code_challenge_method")

	// Validate response_type
	if responseType != "code" {
		return oauth2Error(c, http.StatusBadRequest, "unsupported_response_type", "only 'code' response type is supported")
	}

	// Validate client_id
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

	// Validate redirect_uri
	if redirectURI == "" {
		return oauth2Error(c, http.StatusBadRequest, "invalid_request", "redirect_uri is required")
	}
	if !validateRedirectURI(redirectURI, client.Config.RedirectUris) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uri not registered")
	}

	// Validate PKCE (required)
	if codeChallenge == "" {
		return oauth2ErrorRedirect(c, redirectURI, state, "invalid_request", "code_challenge is required")
	}
	if codeChallengeMethod != "S256" {
		return oauth2ErrorRedirect(c, redirectURI, state, "invalid_request", "code_challenge_method must be S256")
	}

	// Check if user is logged in
	accessToken := getTokenFromEchoRequest(c)
	if accessToken == "" {
		// Redirect to login page with return URL
		loginURL := fmt.Sprintf("/auth/login?redirect=%s", url.QueryEscape(c.Request().URL.String()))
		return c.Redirect(http.StatusFound, loginURL)
	}

	// Return consent page HTML
	return renderConsentPage(c, client.Config.ClientName, clientID, redirectURI, state, codeChallenge, codeChallengeMethod)
}

func (s *Service) handleAuthorizePost(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse form values
	clientID := c.FormValue("client_id")
	redirectURI := c.FormValue("redirect_uri")
	state := c.FormValue("state")
	codeChallenge := c.FormValue("code_challenge")
	codeChallengeMethod := c.FormValue("code_challenge_method")
	action := c.FormValue("action")

	// Validate client
	client, err := s.store.GetOAuth2Client(ctx, clientID)
	if err != nil || client == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_client", "client not found")
	}

	// Validate redirect_uri
	if !validateRedirectURI(redirectURI, client.Config.RedirectUris) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uri not registered")
	}

	// Handle denial
	if action == "deny" {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "user denied the request")
	}

	// Get current user from session
	accessToken := getTokenFromEchoRequest(c)
	if accessToken == "" {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "user not authenticated")
	}

	// Validate the access token and extract user email
	claims := &jwt.RegisteredClaims{}
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

	// Validate audience
	if !audienceContains(claims.Audience, fmt.Sprintf(auth.AccessTokenAudienceFmt, common.ReleaseModeProd)) &&
		!audienceContains(claims.Audience, fmt.Sprintf(auth.AccessTokenAudienceFmt, common.ReleaseModeDev)) {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "invalid token audience")
	}

	// Get user from claims
	principalID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "invalid user ID")
	}

	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil || user == nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "user not found")
	}

	// Generate authorization code
	code, err := generateAuthCode()
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to generate code")
	}

	// Store authorization code
	codeConfig := &storepb.OAuth2AuthorizationCodeConfig{
		RedirectUri:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}
	if _, err := s.store.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
		Code:      code,
		ClientID:  clientID,
		UserID:    user.ID,
		Config:    codeConfig,
		ExpiresAt: time.Now().Add(authCodeExpiry),
	}); err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "server_error", "failed to store code")
	}

	// Update client last active
	_ = s.store.UpdateOAuth2ClientLastActiveAt(ctx, clientID)

	// Redirect with code
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return c.Redirect(http.StatusFound, u.String())
}

func renderConsentPage(c echo.Context, clientName, clientID, redirectURI, state, codeChallenge, codeChallengeMethod string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authorize Application</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #f5f5f5; }
        .container { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); max-width: 400px; width: 100%%; }
        h1 { margin: 0 0 1rem; font-size: 1.5rem; }
        p { color: #666; margin: 0 0 1.5rem; }
        .app-name { font-weight: bold; color: #333; }
        .buttons { display: flex; gap: 1rem; }
        button { flex: 1; padding: 0.75rem; border: none; border-radius: 4px; font-size: 1rem; cursor: pointer; }
        .allow { background: #4f46e5; color: white; }
        .allow:hover { background: #4338ca; }
        .deny { background: #e5e5e5; color: #333; }
        .deny:hover { background: #d4d4d4; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authorize Application</h1>
        <p><span class="app-name">%s</span> is requesting access to your Bytebase account.</p>
        <form method="POST" action="/oauth2/authorize">
            <input type="hidden" name="client_id" value="%s">
            <input type="hidden" name="redirect_uri" value="%s">
            <input type="hidden" name="state" value="%s">
            <input type="hidden" name="code_challenge" value="%s">
            <input type="hidden" name="code_challenge_method" value="%s">
            <div class="buttons">
                <button type="submit" name="action" value="deny" class="deny">Deny</button>
                <button type="submit" name="action" value="allow" class="allow">Allow</button>
            </div>
        </form>
    </div>
</body>
</html>`, clientName, clientID, redirectURI, state, codeChallenge, codeChallengeMethod)
	return c.HTML(http.StatusOK, html)
}

// getTokenFromEchoRequest extracts the access token from an Echo request.
func getTokenFromEchoRequest(c echo.Context) string {
	// Check Authorization header first
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
	}

	// Check HTTP cookies
	cookie, err := c.Cookie(auth.AccessTokenCookieName)
	if err == nil && cookie != nil {
		return cookie.Value
	}

	return ""
}

// audienceContains checks if the audience claim contains the given token.
func audienceContains(audience jwt.ClaimStrings, token string) bool {
	for _, aud := range audience {
		if aud == token {
			return true
		}
	}
	return false
}
