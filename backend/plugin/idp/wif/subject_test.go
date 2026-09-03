package wif

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const testIssuer = "https://token.actions.githubusercontent.com"

// seedIssuer registers a signing key for the test issuer in the JWKS cache and
// returns it, which keeps ValidateToken off the network.
func seedIssuer(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	const issuerURL = testIssuer
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwks := &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{Key: &key.PublicKey, KeyID: "test-key", Algorithm: "RS256", Use: "sig"},
	}}
	// Discovery mode (no jwks_url on the config) caches under the issuer key.
	cacheKey := "issuer:" + issuerURL
	jwksCacheLock.Lock()
	jwksCache[cacheKey] = &cachedJWKS{jwks: jwks, fetchedAt: time.Now()}
	jwksCacheLock.Unlock()

	t.Cleanup(func() {
		jwksCacheLock.Lock()
		delete(jwksCache, cacheKey)
		jwksCacheLock.Unlock()
	})
	return key
}

func signToken(t *testing.T, key *rsa.PrivateKey, subject, audience string) string {
	t.Helper()
	if audience == "" {
		return signTokenWithAudience(t, key, subject, nil)
	}
	return signTokenWithAudience(t, key, subject, jwt.Audience{audience})
}

// signTokenWithAudience sets the aud claim verbatim, so a test can mint the
// `"aud": ""` a legacy row with a blank entry used to match.
func signTokenWithAudience(t *testing.T, key *rsa.PrivateKey, subject string, audience jwt.Audience) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"))
	require.NoError(t, err)

	now := time.Now()
	claims := jwt.Claims{
		Issuer:   testIssuer,
		Subject:  subject,
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}
	claims.Audience = audience
	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	require.NoError(t, err)
	return token
}

func githubConfig() *storepb.WorkloadIdentityConfig {
	return &storepb.WorkloadIdentityConfig{
		ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
		IssuerUrl:        testIssuer,
		AllowedAudiences: []string{"bytebase"},
		SubjectPattern:   "repo:acme-corp/deploy:ref:refs/heads/main",
	}
}

// TestValidateTokenFailsClosedOnUnbindableConfig is the regression for
// BYT-10151, at the caller boundary rather than the helper: validateAudience
// already returned false for an empty allowlist, and ValidateToken's own guard
// short-circuited past it, so a helper-level test passed while the exchange
// stayed open. Every token below is exchanged successfully before this change.
func TestValidateTokenFailsClosedOnUnbindableConfig(t *testing.T) {
	key := seedIssuer(t)
	subject := "repo:acme-corp/deploy:ref:refs/heads/main"

	tests := []struct {
		name   string
		mutate func(*storepb.WorkloadIdentityConfig)
		want   string
	}{
		{"no audiences", func(c *storepb.WorkloadIdentityConfig) { c.AllowedAudiences = nil }, "audience mismatch"},
		{"empty audience list", func(c *storepb.WorkloadIdentityConfig) { c.AllowedAudiences = []string{} }, "audience mismatch"},
		// A row stored before the write paths refused blank entries: go-jose
		// reads `"aud": ""` as one empty string, which used to match it.
		{"blank audience entry", func(c *storepb.WorkloadIdentityConfig) { c.AllowedAudiences = []string{""} }, "audience mismatch"},
		{"whitespace audience entry", func(c *storepb.WorkloadIdentityConfig) { c.AllowedAudiences = []string{"  "} }, "audience mismatch"},
		{"empty subject pattern", func(c *storepb.WorkloadIdentityConfig) { c.SubjectPattern = "" }, "subject mismatch"},
		{"bare wildcard subject pattern", func(c *storepb.WorkloadIdentityConfig) { c.SubjectPattern = "*" }, "subject mismatch"},
		// The migration gives these rows the requestable "bytebase" audience,
		// so the subject is the only thing standing between a broad prefix and
		// every workflow the issuer signs for.
		{"wildcard over every repository", func(c *storepb.WorkloadIdentityConfig) { c.SubjectPattern = "repo:*" }, "subject mismatch"},
		{"wildcard over a partial owner", func(c *storepb.WorkloadIdentityConfig) { c.SubjectPattern = "repo:acme*" }, "subject mismatch"},
		{"wildcard inside the vocabulary marker", func(c *storepb.WorkloadIdentityConfig) { c.SubjectPattern = "r*" }, "subject mismatch"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := githubConfig()
			// A validly signed token from the configured issuer, matching both
			// the subject and the audience the identity was meant to carry:
			// everything except the configuration is correct.
			signed := signToken(t, key, subject, "bytebase")
			tc.mutate(config)
			if len(config.AllowedAudiences) == 1 && strings.TrimSpace(config.AllowedAudiences[0]) == "" {
				// The token has to carry the same blank audience for the row to
				// have matched before this change.
				signed = signTokenWithAudience(t, key, subject,
					jwt.Audience{config.AllowedAudiences[0]})
			}

			_, err := ValidateToken(context.Background(), signed, config)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// TestValidateTokenAudienceBinding is the control: a complete configuration
// accepts only the audience it names.
func TestValidateTokenAudienceBinding(t *testing.T) {
	key := seedIssuer(t)
	config := githubConfig()
	config.AllowedAudiences = []string{"https://bytebase.acme.com/workloadIdentities/ci"}
	subject := "repo:acme-corp/deploy:ref:refs/heads/main"

	claims, err := ValidateToken(context.Background(),
		signToken(t, key, subject, config.AllowedAudiences[0]), config)
	require.NoError(t, err)
	require.Equal(t, subject, claims.Subject)

	for _, wrong := range []string{
		"https://some-other-saas.example.com", // minted for another vendor
		"sts.amazonaws.com",
		"bytebase",                     // what the in-product generator requests
		"https://github.com/acme-corp", // GitHub's default audience
	} {
		_, err := ValidateToken(context.Background(), signToken(t, key, subject, wrong), config)
		require.ErrorContains(t, err, "audience mismatch", "audience %q must not authenticate", wrong)
	}
}

// TestValidateTokenRefusesLegacyPartialMarker is the exchange-level regression
// for the population this PR is about: a row written before provider_type was
// required, bound with a wildcard that stops inside GitHub's subject marker.
// The migration hands such a row the literal "bytebase", which any GitHub
// workflow can request, so the subject is the only thing standing between it
// and every repository the issuer signs for.
func TestValidateTokenRefusesLegacyPartialMarker(t *testing.T) {
	key := seedIssuer(t)

	for _, pattern := range []string{"r*", "repo*", "repo:*", "repo:acme*"} {
		config := githubConfig()
		config.ProviderType = storepb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED
		config.SubjectPattern = pattern

		_, err := ValidateToken(context.Background(),
			signToken(t, key, "repo:attacker-org/evil:ref:refs/heads/main", "bytebase"),
			config)
		require.ErrorContains(t, err, "subject mismatch", "pattern=%q", pattern)
	}

	// The same shapes stay legal on an issuer whose subjects are not GitHub's.
	oidc := githubConfig()
	oidc.ProviderType = storepb.WorkloadIdentityConfig_OIDC
	oidc.SubjectPattern = "r*"
	claims, err := ValidateToken(context.Background(),
		signToken(t, key, "role:admin", "bytebase"), oidc)
	require.NoError(t, err)
	require.Equal(t, "role:admin", claims.Subject)
}

// TestValidateTokenUnspecifiedProviderStillBinds pins the upgrade path for
// identities stored before provider_type was required. Nothing on the token
// path reads that field, so a configuration that is otherwise bound keeps
// working.
func TestValidateTokenUnspecifiedProviderStillBinds(t *testing.T) {
	key := seedIssuer(t)
	config := githubConfig()
	config.ProviderType = storepb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED
	subject := "repo:acme-corp/deploy:ref:refs/heads/main"

	claims, err := ValidateToken(context.Background(), signToken(t, key, subject, "bytebase"), config)
	require.NoError(t, err)
	require.Equal(t, subject, claims.Subject)
}

func TestValidateSubjectPattern(t *testing.T) {
	const github = storepb.WorkloadIdentityConfig_GITHUB
	accepted := []string{
		"repo:acme-corp/deploy:ref:refs/heads/main",
		"repo:acme-corp/*",
		"repo:acme-corp/deploy:*",
		"project_path:grp/*",
		"project_path:grp/proj:ref_type:branch:ref:main",
		// issuer_url is free-form, so a wildcard outside the two vocabularies
		// we model is the operator's call.
		"system:serviceaccount:prod:*",
		"organization:acme:project:prod:workspace:db-prod:*",
		"svc-bytebase-*",
	}
	for _, pattern := range accepted {
		require.NoError(t, ValidateSubjectPattern(github, pattern), "pattern=%q", pattern)
	}

	rejected := map[string]string{
		"":                  "subject_pattern is required",
		"   ":               "subject_pattern is required",
		"*":                 "matches every subject",
		"repo:*":            "matches every repository",
		"repo:acme*":        "matches every repository",
		"r*":                "matches every repository",
		"repo*":             "matches every repository",
		"project_path:*":    "matches every project",
		"project_path:grp*": "matches every project",
	}
	for pattern, want := range rejected {
		require.ErrorContains(t, ValidateSubjectPattern(github, pattern), want, "pattern=%q", pattern)
	}

	// A prefix that stops inside a marker belongs to that vocabulary only if
	// the row could be using it. A declared generic issuer could not.
	for _, pattern := range []string{"r*", "repo*", "project*", "project_path*"} {
		require.NoError(t, ValidateSubjectPattern(
			storepb.WorkloadIdentityConfig_OIDC, pattern), "pattern=%q", pattern)
	}
	// A row that declares no provider predates the requirement, so it is
	// judged against both markers: it is the population the migration repairs.
	for _, pattern := range []string{"r*", "repo*", "project*", "project_path*"} {
		require.ErrorContains(t, ValidateSubjectPattern(
			storepb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED, pattern),
			"matches every", "pattern=%q", pattern)
	}
	require.ErrorContains(t, ValidateSubjectPattern(
		storepb.WorkloadIdentityConfig_GITLAB, "project*"), "matches every project")
	// A full marker names its own vocabulary whatever the provider says.
	require.ErrorContains(t, ValidateSubjectPattern(
		storepb.WorkloadIdentityConfig_OIDC, "repo:*"), "matches every repository")
	// A wildcard outside both vocabularies stays the operator's call, even on
	// a row that declares nothing.
	for _, pattern := range []string{"system:serviceaccount:prod:*", "svc-bytebase-*"} {
		require.NoError(t, ValidateSubjectPattern(
			storepb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED, pattern), "pattern=%q", pattern)
	}
}
