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

	// Verify signature and extract claims
	var claims TokenClaims
	if err := token.Claims(jwks, &claims); err != nil {
		return nil, errors.Wrap(err, "failed to verify token signature")
	}

	// Validate issuer
	if claims.Issuer != config.IssuerUrl {
		return nil, errors.Errorf("issuer mismatch: expected %q, got %q", config.IssuerUrl, claims.Issuer)
	}

	// Validate audience
	if !validateAudience(claims.Audience, config.AllowedAudiences) {
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

	return &claims, nil
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
