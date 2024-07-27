package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetResourceFromRequest(t *testing.T) {
	tests := []struct {
		request any
		method  string
		want    *common.Resource
	}{
		{
			request: &v1pb.LoginRequest{Email: "hello@world.com"},
			method:  "/bytebase.v1.AuthService/Login",
			want:    nil,
		},
		{
			request: &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			method: "/bytebase.v1.ProjectService/CreateProject",
			want:   &common.Resource{Workspace: true},
		},
		{
			request: &v1pb.UpdateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			method: "/bytebase.v1.ProjectService/UpdateProject",
			want:   &common.Resource{Name: "projects/hello"},
		},
		{
			request: &v1pb.ListProjectsRequest{},
			method:  "/bytebase.v1.ProjectService/ListProjects",
			want:    &common.Resource{Workspace: true},
		},
		// The database group has not been annotated with resource yet.
		{
			request: &v1pb.CreateDatabaseGroupRequest{
				Parent: "projects/hello",
			},
			method: "/bytebase.v1.DatabaseGroupService/CreateDatabaseGroup",
			want:   &common.Resource{Name: "projects/hello"},
		},
		{
			// The instance has not been annotated with resource yet.
			request: &v1pb.UpdateInstanceRequest{
				Instance: &v1pb.Instance{
					Name: "instances/hello",
				},
			},
			method: "/bytebase.v1.InstanceService/UpdateInstance",
			want:   &common.Resource{Name: "instances/hello"},
		},
		{
			request: &v1pb.UpdateSubscriptionRequest{
				Patch: &v1pb.PatchSubscription{License: "123"},
			},
			method: "/bytebase.v1.SubscriptionService/UpdateSubscription",
			want:   nil,
		},
		{
			request: &v1pb.RemoveWebhookRequest{
				Webhook: &v1pb.Webhook{
					Name: "projects/aaa/webhooks/bbb",
				},
			},
			method: "/bytebase.v1.ProjectService/RemoveWebhook",
			want:   &common.Resource{Name: "projects/aaa/webhooks/bbb"},
		},
		{
			request: &v1pb.UpdateIdentityProviderRequest{
				IdentityProvider: &v1pb.IdentityProvider{
					Name: "idps/hello",
				},
			},
			method: "/bytebase.v1.IdentityProviderService/UpdateIdentityProvider",
			want: &common.Resource{
				Name: "idps/hello",
			},
		},
	}

	for _, tt := range tests {
		got := getResourceFromRequest(tt.request, tt.method)
		require.Equal(t, tt.want, got, tt.method)
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "DatabaseGroup",
			want:  "database_group",
		},
		{
			input: "User",
			want:  "user",
		},
		{
			input: "Instance",
			want:  "instance",
		},
		{
			input: "IdentityProvider",
			want:  "identity_provider",
		},
	}

	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		require.Equal(t, tt.want, got, tt.input)
	}
}
