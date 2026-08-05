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
	LoginMethod string `json:"login_method,omitempty"`
	// TokenUse is set to TokenUseMCP on OAuth2 MCP tokens and absent on web
	// session tokens. Declared here rather than on oauth2ClaimsMessage because
	// this is the type every verifier parses into.
	TokenUse string `json:"token_use,omitempty"`
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
	return GenerateMFATempTokenWithLoginMethod(userEmail, "", secret, tokenDuration)
}

// GenerateMFATempTokenWithLoginMethod generates a temporary token for MFA with the original login method.
func GenerateMFATempTokenWithLoginMethod(userEmail string, loginMethod string, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateTokenWithLoginMethod(userEmail, "", MFATempTokenAudience, expirationTime, []byte(secret), loginMethod)
}

// generateToken creates a JWT token for web authentication.
func generateToken(userEmail string, workspaceID string, aud string, expirationTime time.Time, secret []byte) (string, error) {
	return generateTokenWithLoginMethod(userEmail, workspaceID, aud, expirationTime, secret, "")
}

func generateTokenWithLoginMethod(userEmail string, workspaceID string, aud string, expirationTime time.Time, secret []byte, loginMethod string) (string, error) {
	claims := &claimsMessage{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{aud},
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   userEmail,
		},
		WorkspaceID: workspaceID,
		LoginMethod: loginMethod,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(secret)
}

// GenerateOAuth2AccessToken generates an access token for OAuth2 clients.
// The clientID is included in the token claims for audit purposes.
//
// audience is the canonical MCP resource URI stored on the grant at consent
// time. It is the caller's value on purpose: minting from the stored grant
// rather than live config means rotating the external URL invalidates
// outstanding tokens at /mcp (audience mismatch, clean 401 driving a re-auth)
// instead of quietly rebinding them to a resource the user never approved.
func GenerateOAuth2AccessToken(userEmail, clientID, workspaceID, audience, secret string, duration time.Duration) (string, error) {
	expirationTime := time.Now().Add(duration)
	return generateOAuth2Token(userEmail, clientID, workspaceID, audience, expirationTime, []byte(secret))
}

// ExpiredTokenClaims holds the claims extracted from an expired JWT.
type ExpiredTokenClaims struct {
	Subject     string
	WorkspaceID string
	Audience    []string
	TokenUse    string
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
		TokenUse:    claims.TokenUse,
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
			TokenUse:    TokenUseMCP,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(secret)
}
