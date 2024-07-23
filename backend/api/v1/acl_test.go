package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetResourceFromRequest(t *testing.T) {
	tests := []struct {
		request any
		method  string
		want    string
	}{
		{
			request: &v1pb.LoginRequest{Email: "hello@world.com"},
			method:  "/bytebase.v1.AuthService/Login",
			want:    "",
		},
		// The database group has not been annotated with resource yet.
		{
			request: &v1pb.CreateDatabaseGroupRequest{
				Parent: "projects/hello",
			},
			method: "/bytebase.v1.DatabaseGroupService/CreateDatabaseGroup",
			want:   "",
		},
		{
			request: &v1pb.UpdateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			method: "/bytebase.v1.ProjectService/UpdateProject",
			want:   "projects/hello",
		},
		{
			// The instance has not been annotated with resource yet.
			request: &v1pb.UpdateInstanceRequest{
				Instance: &v1pb.Instance{
					Name: "instances/hello",
				},
			},
			method: "/bytebase.v1.InstanceService/UpdateInstance",
			want:   "",
		},
	}

	for _, tt := range tests {
		got := getResourceFromRequest(tt.request, tt.method)
		require.Equal(t, tt.want, got, tt.method)
	}
}
