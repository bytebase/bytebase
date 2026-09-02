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
			name: "github compatibility",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType: v1pb.WorkloadIdentityConfig_GITHUB,
			},
		},
		{
			name: "gitlab compatibility",
			config: &v1pb.WorkloadIdentityConfig{
				ProviderType: v1pb.WorkloadIdentityConfig_GITLAB,
			},
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
