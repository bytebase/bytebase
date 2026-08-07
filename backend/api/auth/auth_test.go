package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/permission"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCheckTokenAudience pins the general-API audience policy for P1a PR 5:
// the fixed audiences keep working, an MCP token (token_use=mcp) is refused
// outright — since PR 4's private transport, /mcp tool traffic never presents
// it here, so any appearance is a leaked or misused token, not tool traffic —
// and everything else is refused as an audience mismatch.
func TestCheckTokenAudience(t *testing.T) {
	claimsWith := func(aud string, tokenUse string) *claimsMessage {
		return &claimsMessage{
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{aud}},
			TokenUse:         tokenUse,
		}
	}

	t.Run("web session audience is accepted", func(t *testing.T) {
		err := checkTokenAudience(claimsWith(AccessTokenAudience, ""))
		require.NoError(t, err)
	})

	t.Run("legacy oauth2 audience is refused: it is MCP-minted too", func(t *testing.T) {
		// bb.oauth2.access was only ever minted by the MCP authorization
		// server (pre-PR-3, before tokens carried token_use), so it is an MCP
		// token by provenance and gets the same refusal. Unlike at /mcp, no
		// legitimate traffic drains through here: the pre-PR-4 loopback
		// transport dialed its own replica, so even mid-rolling-upgrade no old
		// replica ever lands a legacy bearer on this chain.
		err := checkTokenAudience(claimsWith(OAuth2AccessTokenAudience, ""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "only accepted at /mcp")
	})

	t.Run("mcp resource-bound token is refused", func(t *testing.T) {
		err := checkTokenAudience(claimsWith(testResource, TokenUseMCP))
		require.Error(t, err)
		require.Contains(t, err.Error(), "only accepted at /mcp")
	})

	t.Run("token_use=mcp is refused even with the accepted fixed audience", func(t *testing.T) {
		// Nothing mints this combination; if one ever appears, the MCP marker
		// must win over the audience allowlist — the rejection keys on what the
		// token IS, not on which audience it also happens to carry.
		err := checkTokenAudience(claimsWith(AccessTokenAudience, TokenUseMCP))
		require.Error(t, err)
		require.Contains(t, err.Error(), "only accepted at /mcp")
	})

	t.Run("unknown audience without token_use is refused", func(t *testing.T) {
		err := checkTokenAudience(claimsWith("wrong.audience", ""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "audience mismatch")
	})
}

func TestPermissionForRequest(t *testing.T) {
	tests := []struct {
		name              string
		request           any
		defaultPermission permission.Permission
		want              permission.Permission
	}{
		{
			name: "keeps default permission for existing instance database listing",
			request: &v1pb.ListInstanceDatabaseRequest{
				Name: "instances/hello",
			},
			defaultPermission: permission.InstancesGet,
			want:              permission.InstancesGet,
		},
		{
			name: "requires create permission for inline instance database preview",
			request: &v1pb.ListInstanceDatabaseRequest{
				Name:     "instances/hello",
				Instance: &v1pb.Instance{},
			},
			defaultPermission: permission.InstancesGet,
			want:              permission.InstancesCreate,
		},
		{
			name:              "keeps default permission for other requests",
			request:           &v1pb.SyncInstanceRequest{Name: "instances/hello"},
			defaultPermission: permission.InstancesSync,
			want:              permission.InstancesSync,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PermissionForRequest(tt.request, tt.defaultPermission)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHasAllowMissingEnabled(t *testing.T) {
	tests := []struct {
		name    string
		request any
		want    bool
	}{
		{
			name: "AllowMissing true",
			request: &v1pb.UpdateRoleRequest{
				AllowMissing: true,
			},
			want: true,
		},
		{
			name: "AllowMissing false",
			request: &v1pb.UpdateRoleRequest{
				AllowMissing: false,
			},
			want: false,
		},
		{
			name: "No AllowMissing field",
			request: &v1pb.GetRoleRequest{
				Name: "roles/test",
			},
			want: false,
		},
		{
			name:    "Nil request",
			request: nil,
			want:    false,
		},
		{
			name: "UpdateGroupRequest with AllowMissing true",
			request: &v1pb.UpdateGroupRequest{
				AllowMissing: true,
			},
			want: true,
		},
		{
			name: "UpdateReviewConfigRequest with AllowMissing true",
			request: &v1pb.UpdateReviewConfigRequest{
				AllowMissing: true,
			},
			want: true,
		},
		{
			name: "UpdateIdentityProviderRequest with AllowMissing false",
			request: &v1pb.UpdateIdentityProviderRequest{
				AllowMissing: false,
			},
			want: false,
		},
		{
			name: "BatchUpdateInstancesRequest with nested AllowMissing true",
			request: &v1pb.BatchUpdateInstancesRequest{
				Requests: []*v1pb.UpdateInstanceRequest{
					{AllowMissing: false},
					{AllowMissing: true},
				},
			},
			want: true,
		},
		{
			name: "BatchUpdateInstancesRequest with nested AllowMissing false",
			request: &v1pb.BatchUpdateInstancesRequest{
				Requests: []*v1pb.UpdateInstanceRequest{{AllowMissing: false}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasAllowMissingEnabled(tt.request)
			require.Equal(t, tt.want, got)
		})
	}
}
