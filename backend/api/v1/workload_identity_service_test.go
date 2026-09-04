package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestValidateWorkloadIdentityConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *v1pb.WorkloadIdentityConfig
		wantErr string
	}{
		{
			name: "nil config",
		},
		{
			name: "github bound",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "repo:acme-corp/deploy:ref:refs/heads/main",
			},
		},
		{
			name: "gitlab bound",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITLAB,
				IssuerUrl:        "https://gitlab.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "project_path:grp/proj:*",
			},
		},
		// BYT-10151: a GitHub or GitLab identity used to be accepted with no
		// binding at all, and the exchange skipped the audience check when the
		// list was empty. Every provider reaches the same exchange, so every
		// provider carries the same requirements.
		{
			name: "github without an audience",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:   v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:      "https://token.actions.githubusercontent.com",
				SubjectPattern: "repo:acme-corp/deploy:ref:refs/heads/main",
			},
			wantErr: "allowed_audiences is required",
		},
		{
			name: "github without a subject pattern",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
			},
			wantErr: "subject_pattern is required",
		},
		{
			name: "github without an issuer",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "repo:acme-corp/deploy:ref:refs/heads/main",
			},
			wantErr: "issuer_url is required",
		},
		// A trailing "*" is a prefix test, so each of these admits every
		// repository the issuer signs for, exactly as a bare "*" does.
		{
			name: "subject matching every subject",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "*",
			},
			wantErr: "matches every subject",
		},
		{
			name: "subject matching every repository",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "repo:*",
			},
			wantErr: "matches every repository",
		},
		{
			name: "subject matching a partial owner",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "repo:acme*",
			},
			wantErr: "matches every repository",
		},
		{
			// The prefix stops inside the vocabulary marker, so it is that
			// whole vocabulary.
			name: "subject wildcard inside the marker",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "r*",
			},
			wantErr: "matches every repository",
		},
		{
			name: "subject matching every gitlab project",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITLAB,
				IssuerUrl:        "https://gitlab.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "project_path:*",
			},
			wantErr: "matches every project",
		},
		{
			// The partial-marker rule reads the declared provider: a generic
			// OIDC issuer may sign "role:admin", so "r*" there is not a
			// GitHub pattern.
			name: "oidc partial-marker wildcard stays legal",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "https://nomad.example.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "r*",
			},
		},
		{
			// issuer_url is free-form, so a wildcard carrying no "/" is the
			// operator's call outside the two vocabularies we model.
			name: "oidc namespace wildcard stays legal",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "https://oidc.eks.us-east-1.amazonaws.com/id/9F8E",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "system:serviceaccount:prod:*",
			},
		},
		{
			name: "github audience with a blank entry",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase", "  "},
				SubjectPattern:   "repo:acme-corp/deploy:ref:refs/heads/main",
			},
			wantErr: "allowed_audiences must not contain an empty value",
		},
		{
			name: "oidc valid",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "https://nomad.example.com",
				JwksUrl:          "https://keys.example.com/jwks.json",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "nomad_job:atlantis:*",
			},
		},
		{
			name: "oidc discovery valid",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "https://nomad.example.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "nomad_job:atlantis:*",
			},
		},
		{
			name: "oidc invalid issuer URL",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "not-a-url",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "nomad_job:atlantis:*",
			},
			wantErr: "issuer URL must use HTTPS",
		},
		{
			name: "oidc invalid JWKS URL",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "https://nomad.example.com",
				JwksUrl:          "http://keys.example.com/jwks.json",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "nomad_job:atlantis:*",
			},
			wantErr: "JWKS URL must use HTTPS",
		},
		{
			name: "oidc missing issuer",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "nomad_job:atlantis:*",
			},
			wantErr: "issuer_url is required",
		},
		{
			name: "oidc missing audience",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:   v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:      "https://nomad.example.com",
				SubjectPattern: "nomad_job:atlantis:*",
			},
			wantErr: "allowed_audiences is required",
		},
		{
			name: "oidc empty audience",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "https://nomad.example.com",
				AllowedAudiences: []string{""},
				SubjectPattern:   "nomad_job:atlantis:*",
			},
			wantErr: "allowed_audiences must not contain an empty value",
		},
		{
			name: "oidc missing subject",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
				IssuerUrl:        "https://nomad.example.com",
				AllowedAudiences: []string{"bytebase"},
			},
			wantErr: "subject_pattern is required",
		},
		{
			name: "provider unspecified",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType: v1pb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED,
			},
			wantErr: "provider_type is required",
		},
		{
			name: "provider unknown",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType: v1pb.WorkloadIdentityConfig_ProviderType(99),
			},
			wantErr: "provider_type is invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateWorkloadIdentityConfig(test.config)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestConvertToStoreWorkloadIdentityConfigNormalizesEveryProvider(t *testing.T) {
	// validateWorkloadIdentityConfig checks the trimmed value, and every
	// comparison at the exchange is exact, so a padded value stored verbatim
	// is an identity that passes Create and can never authenticate.
	stored := convertToStoreWorkloadIdentityConfig(&v1pb.WorkloadIdentityConfig{
		ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
		IssuerUrl:        "  https://token.actions.githubusercontent.com  ",
		AllowedAudiences: []string{" bytebase "},
		SubjectPattern:   " repo:acme-corp/deploy:ref:refs/heads/main ",
	})
	require.Equal(t, "https://token.actions.githubusercontent.com", stored.IssuerUrl)
	require.Equal(t, []string{"bytebase"}, stored.AllowedAudiences)
	require.Equal(t, "repo:acme-corp/deploy:ref:refs/heads/main", stored.SubjectPattern)
}

func TestConvertToStoreWorkloadIdentityConfigNormalizesOIDCValues(t *testing.T) {
	config := &v1pb.WorkloadIdentityConfig{
		ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
		IssuerUrl:        "  https://nomad.example.com  ",
		JwksUrl:          "  https://keys.example.com/jwks.json  ",
		AllowedAudiences: []string{"  bytebase  ", "  terraform  "},
		SubjectPattern:   "  nomad_job:atlantis:*  ",
	}
	require.NoError(t, validateWorkloadIdentityConfig(config))

	converted := convertToStoreWorkloadIdentityConfig(config)

	require.Equal(t, "https://nomad.example.com", converted.IssuerUrl)
	require.Equal(t, "https://keys.example.com/jwks.json", converted.JwksUrl)
	require.Equal(t, []string{"bytebase", "terraform"}, converted.AllowedAudiences)
	require.Equal(t, "nomad_job:atlantis:*", converted.SubjectPattern)
}
