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

// OAuthExchange is the API message for exchange OAuth context.
type OAuthExchange struct {
	ClientID     string
	ClientSecret string
}

// OAuthToken is the API message for OAuthToken.
type OAuthToken struct {
	AccessToken  string `json:"access_token" jsonapi:"attr,accessToken"`
	RefreshToken string `json:"refresh_token" jsonapi:"attr,refreshToken"`
	ExpiresIn    int64  `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
	// ExpiresTs is a derivative from ExpresIn and CreatedAt.
	// ExpiresTs = ExpiresIn == 0 ? 0 : CreatedAt + ExpiresIn
	ExpiresTs int64 `jsonapi:"attr,expiresTs"`
}
