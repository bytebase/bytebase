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
	store     *store.Store
	codeStore *AuthCodeStore
	issuer    string
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
		IDPContext:          state,
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

func (*AuthorizeHandler) errorRedirect(w http.ResponseWriter, _ *http.Request, redirectURI, state, errorCode, errorDesc string) {
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

	http.Redirect(w, nil, redirectURL.String(), http.StatusFound)
}
