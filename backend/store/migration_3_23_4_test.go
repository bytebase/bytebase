package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestMigration3_23_4_WorkloadIdentityAudiences(t *testing.T) {
	ctx := context.Background()
	db, _, _ := newTestDB(t)

	// workload_identity.workspace is a foreign key, and the template carries the
	// real table, so the row the fixtures reference has to exist first.
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('ws')`)
	require.NoError(t, err)

	const ghIssuer = "https://token.actions.githubusercontent.com"
	tests := []struct {
		email    string
		deleted  bool
		config   *storepb.WorkloadIdentityConfig
		expected []string // nil means the row must be left alone
		// The provider the backfill types the row with, for rows stored
		// without one. Zero value means it stays untyped.
		expectedProvider storepb.WorkloadIdentityConfig_ProviderType
	}{
		{
			email:    "legacy-github",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: ghIssuer, SubjectPattern: "repo:acme-corp/*"},
			expected: []string{"bytebase", "https://github.com/acme-corp"},
		},
		{
			// Repositories created after 2026-07-15 carry an immutable id on
			// each segment; the audience names the owner alone.
			email:    "immutable-subject",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: ghIssuer, SubjectPattern: "repo:acme-corp@123456/deploy@789:ref:refs/heads/main"},
			expected: []string{"bytebase", "https://github.com/acme-corp"},
		},
		{
			// GitHub Enterprise Server has its own issuer and its own default
			// audience, so github.com would be the wrong guess. The literal is
			// not a guess: our generator requests it on every provider.
			email:    "enterprise-server",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: "https://ghes.acme.com", SubjectPattern: "repo:acme-corp/*"},
			expected: []string{"bytebase"},
		},
		{
			email:    "gitlab",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITLAB, IssuerUrl: "https://gitlab.acme.com", SubjectPattern: "project_path:grp/*"},
			expected: []string{"bytebase", "https://gitlab.acme.com"},
		},
		{
			email:    "empty-subject",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: ghIssuer},
			expected: []string{"bytebase"},
		},
		{
			email:    "bare-wildcard-subject",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: ghIssuer, SubjectPattern: "*"},
			expected: []string{"bytebase"},
		},
		{
			email:    "wildcard-owner",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: ghIssuer, SubjectPattern: "repo:*/deploy:ref:refs/heads/main"},
			expected: []string{"bytebase"},
		},
		{
			// provider_type is optional and nothing on the token path reads it,
			// so identities created through the API often omit it. protojson
			// drops the zero enum, leaving no key at all. These rows are still
			// repairable from the issuer and the subject.
			email:            "unspecified-provider-github",
			expectedProvider: storepb.WorkloadIdentityConfig_GITHUB,
			config:           &storepb.WorkloadIdentityConfig{IssuerUrl: ghIssuer, SubjectPattern: "repo:acme-corp/*"},
			expected:         []string{"bytebase", "https://github.com/acme-corp"},
		},
		{
			email:            "unspecified-provider-gitlab",
			expectedProvider: storepb.WorkloadIdentityConfig_GITLAB,
			config:           &storepb.WorkloadIdentityConfig{IssuerUrl: "https://gitlab.acme.com", SubjectPattern: "project_path:grp/*"},
			expected:         []string{"bytebase", "https://gitlab.acme.com"},
		},
		{
			// No provider and an unreadable subject: the provider default
			// cannot be derived, but the literal still applies.
			email:    "unspecified-provider-unreadable",
			config:   &storepb.WorkloadIdentityConfig{IssuerUrl: ghIssuer, SubjectPattern: "custom:whatever"},
			expected: []string{"bytebase"},
		},
		{
			// A GitLab row whose subject is not in the project_path shape: the
			// enum alone earns the instance URL, because a GitLab issuer is its
			// instance URL whatever the subject looks like.
			email:    "gitlab-enum-only",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITLAB, IssuerUrl: "https://gitlab.acme.com", SubjectPattern: "custom:deployer"},
			expected: []string{"bytebase", "https://gitlab.acme.com"},
		},
		{
			// A row declaring GITLAB whose issuer is GitHub's: the first arm
			// reads the evidence and the enum does not override it.
			email:    "mislabelled-provider",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITLAB, IssuerUrl: ghIssuer, SubjectPattern: "repo:acme-corp/*"},
			expected: []string{"bytebase", "https://github.com/acme-corp"},
		},
		{
			// The Terraform tutorial's shape: accepted at write time, never able
			// to authenticate.
			email:  "no-issuer",
			config: &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, SubjectPattern: "repo:acme-corp/deploy:ref:refs/heads/main"},
		},
		{
			// A GitLab identity whose stored enum says GITHUB. The arms key on
			// the subject's shape, so it still gets the instance URL.
			email:    "gitlab-mislabelled-github",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: "https://gitlab.acme.com", SubjectPattern: "project_path:grp/*"},
			expected: []string{"bytebase", "https://gitlab.acme.com"},
		},
		{
			email:    "already-bound",
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: ghIssuer, SubjectPattern: "repo:acme-corp/*", AllowedAudiences: []string{"https://bytebase.acme.com/workloadIdentities/ci@workload.bytebase.com"}},
			expected: []string{"https://bytebase.acme.com/workloadIdentities/ci@workload.bytebase.com"},
		},
		{
			// Undeleting a row must not resurrect an identity that can no
			// longer authenticate.
			email:    "soft-deleted",
			deleted:  true,
			config:   &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB, IssuerUrl: ghIssuer, SubjectPattern: "repo:acme-corp/*"},
			expected: []string{"bytebase", "https://github.com/acme-corp"},
		},
	}

	for _, tc := range tests {
		// Seeded through the real serializer, so the test exercises the key
		// names the store actually writes rather than repeating a guess.
		payload, err := protojson.Marshal(tc.config)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx,
			`INSERT INTO workload_identity (name, email, workspace, deleted, config) VALUES ($1, $1, 'ws', $2, $3)`,
			tc.email, tc.deleted, string(payload))
		require.NoError(t, err)
	}

	statement, err := os.ReadFile("../migrator/migration/3.23/0004##workload_identity_audiences.sql")
	require.NoError(t, err)
	// Applied twice: the migration must be idempotent, because its own guard
	// (a missing audience list) is what stops it re-running.
	for range 2 {
		_, err = db.ExecContext(ctx, string(statement))
		require.NoError(t, err)
	}

	for _, tc := range tests {
		t.Run(tc.email, func(t *testing.T) {
			var raw string
			require.NoError(t, db.QueryRowContext(ctx,
				`SELECT config FROM workload_identity WHERE email = $1`, tc.email).Scan(&raw))
			var got storepb.WorkloadIdentityConfig
			require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(raw), &got))
			require.Equal(t, tc.expected, got.AllowedAudiences)
			if tc.config.ProviderType == storepb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED {
				// "unspecified-provider-unreadable" names no vocabulary, so it
				// stays untyped and the zero value is the assertion.
				require.Equal(t, tc.expectedProvider, got.ProviderType)
			}
		})
	}
}
