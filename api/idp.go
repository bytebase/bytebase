package api

// IdentityProviderType is the type of an identity provider.
type IdentityProviderType string

const (
	// OAuth2IdentityProvider is the identity provider type for OAuth2.
	OAuth2IdentityProvider IdentityProviderType = "OAUTH2"
	// OIDCIdentityProvider is the identity provider type for OIDC.
	OIDCIdentityProvider IdentityProviderType = "OIDC"
)
