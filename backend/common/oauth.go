package common

// TokenRefresher is a function refreshes the oauth token and updates the repository.
type TokenRefresher func(token, refreshToken string, expiresTs int64) error

// OauthContext encapsulated the oauth info.
type OauthContext struct {
	AccessToken string
}

// OAuthExchange encapsulated the exchange OAuth context.
type OAuthExchange struct {
	ClientID     string
	ClientSecret string
	Code         string
	RedirectURL  string
}
