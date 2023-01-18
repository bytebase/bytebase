package api

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// IdentityProviderType is the type of an identity provider.
type IdentityProviderType string

const (
	// OAuth2IdentityProvider is the identity provider type for OAuth2.
	OAuth2IdentityProvider IdentityProviderType = "OAUTH2"
	// OIDCIdentityProvider is the identity provider type for OIDC.
	OIDCIdentityProvider IdentityProviderType = "OIDC"
)

// FieldMapping uses for mapping user info from identity provider to Bytebase.
type FieldMapping struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}

// OAuth2IdentityProviderConfig is the structure for OAuth2 identity provider config.
type OAuth2IdentityProviderConfig struct {
	AuthURL      string       `json:"authUrl"`
	TokenURL     string       `json:"tokenUrl"`
	UserInfoURL  string       `json:"userInfoUrl"`
	ClientID     string       `json:"clientId"`
	ClientSecret string       `json:"clientSecret"`
	Scopes       string       `json:"scopes"`
	FieldMapping FieldMapping `json:"fieldMapping"`
}

// OIDCIdentityProviderConfig is the structure for OIDC identity provider config.
type OIDCIdentityProviderConfig struct {
	Issuer       string       `json:"issuer"`
	ClientID     string       `json:"clientId"`
	ClientSecret string       `json:"clientSecret"`
	FieldMapping FieldMapping `json:"fieldMapping"`
}

// ValidIdentityProviderConfig validates the identity provider's config is a valid JSON.
func ValidIdentityProviderConfig(identityProviderType IdentityProviderType, configString string) error {
	if identityProviderType == OAuth2IdentityProvider {
		formatedConfig := &OAuth2IdentityProviderConfig{}
		if err := json.Unmarshal([]byte(configString), formatedConfig); err != nil {
			return errors.Wrap(err, "failed to unmarshal config")
		}
	} else if identityProviderType == OIDCIdentityProvider {
		formatedConfig := &OIDCIdentityProviderConfig{}
		if err := json.Unmarshal([]byte(configString), formatedConfig); err != nil {
			return errors.Wrap(err, "failed to unmarshal config")
		}
	}
	return nil
}
