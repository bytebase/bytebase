// Package oidc is the plugin for OIDC Identity Provider.
package oidc

import (
	"context"
	"encoding/json"

	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// IdentityProvider represents an OIDC Identity Provider.
type IdentityProvider struct {
	provider *oidc.Provider
	config   IdentityProviderConfig
}

// IdentityProviderConfig is the configuration to be consumed by the OIDC
// Identity Provider.
type IdentityProviderConfig struct {
	Issuer       string                `json:"issuer"`
	ClientID     string                `json:"clientId"`
	ClientSecret string                `json:"clientSecret"`
	FieldMapping *storepb.FieldMapping `json:"fieldMapping"`
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

	p, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, errors.Wrap(err, "create new provider")
	}
	return &IdentityProvider{
		provider: p,
		config:   config,
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
func (p *IdentityProvider) UserInfo(ctx context.Context, token *oauth2.Token, nonce string) (*storepb.IdentityProviderUserInfo, error) {
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

	var rawClaims json.RawMessage
	err = rawUserInfo.Claims(&rawClaims)
	if err != nil {
		return nil, errors.Wrap(err, "get raw claims")
	}

	var claims map[string]any
	err = json.Unmarshal(rawClaims, &claims)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal claims")
	}

	userInfo := &storepb.IdentityProviderUserInfo{}
	if v, ok := claims[p.config.FieldMapping.Identifier].(string); ok {
		userInfo.Identifier = v
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
