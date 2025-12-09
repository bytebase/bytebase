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
		h.redirectToIDP(w, r, idps[0], state)
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
	_ = json.NewEncoder(w).Encode(map[string]any{
		"identity_providers": idpList,
		"message":            "Select an identity provider to continue",
	})
}

// ServeLoginWithIDP handles GET /oauth/login/{idp} - redirects to specific IdP.
func (h *LoginHandler) ServeLoginWithIDP(w http.ResponseWriter, r *http.Request, idpName string) {
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

	h.redirectToIDP(w, r, idp, state)
}

func (h *LoginHandler) redirectToIDP(w http.ResponseWriter, r *http.Request, idp *store.IdentityProviderMessage, state string) {
	callbackURL := fmt.Sprintf("%s/oauth/callback", h.issuer)

	var authURL string
	ctx := r.Context()

	switch idp.Type {
	case storepb.IdentityProviderType_OAUTH2:
		oauth2Config := idp.Config.GetOauth2Config()
		oauth2IDP, err := oauth2.NewIdentityProvider(oauth2Config)
		if err != nil {
			http.Error(w, "failed to initialize OAuth2 provider", http.StatusInternalServerError)
			return
		}
		authURL = buildOAuth2AuthURL(oauth2IDP, oauth2Config, callbackURL, state)
	case storepb.IdentityProviderType_OIDC:
		oidcConfig := idp.Config.GetOidcConfig()
		oidcIDP, err := oidc.NewIdentityProvider(ctx, oidcConfig)
		if err != nil {
			http.Error(w, "failed to initialize OIDC provider", http.StatusInternalServerError)
			return
		}
		authURL, err = buildOIDCAuthURL(oidcIDP, oidcConfig, callbackURL, state)
		if err != nil {
			http.Error(w, "failed to build OIDC auth URL", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "unsupported identity provider type", http.StatusBadRequest)
		return
	}

	// Store IdP name in pending auth
	pending, ok := h.codeStore.ConsumePending(state)
	if ok {
		pending.IDPName = idp.ResourceID
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
		ResourceID: &pending.IDPName,
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

func (*LoginHandler) exchangeForUserInfo(ctx context.Context, idp *store.IdentityProviderMessage, code, callbackURL string) (*userInfo, error) {
	switch idp.Type {
	case storepb.IdentityProviderType_OAUTH2:
		oauth2Config := idp.Config.GetOauth2Config()
		oauth2IDP, err := oauth2.NewIdentityProvider(oauth2Config)
		if err != nil {
			return nil, err
		}
		token, err := oauth2IDP.ExchangeToken(ctx, callbackURL, code)
		if err != nil {
			return nil, err
		}
		info, _, err := oauth2IDP.UserInfo(token)
		if err != nil {
			return nil, err
		}
		return &userInfo{Email: info.Identifier, Name: info.DisplayName}, nil

	case storepb.IdentityProviderType_OIDC:
		oidcConfig := idp.Config.GetOidcConfig()
		oidcIDP, err := oidc.NewIdentityProvider(ctx, oidcConfig)
		if err != nil {
			return nil, err
		}
		token, err := oidcIDP.ExchangeToken(ctx, callbackURL, code)
		if err != nil {
			return nil, err
		}
		info, _, err := oidcIDP.UserInfo(ctx, token, "")
		if err != nil {
			return nil, err
		}
		return &userInfo{Email: info.Identifier, Name: info.DisplayName}, nil

	default:
		return nil, errors.New("unsupported identity provider type")
	}
}

func (h *LoginHandler) findOrCreateUser(ctx context.Context, info *userInfo, _ *store.IdentityProviderMessage) (*store.UserMessage, error) {
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

// buildOAuth2AuthURL builds the authorization URL for OAuth2 providers.
func buildOAuth2AuthURL(_ *oauth2.IdentityProvider, config *storepb.OAuth2IdentityProviderConfig, redirectURL, state string) string {
	params := url.Values{}
	params.Set("client_id", config.ClientId)
	params.Set("redirect_uri", redirectURL)
	params.Set("response_type", "code")
	params.Set("state", state)
	if len(config.Scopes) > 0 {
		params.Set("scope", config.Scopes[0])
		for i := 1; i < len(config.Scopes); i++ {
			params.Add("scope", config.Scopes[i])
		}
	}
	return config.AuthUrl + "?" + params.Encode()
}

// buildOIDCAuthURL builds the authorization URL for OIDC providers.
func buildOIDCAuthURL(_ *oidc.IdentityProvider, config *storepb.OIDCIdentityProviderConfig, redirectURL, state string) (string, error) {
	params := url.Values{}
	params.Set("client_id", config.ClientId)
	params.Set("redirect_uri", redirectURL)
	params.Set("response_type", "code")
	params.Set("state", state)
	if len(config.Scopes) > 0 {
		scope := ""
		for i, s := range config.Scopes {
			if i > 0 {
				scope += " "
			}
			scope += s
		}
		params.Set("scope", scope)
	}

	// Get authorization endpoint from OIDC configuration
	oidcConfig, err := oidc.GetOpenIDConfiguration(config.Issuer, config.SkipTlsVerify)
	if err != nil {
		return "", err
	}
	return oidcConfig.AuthorizationEndpoint + "?" + params.Encode(), nil
}
