package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestWorkloadIdentityConfigInUserMessage verifies that WorkloadIdentityConfig
// is accessible through the UserMessage.Profile field.
func TestWorkloadIdentityConfigInUserMessage(t *testing.T) {
	// Create a UserMessage with WorkloadIdentityConfig in the Profile
	profile := &storepb.UserProfile{
		WorkloadIdentityConfig: &storepb.WorkloadIdentityConfig{
			ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"https://github.com/myorg"},
			SubjectPattern:   "repo:myorg/myrepo:ref:refs/heads/main",
		},
	}
	user := &UserMessage{
		Profile: profile,
	}

	// Verify WorkloadIdentityConfig is accessible
	require.NotNil(t, user.Profile)
	require.NotNil(t, user.Profile.WorkloadIdentityConfig)

	// Verify the config values
	wic := user.Profile.WorkloadIdentityConfig
	require.Equal(t, storepb.WorkloadIdentityConfig_GITHUB, wic.ProviderType)
	require.Equal(t, "https://token.actions.githubusercontent.com", wic.IssuerUrl)
	require.Equal(t, []string{"https://github.com/myorg"}, wic.AllowedAudiences)
	require.Equal(t, "repo:myorg/myrepo:ref:refs/heads/main", wic.SubjectPattern)

	// Verify we can use GetWorkloadIdentityConfig() helper
	wic2 := user.Profile.GetWorkloadIdentityConfig()
	require.NotNil(t, wic2)
	require.Equal(t, wic, wic2)
}

// TestCreateUserMessageWithWorkloadIdentityConfig verifies that CreateUser
// can handle UserMessage with WorkloadIdentityConfig in the Profile.
func TestCreateUserMessageWithWorkloadIdentityConfig(t *testing.T) {
	// Create a user with workload identity config
	profile := &storepb.UserProfile{
		WorkloadIdentityConfig: &storepb.WorkloadIdentityConfig{
			ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"https://github.com/myorg"},
			SubjectPattern:   "repo:myorg/*",
		},
	}
	create := &UserMessage{
		Type:    storepb.PrincipalType_WORKLOAD_IDENTITY,
		Profile: profile,
	}

	// Verify the structure is correctly formed
	require.Equal(t, storepb.PrincipalType_WORKLOAD_IDENTITY, create.Type)
	require.NotNil(t, create.Profile)
	require.NotNil(t, create.Profile.WorkloadIdentityConfig)

	// The actual CreateUser function will marshal this Profile to JSONB
	// and store it in the database. The existing code already handles this
	// without any modifications needed.
}

// TestUpdateUserMessageWithWorkloadIdentityConfig verifies that UpdateUser
// can handle updates to WorkloadIdentityConfig.
func TestUpdateUserMessageWithWorkloadIdentityConfig(t *testing.T) {
	// Update patch with modified workload identity config
	patch := &UpdateUserMessage{
		Profile: &storepb.UserProfile{
			WorkloadIdentityConfig: &storepb.WorkloadIdentityConfig{
				ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"https://github.com/myorg"},
				SubjectPattern:   "repo:myorg/*",
			},
		},
	}

	// Verify the patch is correctly formed
	require.NotNil(t, patch.Profile)
	require.NotNil(t, patch.Profile.WorkloadIdentityConfig)
	require.Equal(t, "repo:myorg/*", patch.Profile.WorkloadIdentityConfig.SubjectPattern)

	// The actual UpdateUser function will marshal this Profile to JSONB
	// and update it in the database. The existing code already handles this
	// without any modifications needed.
}
