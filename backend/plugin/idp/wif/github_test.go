package wif

import (
	"context"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"
)

// acmeDeploy is the claims GitHub puts in every token for acme-corp/deploy,
// whichever subject format the repository uses.
var acmeDeploy = githubClaims{
	RepositoryOwner:   "acme-corp",
	RepositoryOwnerID: "111",
	Repository:        "acme-corp/deploy",
	RepositoryID:      "222",
}

const (
	nameSubject = "repo:acme-corp/deploy:ref:refs/heads/main"
	idSubject   = "repo:acme-corp@111/deploy@222:ref:refs/heads/main"
)

func TestMatchSubject(t *testing.T) {
	other := githubClaims{
		RepositoryOwner:   "evil-corp",
		RepositoryOwnerID: "999",
		Repository:        "evil-corp/deploy",
		RepositoryID:      "998",
	}
	renamedRepo := acmeDeploy
	renamedRepo.Repository = "acme-corp/deploy-v2"

	tests := []struct {
		name     string
		subject  string
		pattern  string
		claims   githubClaims
		expected bool
	}{
		// Unchanged: everything the prefix match accepts is still accepted.
		{"name subject, repo pattern", nameSubject, "repo:acme-corp/deploy:*", acmeDeploy, true},
		{"name subject, exact pattern", nameSubject, nameSubject, acmeDeploy, true},
		{"loose prefix still admits a sibling repository", "repo:acme-corp/deploy-test:pull_request", "repo:acme-corp/deploy*", githubClaims{}, true},
		{"custom subject template still matches by prefix", "repo:acme-corp/deploy:environment:prod:job_workflow_ref:x", "repo:acme-corp/deploy:environment:prod:*", githubClaims{}, true},
		{"id pattern, id subject", idSubject, "repo:acme-corp@111/deploy@222:*", acmeDeploy, true},

		// BYT-10349: ID-based subjects now match name patterns through the claims.
		{"id subject, owner pattern", idSubject, "repo:acme-corp/*", acmeDeploy, true},
		{"id subject, repo pattern", idSubject, "repo:acme-corp/deploy:*", acmeDeploy, true},
		{"id subject, branch pattern", idSubject, "repo:acme-corp/deploy:ref:refs/heads/main", acmeDeploy, true},
		{"id subject, other branch", idSubject, "repo:acme-corp/deploy:ref:refs/heads/dev", acmeDeploy, false},
		{"id subject, environment pattern", "repo:acme-corp@111/deploy@222:environment:prod", "repo:acme-corp/deploy:environment:*", acmeDeploy, true},
		{"id subject, owner id pattern", idSubject, "repo:acme-corp@111/deploy:*", acmeDeploy, true},
		{"name subject, owner id pattern", nameSubject, "repo:acme-corp@111/deploy:*", acmeDeploy, true},
		{"name subject, repo id pattern", nameSubject, "repo:acme-corp/deploy@222:*", acmeDeploy, true},

		// Never admits another owner, owner ID, repository or repository ID.
		{"other owner", "repo:evil-corp@999/deploy@998:ref:refs/heads/main", "repo:acme-corp/*", other, false},
		{"other owner id", idSubject, "repo:acme-corp@123/deploy:*", acmeDeploy, false},
		{"other repository", idSubject, "repo:acme-corp/deploy:*", renamedRepo, false},
		{"other repository id", idSubject, "repo:acme-corp/deploy@333:*", acmeDeploy, false},
		{"owner name compared exactly", idSubject, "repo:Acme-Corp/deploy:*", acmeDeploy, false},
		{"no claims", idSubject, "repo:acme-corp/deploy:*", githubClaims{}, false},

		// Patterns the prefix rule refuses stay refused.
		{"bare marker wildcard", idSubject, "repo:*", acmeDeploy, false},
		{"partial owner wildcard", idSubject, "repo:acme*", acmeDeploy, false},
		{"empty pattern", idSubject, "", acmeDeploy, false},

		// Patterns outside GitHub's vocabulary never read the claims.
		{"gitlab pattern", idSubject, "project_path:acme-corp/deploy:*", acmeDeploy, false},
		{"custom pattern", idSubject, "system:serviceaccount:prod:*", acmeDeploy, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{
				"repository_owner":    tc.claims.RepositoryOwner,
				"repository_owner_id": tc.claims.RepositoryOwnerID,
				"repository":          tc.claims.Repository,
				"repository_id":       tc.claims.RepositoryID,
			}
			result := matchSubject(tc.subject, tc.pattern, payload)
			require.Equal(t, tc.expected, result, "subject=%q pattern=%q", tc.subject, tc.pattern)
		})
	}
}

func TestParseGitHubSubjectPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    githubSubjectPattern
		ok      bool
	}{
		{"repo:acme-corp/*", githubSubjectPattern{owner: "acme-corp"}, true},
		{"repo:acme-corp@111/*", githubSubjectPattern{owner: "acme-corp", ownerID: "111"}, true},
		{"repo:acme-corp/deploy:*", githubSubjectPattern{owner: "acme-corp", repo: "deploy", rest: ":*"}, true},
		{"repo:acme-corp@111/deploy@222:ref:refs/heads/main", githubSubjectPattern{owner: "acme-corp", ownerID: "111", repo: "deploy", repoID: "222", rest: ":ref:refs/heads/main"}, true},
		{"repo:acme-corp/deploy*", githubSubjectPattern{}, false},
		{"repo:acme-corp/deploy", githubSubjectPattern{}, false},
		{"repo:acme-corp@abc/*", githubSubjectPattern{}, false},
		{"repo:acme*", githubSubjectPattern{}, false},
		{"repo:/deploy:*", githubSubjectPattern{}, false},
		{"project_path:acme/deploy:*", githubSubjectPattern{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			got, ok := parseGitHubSubjectPattern(tc.pattern)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestClaimStringAcceptsNumbers(t *testing.T) {
	payload := map[string]any{"repository_owner_id": float64(111), "repository_id": "222", "repository": true}
	require.Equal(t, "111", claimString(payload, "repository_owner_id"))
	require.Equal(t, "222", claimString(payload, "repository_id"))
	require.Equal(t, "", claimString(payload, "repository"))
	require.Equal(t, "", claimString(payload, "missing"))
}

// signGitHubToken mints a token the way GitHub Actions does: the subject plus
// the repository claims.
func signGitHubToken(t *testing.T, key *rsa.PrivateKey, subject string, claims githubClaims) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"))
	require.NoError(t, err)

	now := time.Now()
	registered := jwt.Claims{
		Issuer:   testIssuer,
		Subject:  subject,
		Audience: jwt.Audience{"bytebase"},
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}
	extra := map[string]any{
		"repository_owner":    claims.RepositoryOwner,
		"repository_owner_id": claims.RepositoryOwnerID,
		"repository":          claims.Repository,
		"repository_id":       claims.RepositoryID,
	}
	token, err := jwt.Signed(signer).Claims(registered).Claims(extra).Serialize()
	require.NoError(t, err)
	return token
}

// TestValidateTokenAcceptsIDBasedGitHubSubject is the exchange-level check for
// BYT-10349: an identity bound with the name pattern the form writes keeps
// working after GitHub moves the repository to ID-based subjects.
func TestValidateTokenAcceptsIDBasedGitHubSubject(t *testing.T) {
	key := seedIssuer(t)
	config := githubConfig()
	config.SubjectPattern = "repo:acme-corp/deploy:*"

	for _, subject := range []string{nameSubject, idSubject} {
		claims, err := ValidateToken(context.Background(), signGitHubToken(t, key, subject, acmeDeploy), config)
		require.NoError(t, err, "subject=%q", subject)
		require.Equal(t, subject, claims.Subject)
	}

	evil := githubClaims{RepositoryOwner: "evil-corp", RepositoryOwnerID: "999", Repository: "evil-corp/deploy", RepositoryID: "998"}
	_, err := ValidateToken(context.Background(),
		signGitHubToken(t, key, "repo:evil-corp@999/deploy@998:ref:refs/heads/main", evil), config)
	require.ErrorContains(t, err, "subject mismatch")
}
