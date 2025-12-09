# MCP OAuth Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add OAuth 2.1 with PKCE support to the MCP server, enabling seamless authentication via Bytebase's existing IdP integrations (Google, GitHub, OIDC).

**Architecture:** Bytebase acts as OAuth Authorization Server, delegating user authentication to configured IdPs. Uses MCP SDK's `auth` package for middleware and `oauthex` for metadata types.

**Tech Stack:** `github.com/modelcontextprotocol/go-sdk/auth`, `github.com/modelcontextprotocol/go-sdk/oauthex`, existing Bytebase IdP plugins

**Design Doc:** `docs/plans/2024-12-08-mcp-server-design.md` (Authentication section)

---

## Task 1: Add OAuth Metadata Endpoints

**Files:**
- Create: `backend/api/mcp/oauth/metadata.go`

**Step 1: Create oauth package directory**

```bash
mkdir -p backend/api/mcp/oauth
```

**Step 2: Write metadata.go**

```go
// Package oauth provides OAuth 2.1 authorization server for MCP.
package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// MetadataServer serves OAuth metadata endpoints.
type MetadataServer struct {
	issuer string
}

// NewMetadataServer creates a new metadata server.
func NewMetadataServer(issuer string) *MetadataServer {
	return &MetadataServer{issuer: issuer}
}

// ProtectedResourceMetadata handles GET /.well-known/oauth-protected-resource
func (s *MetadataServer) ProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:             s.issuer + "/mcp",
		AuthorizationServers: []string{s.issuer},
		ScopesSupported:      []string{"mcp:tools"},
		BearerMethodsSupported: []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(metadata)
}

// AuthorizationServerMetadata handles GET /.well-known/oauth-authorization-server
func (s *MetadataServer) AuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":               s.issuer + "/oauth/authorize",
		"token_endpoint":                       s.issuer + "/oauth/token",
		"response_types_supported":             []string{"code"},
		"grant_types_supported":                []string{"authorization_code"},
		"code_challenge_methods_supported":     []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                     []string{"mcp:tools"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(metadata)
}
```

**Step 3: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

**Step 4: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): add OAuth metadata endpoints"
```

---

## Task 2: Create PKCE Utilities

**Files:**
- Create: `backend/api/mcp/oauth/pkce.go`

**Step 1: Write pkce.go**

```go
package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/pkg/errors"
)

// VerifyPKCE verifies that the code_verifier matches the code_challenge.
// Only S256 method is supported per OAuth 2.1 requirements.
func VerifyPKCE(codeVerifier, codeChallenge, codeChallengeMethod string) error {
	if codeChallengeMethod != "S256" {
		return errors.New("only S256 code_challenge_method is supported")
	}

	// S256: BASE64URL(SHA256(code_verifier)) == code_challenge
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])

	if computed != codeChallenge {
		return errors.New("code_verifier does not match code_challenge")
	}

	return nil
}

// ValidateCodeVerifier checks that the code_verifier meets RFC 7636 requirements.
func ValidateCodeVerifier(verifier string) error {
	// code_verifier must be 43-128 characters
	if len(verifier) < 43 || len(verifier) > 128 {
		return errors.New("code_verifier must be 43-128 characters")
	}

	// Must only contain unreserved characters: [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
	for _, c := range verifier {
		if !isUnreserved(c) {
			return errors.New("code_verifier contains invalid characters")
		}
	}

	return nil
}

// ValidateCodeChallenge checks that the code_challenge is valid base64url.
func ValidateCodeChallenge(challenge string) error {
	// S256 challenge is 43 characters (256 bits in base64url)
	if len(challenge) != 43 {
		return errors.New("code_challenge must be 43 characters for S256")
	}

	// Must be valid base64url (no padding)
	if strings.ContainsAny(challenge, "+/=") {
		return errors.New("code_challenge must be base64url encoded without padding")
	}

	return nil
}

func isUnreserved(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

**Step 3: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): add PKCE verification utilities"
```

---

## Task 3: Create Authorization Code Store

**Files:**
- Create: `backend/api/mcp/oauth/store.go`

**Step 1: Write store.go**

```go
package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// AuthorizationCode represents a pending OAuth authorization.
type AuthorizationCode struct {
	Code                string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scopes              []string
	UserID              int      // Bytebase user ID after IdP auth
	UserEmail           string
	ExpiresAt           time.Time
	IdPName             string   // Which IdP was used
	IdPContext          string   // IdP-specific context (e.g., state)
}

// AuthCodeStore manages authorization codes in memory.
// For production, consider using Redis or database storage.
type AuthCodeStore struct {
	mu    sync.RWMutex
	codes map[string]*AuthorizationCode
}

// NewAuthCodeStore creates a new authorization code store.
func NewAuthCodeStore() *AuthCodeStore {
	store := &AuthCodeStore{
		codes: make(map[string]*AuthorizationCode),
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// GenerateCode creates a new authorization code.
func (s *AuthCodeStore) GenerateCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Store saves an authorization code.
func (s *AuthCodeStore) Store(code *AuthorizationCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code.Code] = code
}

// Consume retrieves and deletes an authorization code (one-time use).
func (s *AuthCodeStore) Consume(code string) (*AuthorizationCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCode, ok := s.codes[code]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(authCode.ExpiresAt) {
		delete(s.codes, code)
		return nil, false
	}

	// Delete after retrieval (one-time use)
	delete(s.codes, code)
	return authCode, true
}

// StorePending stores a pending authorization (before IdP callback).
func (s *AuthCodeStore) StorePending(state string, code *AuthorizationCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Use state as temporary key
	s.codes["pending:"+state] = code
}

// ConsumePending retrieves pending authorization by state.
func (s *AuthCodeStore) ConsumePending(state string) (*AuthorizationCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCode, ok := s.codes["pending:"+state]
	if !ok {
		return nil, false
	}

	delete(s.codes, "pending:"+state)
	return authCode, true
}

// cleanup removes expired codes periodically.
func (s *AuthCodeStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for code, authCode := range s.codes {
			if now.After(authCode.ExpiresAt) {
				delete(s.codes, code)
			}
		}
		s.mu.Unlock()
	}
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

**Step 3: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): add authorization code store"
```

---

## Task 4: Create Authorization Endpoint

**Files:**
- Create: `backend/api/mcp/oauth/authorize.go`

**Step 1: Write authorize.go**

```go
package oauth

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
)

// AuthorizeHandler handles the /oauth/authorize endpoint.
type AuthorizeHandler struct {
	store       *store.Store
	codeStore   *AuthCodeStore
	issuer      string
}

// NewAuthorizeHandler creates a new authorize handler.
func NewAuthorizeHandler(store *store.Store, codeStore *AuthCodeStore, issuer string) *AuthorizeHandler {
	return &AuthorizeHandler{
		store:     store,
		codeStore: codeStore,
		issuer:    issuer,
	}
}

// ServeHTTP handles GET /oauth/authorize
func (h *AuthorizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse required parameters
	responseType := r.URL.Query().Get("response_type")
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
	state := r.URL.Query().Get("state")
	scope := r.URL.Query().Get("scope")

	// Validate required parameters
	if responseType != "code" {
		h.errorRedirect(w, r, redirectURI, state, "unsupported_response_type", "only code is supported")
		return
	}

	if codeChallenge == "" || codeChallengeMethod == "" {
		h.errorRedirect(w, r, redirectURI, state, "invalid_request", "PKCE is required")
		return
	}

	if codeChallengeMethod != "S256" {
		h.errorRedirect(w, r, redirectURI, state, "invalid_request", "only S256 is supported")
		return
	}

	if err := ValidateCodeChallenge(codeChallenge); err != nil {
		h.errorRedirect(w, r, redirectURI, state, "invalid_request", err.Error())
		return
	}

	// Store pending authorization
	pendingAuth := &AuthorizationCode{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Scopes:              []string{scope},
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		IdPContext:          state,
	}
	h.codeStore.StorePending(state, pendingAuth)

	// Redirect to Bytebase login page with OAuth context
	// The login page will handle IdP selection and redirect back
	loginURL := fmt.Sprintf("%s/oauth/login?state=%s", h.issuer, url.QueryEscape(state))
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// CompleteAuthorization is called after successful IdP authentication.
func (h *AuthorizeHandler) CompleteAuthorization(w http.ResponseWriter, r *http.Request, state string, userID int, userEmail string) error {
	// Retrieve pending authorization
	pending, ok := h.codeStore.ConsumePending(state)
	if !ok {
		return errors.New("invalid or expired state")
	}

	// Generate authorization code
	code, err := h.codeStore.GenerateCode()
	if err != nil {
		return errors.Wrap(err, "failed to generate code")
	}

	// Store the authorization code with user info
	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            pending.ClientID,
		RedirectURI:         pending.RedirectURI,
		CodeChallenge:       pending.CodeChallenge,
		CodeChallengeMethod: pending.CodeChallengeMethod,
		Scopes:              pending.Scopes,
		UserID:              userID,
		UserEmail:           userEmail,
		ExpiresAt:           time.Now().Add(5 * time.Minute),
	}
	h.codeStore.Store(authCode)

	// Redirect back to client with authorization code
	redirectURL, _ := url.Parse(pending.RedirectURI)
	q := redirectURL.Query()
	q.Set("code", code)
	q.Set("state", state)
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
	return nil
}

func (h *AuthorizeHandler) errorRedirect(w http.ResponseWriter, r *http.Request, redirectURI, state, errorCode, errorDesc string) {
	if redirectURI == "" {
		http.Error(w, errorDesc, http.StatusBadRequest)
		return
	}

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	q := redirectURL.Query()
	q.Set("error", errorCode)
	q.Set("error_description", errorDesc)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

**Step 3: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): add OAuth authorize endpoint"
```

---

## Task 5: Create Token Endpoint

**Files:**
- Create: `backend/api/mcp/oauth/token.go`

**Step 1: Write token.go**

```go
package oauth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/store"
)

// TokenHandler handles the /oauth/token endpoint.
type TokenHandler struct {
	store     *store.Store
	codeStore *AuthCodeStore
	secret    string
}

// NewTokenHandler creates a new token handler.
func NewTokenHandler(store *store.Store, codeStore *AuthCodeStore, secret string) *TokenHandler {
	return &TokenHandler{
		store:     store,
		codeStore: codeStore,
		secret:    secret,
	}
}

// TokenResponse represents the OAuth token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// TokenErrorResponse represents an OAuth error response.
type TokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// ServeHTTP handles POST /oauth/token
func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid_request", "failed to parse form")
		return
	}

	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	// Validate grant type
	if grantType != "authorization_code" {
		h.errorResponse(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code is supported")
		return
	}

	// Validate code verifier
	if err := ValidateCodeVerifier(codeVerifier); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Consume authorization code
	authCode, ok := h.codeStore.Consume(code)
	if !ok {
		h.errorResponse(w, http.StatusBadRequest, "invalid_grant", "invalid or expired code")
		return
	}

	// Verify redirect_uri matches
	if authCode.RedirectURI != redirectURI {
		h.errorResponse(w, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
		return
	}

	// Verify PKCE
	if err := VerifyPKCE(codeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}

	// Generate Bytebase access token (JWT)
	expiresIn := 7 * 24 * time.Hour // 7 days
	accessToken, err := auth.GenerateAccessToken(
		authCode.UserEmail,
		authCode.UserID,
		time.Now().Add(expiresIn),
		h.secret,
	)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "server_error", "failed to generate token")
		return
	}

	// Return token response
	response := &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(expiresIn.Seconds()),
		Scope:       "mcp:tools",
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(response)
}

func (h *TokenHandler) errorResponse(w http.ResponseWriter, status int, errorCode, errorDesc string) {
	response := &TokenErrorResponse{
		Error:            errorCode,
		ErrorDescription: errorDesc,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

**Step 3: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): add OAuth token endpoint"
```

---

## Task 6: Create OAuth Login Handler

**Files:**
- Create: `backend/api/mcp/oauth/login.go`

**Step 1: Write login.go**

This handler bridges the OAuth flow with Bytebase's existing IdP integration.

```go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/store"
)

// LoginHandler handles /oauth/login and /oauth/callback.
type LoginHandler struct {
	store            *store.Store
	codeStore        *AuthCodeStore
	authorizeHandler *AuthorizeHandler
	issuer           string
}

// NewLoginHandler creates a new login handler.
func NewLoginHandler(store *store.Store, codeStore *AuthCodeStore, authorizeHandler *AuthorizeHandler, issuer string) *LoginHandler {
	return &LoginHandler{
		store:            store,
		codeStore:        codeStore,
		authorizeHandler: authorizeHandler,
		issuer:           issuer,
	}
}

// ServeLogin handles GET /oauth/login - shows IdP selection or redirects to IdP.
func (h *LoginHandler) ServeLogin(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	// Get configured IdPs
	ctx := r.Context()
	idps, err := h.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
	if err != nil {
		http.Error(w, "failed to list identity providers", http.StatusInternalServerError)
		return
	}

	// If only one IdP, redirect directly
	if len(idps) == 1 {
		h.redirectToIdP(w, r, idps[0], state)
		return
	}

	// If multiple IdPs, return JSON list for client to choose
	// (In a full implementation, this would render an HTML page)
	idpList := make([]map[string]string, 0, len(idps))
	for _, idp := range idps {
		idpList = append(idpList, map[string]string{
			"name":  idp.ResourceID,
			"title": idp.Title,
			"type":  string(idp.Type),
			"url":   fmt.Sprintf("%s/oauth/login/%s?state=%s", h.issuer, idp.ResourceID, url.QueryEscape(state)),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"identity_providers": idpList,
		"message":            "Select an identity provider to continue",
	})
}

// ServeLoginWithIdP handles GET /oauth/login/{idp} - redirects to specific IdP.
func (h *LoginHandler) ServeLoginWithIdP(w http.ResponseWriter, r *http.Request, idpName string) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	idp, err := h.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &idpName,
	})
	if err != nil || idp == nil {
		http.Error(w, "identity provider not found", http.StatusNotFound)
		return
	}

	h.redirectToIdP(w, r, idp, state)
}

func (h *LoginHandler) redirectToIdP(w http.ResponseWriter, r *http.Request, idp *store.IdentityProviderMessage, state string) {
	callbackURL := fmt.Sprintf("%s/oauth/callback", h.issuer)

	var authURL string
	switch idp.Type {
	case storepb.IdentityProviderType_OAUTH2:
		oauth2Config := idp.Config.GetOauth2Config()
		authURL = oauth2.BuildAuthURL(oauth2Config, callbackURL, state)
	case storepb.IdentityProviderType_OIDC:
		oidcConfig := idp.Config.GetOidcConfig()
		authURL = oidc.BuildAuthURL(oidcConfig, callbackURL, state)
	default:
		http.Error(w, "unsupported identity provider type", http.StatusBadRequest)
		return
	}

	// Store IdP name in pending auth
	pending, ok := h.codeStore.ConsumePending(state)
	if ok {
		pending.IdPName = idp.ResourceID
		h.codeStore.StorePending(state, pending)
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// ServeCallback handles GET /oauth/callback - IdP redirects back here.
func (h *LoginHandler) ServeCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get state and code from IdP
	state := r.URL.Query().Get("state")
	idpCode := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("IdP error: %s - %s", errorParam, errorDesc), http.StatusBadRequest)
		return
	}

	// Get pending authorization
	pending, ok := h.codeStore.ConsumePending(state)
	if !ok {
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
		return
	}

	// Get IdP configuration
	idp, err := h.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &pending.IdPName,
	})
	if err != nil || idp == nil {
		http.Error(w, "identity provider not found", http.StatusInternalServerError)
		return
	}

	// Exchange code for user info
	callbackURL := fmt.Sprintf("%s/oauth/callback", h.issuer)
	userInfo, err := h.exchangeForUserInfo(ctx, idp, idpCode, callbackURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get user info: %v", err), http.StatusInternalServerError)
		return
	}

	// Find or create Bytebase user
	user, err := h.findOrCreateUser(ctx, userInfo, idp)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find user: %v", err), http.StatusInternalServerError)
		return
	}

	// Re-store pending for CompleteAuthorization
	h.codeStore.StorePending(state, pending)

	// Complete the OAuth flow
	if err := h.authorizeHandler.CompleteAuthorization(w, r, state, user.ID, user.Email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type userInfo struct {
	Email string
	Name  string
}

func (h *LoginHandler) exchangeForUserInfo(ctx context.Context, idp *store.IdentityProviderMessage, code, callbackURL string) (*userInfo, error) {
	switch idp.Type {
	case storepb.IdentityProviderType_OAUTH2:
		oauth2Config := idp.Config.GetOauth2Config()
		info, err := oauth2.ExchangeToken(ctx, oauth2Config, code, callbackURL)
		if err != nil {
			return nil, err
		}
		return &userInfo{Email: info.Email, Name: info.DisplayName}, nil

	case storepb.IdentityProviderType_OIDC:
		oidcConfig := idp.Config.GetOidcConfig()
		info, err := oidc.ExchangeToken(ctx, oidcConfig, code, callbackURL)
		if err != nil {
			return nil, err
		}
		return &userInfo{Email: info.Email, Name: info.DisplayName}, nil

	default:
		return nil, errors.New("unsupported identity provider type")
	}
}

func (h *LoginHandler) findOrCreateUser(ctx context.Context, info *userInfo, idp *store.IdentityProviderMessage) (*store.UserMessage, error) {
	// Try to find existing user by email
	user, err := h.store.GetUserByEmail(ctx, info.Email)
	if err != nil {
		return nil, err
	}

	if user != nil {
		return user, nil
	}

	// User not found - return error (don't auto-create for security)
	return nil, errors.Errorf("user %s not found in Bytebase", info.Email)
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

**Step 3: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): add OAuth login handler with IdP integration"
```

---

## Task 7: Add Token Verifier and Auth Middleware

**Files:**
- Modify: `backend/api/mcp/server.go`

**Step 1: Update server.go to add auth middleware**

Add the token verifier and wrap handler with auth middleware:

```go
// Add to imports
import (
	"github.com/modelcontextprotocol/go-sdk/auth"
	// ... existing imports
)

// Add token verifier function
func (s *Server) tokenVerifier(secret string) auth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		// Validate Bytebase JWT token
		claims, err := bbauth.ValidateAccessToken(token, secret)
		if err != nil {
			return nil, auth.ErrInvalidToken
		}

		return &auth.TokenInfo{
			Scopes:     []string{"mcp:tools"},
			Expiration: time.Unix(claims.ExpiresAt, 0),
			UserID:     fmt.Sprintf("%d", claims.UserID),
			Extra: map[string]any{
				"email": claims.Email,
			},
		}, nil
	}
}

// Update Handler() to add auth middleware
func (s *Server) Handler(secret, resourceMetadataURL string) http.Handler {
	sdkHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			auth := r.Header.Get("Authorization")
			if auth != "" {
				ctx := WithAuthHeader(r.Context(), auth)
				*r = *r.WithContext(ctx)
			}
			return s.mcpServer
		},
		nil,
	)

	// Wrap with auth middleware
	return auth.RequireBearerToken(s.tokenVerifier(secret), &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: resourceMetadataURL,
		Scopes:              []string{"mcp:tools"},
	})(sdkHandler)
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

**Step 3: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): add token verifier and auth middleware"
```

---

## Task 8: Register OAuth Routes in Echo

**Files:**
- Modify: `backend/server/echo_routes.go`

**Step 1: Add OAuth route registrations**

```go
// Add after MCP routes
// OAuth endpoints for MCP authentication.
e.GET("/.well-known/oauth-protected-resource", echo.WrapHandler(http.HandlerFunc(oauthMetadata.ProtectedResourceMetadata)))
e.GET("/.well-known/oauth-authorization-server", echo.WrapHandler(http.HandlerFunc(oauthMetadata.AuthorizationServerMetadata)))
e.GET("/oauth/authorize", echo.WrapHandler(oauthAuthorize))
e.POST("/oauth/token", echo.WrapHandler(oauthToken))
e.GET("/oauth/login", echo.WrapHandler(http.HandlerFunc(oauthLogin.ServeLogin)))
e.GET("/oauth/login/:idp", func(c echo.Context) error {
	oauthLogin.ServeLoginWithIdP(c.Response(), c.Request(), c.Param("idp"))
	return nil
})
e.GET("/oauth/callback", echo.WrapHandler(http.HandlerFunc(oauthLogin.ServeCallback)))
```

**Step 2: Update server.go to initialize OAuth handlers**

**Step 3: Verify it compiles**

```bash
go build ./backend/server/...
```

**Step 4: Commit**

```bash
but commit mcp-implementation -m "feat(mcp): register OAuth routes in Echo"
```

---

## Task 9: Add OAuth Tests

**Files:**
- Create: `backend/api/mcp/oauth/pkce_test.go`
- Create: `backend/api/mcp/oauth/store_test.go`

**Step 1: Write pkce_test.go**

```go
package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPKCE(t *testing.T) {
	// Generate a valid code_verifier
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	// Compute the challenge
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	tests := []struct {
		name        string
		verifier    string
		challenge   string
		method      string
		wantErr     bool
	}{
		{
			name:      "valid S256",
			verifier:  verifier,
			challenge: challenge,
			method:    "S256",
			wantErr:   false,
		},
		{
			name:      "invalid verifier",
			verifier:  "wrong-verifier",
			challenge: challenge,
			method:    "S256",
			wantErr:   true,
		},
		{
			name:      "unsupported method",
			verifier:  verifier,
			challenge: challenge,
			method:    "plain",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPKCE(tt.verifier, tt.challenge, tt.method)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCodeVerifier(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
		wantErr  bool
	}{
		{
			name:     "valid verifier",
			verifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantErr:  false,
		},
		{
			name:     "too short",
			verifier: "short",
			wantErr:  true,
		},
		{
			name:     "invalid characters",
			verifier: "invalid!verifier@with#special$characters%that^are&not*allowed",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCodeVerifier(tt.verifier)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```

**Step 2: Run tests**

```bash
go test -v ./backend/api/mcp/oauth/...
```

**Step 3: Commit**

```bash
but commit mcp-implementation -m "test(mcp): add OAuth PKCE and store tests"
```

---

## Task 10: Build and Integration Test

**Step 1: Build full binary**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

**Step 2: Run linter**

```bash
golangci-lint run --allow-parallel-runners ./backend/api/mcp/...
```

**Step 3: Test OAuth metadata endpoints**

```bash
curl http://localhost:8080/.well-known/oauth-protected-resource
curl http://localhost:8080/.well-known/oauth-authorization-server
```

**Step 4: Commit final changes**

```bash
but commit mcp-implementation -m "feat(mcp): complete OAuth 2.1 implementation"
```

---

## Summary

After completing all tasks:

1. **OAuth metadata endpoints** serve RFC 9728/8414 discovery documents
2. **PKCE utilities** validate S256 code challenges per RFC 7636
3. **Authorization code store** manages pending authorizations in memory
4. **Authorize endpoint** initiates OAuth flow, redirects to IdP
5. **Token endpoint** exchanges authorization codes for Bytebase JWTs
6. **Login handler** bridges OAuth flow with existing IdP integrations
7. **Auth middleware** uses SDK's `RequireBearerToken` with custom verifier
8. **Routes registered** in Echo server
9. **Tests** cover PKCE and store functionality

The MCP server now supports seamless OAuth 2.1 authentication via any configured Bytebase IdP (Google, GitHub, OIDC).
