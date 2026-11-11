package wif

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestValidateTokenSubject(t *testing.T) {
	const (
		issuer   = "https://token.actions.githubusercontent.com"
		audience = "bytebase"
	)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithHeader("kid", "test-key"),
	)
	require.NoError(t, err)

	setJWKSHTTPClientForTest(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("the test seeds the JWKS cache; nothing should be fetched")
	}))
	setCachedJWKS(&cachedJWKS{
		jwks: &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &key.PublicKey, KeyID: "test-key", Algorithm: string(jose.RS256), Use: "sig"},
		}},
		fetchedAt: time.Now(),
	}, "issuer:"+issuer)

	config := &storepb.WorkloadIdentityConfig{
		IssuerUrl:        issuer,
		AllowedAudiences: []string{audience},
		SubjectPattern:   "repo:my-org/*",
	}
	sign := func(t *testing.T, subject string) string {
		t.Helper()
		token, err := jwt.Signed(signer).Claims(jwt.Claims{
			Issuer:   issuer,
			Subject:  subject,
			Audience: jwt.Audience{audience},
			Expiry:   jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		}).Serialize()
		require.NoError(t, err)
		return token
	}

	t.Run("returns the issuer and the subject that matched the pattern", func(t *testing.T) {
		claims, err := ValidateToken(context.Background(), sign(t, "repo:my-org/app:ref:refs/heads/main"), config)
		require.NoError(t, err)
		require.Equal(t, issuer, claims.Issuer)
		require.Equal(t, "repo:my-org/app:ref:refs/heads/main", claims.Subject)
	})

	t.Run("refuses a subject too long to record", func(t *testing.T) {
		subject := "repo:my-org/" + strings.Repeat("a", maxSubjectLength)
		_, err := ValidateToken(context.Background(), sign(t, subject), config)
		require.ErrorContains(t, err, "longer than the 1024 allowed")
	})
}
