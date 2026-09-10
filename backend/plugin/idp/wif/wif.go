package wif

import (
	"context"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TokenClaims represents the claims from an OIDC token.
type TokenClaims struct {
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience []string `json:"aud"`
	Expiry   int64    `json:"exp"`
	IssuedAt int64    `json:"iat"`
}

// ValidateToken validates an OIDC token against a workload identity configuration.
func ValidateToken(ctx context.Context, tokenString string, config *storepb.WorkloadIdentityConfig) (*TokenClaims, error) {
	// Parse the token
	token, err := jwt.ParseSigned(tokenString, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse token")
	}

	// Get JWKS from issuer
	jwks, err := FetchJWKS(ctx, config.IssuerUrl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch JWKS")
	}

	// Use jwt.Claims which handles audience as both string and []string
	var registeredClaims jwt.Claims
	if err := token.Claims(jwks, &registeredClaims); err != nil {
		return nil, errors.Wrap(err, "failed to verify token signature")
	}

	// Convert to our TokenClaims format
	claims := &TokenClaims{
		Issuer:   registeredClaims.Issuer,
		Subject:  registeredClaims.Subject,
		Audience: registeredClaims.Audience,
	}
	if registeredClaims.Expiry != nil {
		claims.Expiry = registeredClaims.Expiry.Time().Unix()
	}
	if registeredClaims.IssuedAt != nil {
		claims.IssuedAt = registeredClaims.IssuedAt.Time().Unix()
	}

	// Validate issuer
	if claims.Issuer != config.IssuerUrl {
		return nil, errors.Errorf("issuer mismatch: expected %q, got %q", config.IssuerUrl, claims.Issuer)
	}

	// Validate audience (skip if no allowed audiences configured)
	if len(config.AllowedAudiences) > 0 && !validateAudience(claims.Audience, config.AllowedAudiences) {
		return nil, errors.Errorf("audience mismatch: token has %v, allowed %v", claims.Audience, config.AllowedAudiences)
	}

	// Validate subject pattern
	if !matchSubjectPattern(claims.Subject, config.SubjectPattern) {
		return nil, errors.Errorf("subject mismatch: expected pattern %q, got %q", config.SubjectPattern, claims.Subject)
	}

	// Validate expiry
	if time.Now().Unix() > claims.Expiry {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

func validateAudience(tokenAudience []string, allowedAudiences []string) bool {
	for _, allowed := range allowedAudiences {
		for _, aud := range tokenAudience {
			if aud == allowed {
				return true
			}
		}
	}
	return false
}

func matchSubjectPattern(subject, pattern string) bool {
	// Simple pattern matching - exact match or wildcard suffix
	if pattern == "" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(subject, prefix)
	}
	return subject == pattern
}
