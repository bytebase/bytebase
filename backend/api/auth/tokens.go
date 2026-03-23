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
	WorkspaceID string `json:"workspace_id,omitempty"`
}

// oauth2ClaimsMessage extends claimsMessage with OAuth2-specific fields.
type oauth2ClaimsMessage struct {
	claimsMessage
	ClientID string `json:"client_id,omitempty"`
}

// GenerateAPIToken generates an API token.
func GenerateAPIToken(userEmail string, workspaceID, secret string) (string, error) {
	expirationTime := time.Now().Add(apiTokenDuration)
	return generateToken(userEmail, workspaceID, AccessTokenAudience, expirationTime, []byte(secret))
}

// GenerateAccessToken generates an access token for web.
func GenerateAccessToken(userEmail string, workspaceID string, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, workspaceID, AccessTokenAudience, expirationTime, []byte(secret))
}

// GenerateMFATempToken generates a temporary token for MFA.
func GenerateMFATempToken(userEmail string, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, "", MFATempTokenAudience, expirationTime, []byte(secret))
}

// generateToken creates a JWT token for web authentication.
func generateToken(userEmail string, workspaceID string, aud string, expirationTime time.Time, secret []byte) (string, error) {
	claims := &claimsMessage{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{aud},
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   userEmail,
		},
		WorkspaceID: workspaceID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(secret)
}

// GenerateOAuth2AccessToken generates an access token for OAuth2 clients.
// The clientID is included in the token claims for audit purposes.
func GenerateOAuth2AccessToken(userEmail, clientID, workspaceID, secret string, duration time.Duration) (string, error) {
	expirationTime := time.Now().Add(duration)
	return generateOAuth2Token(userEmail, clientID, workspaceID, OAuth2AccessTokenAudience, expirationTime, []byte(secret))
}

// ExpiredTokenClaims holds the claims extracted from an expired JWT.
type ExpiredTokenClaims struct {
	Subject     string
	WorkspaceID string
	Audience    []string
}

// ExtractClaimsFromExpiredToken parses a JWT (even if expired) and returns key claims.
// Signature is still verified. Used by the Refresh endpoint to bind workspace to the session.
func ExtractClaimsFromExpiredToken(tokenString, secret string) (*ExpiredTokenClaims, error) {
	claims := &claimsMessage{}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return &ExpiredTokenClaims{
		Subject:     claims.Subject,
		WorkspaceID: claims.WorkspaceID,
		Audience:    claims.Audience,
	}, nil
}

// generateOAuth2Token creates a JWT token with OAuth2-specific claims including client_id.
func generateOAuth2Token(userEmail, clientID, workspaceID, aud string, expirationTime time.Time, secret []byte) (string, error) {
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
			WorkspaceID: workspaceID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(secret)
}
