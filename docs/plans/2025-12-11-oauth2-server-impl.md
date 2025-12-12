# OAuth2 Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add OAuth2 Authorization Server to Bytebase enabling third-party apps (MCP servers) to authenticate users via Authorization Code + PKCE flow.

**Architecture:** Standard HTTP endpoints at `/oauth2/*` using Echo framework. JWT access tokens reusing existing auth infrastructure. All state in PostgreSQL with proto-defined JSONB configs.

**Tech Stack:** Go, Echo framework, PostgreSQL, protobuf/protojson, bcrypt, crypto/rand

**Design Doc:** `docs/plans/2025-12-11-oauth2-server-design.md`

---

## Task 1: Proto Definitions

**Files:**
- Create: `proto/store/oauth2.proto`

**Step 1: Create proto file**

```protobuf
syntax = "proto3";

package bytebase.store;

option go_package = "generated-go/store";

message OAuth2ClientConfig {
  string client_name = 1;
  repeated string redirect_uris = 2;
  repeated string grant_types = 3;
  string token_endpoint_auth_method = 4;
}

message OAuth2AuthorizationCodeConfig {
  string redirect_uri = 1;
  string code_challenge = 2;
  string code_challenge_method = 3;
}
```

**Step 2: Generate Go code**

Run: `cd proto && buf generate`

**Step 3: Verify generation**

Run: `ls backend/generated-go/store/oauth2.pb.go`
Expected: File exists

**Step 4: Commit**

```bash
but commit <branch> -m "proto: add OAuth2 store proto definitions"
```

---

## Task 2: Database Migration

**Files:**
- Create: `backend/migrator/migration/3.13/0020##add_oauth2_tables.sql`
- Modify: `backend/migrator/migration/LATEST.sql`
- Modify: `backend/migrator/migrator_test.go:15`

**Step 1: Create migration file**

```sql
CREATE TABLE oauth2_client (
    client_id TEXT PRIMARY KEY,
    client_secret_hash TEXT NOT NULL,
    config JSONB NOT NULL,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE oauth2_authorization_code (
    code TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES principal(id),
    config JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE oauth2_refresh_token (
    token_hash TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES principal(id),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth2_authorization_code_expires_at ON oauth2_authorization_code(expires_at);
CREATE INDEX idx_oauth2_refresh_token_expires_at ON oauth2_refresh_token(expires_at);
CREATE INDEX idx_oauth2_client_last_active_at ON oauth2_client(last_active_at);
```

**Step 2: Update LATEST.sql**

Add the same table definitions to `backend/migrator/migration/LATEST.sql` after the existing table definitions.

**Step 3: Update migrator_test.go**

Change line 15 from:
```go
require.Equal(t, semver.MustParse("3.13.19"), *files[len(files)-1].version)
```
to:
```go
require.Equal(t, semver.MustParse("3.13.20"), *files[len(files)-1].version)
```

**Step 4: Run migration tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run ^TestLatestVersion$`
Expected: PASS

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run ^TestVersionUnique$`
Expected: PASS

**Step 5: Commit**

```bash
but commit <branch> -m "migration: add OAuth2 tables"
```

---

## Task 3: Store Layer - OAuth2 Client

**Files:**
- Create: `backend/store/oauth2_client.go`

**Step 1: Create store file**

```go
package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type OAuth2ClientMessage struct {
	ClientID         string
	ClientSecretHash string
	Config           *storepb.OAuth2ClientConfig
	LastActiveAt     time.Time
}

type FindOAuth2ClientMessage struct {
	ClientID *string
}

func (s *Store) CreateOAuth2Client(ctx context.Context, create *OAuth2ClientMessage) (*OAuth2ClientMessage, error) {
	configBytes, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}

	q := qb.Q().Space(`
		INSERT INTO oauth2_client (client_id, client_secret_hash, config, last_active_at)
		VALUES (?, ?, ?, NOW())
		RETURNING last_active_at
	`, create.ClientID, create.ClientSecretHash, configBytes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.LastActiveAt); err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth2 client")
	}
	return create, nil
}

func (s *Store) GetOAuth2Client(ctx context.Context, clientID string) (*OAuth2ClientMessage, error) {
	clients, err := s.ListOAuth2Clients(ctx, &FindOAuth2ClientMessage{ClientID: &clientID})
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, nil
	}
	return clients[0], nil
}

func (s *Store) ListOAuth2Clients(ctx context.Context, find *FindOAuth2ClientMessage) ([]*OAuth2ClientMessage, error) {
	q := qb.Q().Space(`
		SELECT client_id, client_secret_hash, config, last_active_at
		FROM oauth2_client
		WHERE TRUE
	`)

	if v := find.ClientID; v != nil {
		q.And("client_id = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query OAuth2 clients")
	}
	defer rows.Close()

	var clients []*OAuth2ClientMessage
	for rows.Next() {
		client := &OAuth2ClientMessage{}
		var configBytes []byte
		if err := rows.Scan(&client.ClientID, &client.ClientSecretHash, &configBytes, &client.LastActiveAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan OAuth2 client")
		}
		client.Config = &storepb.OAuth2ClientConfig{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, client.Config); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal config")
		}
		clients = append(clients, client)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate OAuth2 clients")
	}
	return clients, nil
}

func (s *Store) UpdateOAuth2ClientLastActiveAt(ctx context.Context, clientID string) error {
	q := qb.Q().Space(`
		UPDATE oauth2_client
		SET last_active_at = NOW()
		WHERE client_id = ?
	`, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to update OAuth2 client last active at")
	}
	return nil
}

func (s *Store) DeleteOAuth2Client(ctx context.Context, clientID string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_client
		WHERE client_id = ?
	`, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 client")
	}
	return nil
}

func (s *Store) DeleteExpiredOAuth2Clients(ctx context.Context, expireBefore time.Time) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM oauth2_client
		WHERE last_active_at < ?
	`, expireBefore)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired OAuth2 clients")
	}
	return result.RowsAffected()
}
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/store/oauth2_client.go`
Expected: No errors (or only unrelated warnings)

**Step 3: Commit**

```bash
but commit <branch> -m "store: add OAuth2 client CRUD"
```

---

## Task 4: Store Layer - Authorization Code

**Files:**
- Create: `backend/store/oauth2_authorization_code.go`

**Step 1: Create store file**

```go
package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type OAuth2AuthorizationCodeMessage struct {
	Code      string
	ClientID  string
	UserID    int
	Config    *storepb.OAuth2AuthorizationCodeConfig
	ExpiresAt time.Time
}

func (s *Store) CreateOAuth2AuthorizationCode(ctx context.Context, create *OAuth2AuthorizationCodeMessage) (*OAuth2AuthorizationCodeMessage, error) {
	configBytes, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}

	q := qb.Q().Space(`
		INSERT INTO oauth2_authorization_code (code, client_id, user_id, config, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, create.Code, create.ClientID, create.UserID, configBytes, create.ExpiresAt)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth2 authorization code")
	}
	return create, nil
}

func (s *Store) GetOAuth2AuthorizationCode(ctx context.Context, code string) (*OAuth2AuthorizationCodeMessage, error) {
	q := qb.Q().Space(`
		SELECT code, client_id, user_id, config, expires_at
		FROM oauth2_authorization_code
		WHERE code = ?
	`, code)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &OAuth2AuthorizationCodeMessage{}
	var configBytes []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&msg.Code, &msg.ClientID, &msg.UserID, &configBytes, &msg.ExpiresAt,
	); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get OAuth2 authorization code")
	}

	msg.Config = &storepb.OAuth2AuthorizationCodeConfig{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, msg.Config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}
	return msg, nil
}

func (s *Store) DeleteOAuth2AuthorizationCode(ctx context.Context, code string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_authorization_code
		WHERE code = ?
	`, code)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 authorization code")
	}
	return nil
}

func (s *Store) DeleteExpiredOAuth2AuthorizationCodes(ctx context.Context) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM oauth2_authorization_code
		WHERE expires_at < NOW()
	`)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired OAuth2 authorization codes")
	}
	return result.RowsAffected()
}
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/store/oauth2_authorization_code.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "store: add OAuth2 authorization code CRUD"
```

---

## Task 5: Store Layer - Refresh Token

**Files:**
- Create: `backend/store/oauth2_refresh_token.go`

**Step 1: Create store file**

```go
package store

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

type OAuth2RefreshTokenMessage struct {
	TokenHash string
	ClientID  string
	UserID    int
	ExpiresAt time.Time
}

func (s *Store) CreateOAuth2RefreshToken(ctx context.Context, create *OAuth2RefreshTokenMessage) (*OAuth2RefreshTokenMessage, error) {
	q := qb.Q().Space(`
		INSERT INTO oauth2_refresh_token (token_hash, client_id, user_id, expires_at)
		VALUES (?, ?, ?, ?)
	`, create.TokenHash, create.ClientID, create.UserID, create.ExpiresAt)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth2 refresh token")
	}
	return create, nil
}

func (s *Store) GetOAuth2RefreshToken(ctx context.Context, tokenHash string) (*OAuth2RefreshTokenMessage, error) {
	q := qb.Q().Space(`
		SELECT token_hash, client_id, user_id, expires_at
		FROM oauth2_refresh_token
		WHERE token_hash = ?
	`, tokenHash)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &OAuth2RefreshTokenMessage{}
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&msg.TokenHash, &msg.ClientID, &msg.UserID, &msg.ExpiresAt,
	); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get OAuth2 refresh token")
	}
	return msg, nil
}

func (s *Store) DeleteOAuth2RefreshToken(ctx context.Context, tokenHash string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE token_hash = ?
	`, tokenHash)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 refresh token")
	}
	return nil
}

func (s *Store) DeleteOAuth2RefreshTokensByUserAndClient(ctx context.Context, userID int, clientID string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE user_id = ? AND client_id = ?
	`, userID, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 refresh tokens")
	}
	return nil
}

func (s *Store) DeleteExpiredOAuth2RefreshTokens(ctx context.Context) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE expires_at < NOW()
	`)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired OAuth2 refresh tokens")
	}
	return result.RowsAffected()
}
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/store/oauth2_refresh_token.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "store: add OAuth2 refresh token CRUD"
```

---

## Task 6: OAuth2 Handler - Utilities

**Files:**
- Create: `backend/api/oauth2/oauth2.go`

**Step 1: Create main handler file with utilities**

```go
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

const (
	clientIDPrefix       = "bb_oauth_"
	clientSecretPrefix   = "bb_secret_"
	refreshTokenPrefix   = "bb_refresh_"
	authCodePrefix       = "bb_code_"

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
	g.POST("/oauth2/register", s.handleRegister)
	g.GET("/oauth2/authorize", s.handleAuthorizeGet)
	g.POST("/oauth2/authorize", s.handleAuthorizePost)
	g.POST("/oauth2/token", s.handleToken)
	g.POST("/oauth2/revoke", s.handleRevoke)
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func generateClientID() (string, error) {
	random, err := generateRandomString(24)
	if err != nil {
		return "", err
	}
	return clientIDPrefix + random, nil
}

func generateClientSecret() (string, error) {
	random, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	return clientSecretPrefix + random, nil
}

func generateAuthCode() (string, error) {
	random, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	return authCodePrefix + random, nil
}

func generateRefreshToken() (string, error) {
	random, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	return refreshTokenPrefix + random, nil
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

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func verifyPKCE(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	return computed == codeChallenge
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

func (s *Service) issuer() string {
	return s.externalURL
}

func (s *Service) authorizationEndpoint() string {
	return fmt.Sprintf("%s/oauth2/authorize", s.externalURL)
}

func (s *Service) tokenEndpoint() string {
	return fmt.Sprintf("%s/oauth2/token", s.externalURL)
}

func (s *Service) registrationEndpoint() string {
	return fmt.Sprintf("%s/oauth2/register", s.externalURL)
}

func (s *Service) revocationEndpoint() string {
	return fmt.Sprintf("%s/oauth2/revoke", s.externalURL)
}
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/oauth2/oauth2.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "api: add OAuth2 service with utilities"
```

---

## Task 7: OAuth2 Handler - Discovery

**Files:**
- Create: `backend/api/oauth2/discovery.go`

**Step 1: Create discovery handler**

```go
package oauth2

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type authorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

func (s *Service) handleDiscovery(c echo.Context) error {
	metadata := &authorizationServerMetadata{
		Issuer:                            s.issuer(),
		AuthorizationEndpoint:             s.authorizationEndpoint(),
		TokenEndpoint:                     s.tokenEndpoint(),
		RegistrationEndpoint:              s.registrationEndpoint(),
		RevocationEndpoint:                s.revocationEndpoint(),
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
	}
	return c.JSON(http.StatusOK, metadata)
}
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/oauth2/discovery.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "api: add OAuth2 discovery endpoint"
```

---

## Task 8: OAuth2 Handler - Dynamic Client Registration

**Files:**
- Create: `backend/api/oauth2/register.go`

**Step 1: Create DCR handler**

```go
package oauth2

import (
	"net/http"
	"net/url"
	"slices"

	"github.com/labstack/echo/v4"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type clientRegistrationRequest struct {
	ClientName                string   `json:"client_name"`
	RedirectURIs              []string `json:"redirect_uris"`
	GrantTypes                []string `json:"grant_types"`
	TokenEndpointAuthMethod   string   `json:"token_endpoint_auth_method"`
}

type clientRegistrationResponse struct {
	ClientID                  string   `json:"client_id"`
	ClientSecret              string   `json:"client_secret"`
	ClientName                string   `json:"client_name"`
	RedirectURIs              []string `json:"redirect_uris"`
	GrantTypes                []string `json:"grant_types"`
	TokenEndpointAuthMethod   string   `json:"token_endpoint_auth_method"`
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
		RedirectUris:           req.RedirectURIs,
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
```

**Step 2: Add missing import**

Add `"github.com/bytebase/bytebase/backend/store"` to imports.

**Step 3: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/oauth2/register.go`
Expected: No errors

**Step 4: Commit**

```bash
but commit <branch> -m "api: add OAuth2 dynamic client registration"
```

---

## Task 9: OAuth2 Handler - Authorization Endpoint

**Files:**
- Create: `backend/api/oauth2/authorize.go`

**Step 1: Create authorization handler**

```go
package oauth2

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	accessToken, err := auth.GetTokenFromRequest(c.Request())
	if err != nil || accessToken == "" {
		// Redirect to login page with return URL
		loginURL := fmt.Sprintf("/auth/login?redirect=%s", url.QueryEscape(c.Request().URL.String()))
		return c.Redirect(http.StatusFound, loginURL)
	}

	// Return consent page HTML
	return s.renderConsentPage(c, client.Config.ClientName, clientID, redirectURI, state, codeChallenge, codeChallengeMethod)
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
	accessToken, err := auth.GetTokenFromRequest(c.Request())
	if err != nil || accessToken == "" {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "user not authenticated")
	}

	claims, err := auth.ValidateAccessToken(accessToken, s.secret)
	if err != nil {
		return oauth2ErrorRedirect(c, redirectURI, state, "access_denied", "invalid session")
	}

	// Get user from claims
	user, err := s.store.GetUserByEmail(ctx, claims.Subject)
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

func (s *Service) renderConsentPage(c echo.Context, clientName, clientID, redirectURI, state, codeChallenge, codeChallengeMethod string) error {
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
```

**Step 2: Check auth helpers exist**

The code references `auth.GetTokenFromRequest` and `auth.ValidateAccessToken`. These may need to be added or use existing functions. Check `backend/api/auth/auth.go` for existing patterns and adapt as needed.

**Step 3: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/oauth2/authorize.go`
Expected: No errors (fix any import issues)

**Step 4: Commit**

```bash
but commit <branch> -m "api: add OAuth2 authorization endpoint"
```

---

## Task 10: OAuth2 Handler - Token Endpoint

**Files:**
- Create: `backend/api/oauth2/token.go`

**Step 1: Create token handler**

```go
package oauth2

import (
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

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
	if !verifySecret(client.ClientSecretHash, clientSecret) {
		return oauth2Error(c, http.StatusUnauthorized, "invalid_client", "invalid client credentials")
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

func (s *Service) extractClientCredentials(c echo.Context, req *tokenRequest) (clientID, clientSecret string) {
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

	// Delete code immediately (single use)
	_ = s.store.DeleteOAuth2AuthorizationCode(ctx, req.Code)

	// Validate code belongs to this client
	if authCode.ClientID != client.ClientID {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "code was not issued to this client")
	}

	// Validate code not expired
	if time.Now().After(authCode.ExpiresAt) {
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

	// Get user
	user, err := s.store.GetUserByID(ctx, authCode.UserID)
	if err != nil || user == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "user not found")
	}

	// Generate tokens
	return s.issueTokens(c, client, user.ID, user.Email)
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

	tokenHash := hashToken(req.RefreshToken)
	refreshToken, err := s.store.GetOAuth2RefreshToken(ctx, tokenHash)
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to lookup refresh token")
	}
	if refreshToken == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "invalid refresh token")
	}

	// Delete token immediately (single use, will issue new one)
	_ = s.store.DeleteOAuth2RefreshToken(ctx, tokenHash)

	// Validate token belongs to this client
	if refreshToken.ClientID != client.ClientID {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "refresh token was not issued to this client")
	}

	// Validate not expired
	if time.Now().After(refreshToken.ExpiresAt) {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "refresh token has expired")
	}

	// Get user
	user, err := s.store.GetUserByID(ctx, refreshToken.UserID)
	if err != nil || user == nil {
		return oauth2Error(c, http.StatusBadRequest, "invalid_grant", "user not found")
	}

	// Issue new tokens
	return s.issueTokens(c, client, user.ID, user.Email)
}

func (s *Service) issueTokens(c echo.Context, client *store.OAuth2ClientMessage, userID int, userEmail string) error {
	ctx := c.Request().Context()

	// Generate access token (JWT)
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "bytebase",
		"sub":       userEmail,
		"aud":       "bb.oauth2.access",
		"exp":       now.Add(accessTokenExpiry).Unix(),
		"iat":       now.Unix(),
		"client_id": client.ClientID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate access token")
	}

	// Generate refresh token if allowed
	var refreshTokenStr string
	if slices.Contains(client.Config.GrantTypes, "refresh_token") {
		refreshTokenStr, err = generateRefreshToken()
		if err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to generate refresh token")
		}

		// Store refresh token
		if _, err := s.store.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: hashToken(refreshTokenStr),
			ClientID:  client.ClientID,
			UserID:    userID,
			ExpiresAt: now.Add(refreshTokenExpiry),
		}); err != nil {
			return oauth2Error(c, http.StatusInternalServerError, "server_error", "failed to store refresh token")
		}
	}

	// Update client last active
	_ = s.store.UpdateOAuth2ClientLastActiveAt(ctx, client.ClientID)

	return c.JSON(http.StatusOK, &tokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenExpiry.Seconds()),
		RefreshToken: refreshTokenStr,
	})
}
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/oauth2/token.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "api: add OAuth2 token endpoint"
```

---

## Task 11: OAuth2 Handler - Revocation Endpoint

**Files:**
- Create: `backend/api/oauth2/revoke.go`

**Step 1: Create revoke handler**

```go
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
	clientID, clientSecret := s.extractRevokeClientCredentials(c, &req)
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
	tokenHash := hashToken(req.Token)
	if err := s.store.DeleteOAuth2RefreshToken(ctx, tokenHash); err != nil {
		// Ignore errors - RFC 7009 says to return 200 even if token is invalid
	}

	// Return success (RFC 7009: always return 200)
	return c.NoContent(http.StatusOK)
}

func (s *Service) extractRevokeClientCredentials(c echo.Context, req *revokeRequest) (clientID, clientSecret string) {
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
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/oauth2/revoke.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "api: add OAuth2 revocation endpoint"
```

---

## Task 12: Register OAuth2 Routes in Server

**Files:**
- Modify: `backend/server/server.go`
- Modify: `backend/server/echo_routes.go`

**Step 1: Add OAuth2 service to server struct**

In `backend/server/server.go`, add import and initialize OAuth2 service in `NewServer`:

```go
import (
	// ... existing imports
	"github.com/bytebase/bytebase/backend/api/oauth2"
)

// In NewServer function, after other service initializations:
oauth2Service := oauth2.NewService(stores, secret, profile.ExternalURL)
```

**Step 2: Register OAuth2 routes in echo_routes.go**

In `backend/server/echo_routes.go`, add OAuth2 route registration in `configureEchoRouters`:

```go
func configureEchoRouters(
	e *echo.Echo,
	lspServer *lsp.Server,
	directorySyncServer *directorysync.Service,
	oauth2Service *oauth2.Service,  // Add parameter
	profile *config.Profile,
) {
	// ... existing code ...

	// OAuth2 routes
	oauth2Service.RegisterRoutes(e.Group(""))
}
```

**Step 3: Update configureEchoRouters call**

In `backend/server/server.go`, update the call to pass oauth2Service.

**Step 4: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/server/...`
Expected: No errors

**Step 5: Commit**

```bash
but commit <branch> -m "server: register OAuth2 routes"
```

---

## Task 13: Auth Interceptor - Accept OAuth2 Tokens

**Files:**
- Modify: `backend/api/auth/auth.go`

**Step 1: Update token validation to accept OAuth2 audience**

Find the token validation logic and add `bb.oauth2.access` as a valid audience:

```go
// In the ValidateAccessToken or similar function, update audience check:
validAudiences := []string{
	fmt.Sprintf("bb.user.access.%s", mode),
	"bb.oauth2.access",  // Add this
}
```

**Step 2: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/auth/...`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "auth: accept OAuth2 access tokens"
```

---

## Task 14: Add Store Helper Methods

**Files:**
- Modify: `backend/store/user.go` (or principal.go)

**Step 1: Add GetUserByEmail if not exists**

Check if `GetUserByEmail` exists. If not, add:

```go
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	users, err := s.ListUsers(ctx, &FindUserMessage{Email: &email})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}
```

**Step 2: Add GetUserByID if not exists**

```go
func (s *Store) GetUserByID(ctx context.Context, id int) (*UserMessage, error) {
	users, err := s.ListUsers(ctx, &FindUserMessage{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}
```

**Step 3: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/store/...`
Expected: No errors

**Step 4: Commit**

```bash
but commit <branch> -m "store: add user lookup helpers"
```

---

## Task 15: Build and Test

**Step 1: Build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run linter on all changes**

Run: `golangci-lint run --allow-parallel-runners ./backend/...`
Expected: No errors (run repeatedly until clean)

**Step 3: Manual test (optional)**

Start the server and test the OAuth2 flow:

1. Discovery: `curl http://localhost:8080/.well-known/oauth-authorization-server`
2. Register client: `curl -X POST http://localhost:8080/oauth2/register -H "Content-Type: application/json" -d '{"client_name":"Test","redirect_uris":["http://localhost:3000/callback"]}'`
3. Open authorize URL in browser
4. Exchange code for token

**Step 4: Commit**

```bash
but commit <branch> -m "build: verify OAuth2 implementation compiles"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Proto definitions | `proto/store/oauth2.proto` |
| 2 | Database migration | `backend/migrator/migration/3.13/0020##...` |
| 3 | Store: OAuth2 Client | `backend/store/oauth2_client.go` |
| 4 | Store: Authorization Code | `backend/store/oauth2_authorization_code.go` |
| 5 | Store: Refresh Token | `backend/store/oauth2_refresh_token.go` |
| 6 | Handler: Utilities | `backend/api/oauth2/oauth2.go` |
| 7 | Handler: Discovery | `backend/api/oauth2/discovery.go` |
| 8 | Handler: DCR | `backend/api/oauth2/register.go` |
| 9 | Handler: Authorize | `backend/api/oauth2/authorize.go` |
| 10 | Handler: Token | `backend/api/oauth2/token.go` |
| 11 | Handler: Revoke | `backend/api/oauth2/revoke.go` |
| 12 | Server registration | `backend/server/server.go`, `echo_routes.go` |
| 13 | Auth interceptor | `backend/api/auth/auth.go` |
| 14 | Store helpers | `backend/store/user.go` |
| 15 | Build and test | - |
