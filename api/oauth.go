package api

// OAuthToken is the API message for OAuthToken.
type OAuthToken struct {
	AccessToken  string `jsonapi:"attr,accessToken" `
	RefreshToken string `jsonapi:"attr,refreshToken"`
	ExpiresTs    int64  `jsonapi:"attr,expiresAt"`
}

// OAuthConfig is the API message for OAuthConfig.
type OAuthConfig struct {
	Endpoint      string `jsonapi:"attr,endpoint"`
	ApplicationID string `jsonapi:"attr,applicationId"`
	Secret        string `jsonapi:"attr,secret"`
	RedirectURL   string `jsonapi:"attr,redirectUrl"`
}

// ExchangeToken is the API message for exchanging token.
type ExchangeToken struct {
	ID     int         `jsonapi:"attr,vcsId"`
	Code   string      `jsonapi:"attr,code"`
	Config OAuthConfig `jsonapi:"relation,config"`
}
