package oauth2

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/backend/store"
)

// nolint:unused
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
	store       *store.Store
	secret      string
	externalURL string
}

func NewService(store *store.Store, secret, externalURL string) *Service {
	return &Service{
		store:       store,
		secret:      secret,
		externalURL: strings.TrimSuffix(externalURL, "/"),
	}
}

func (s *Service) RegisterRoutes(g *echo.Group) {
	g.GET("/.well-known/oauth-authorization-server", s.handleDiscovery)
	g.GET("/.well-known/oauth-authorization-server/*", s.handleDiscovery)
	g.GET("/.well-known/oauth-protected-resource", s.handleProtectedResourceMetadata)
	g.GET("/.well-known/oauth-protected-resource/*", s.handleProtectedResourceMetadata)
	g.POST("/oauth2/register", s.handleRegister)
	g.GET("/oauth2/authorize", s.handleAuthorizeGet)
	g.POST("/oauth2/authorize", s.handleAuthorizePost)
	g.GET("/oauth2/clients/:clientID", s.handleGetClient)
	g.POST("/oauth2/token", s.handleToken)
	g.POST("/oauth2/revoke", s.handleRevoke)
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

// nolint:unused
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// nolint:unused
func generateClientID() (string, error) {
	random, err := generateRandomString(24)
	if err != nil {
		return "", err
	}
	return clientIDPrefix + random, nil
}

// nolint:unused
func generateClientSecret() (string, error) {
	random, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	return clientSecretPrefix + random, nil
}

// nolint:unused
func generateAuthCode() (string, error) {
	random, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	return authCodePrefix + random, nil
}

// nolint:unused
func generateRefreshToken() (string, error) {
	random, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	return refreshTokenPrefix + random, nil
}

// nolint:unused
func hashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// nolint:unused
func verifySecret(hash, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}

// nolint:unused
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// nolint:unused
func verifyPKCE(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	return computed == codeChallenge
}

// nolint:unused
func validateRedirectURI(uri string, allowedURIs []string) bool {
	return slices.Contains(allowedURIs, uri)
}

// nolint:unused
func isLocalhostURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// nolint:unused
func oauth2Error(c echo.Context, statusCode int, errorCode, description string) error {
	return c.JSON(statusCode, map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
}

// nolint:unused
func oauth2ErrorRedirect(c echo.Context, redirectURI, state, errorCode, description string) error {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", errorCode)
	q.Set("error_description", description)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return c.Redirect(http.StatusFound, u.String())
}

// nolint:unused
func (s *Service) issuer() string {
	return s.externalURL
}

// nolint:unused
func (s *Service) authorizationEndpoint() string {
	return fmt.Sprintf("%s/oauth2/authorize", s.externalURL)
}

// nolint:unused
func (s *Service) tokenEndpoint() string {
	return fmt.Sprintf("%s/oauth2/token", s.externalURL)
}

// nolint:unused
func (s *Service) registrationEndpoint() string {
	return fmt.Sprintf("%s/oauth2/register", s.externalURL)
}

// nolint:unused
func (s *Service) revocationEndpoint() string {
	return fmt.Sprintf("%s/oauth2/revoke", s.externalURL)
}
