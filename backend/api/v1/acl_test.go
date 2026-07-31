package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestGetResourceRoute(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  resourceRoute
	}{
		{
			name:  "workspace",
			parts: []string{"workspaces", "default"},
			want:  resourceRoute{"workspaces"},
		},
		{
			name:  "project",
			parts: []string{"projects", "project-a"},
			want:  resourceRoute{"projects"},
		},
		{
			name:  "project instance",
			parts: []string{"projects", "project-a", "instances", "instance-a"},
			want:  resourceRoute{"projects", "instances"},
		},
		{
			name:  "project database",
			parts: []string{"projects", "project-a", "instances", "instance-a", "databases", "app"},
			want:  resourceRoute{"projects", "instances", "databases"},
		},
		{
			name:  "workspace instance",
			parts: []string{"instances", "instance-a"},
			want:  resourceRoute{"instances"},
		},
		{
			name:  "workspace database",
			parts: []string{"instances", "instance-a", "databases", "app"},
			want:  resourceRoute{"instances", "databases"},
		},
		{
			name:  "project instance role",
			parts: []string{"projects", "project-a", "instances", "instance-a", "roles", "role-a"},
			want:  resourceRoute{"projects", "instances", "roles"},
		},
		{
			name:  "project database revision",
			parts: []string{"projects", "project-a", "instances", "instance-a", "databases", "app", "revisions", "1"},
			want:  resourceRoute{"projects", "instances", "databases"},
		},
		{
			name:  "project database changelog",
			parts: []string{"projects", "project-a", "instances", "instance-a", "databases", "app", "changelogs", "1"},
			want:  resourceRoute{"projects", "instances", "databases"},
		},
		{
			name:  "project database schema",
			parts: []string{"projects", "project-a", "instances", "instance-a", "databases", "app", "schema"},
			want:  resourceRoute{"projects", "instances", "databases"},
		},
		{
			name:  "workspace instance role",
			parts: []string{"instances", "instance-a", "roles", "role-a"},
			want:  resourceRoute{"instances", "roles"},
		},
		{
			name:  "workspace database revision",
			parts: []string{"instances", "instance-a", "databases", "app", "revisions", "1"},
			want:  resourceRoute{"instances", "databases", "revisions"},
		},
		{
			name:  "workspace database schema",
			parts: []string{"instances", "instance-a", "databases", "app", "schema"},
			want:  resourceRoute{"instances", "databases", "schema"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, getResourceRoute(tt.parts))
		})
	}
}

func TestFindResourceResolver(t *testing.T) {
	tests := []struct {
		name       string
		route      resourceRoute
		wantRoute  resourceRoute
		wantExists bool
	}{
		{name: "workspace", route: resourceRoute{"workspaces"}, wantRoute: resourceRoute{"workspaces"}, wantExists: true},
		{name: "project", route: resourceRoute{"projects"}, wantRoute: resourceRoute{"projects"}, wantExists: true},
		{name: "project instance", route: resourceRoute{"projects", "instances"}, wantRoute: resourceRoute{"projects", "instances"}, wantExists: true},
		{name: "project database", route: resourceRoute{"projects", "instances", "databases"}, wantRoute: resourceRoute{"projects", "instances", "databases"}, wantExists: true},
		{name: "workspace instance", route: resourceRoute{"instances"}, wantRoute: resourceRoute{"instances"}, wantExists: true},
		{name: "workspace database", route: resourceRoute{"instances", "databases"}, wantRoute: resourceRoute{"instances", "databases"}, wantExists: true},
		{name: "project instance role", route: resourceRoute{"projects", "instances", "roles"}, wantRoute: resourceRoute{"projects", "instances"}, wantExists: true},
		{name: "project database revision", route: resourceRoute{"projects", "instances", "databases"}, wantRoute: resourceRoute{"projects", "instances", "databases"}, wantExists: true},
		{name: "project database changelog", route: resourceRoute{"projects", "instances", "databases"}, wantRoute: resourceRoute{"projects", "instances", "databases"}, wantExists: true},
		{name: "project database schema", route: resourceRoute{"projects", "instances", "databases"}, wantRoute: resourceRoute{"projects", "instances", "databases"}, wantExists: true},
		{name: "workspace instance role", route: resourceRoute{"instances", "roles"}, wantRoute: resourceRoute{"instances"}, wantExists: true},
		{name: "workspace database revision", route: resourceRoute{"instances", "databases", "revisions"}, wantRoute: resourceRoute{"instances", "databases"}, wantExists: true},
		{name: "workspace database schema", route: resourceRoute{"instances", "databases", "schema"}, wantRoute: resourceRoute{"instances", "databases"}, wantExists: true},
		{name: "ordinary project descendant", route: resourceRoute{"projects", "issues"}, wantRoute: resourceRoute{"projects"}, wantExists: true},
		{name: "unknown root", route: resourceRoute{"unknowns"}, wantExists: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRoute, resolver, gotExists := findResourceResolver(tt.route)
			require.Equal(t, tt.wantExists, gotExists)
			if !tt.wantExists {
				require.Nil(t, resolver)
				return
			}
			require.NotNil(t, resolver)
			require.Equal(t, tt.wantRoute, gotRoute)
		})
	}
}

func TestResolveRawResource(t *testing.T) {
	ctx, stores, instanceID, _, _, _ := setupProjectInstanceDescendantServiceTest(t)

	t.Run("rejects project instance in another project", func(t *testing.T) {
		resource, err := resolveRawResource(ctx, stores, common.FormatProjectInstance("project-b", instanceID)+"/roles/role-a")
		require.Error(t, err)
		require.Nil(t, resource)
	})

	t.Run("rejects missing project database", func(t *testing.T) {
		resource, err := resolveRawResource(ctx, stores, common.FormatProjectDatabase("project-a", instanceID, "missing")+"/schema")
		require.Error(t, err)
		require.Nil(t, resource)
	})

	t.Run("requires resolver identifiers", func(t *testing.T) {
		for _, name := range []string{
			"workspaces/",
			"projects/",
			"projects/project-a/instances/",
			"projects/project-a/instances/instance-a/databases/",
			"instances/",
			"instances/instance-a/databases/",
		} {
			resource, err := resolveRawResource(ctx, nil, name)
			require.Error(t, err, name)
			require.Nil(t, resource, name)
		}
	})
}

func TestResolveRawResourceWorkspaceDatabaseUsesDatabaseProject(t *testing.T) {
	ctx, stores, instanceID, databaseName, _, _ := setupWorkspaceInstanceDescendantServiceTest(t)
	resource, err := resolveRawResource(ctx, stores, common.FormatDatabase(instanceID, databaseName)+"/revisions/1")
	require.NoError(t, err)
	require.Equal(t, &common.Resource{Type: common.ResourceTypeProject, ID: "project-a"}, resource)
}

func TestPopulateRawResourcesUsesWorkspaceFallback(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	for _, request := range []any{
		&v1pb.GetInstanceRequest{Name: "projects/-"},
		&v1pb.GetDatabaseRequest{Name: "instances/-/databases/app"},
	} {
		resources, err := populateRawResources(ctx, nil, request, "/bytebase.v1.TestService/Get")
		require.NoError(t, err)
		require.Equal(t, []*common.Resource{{Type: common.ResourceTypeWorkspace, ID: "default"}}, resources)
	}

	resources, err := populateRawResources(context.Background(), nil, &v1pb.ListProjectsRequest{}, "/bytebase.v1.ProjectService/ListProjects")
	require.NoError(t, err)
	require.Empty(t, resources)
}

func TestGetResourceFromRequest(t *testing.T) {
	tests := []struct {
		request any
		method  string
		want    []string
	}{
		{
			request: &v1pb.LoginRequest{Email: "hello@world.com"},
			method:  "/bytebase.v1.AuthService/Login",
			want:    []string{""},
		},
		{
			request: &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			method: "/bytebase.v1.ProjectService/CreateProject",
			want:   []string{""},
		},
		{
			request: &v1pb.UpdateProjectRequest{
				Project: &v1pb.Project{
					Name: "projects/hello",
				},
			},
			method: "/bytebase.v1.ProjectService/UpdateProject",
			want:   []string{"projects/hello"},
		},
		{
			request: &v1pb.ListProjectsRequest{},
			method:  "/bytebase.v1.ProjectService/ListProjects",
			want:    []string{""},
		},
		// The database group has not been annotated with resource yet.
		{
			request: &v1pb.CreateDatabaseGroupRequest{
				Parent: "projects/hello",
			},
			method: "/bytebase.v1.DatabaseGroupService/CreateDatabaseGroup",
			want:   []string{"projects/hello"},
		},
		{
			// The instance has not been annotated with resource yet.
			request: &v1pb.UpdateInstanceRequest{
				Instance: &v1pb.Instance{
					Name: "instances/hello",
				},
			},
			method: "/bytebase.v1.InstanceService/UpdateInstance",
			want:   []string{"instances/hello"},
		},
		{
			request: &v1pb.CreateInstanceRequest{
				Parent: new("projects/default"),
			},
			method: "/bytebase.v1.InstanceService/CreateInstance",
			want:   []string{""},
		},
		{
			request: &v1pb.CreateInstanceRequest{
				Parent: new("projects/hello"),
			},
			method: "/bytebase.v1.InstanceService/CreateInstance",
			want:   []string{"projects/hello"},
		},
		{
			request: &v1pb.UpdateInstanceRequest{
				Instance:     &v1pb.Instance{Name: "projects/hello/instances/new-instance"},
				AllowMissing: true,
			},
			method: "/bytebase.v1.InstanceService/UpdateInstance",
			want:   []string{"projects/hello"},
		},
		{
			request: &v1pb.UploadLicenseRequest{
				License: "123",
			},
			method: "/bytebase.v1.SubscriptionService/UpdateSubscription",
			want:   []string{""},
		},
		{
			request: &v1pb.RemoveWebhookRequest{
				Webhook: &v1pb.Webhook{
					Name: "projects/aaa/webhooks/bbb",
				},
			},
			method: "/bytebase.v1.ProjectService/RemoveWebhook",
			want:   []string{"projects/aaa/webhooks/bbb"},
		},
		{
			request: &v1pb.UpdateIdentityProviderRequest{
				IdentityProvider: &v1pb.IdentityProvider{
					Name: "idps/hello",
				},
			},
			method: "/bytebase.v1.IdentityProviderService/UpdateIdentityProvider",
			want:   []string{"idps/hello"},
		},
		{
			request: &v1pb.TestIdentityProviderRequest{
				IdentityProvider: &v1pb.IdentityProvider{
					Name: "idps/hello",
				},
			},
			method: "/bytebase.v1.IdentityProviderService/TestIdentityProvider",
			want:   []string{"idps/hello"},
		},
		{
			request: &v1pb.ListReviewConfigsRequest{},
			method:  "/bytebase.v1.ReviewConfigService/ListReviewConfigs",
			want:    []string{""},
		},
		{
			request: &v1pb.BatchUpdateDatabasesRequest{
				Requests: []*v1pb.UpdateDatabaseRequest{
					{Database: &v1pb.Database{Name: "instances/hello/databases/hello"}},
					{Database: &v1pb.Database{Name: "instances/world/databases/world"}},
				},
			},
			method: "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
			want:   []string{"instances/hello/databases/hello", "instances/world/databases/world"},
		},
		{
			request: &v1pb.BatchUpdateDatabasesRequest{
				Requests: []*v1pb.UpdateDatabaseRequest{
					{Database: &v1pb.Database{Name: "instances/hello/databases/hello", Project: "projects/a"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"project"}}},
					{Database: &v1pb.Database{Name: "instances/world/databases/world", Project: "projects/b"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"project"}}},
				},
			},
			method: "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
			want:   []string{"projects/a", "projects/b", "instances/hello/databases/hello", "instances/world/databases/world"},
		},
		{
			request: &v1pb.SyncInstanceRequest{
				Name: "instances/hello",
			},
			method: "/bytebase.v1.InstanceService/SyncInstance",
			want:   []string{"instances/hello"},
		},
		{
			request: &v1pb.ListInstanceDatabaseRequest{
				Name: "instances/hello",
			},
			method: "/bytebase.v1.InstanceService/ListInstanceDatabase",
			want:   []string{"instances/hello"},
		},
		{
			request: &v1pb.ListInstanceDatabaseRequest{
				Name:     "instances/hello",
				Instance: &v1pb.Instance{},
			},
			method: "/bytebase.v1.InstanceService/ListInstanceDatabase",
			want:   []string{""},
		},
		{
			request: &v1pb.ListInstanceDatabaseRequest{
				Name:     "projects/project-a/instances/hello",
				Instance: &v1pb.Instance{},
			},
			method: "/bytebase.v1.InstanceService/ListInstanceDatabase",
			want:   []string{"projects/project-a"},
		},
		{
			request: &v1pb.BatchSyncInstancesRequest{
				Requests: []*v1pb.SyncInstanceRequest{
					{Name: "instances/hello"},
					{Name: "instances/world"},
				},
			},
			method: "/bytebase.v1.InstanceService/BatchSyncInstances",
			want:   []string{"instances/hello", "instances/world"},
		},
		{
			request: &v1pb.BatchSyncInstancesRequest{
				Parent: new("projects/project-a"),
				Requests: []*v1pb.SyncInstanceRequest{
					{Name: "projects/project-a/instances/hello"},
				},
			},
			method: "/bytebase.v1.InstanceService/BatchSyncInstances",
			want:   []string{"projects/project-a", "projects/project-a/instances/hello"},
		},
		{
			request: &v1pb.CancelPlanCheckRunRequest{
				Name: "projects/hello/plans/world/planCheckRun",
			},
			method: "/bytebase.v1.PlanService/CancelPlanCheckRun",
			want:   []string{"projects/hello/plans/world/planCheckRun"},
		},
	}

	for _, tt := range tests {
		got, _ := getResourceFromRequest(context.Background(), tt.request, tt.method)
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

func TestGetPermissionForRequest(t *testing.T) {
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
			got := getPermissionForRequest(tt.request, tt.defaultPermission)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAllowMissingEnabled(tt.request)
			require.Equal(t, tt.want, got)
		})
	}
}
