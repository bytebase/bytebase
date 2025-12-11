package oauth2

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type authorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

func (s *Service) handleDiscovery(c echo.Context) error {
	metadata := &authorizationServerMetadata{
		Issuer:                            s.issuer(),
		AuthorizationEndpoint:             s.authorizationEndpoint(),
		TokenEndpoint:                     s.tokenEndpoint(),
		RegistrationEndpoint:              s.registrationEndpoint(),
		RevocationEndpoint:                s.revocationEndpoint(),
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
	}
	return c.JSON(http.StatusOK, metadata)
}
