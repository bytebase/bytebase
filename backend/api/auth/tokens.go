package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateOpaqueToken creates a cryptographically random opaque token.
// Returns 32 bytes encoded as base64url (43 characters).
func GenerateOpaqueToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashToken returns the SHA256 hash of a token, encoded as base64url.
// Used for secure storage of opaque tokens in the database.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// claimsMessage is the JWT claims structure for web authentication tokens.
type claimsMessage struct {
	jwt.RegisteredClaims
}

// oauth2ClaimsMessage extends claimsMessage with OAuth2-specific fields.
type oauth2ClaimsMessage struct {
	claimsMessage
	ClientID string `json:"client_id,omitempty"`
}

// GenerateAPIToken generates an API token.
func GenerateAPIToken(userEmail string, secret string) (string, error) {
	expirationTime := time.Now().Add(apiTokenDuration)
	return generateToken(userEmail, AccessTokenAudience, expirationTime, []byte(secret))
}

// GenerateAccessToken generates an access token for web.
func GenerateAccessToken(userEmail string, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, AccessTokenAudience, expirationTime, []byte(secret))
}

// GenerateMFATempToken generates a temporary token for MFA.
func GenerateMFATempToken(userEmail string, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, MFATempTokenAudience, expirationTime, []byte(secret))
}

// generateToken creates a JWT token for web authentication.
func generateToken(userEmail string, aud string, expirationTime time.Time, secret []byte) (string, error) {
	claims := &claimsMessage{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{aud},
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   userEmail,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(secret)
}

// GenerateOAuth2AccessToken generates an access token for OAuth2 clients.
// The clientID is included in the token claims for audit purposes.
func GenerateOAuth2AccessToken(userEmail, clientID, secret string, duration time.Duration) (string, error) {
	expirationTime := time.Now().Add(duration)
	return generateOAuth2Token(userEmail, clientID, OAuth2AccessTokenAudience, expirationTime, []byte(secret))
}

// generateOAuth2Token creates a JWT token with OAuth2-specific claims including client_id.
func generateOAuth2Token(userEmail, clientID, aud string, expirationTime time.Time, secret []byte) (string, error) {
	claims := &oauth2ClaimsMessage{
		ClientID: clientID,
		claimsMessage: claimsMessage{
			RegisteredClaims: jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{aud},
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    issuer,
				Subject:   userEmail,
			},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(secret)
}
