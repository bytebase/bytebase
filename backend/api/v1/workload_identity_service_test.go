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
			err := validateWorkloadIdentityConfig(
				convertToStoreWorkloadIdentityConfig(test.config))
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestConvertToStoreWorkloadIdentityConfigNormalizes(t *testing.T) {
	// Every comparison at the exchange is exact, so a padded value stored
	// verbatim is an identity that passes Create and can never authenticate.
	// The write path validates this normalized form for that reason.
	config := &v1pb.WorkloadIdentityConfig{
		ProviderType:     v1pb.WorkloadIdentityConfig_OIDC,
		IssuerUrl:        "  https://nomad.example.com  ",
		JwksUrl:          "  https://keys.example.com/jwks.json  ",
		AllowedAudiences: []string{"  bytebase  ", "  terraform  "},
		SubjectPattern:   "  nomad_job:atlantis:*  ",
	}
	stored := convertToStoreWorkloadIdentityConfig(config)
	require.NoError(t, validateWorkloadIdentityConfig(stored))
	require.Equal(t, "https://nomad.example.com", stored.IssuerUrl)
	require.Equal(t, "https://keys.example.com/jwks.json", stored.JwksUrl)
	require.Equal(t, []string{"bytebase", "terraform"}, stored.AllowedAudiences)
	require.Equal(t, "nomad_job:atlantis:*", stored.SubjectPattern)

	// A pattern that only looks bindable before normalization. Validating the
	// request instead of the stored form let these through, and the matcher
	// then refused them forever.
	for _, pattern := range []string{" * ", " repo:* "} {
		padded := convertToStoreWorkloadIdentityConfig(&v1pb.WorkloadIdentityConfig{
			ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"bytebase"},
			SubjectPattern:   pattern,
		})
		require.Error(t, validateWorkloadIdentityConfig(padded), "pattern=%q", pattern)
	}
}
