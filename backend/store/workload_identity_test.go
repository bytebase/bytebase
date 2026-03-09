package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestWorkloadIdentityMessageConfig(t *testing.T) {
	config := &storepb.WorkloadIdentityConfig{
		ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
		IssuerUrl:        "https://token.actions.githubusercontent.com",
		AllowedAudiences: []string{"https://github.com/myorg"},
		SubjectPattern:   "repo:myorg/myrepo:ref:refs/heads/main",
	}
	wi := &WorkloadIdentityMessage{
		Config: config,
	}

	require.NotNil(t, wi.Config)
	require.Equal(t, storepb.WorkloadIdentityConfig_GITHUB, wi.Config.ProviderType)
	require.Equal(t, "https://token.actions.githubusercontent.com", wi.Config.IssuerUrl)
	require.Equal(t, []string{"https://github.com/myorg"}, wi.Config.AllowedAudiences)
	require.Equal(t, "repo:myorg/myrepo:ref:refs/heads/main", wi.Config.SubjectPattern)
}

func TestCreateWorkloadIdentityMessageConfig(t *testing.T) {
	create := &CreateWorkloadIdentityMessage{
		Email: "test@workload.bytebase.com",
		Name:  "test",
		Config: &storepb.WorkloadIdentityConfig{
			ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"https://github.com/myorg"},
			SubjectPattern:   "repo:myorg/*",
		},
	}

	require.Equal(t, "test@workload.bytebase.com", create.Email)
	require.Equal(t, "test", create.Name)
	require.NotNil(t, create.Config)
	require.Equal(t, storepb.WorkloadIdentityConfig_GITHUB, create.Config.ProviderType)
}

func TestUpdateWorkloadIdentityMessageConfig(t *testing.T) {
	patch := &UpdateWorkloadIdentityMessage{
		Config: &storepb.WorkloadIdentityConfig{
			ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"https://github.com/myorg"},
			SubjectPattern:   "repo:myorg/*",
		},
	}

	require.NotNil(t, patch.Config)
	require.Equal(t, "repo:myorg/*", patch.Config.SubjectPattern)
}
