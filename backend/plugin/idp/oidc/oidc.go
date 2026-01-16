// Package oidc is the plugin for OIDC Identity Provider.
package oidc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/idp"
)

// IdentityProvider represents an OIDC Identity Provider.
type IdentityProvider struct {
	client   *http.Client
	provider *oidc.Provider
	config   *storepb.OIDCIdentityProviderConfig
}

// NewIdentityProvider initializes a new OIDC Identity Provider with the given
// configuration.
func NewIdentityProvider(ctx context.Context, config *storepb.OIDCIdentityProviderConfig) (*IdentityProvider, error) {
	for v, field := range map[string]string{
		config.Issuer:                  "issuer",
		config.ClientId:                "clientId",
		config.ClientSecret:            "clientSecret",
		config.FieldMapping.Identifier: "fieldMapping.identifier",
	} {
		if v == "" {
			return nil, errors.Errorf("the field %q is empty but required", field)
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.SkipTlsVerify,
			},
		},
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
	p, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, errors.Wrap(err, "create new provider")
	}
	return &IdentityProvider{
		client:   client,
		provider: p,
		config:   config,
	}, nil
}

// ExchangeToken returns the exchanged OAuth2 token using the given redirect
// URL and authorization code.
func (p *IdentityProvider) ExchangeToken(ctx context.Context, redirectURL, code string) (*oauth2.Token, error) {
	oauth2Config := oauth2.Config{
		ClientID:     p.config.ClientId,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  redirectURL,

		// Discovery returns the OAuth2 endpoints.
		Endpoint: p.provider.Endpoint(),
		Scopes:   p.config.Scopes,
	}

	authStyle := oauth2.AuthStyleInParams
	if p.config.AuthStyle == storepb.OAuth2AuthStyle_IN_HEADER {
		authStyle = oauth2.AuthStyleInHeader
	}
	oauth2Config.Endpoint.AuthStyle = authStyle

	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "exchange token")
	}
	return token, nil
}

// UserInfo returns the parsed user information using the given OAuth2 token.
// The nonce is used for request validation, which should be the same value as
// it was sent to the issuer as part of the Authentication Request, see
// https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest.
func (p *IdentityProvider) UserInfo(ctx context.Context, token *oauth2.Token, nonce string) (*storepb.IdentityProviderUserInfo, map[string]any, error) {
	// Extract the ID Token from the access token, see http://openid.net/specs/openid-connect-core-1_0.html#TokenResponse.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, nil, errors.New(`missing "id_token" from the issuer's authorization response`)
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	verifier := p.provider.Verifier(&oidc.Config{ClientID: p.config.ClientId})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, nil, errors.Wrap(err, "verify raw ID Token")
	}

	// Parse ID token claims - these are authoritative and verified
	var idTokenClaims map[string]any
	if err := idToken.Claims(&idTokenClaims); err != nil {
		return nil, nil, errors.Wrap(err, "parse ID token claims")
	}

	// NOTE: Skip checking nonce if the expected nonce is empty. It is OK because
	// we've given away the security benefits the nonce brings with an empty nonce,
	// and some IdP implementations are just behaving strangely that would return a
	// random nonce when we send an empty nonce to them.
	if nonce != "" && nonce != idToken.Nonce {
		return nil, nil, errors.Errorf("mismatched nonce, want %q but got %q", nonce, idToken.Nonce)
	}

	// Start with ID token claims as the base
	claims := make(map[string]any)
	for k, v := range idTokenClaims {
		claims[k] = v
	}

	// Try to fetch UserInfo to get additional/updated claims
	// But if it fails, we can still proceed with ID token claims
	rawUserInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		// Log as warning, not error, since we can proceed with ID token claims
		slog.Warn("Failed to fetch UserInfo endpoint, using ID token claims only",
			slog.String("error", err.Error()),
			slog.String("issuer", p.config.Issuer))
	} else {
		var userInfoClaims map[string]any
		err = rawUserInfo.Claims(&userInfoClaims)
		if err != nil {
			slog.Warn("Failed to parse UserInfo claims, using ID token claims only",
				slog.String("error", err.Error()))
		} else {
			// UserInfo claims override ID token claims (fresher data)
			for key, value := range userInfoClaims {
				claims[key] = value
			}
		}
	}

	// Log the merged claims for debugging
	slog.Debug("OIDC merged claims", slog.Any("claims", claims))

	userInfo := &storepb.IdentityProviderUserInfo{}
	if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Identifier).(string); ok {
		userInfo.Identifier = v
	}
	if p.config.FieldMapping.DisplayName != "" {
		if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.DisplayName).(string); ok {
			userInfo.DisplayName = v
		}
	}
	if userInfo.DisplayName == "" {
		userInfo.DisplayName = userInfo.Identifier
	}
	if p.config.FieldMapping.Phone != "" {
		if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Phone).(string); ok {
			// Only set phone if it's valid.
			if err := common.ValidatePhone(v); err == nil {
				userInfo.Phone = v
			}
		}
	}
	if p.config.FieldMapping.Groups != "" {
		if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Groups).([]any); ok {
			slog.Debug("User groups", slog.Any("groups", v))
			userInfo.HasGroups = true
			for _, group := range v {
				// Only handle string type here.
				if groupStr, ok := group.(string); ok {
					// Try to parse as JSON array if it looks like one (starts with '[')
					if strings.HasPrefix(groupStr, "[") && strings.HasSuffix(groupStr, "]") {
						var nestedGroups []string
						if err := json.Unmarshal([]byte(groupStr), &nestedGroups); err == nil {
							userInfo.Groups = append(userInfo.Groups, nestedGroups...)
						} else {
							// If JSON parsing fails, treat as regular string
							userInfo.Groups = append(userInfo.Groups, groupStr)
						}
					} else {
						userInfo.Groups = append(userInfo.Groups, groupStr)
					}
				}
			}
		}
	}
	return userInfo, claims, nil
}

// The common OIDC configuration response.
// Refer to https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata.
type OpenIDConfigurationResponse struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
}

// openidConfigResponseCache is a concurrent-safe LRU cache with TTL for OpenID Configuration responses.
var openidConfigResponseCache = expirable.NewLRU[string, *OpenIDConfigurationResponse](128, nil, 5*time.Minute)

// GetOpenIDConfiguration fetches the OpenID Configuration from the given issuer.
func GetOpenIDConfiguration(issuer string, insecureSkipVerify bool) (*OpenIDConfigurationResponse, error) {
	// Return from cache if available.
	if config, found := openidConfigResponseCache.Get(issuer); found {
		return config, nil
	}

	req, err := http.NewRequest(http.MethodGet, strings.TrimSuffix(issuer, "/")+"/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, errors.Wrap(err, "construct GET request")
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch openid configuration")
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read body")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("received non-200 response code, code: %d, body: %s", resp.StatusCode, string(b))
	}

	var config OpenIDConfigurationResponse
	if err := json.Unmarshal(b, &config); err != nil {
		return nil, errors.Wrapf(err, "unmarshal openid configuration, body: %s", string(b))
	}

	openidConfigResponseCache.Add(issuer, &config)
	return &config, nil
}
