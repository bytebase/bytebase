package oidc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/bytebase/bytebase/plugin/idp"
)

// IdentityProvider represents an OIDC Identity Provider.
type IdentityProvider struct {
	provider *oidc.Provider
	config   identityProviderConfig
}

// identityProviderConfig is the configuration to be consumed by the OIDC
// Identity Provider.
type identityProviderConfig struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	FieldMapping struct {
		Identifier  string `json:"identifier"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
	} `json:"fieldMapping"`
}

// NewIdentityProvider initializes a new OIDC Identity Provider with the given
// configuration.
func NewIdentityProvider(ctx context.Context, config string) (*IdentityProvider, error) {
	var idpConfig identityProviderConfig
	err := json.Unmarshal([]byte(config), &idpConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal config")
	}

	for v, field := range map[string]string{
		idpConfig.Issuer:                  "issuer",
		idpConfig.ClientID:                "clientId",
		idpConfig.ClientSecret:            "clientSecret",
		idpConfig.FieldMapping.Identifier: "fieldMapping.identifier",
	} {
		if v == "" {
			return nil, errors.Errorf("the field %q is empty but required", field)
		}
	}

	p, err := oidc.NewProvider(ctx, idpConfig.Issuer)
	if err != nil {
		return nil, errors.Wrap(err, "create new provider")
	}
	return &IdentityProvider{
		provider: p,
		config:   idpConfig,
	}, nil
}

// ExchangeToken returns the exchanged OAuth2 token using the given redirect
// URL and authorization code.
func (p *IdentityProvider) ExchangeToken(ctx context.Context, redirectURL, code string) (*oauth2.Token, error) {
	oauth2Config := oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  redirectURL,

		// Discovery returns the OAuth2 endpoints.
		Endpoint: p.provider.Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}
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
func (p *IdentityProvider) UserInfo(ctx context.Context, token *oauth2.Token, nonce string) (*idp.UserInfo, error) {
	// Extract the ID Token from the access token, see http://openid.net/specs/openid-connect-core-1_0.html#TokenResponse.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New(`missing "id_token" from the issuer's authorization response`)
	}

	verifier := p.provider.Verifier(&oidc.Config{ClientID: p.config.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, errors.Wrap(err, "verify raw ID Token")
	}

	if idToken.Nonce != nonce {
		return nil, errors.Errorf("mismatched nonce")
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

	source, err := json.Marshal(claims)
	if err != nil {
		return nil, errors.Wrap(err, "marshal claims")
	}
	userInfo := &idp.UserInfo{Source: source}
	if v, ok := claims[p.config.FieldMapping.Identifier]; ok {
		// Both string (e.g. login) and integer (e.g. numeric ID) are valid types we
		// accept for the identifier field.
		switch id := v.(type) {
		case string:
			userInfo.Identifier = id
		case int, int32, int64:
			userInfo.Identifier = fmt.Sprintf("%d", id)
		default:
			return nil, errors.Errorf("unsupported value type of the field %q in claims, only string and integer types are expected", p.config.FieldMapping.Identifier)
		}
	}
	if userInfo.Identifier == "" {
		return nil, errors.Errorf("the field %q is not found in claims or has empty value", p.config.FieldMapping.Identifier)
	}

	// Best effort to map optional fields
	if p.config.FieldMapping.DisplayName != "" {
		if v, ok := claims[p.config.FieldMapping.DisplayName].(string); ok {
			userInfo.DisplayName = v
		}
	}
	if p.config.FieldMapping.Email != "" {
		if v, ok := claims[p.config.FieldMapping.Email].(string); ok {
			userInfo.Email = v
		}
	}
	return userInfo, nil
}
