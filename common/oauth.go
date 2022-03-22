package common

// TokenRefresher is a function refreshes the oauth token and updates the repository.
type TokenRefresher func(token, refreshToken string, expiresTs int64) error

// OauthContext encapsulated the oauth info
type OauthContext struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
	Refresher    TokenRefresher
}

// ExchangeOAuth is the API message for ExchangeOAuth context
type ExchangeOAuth struct {
	Code string `jsonapi:"attr,code"`
}

// OAuthToken is the API message for OAuthToken
type OAuthToken struct {
	AccessToken  string `json:"access_token" jsonapi:"attr,accessToken"`
	RefreshToken string `json:"refresh_token" jsonapi:"attr,refreshToken"`
	ExpiresIn    int64  `json:"expires_token" jsonapi:"attr,expiresIn"`
	CreatedAt    int64  `json:"created_token" jsonapi:"attr,createdAt"`
}
