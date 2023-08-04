// Package oidc is the plugin for OIDC Identity Provider.
package oidc

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/idp"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// IdentityProvider represents an OIDC Identity Provider.
type IdentityProvider struct {
	client   *http.Client
	provider *oidc.Provider
	config   IdentityProviderConfig
}

// IdentityProviderConfig is the configuration to be consumed by the OIDC
// Identity Provider.
type IdentityProviderConfig struct {
	Issuer        string                  `json:"issuer"`
	ClientID      string                  `json:"clientId"`
	ClientSecret  string                  `json:"clientSecret"`
	FieldMapping  *storepb.FieldMapping   `json:"fieldMapping"`
	SkipTLSVerify bool                    `json:"skipTlsVerify"`
	AuthStyle     storepb.OAuth2AuthStyle `json:"authStyle"`
}

// NewIdentityProvider initializes a new OIDC Identity Provider with the given
// configuration.
func NewIdentityProvider(ctx context.Context, config IdentityProviderConfig) (*IdentityProvider, error) {
	for v, field := range map[string]string{
		config.Issuer:                  "issuer",
		config.ClientID:                "clientId",
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
				InsecureSkipVerify: config.SkipTLSVerify,
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

// DefaultScopes is a list of scopes that are part of OIDC standard claims, see
// https://auth0.com/docs/get-started/apis/scopes/openid-connect-scopes#standard-claims.
var DefaultScopes = []string{oidc.ScopeOpenID, "profile", "email"}

// ExchangeToken returns the exchanged OAuth2 token using the given redirect
// URL and authorization code.
func (p *IdentityProvider) ExchangeToken(ctx context.Context, redirectURL, code string) (*oauth2.Token, error) {
	oauth2Config := oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  redirectURL,

		// Discovery returns the OAuth2 endpoints.
		Endpoint: p.provider.Endpoint(),
		Scopes:   DefaultScopes,
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
func (p *IdentityProvider) UserInfo(ctx context.Context, token *oauth2.Token, nonce string) (*storepb.IdentityProviderUserInfo, error) {
	// Extract the ID Token from the access token, see http://openid.net/specs/openid-connect-core-1_0.html#TokenResponse.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New(`missing "id_token" from the issuer's authorization response`)
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	verifier := p.provider.Verifier(&oidc.Config{ClientID: p.config.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, errors.Wrap(err, "verify raw ID Token")
	}

	// NOTE: Skip checking nonce if the expected nonce is empty. It is OK because
	// we've given away the security benefits the nonce brings with an empty nonce,
	// and some IdP implementations are just behaving strangely that would return a
	// random nonce when we send an empty nonce to them.
	if nonce != "" && nonce != idToken.Nonce {
		return nil, errors.Errorf("mismatched nonce, want %q but got %q", nonce, idToken.Nonce)
	}

	rawUserInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, errors.Wrap(err, "fetch user info")
	}

	var claims map[string]any
	err = rawUserInfo.Claims(&claims)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal claims")
	}
	log.Debug("User info", zap.Any("claims", claims))

	userInfo := &storepb.IdentityProviderUserInfo{}
	if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Identifier).(string); ok {
		userInfo.Identifier = v
	}
	if userInfo.Identifier == "" {
		return nil, errors.Errorf("the field %q is not found in claims or has empty value", p.config.FieldMapping.Identifier)
	}

	// Best effort to map optional fields
	if p.config.FieldMapping.DisplayName != "" {
		if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.DisplayName).(string); ok {
			userInfo.DisplayName = v
		}
	}
	if userInfo.DisplayName == "" {
		userInfo.DisplayName = userInfo.Identifier
	}
	if p.config.FieldMapping.Email != "" {
		if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Email).(string); ok {
			userInfo.Email = v
		}
	}
	if p.config.FieldMapping.Phone != "" {
		if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Phone).(string); ok {
			// Only set phone if it's valid.
			if err := common.ValidatePhone(v); err == nil {
				userInfo.Phone = v
			}
		}
	}
	return userInfo, nil
}
