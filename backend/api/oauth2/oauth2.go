package oauth2

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	clientIDPrefix     = "bb_oauth_"
	clientSecretPrefix = "bb_secret_"
	refreshTokenPrefix = "bb_refresh_"
	authCodePrefix     = "bb_code_"

	authCodeExpiry       = 10 * time.Minute
	accessTokenExpiry    = 1 * time.Hour
	refreshTokenExpiry   = 30 * 24 * time.Hour
	clientInactiveExpiry = 30 * 24 * time.Hour
)

type Service struct {
	store   *store.Store
	profile *config.Profile
	secret  string
}

func NewService(store *store.Store, profile *config.Profile, secret string) *Service {
	return &Service{
		store:   store,
		profile: profile,
		secret:  secret,
	}
}

func (s *Service) RegisterRoutes(e *echo.Echo) {
	// .well-known endpoints must stay at root (RFC 8615)
	e.GET("/.well-known/oauth-authorization-server", s.handleDiscovery)
	e.GET("/.well-known/oauth-authorization-server/*", s.handleDiscovery)
	e.GET("/.well-known/oauth-protected-resource", s.handleProtectedResourceMetadata)
	e.GET("/.well-known/oauth-protected-resource/*", s.handleProtectedResourceMetadata)
	// OAuth2 endpoints under /api prefix
	e.POST("/api/oauth2/register", s.handleRegister)
	e.GET("/api/oauth2/authorize", s.handleAuthorizeGet)
	e.POST("/api/oauth2/authorize", s.handleAuthorizePost)
	e.GET("/api/oauth2/clients/:clientID", s.handleGetClient)
	e.POST("/api/oauth2/token", s.handleToken)
	e.POST("/api/oauth2/revoke", s.handleRevoke)
}

// handleGetClient returns public client info for the consent page.
func (s *Service) handleGetClient(c echo.Context) error {
	ctx := c.Request().Context()
	clientID := c.Param("clientID")

	client, err := s.store.GetOAuth2Client(ctx, clientID)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to lookup client")
	}
	if client == nil {
		return oauth2Error(c, http.StatusNotFound, "invalid_client", "client not found")
	}

	// Return only public info (no secret)
	return c.JSON(http.StatusOK, map[string]any{
		"client_id":     client.ClientID,
		"client_name":   client.Config.ClientName,
		"redirect_uris": client.Config.RedirectUris,
	})
}

func generateClientID() (string, error) {
	// Use 24 bytes for client ID (shorter than refresh tokens)
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return clientIDPrefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func generateClientSecret() (string, error) {
	token, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", err
	}
	return clientSecretPrefix + token, nil
}

func generateAuthCode() (string, error) {
	token, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", err
	}
	return authCodePrefix + token, nil
}

func generateRefreshToken() (string, error) {
	token, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", err
	}
	return refreshTokenPrefix + token, nil
}

func hashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func verifySecret(hash, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}

func verifyPKCE(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}
	// RFC 7636: code_verifier must be 43-128 characters
	if len(codeVerifier) < 43 || len(codeVerifier) > 128 {
		return false
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(computed), []byte(codeChallenge)) == 1
}

func validateRedirectURI(uri string, allowedURIs []string) bool {
	return slices.Contains(allowedURIs, uri)
}

func isLocalhostURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func oauth2Error(c echo.Context, statusCode int, errorCode, description string) error {
	return c.JSON(statusCode, map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
}

func oauth2ErrorRedirect(c echo.Context, redirectURI, state, errorCode, description string) error {
	u, err := url.Parse(redirectURI)
	if err != nil {
		slog.Error("failed to parse redirect URI for OAuth2 error redirect", slog.String("redirectURI", redirectURI), log.BBError(err))
		return oauth2Error(c, http.StatusInternalServerError, errorCode, description)
	}
	q := u.Query()
	q.Set("error", errorCode)
	q.Set("error_description", description)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	// Return HTML page that redirects to callback URL
	// This avoids CSP form-action restrictions
	return c.HTML(http.StatusOK, buildRedirectHTML(u.String()))
}

// buildRedirectHTML creates an HTML page that redirects to the given URL.
// This is used to work around CSP form-action restrictions.
func buildRedirectHTML(redirectURL string) string {
	// HTML escape the URL for safe embedding
	escaped := html.EscapeString(redirectURL)
	return `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="refresh" content="0;url=` + escaped + `">
<title>Redirecting...</title>
</head>
<body>
<p>Redirecting to application...</p>
<noscript><a href="` + escaped + `">Click here to continue</a></noscript>
</body>
</html>`
}
