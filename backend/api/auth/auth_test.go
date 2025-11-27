package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAllowMissingRequiredPermission(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		wantPerm   string
		wantErr    bool
	}{
		{
			name:       "RoleService.UpdateRole has annotation",
			fullMethod: "/bytebase.v1.RoleService/UpdateRole",
			wantPerm:   "bb.roles.create",
			wantErr:    false,
		},
		{
			name:       "GroupService.UpdateGroup has annotation",
			fullMethod: "/bytebase.v1.GroupService/UpdateGroup",
			wantPerm:   "bb.groups.create",
			wantErr:    false,
		},
		{
			name:       "ReviewConfigService.UpdateReviewConfig has annotation",
			fullMethod: "/bytebase.v1.ReviewConfigService/UpdateReviewConfig",
			wantPerm:   "bb.reviewConfigs.create",
			wantErr:    false,
		},
		{
			name:       "IdentityProviderService.UpdateIdentityProvider has annotation",
			fullMethod: "/bytebase.v1.IdentityProviderService/UpdateIdentityProvider",
			wantPerm:   "bb.identityProviders.create",
			wantErr:    false,
		},
		{
			name:       "DatabaseGroupService.UpdateDatabaseGroup has annotation",
			fullMethod: "/bytebase.v1.DatabaseGroupService/UpdateDatabaseGroup",
			wantPerm:   "bb.projects.update",
			wantErr:    false,
		},
		{
			name:       "ReleaseService.UpdateRelease has annotation",
			fullMethod: "/bytebase.v1.ReleaseService/UpdateRelease",
			wantPerm:   "bb.releases.create",
			wantErr:    false,
		},
		{
			name:       "RoleService.GetRole no annotation",
			fullMethod: "/bytebase.v1.RoleService/GetRole",
			wantPerm:   "",
			wantErr:    false,
		},
		{
			name:       "RoleService.CreateRole no annotation",
			fullMethod: "/bytebase.v1.RoleService/CreateRole",
			wantPerm:   "",
			wantErr:    false,
		},
		{
			name:       "Invalid method path - too short",
			fullMethod: "invalid",
			wantPerm:   "",
			wantErr:    true,
		},
		{
			name:       "Invalid method path - missing slash",
			fullMethod: "bytebase.v1.RoleService.UpdateRole",
			wantPerm:   "",
			wantErr:    true,
		},
		{
			name:       "Non-existent service",
			fullMethod: "/bytebase.v1.NonExistentService/SomeMethod",
			wantPerm:   "",
			wantErr:    true,
		},
		{
			name:       "Non-existent method",
			fullMethod: "/bytebase.v1.RoleService/NonExistentMethod",
			wantPerm:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPerm, err := GetAllowMissingRequiredPermission(tt.fullMethod)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantPerm, gotPerm)
			}
		})
	}
}
