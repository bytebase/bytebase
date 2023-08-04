// Package oauth2 is the plugin for OAuth2 Identity Provider.
package oauth2

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/zap"
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
		log.Error("Failed to exchange access token", zap.String("code", code), zap.Error(err))
		return "", errors.Wrap(err, "failed to exchange access token")
	}

	accessToken, ok := token.Extra("access_token").(string)
	if !ok {
		log.Error(`Missing "access_token" from authorization response`, zap.String("code", code), zap.Any("token", token))
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
		log.Error("Failed to get user information", zap.String("token", token), zap.Error(err))
		return nil, errors.Wrap(err, "failed to get user information")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", zap.String("token", token), zap.Error(err))
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var claims map[string]any
	err = json.Unmarshal(body, &claims)
	if err != nil {
		log.Error("Failed to unmarshal response body", zap.String("token", token), zap.String("body", string(body)), zap.Error(err))
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	log.Debug("User info", zap.Any("claims", claims))

	userInfo := &storepb.IdentityProviderUserInfo{}
	if v, ok := idp.GetValueWithKey(claims, p.config.FieldMapping.Identifier).(string); ok {
		userInfo.Identifier = v
	}
	if userInfo.Identifier == "" {
		log.Error("Missing identifier in response body", zap.String("token", token), zap.Any("claims", claims), zap.Error(err))
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
