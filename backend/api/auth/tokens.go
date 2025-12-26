package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/bytebase/bytebase/backend/common"
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
	Name string `json:"name"`
	jwt.RegisteredClaims
}

// oauth2ClaimsMessage extends claimsMessage with OAuth2-specific fields.
type oauth2ClaimsMessage struct {
	claimsMessage
	ClientID string `json:"client_id,omitempty"`
}

// GenerateAPIToken generates an API token.
func GenerateAPIToken(userEmail string, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(apiTokenDuration)
	return generateToken(userEmail, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateAccessToken generates an access token for web.
func GenerateAccessToken(userEmail string, mode common.ReleaseMode, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateMFATempToken generates a temporary token for MFA.
func GenerateMFATempToken(userEmail string, mode common.ReleaseMode, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, fmt.Sprintf(MFATempTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// generateToken creates a JWT token for web authentication.
func generateToken(userEmail string, aud string, expirationTime time.Time, secret []byte) (string, error) {
	claims := &claimsMessage{
		Name: userEmail,
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
			Name: userEmail,
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
