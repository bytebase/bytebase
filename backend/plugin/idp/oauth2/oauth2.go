// Package oauth2 is the plugin for OAuth2 Identity Provider.
package oauth2

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/idp"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// IdentityProvider represents an OAuth2 Identity Provider.
type IdentityProvider struct {
	client *http.Client
	config *storepb.OAuth2IdentityProviderConfig
}

// NewIdentityProvider initializes a new OAuth2 Identity Provider with the given configuration.
func NewIdentityProvider(config *storepb.OAuth2IdentityProviderConfig) (*IdentityProvider, error) {
	for v, field := range map[string]string{
		config.ClientId:                "clientId",
		config.ClientSecret:            "clientSecret",
		config.TokenUrl:                "tokenUrl",
		config.UserInfoUrl:             "userInfoUrl",
		config.FieldMapping.Identifier: "fieldMapping.identifier",
	} {
		if v == "" {
			return nil, errors.Errorf(`the field "%s" is empty but required`, field)
		}
	}

	return &IdentityProvider{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: config.SkipTlsVerify,
				},
			},
		},
		config: config,
	}, nil
}

// ExchangeToken returns the exchanged OAuth2 token using the given authorization code.
func (p *IdentityProvider) ExchangeToken(ctx context.Context, redirectURL, code string) (string, error) {
	authStyle := oauth2.AuthStyleInParams
	if p.config.GetAuthStyle() == storepb.OAuth2AuthStyle_IN_HEADER {
		authStyle = oauth2.AuthStyleInHeader
	}
	conf := &oauth2.Config{
		ClientID:     p.config.ClientId,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       p.config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   p.config.AuthUrl,
			TokenURL:  p.config.TokenUrl,
			AuthStyle: authStyle,
		},
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := conf.Exchange(ctx, code)
	if err != nil {
		slog.Error("Failed to exchange access token", slog.String("code", code), log.BBError(err))
		return "", errors.Wrap(err, "failed to exchange access token")
	}

	accessToken, ok := token.Extra("access_token").(string)
	if !ok {
		slog.Error(`Missing "access_token" from authorization response`, slog.String("code", code), slog.Any("token", token))
		return "", errors.New(`missing "access_token" from authorization response`)
	}

	return accessToken, nil
}

// UserInfo returns the parsed user information using the given OAuth2 token.
func (p *IdentityProvider) UserInfo(token string) (*storepb.IdentityProviderUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, p.config.UserInfoUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to new http request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := p.client.Do(req)
	if err != nil {
		slog.Error("Failed to get user information", slog.String("token", token), log.BBError(err))
		return nil, errors.Wrap(err, "failed to get user information")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response body", slog.String("token", token), log.BBError(err))
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var claims map[string]any
	err = json.Unmarshal(body, &claims)
	if err != nil {
		slog.Error("Failed to unmarshal response body", slog.String("token", token), slog.String("body", string(body)), log.BBError(err))
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	slog.Debug("User info", slog.Any("claims", claims))

	userInfo := &storepb.IdentityProviderUserInfo{}
	if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Identifier).(string); ok {
		userInfo.Identifier = v
	}
	if userInfo.Identifier == "" {
		slog.Error("Missing identifier in response body", slog.String("token", token), slog.Any("claims", claims))
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
