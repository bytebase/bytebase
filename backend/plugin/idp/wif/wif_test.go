package wif

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestMatchSubjectPattern(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		pattern  string
		expected bool
	}{
		{
			name:     "empty pattern matches any subject",
			subject:  "repo:owner/repo:ref:refs/heads/main",
			pattern:  "",
			expected: true,
		},
		{
			name:     "exact match",
			subject:  "repo:owner/repo:ref:refs/heads/main",
			pattern:  "repo:owner/repo:ref:refs/heads/main",
			expected: true,
		},
		{
			name:     "exact match failure",
			subject:  "repo:owner/repo:ref:refs/heads/main",
			pattern:  "repo:owner/repo:ref:refs/heads/develop",
			expected: false,
		},
		{
			name:     "wildcard suffix matches",
			subject:  "repo:owner/repo:ref:refs/heads/main",
			pattern:  "repo:owner/repo:*",
			expected: true,
		},
		{
			name:     "wildcard suffix matches prefix",
			subject:  "repo:owner/repo:ref:refs/heads/main",
			pattern:  "repo:owner/*",
			expected: true,
		},
		{
			name:     "wildcard suffix does not match different prefix",
			subject:  "repo:owner/repo:ref:refs/heads/main",
			pattern:  "repo:other-owner/*",
			expected: false,
		},
		{
			name:     "gitlab project path pattern",
			subject:  "project_path:mygroup/myproject:ref_type:branch:ref:main",
			pattern:  "project_path:mygroup/myproject:*",
			expected: true,
		},
		{
			name:     "gitlab group wildcard",
			subject:  "project_path:mygroup/myproject:ref_type:branch:ref:main",
			pattern:  "project_path:mygroup/*",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchSubjectPattern(tc.subject, tc.pattern)
			require.Equal(t, tc.expected, result, "subject=%q pattern=%q", tc.subject, tc.pattern)
		})
	}
}

func TestValidateAudience(t *testing.T) {
	tests := []struct {
		name             string
		tokenAudience    []string
		allowedAudiences []string
		expected         bool
	}{
		{
			name:             "single audience match",
			tokenAudience:    []string{"https://github.com/owner"},
			allowedAudiences: []string{"https://github.com/owner"},
			expected:         true,
		},
		{
			name:             "multiple token audiences, one matches",
			tokenAudience:    []string{"aud1", "aud2", "aud3"},
			allowedAudiences: []string{"aud2"},
			expected:         true,
		},
		{
			name:             "multiple allowed audiences, one matches",
			tokenAudience:    []string{"aud2"},
			allowedAudiences: []string{"aud1", "aud2", "aud3"},
			expected:         true,
		},
		{
			name:             "no match",
			tokenAudience:    []string{"aud1"},
			allowedAudiences: []string{"aud2"},
			expected:         false,
		},
		{
			name:             "empty token audience",
			tokenAudience:    []string{},
			allowedAudiences: []string{"aud1"},
			expected:         false,
		},
		{
			name:             "empty allowed audiences",
			tokenAudience:    []string{"aud1"},
			allowedAudiences: []string{},
			expected:         false,
		},
		{
			name:             "gitlab audience",
			tokenAudience:    []string{"https://gitlab.com"},
			allowedAudiences: []string{"https://gitlab.com"},
			expected:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := validateAudience(tc.tokenAudience, tc.allowedAudiences)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildSubjectPattern(t *testing.T) {
	tests := []struct {
		name         string
		providerType storepb.ProviderType
		owner        string
		repo         string
		branch       string
		expected     string
	}{
		{
			name:         "github with all fields",
			providerType: storepb.ProviderType_PROVIDER_GITHUB,
			owner:        "myorg",
			repo:         "myrepo",
			branch:       "main",
			expected:     "repo:myorg/myrepo:ref:refs/heads/main",
		},
		{
			name:         "github without branch",
			providerType: storepb.ProviderType_PROVIDER_GITHUB,
			owner:        "myorg",
			repo:         "myrepo",
			branch:       "",
			expected:     "repo:myorg/myrepo:*",
		},
		{
			name:         "github without repo",
			providerType: storepb.ProviderType_PROVIDER_GITHUB,
			owner:        "myorg",
			repo:         "",
			branch:       "",
			expected:     "repo:myorg/*",
		},
		{
			name:         "gitlab with all fields",
			providerType: storepb.ProviderType_PROVIDER_GITLAB,
			owner:        "mygroup",
			repo:         "myproject",
			branch:       "develop",
			expected:     "project_path:mygroup/myproject:ref_type:branch:ref:develop",
		},
		{
			name:         "gitlab without branch",
			providerType: storepb.ProviderType_PROVIDER_GITLAB,
			owner:        "mygroup",
			repo:         "myproject",
			branch:       "",
			expected:     "project_path:mygroup/myproject:*",
		},
		{
			name:         "gitlab without repo",
			providerType: storepb.ProviderType_PROVIDER_GITLAB,
			owner:        "mygroup",
			repo:         "",
			branch:       "",
			expected:     "project_path:mygroup/*",
		},
		{
			name:         "unsupported provider",
			providerType: storepb.ProviderType_PROVIDER_TYPE_UNSPECIFIED,
			owner:        "owner",
			repo:         "repo",
			branch:       "main",
			expected:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildSubjectPattern(tc.providerType, tc.owner, tc.repo, tc.branch)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestGetPlatformPreset(t *testing.T) {
	tests := []struct {
		name         string
		providerType storepb.ProviderType
		expectNil    bool
		issuerURL    string
	}{
		{
			name:         "github preset",
			providerType: storepb.ProviderType_PROVIDER_GITHUB,
			expectNil:    false,
			issuerURL:    "https://token.actions.githubusercontent.com",
		},
		{
			name:         "gitlab preset",
			providerType: storepb.ProviderType_PROVIDER_GITLAB,
			expectNil:    false,
			issuerURL:    "https://gitlab.com",
		},
		{
			name:         "bitbucket preset",
			providerType: storepb.ProviderType_PROVIDER_BITBUCKET,
			expectNil:    false,
			issuerURL:    "https://api.bitbucket.org/2.0/workspaces/%s/pipelines-config/identity/oidc",
		},
		{
			name:         "azure devops preset",
			providerType: storepb.ProviderType_PROVIDER_AZURE_DEVOPS,
			expectNil:    false,
			issuerURL:    "https://vstoken.dev.azure.com/%s",
		},
		{
			name:         "unspecified provider returns nil",
			providerType: storepb.ProviderType_PROVIDER_TYPE_UNSPECIFIED,
			expectNil:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preset := GetPlatformPreset(tc.providerType)
			if tc.expectNil {
				require.Nil(t, preset)
			} else {
				require.NotNil(t, preset)
				require.Equal(t, tc.issuerURL, preset.IssuerURL)
			}
		})
	}
}
