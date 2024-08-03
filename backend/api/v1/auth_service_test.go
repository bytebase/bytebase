package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{
			domain: "www.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com.cn",
			want:   "google.com.cn",
		},
		{
			domain: "google.com",
			want:   "google.com",
		},
	}

	for _, test := range tests {
		got := extractDomain(test.domain)
		if got != test.want {
			t.Errorf("extractDomain %s, got %s, want %s", test.domain, got, test.want)
		}
	}
}

func TestHasWorkspaceAdmin(t *testing.T) {
	tests := []struct {
		policy *storepb.IamPolicy
		userID int
		want   bool
	}{
		{
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/123"},
					},
				},
			},
			userID: 123,
			want:   false,
		},
		{
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/1", "users/123"},
					},
				},
			},
			userID: 123,
			want:   false,
		},
		{
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/180", "users/123"},
					},
				},
			},
			userID: 123,
			want:   true,
		},
	}

	for _, test := range tests {
		got := hasExtraWorkspaceAdmin(test.policy, test.userID)
		if got != test.want {
			require.Equal(t, test.want, got)
		}
	}
}
