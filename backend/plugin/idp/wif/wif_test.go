package wif

import (
	"testing"

	"github.com/stretchr/testify/require"
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
			name:     "gitlab exact match",
			subject:  "project_path:mygroup/myproject:ref_type:branch:ref:main",
			pattern:  "project_path:mygroup/myproject:ref_type:branch:ref:main",
			expected: true,
		},
		{
			name:     "gitlab wildcard suffix matches",
			subject:  "project_path:mygroup/myproject:ref_type:branch:ref:main",
			pattern:  "project_path:mygroup/myproject:*",
			expected: true,
		},
		{
			name:     "gitlab group wildcard matches",
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := validateAudience(tc.tokenAudience, tc.allowedAudiences)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateIssuerURL(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		wantErr   bool
	}{
		{
			name:      "valid https url",
			issuerURL: "https://token.actions.githubusercontent.com",
			wantErr:   false,
		},
		{
			name:      "valid gitlab url",
			issuerURL: "https://gitlab.com",
			wantErr:   false,
		},
		{
			name:      "valid self-hosted gitlab url",
			issuerURL: "https://gitlab.example.com",
			wantErr:   false,
		},
		{
			name:      "http url rejected",
			issuerURL: "http://example.com",
			wantErr:   true,
		},
		{
			name:      "localhost rejected",
			issuerURL: "https://localhost",
			wantErr:   true,
		},
		{
			name:      "private ip 127.x rejected",
			issuerURL: "https://127.0.0.1",
			wantErr:   true,
		},
		{
			name:      "private ip 10.x rejected",
			issuerURL: "https://10.0.0.1",
			wantErr:   true,
		},
		{
			name:      "private ip 192.168.x rejected",
			issuerURL: "https://192.168.1.1",
			wantErr:   true,
		},
		{
			name:      "empty url rejected",
			issuerURL: "",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIssuerURL(tc.issuerURL)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
