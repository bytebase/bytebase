package oauth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

// TokenHandler handles the /oauth/token endpoint.
type TokenHandler struct {
	store     *store.Store
	codeStore *AuthCodeStore
	secret    string
	mode      common.ReleaseMode
}

// NewTokenHandler creates a new token handler.
func NewTokenHandler(store *store.Store, codeStore *AuthCodeStore, secret string, mode common.ReleaseMode) *TokenHandler {
	return &TokenHandler{
		store:     store,
		codeStore: codeStore,
		secret:    secret,
		mode:      mode,
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
		h.mode,
		h.secret,
		expiresIn,
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
	_ = json.NewEncoder(w).Encode(response)
}

func (*TokenHandler) errorResponse(w http.ResponseWriter, status int, errorCode, errorDesc string) {
	response := &TokenErrorResponse{
		Error:            errorCode,
		ErrorDescription: errorDesc,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
