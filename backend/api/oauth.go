package api

// OAuthToken is the API message for OAuthToken.
type OAuthToken struct {
	AccessToken  string `jsonapi:"attr,accessToken" `
	RefreshToken string `jsonapi:"attr,refreshToken"`
	ExpiresTs    int64  `jsonapi:"attr,expiresAt"`
}
