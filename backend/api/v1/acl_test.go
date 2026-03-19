package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestGetResourceFromRequest(t *testing.T) {
	tests := []struct {
		request any
		method  string
		want    []*rawResource
	}{
		{
			request: &v1pb.LoginRequest{Email: "hello@world.com"},
			method:  "/bytebase.v1.AuthService/Login",
			want: []*rawResource{
				{isWorkspace: true},
			},
		},
		{
			request: &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			method: "/bytebase.v1.ProjectService/CreateProject",
			want: []*rawResource{
				{isWorkspace: true},
			},
		},
		{
			request: &v1pb.UpdateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			method: "/bytebase.v1.ProjectService/UpdateProject",
			want: []*rawResource{
				{name: "projects/hello"},
			},
		},
		{
			request: &v1pb.ListProjectsRequest{},
			method:  "/bytebase.v1.ProjectService/ListProjects",
			want: []*rawResource{
				{isWorkspace: true},
			},
		},
		// The database group has not been annotated with resource yet.
		{
			request: &v1pb.CreateDatabaseGroupRequest{
				Parent: "projects/hello",
			},
			method: "/bytebase.v1.DatabaseGroupService/CreateDatabaseGroup",
			want: []*rawResource{
				{name: "projects/hello"},
			},
		},
		{
			// The instance has not been annotated with resource yet.
			request: &v1pb.UpdateInstanceRequest{
				Instance: &v1pb.Instance{
					Name: "instances/hello",
				},
			},
			method: "/bytebase.v1.InstanceService/UpdateInstance",
			want: []*rawResource{
				{name: "instances/hello"},
			},
		},
		{
			request: &v1pb.UpdateSubscriptionRequest{
				License: "123",
			},
			method: "/bytebase.v1.SubscriptionService/UpdateSubscription",
			want: []*rawResource{
				{isWorkspace: true},
			},
		},
		{
			request: &v1pb.RemoveWebhookRequest{
				Webhook: &v1pb.Webhook{
					Name: "projects/aaa/webhooks/bbb",
				},
			},
			method: "/bytebase.v1.ProjectService/RemoveWebhook",
			want: []*rawResource{
				{name: "projects/aaa/webhooks/bbb"},
			},
		},
		{
			request: &v1pb.UpdateIdentityProviderRequest{
				IdentityProvider: &v1pb.IdentityProvider{
					Name: "idps/hello",
				},
			},
			method: "/bytebase.v1.IdentityProviderService/UpdateIdentityProvider",
			want: []*rawResource{
				{name: "idps/hello"},
			},
		},
		{
			request: &v1pb.TestIdentityProviderRequest{
				IdentityProvider: &v1pb.IdentityProvider{
					Name: "idps/hello",
				},
			},
			method: "/bytebase.v1.IdentityProviderService/TestIdentityProvider",
			want: []*rawResource{
				{name: "idps/hello"},
			},
		},
		{
			request: &v1pb.ListReviewConfigsRequest{},
			method:  "/bytebase.v1.ReviewConfigService/ListReviewConfigs",
			want: []*rawResource{
				{isWorkspace: true},
			},
		},
		{
			request: &v1pb.BatchUpdateDatabasesRequest{
				Requests: []*v1pb.UpdateDatabaseRequest{
					{Database: &v1pb.Database{Name: "instances/hello/databases/hello"}},
					{Database: &v1pb.Database{Name: "instances/world/databases/world"}},
				},
			},
			method: "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
			want: []*rawResource{
				{name: "instances/hello/databases/hello"},
				{name: "instances/world/databases/world"},
			},
		},
		{
			request: &v1pb.BatchUpdateDatabasesRequest{
				Requests: []*v1pb.UpdateDatabaseRequest{
					{Database: &v1pb.Database{Name: "instances/hello/databases/hello", Project: "projects/a"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"project"}}},
					{Database: &v1pb.Database{Name: "instances/world/databases/world", Project: "projects/b"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"project"}}},
				},
			},
			method: "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
			want: []*rawResource{
				{name: "projects/a"},
				{name: "projects/b"},
				{name: "instances/hello/databases/hello"},
				{name: "instances/world/databases/world"},
			},
		},
		{
			request: &v1pb.SyncInstanceRequest{
				Name: "instances/hello",
			},
			method: "/bytebase.v1.InstanceService/SyncInstance",
			want: []*rawResource{
				{name: "instances/hello"},
			},
		},
		{
			request: &v1pb.BatchSyncInstancesRequest{
				Requests: []*v1pb.SyncInstanceRequest{
					{Name: "instances/hello"},
					{Name: "instances/world"},
				},
			},
			method: "/bytebase.v1.InstanceService/BatchSyncInstances",
			want: []*rawResource{
				{name: "instances/hello"},
				{name: "instances/world"},
			},
		},
		{
			request: &v1pb.CancelPlanCheckRunRequest{
				Name: "projects/hello/plans/world/planCheckRun",
			},
			method: "/bytebase.v1.PlanService/CancelPlanCheckRun",
			want: []*rawResource{
				{name: "projects/hello/plans/world/planCheckRun"},
			},
		},
	}

	for _, tt := range tests {
		got, _ := getResourceFromRequest(tt.request, tt.method)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAllowMissingEnabled(tt.request)
			require.Equal(t, tt.want, got)
		})
	}
}
