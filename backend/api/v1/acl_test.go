package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetResourceFromRequest(t *testing.T) {
	tests := []struct {
		request any
		want    string
	}{
		{
			request: &v1pb.LoginRequest{Email: "hello@world.com"},
			want:    "",
		},
		{
			request: &v1pb.CreateDatabaseGroupRequest{
				Parent: "projects/hello",
			},
			want: "projects/hello",
		},
		{
			request: &v1pb.UpdateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			want: "projects/hello",
		},
		{
			// The instance has not been annotated with resource yet.
			request: &v1pb.UpdateInstanceRequest{
				Instance: &v1pb.Instance{
					Name: "instances/hello",
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		got := getResourceFromRequest(tt.request)
		require.Equal(t, tt.want, got)
	}
}
