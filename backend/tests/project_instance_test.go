package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestProjectInstanceCoreBehavior exercises the public v1 collection contract
// against PostgreSQL, including database discovery and the nested resource names
// that clients use after an instance is assigned to a project.
func TestProjectInstanceCoreBehavior(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	const databaseID = "bot35_project_database"
	createPgDatabase(t, pg, databaseID)

	otherProject := createProjectForProjectInstanceTest(ctx, t, ctl, "bot35-other-project")

	workspaceID := "bot35-workspace-instance"
	workspace := createProjectInstanceTestInstance(ctx, t, ctl, nil, workspaceID, "workspace instance", pg)
	a.Equal("instances/"+workspaceID, workspace.Name)

	projectParent := ctl.project.Name
	projectID := "bot35-project-instance"
	projectInstance := createProjectInstanceTestInstance(ctx, t, ctl, &projectParent, projectID, "project instance", pg)
	a.Equal(fmt.Sprintf("%s/instances/%s", ctl.project.Name, projectID), projectInstance.Name)

	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{
		Name: projectInstance.Name,
	}))
	a.NoError(err)

	databaseName := fmt.Sprintf("%s/databases/%s", projectInstance.Name, databaseID)
	database, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseName}))
	a.NoError(err)
	a.Equal(databaseName, database.Msg.Name)
	a.Equal(ctl.project.Name, database.Msg.Project)

	for _, name := range []string{
		projectInstance.Name + "/databases/missing",
		workspace.Name + "/databases/missing",
	} {
		_, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: name}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	}

	projectInstances, err := ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{
		Parent:   &projectParent,
		PageSize: 100,
	}))
	a.NoError(err)
	a.Len(projectInstances.Msg.Instances, 1)
	a.Equal(projectInstance.Name, projectInstances.Msg.Instances[0].Name)

	workspaceInstances, err := ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{PageSize: 100}))
	a.NoError(err)
	a.Contains(projectInstanceTestInstanceNames(workspaceInstances.Msg.Instances), workspace.Name)
	a.NotContains(projectInstanceTestInstanceNames(workspaceInstances.Msg.Instances), projectInstance.Name)

	_, err = ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{
		Parent: &projectParent,
		Filter: fmt.Sprintf("project == %q", otherProject.Name),
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// A project instance is never reachable through the workspace alias.
	_, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
		Name: "instances/" + projectID,
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	a.Equal(fmt.Sprintf("not_found: instance %q not found", "instances/"+projectID), err.Error())

	// Instance IDs are workspace-global even though project instances have a
	// project-nested resource name.
	_, err = ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: projectID,
		Instance:   projectInstanceTestSpec("duplicate", pg),
	}))
	a.Error(err)
	a.Equal(connect.CodeAlreadyExists, connect.CodeOf(err))

	allowMissingID := "bot35-allow-missing"
	allowMissingName := fmt.Sprintf("%s/instances/%s", ctl.project.Name, allowMissingID)
	allowMissing, err := ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
		Instance: func() *v1pb.Instance {
			instance := projectInstanceTestSpec("created with allow missing", pg)
			instance.Name = allowMissingName
			// Keep this lifecycle target database-free. SyncDatabases is a
			// whitelist, and an empty list excludes every database the server
			// itself uses for this integration test.
			instance.SyncDatabases = &v1pb.SyncDatabases{}
			return instance
		}(),
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		AllowMissing: true,
	}))
	a.NoError(err)
	a.Equal(allowMissingName, allowMissing.Msg.Name)

	filteredProjectInstances, err := ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{
		Parent: &projectParent,
		Filter: fmt.Sprintf("project == %q", projectParent),
	}))
	a.NoError(err)
	a.Contains(projectInstanceTestInstanceNames(filteredProjectInstances.Msg.Instances), allowMissingName)

	_, err = ctl.databaseServiceClient.UpdateDatabase(ctx, connect.NewRequest(&v1pb.UpdateDatabaseRequest{
		Database: &v1pb.Database{
			Name:    databaseName,
			Project: otherProject.Name,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"project"}},
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	validateOnlyID := "bot35-validate-only"
	_, err = ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		Parent:       &projectParent,
		InstanceId:   validateOnlyID,
		Instance:     projectInstanceTestSpec("validate only", pg),
		ValidateOnly: true,
	}))
	a.NoError(err)
	_, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
		Name: fmt.Sprintf("%s/instances/%s", ctl.project.Name, validateOnlyID),
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// Batch requests validate every target against the exact parent collection
	// before updating anything.
	updated, err := ctl.instanceServiceClient.BatchUpdateInstances(ctx, connect.NewRequest(&v1pb.BatchUpdateInstancesRequest{
		Parent: &projectParent,
		Requests: []*v1pb.UpdateInstanceRequest{{
			Instance:   &v1pb.Instance{Name: allowMissingName, Title: "exact batch parent"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}},
	}))
	a.NoError(err)
	a.Len(updated.Msg.Instances, 1)
	a.Equal("exact batch parent", updated.Msg.Instances[0].Title)

	beforeProject, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: allowMissingName}))
	a.NoError(err)
	beforeWorkspace, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: workspace.Name}))
	a.NoError(err)
	_, err = ctl.instanceServiceClient.BatchUpdateInstances(ctx, connect.NewRequest(&v1pb.BatchUpdateInstancesRequest{
		Parent: &projectParent,
		Requests: []*v1pb.UpdateInstanceRequest{
			{
				Instance:   &v1pb.Instance{Name: allowMissingName, Title: "must not update"},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
			},
			{
				Instance:   &v1pb.Instance{Name: workspace.Name, Title: "must not update"},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
			},
		},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	afterProject, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: allowMissingName}))
	a.NoError(err)
	afterWorkspace, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: workspace.Name}))
	a.NoError(err)
	a.Equal(beforeProject.Msg.Title, afterProject.Msg.Title)
	a.Equal(beforeWorkspace.Msg.Title, afterWorkspace.Msg.Title)

	// A project instance cannot force-transfer its databases, but can be
	// archived normally even after synchronization discovers databases.
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{
		Name:  projectInstance.Name,
		Force: true,
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: projectInstance.Name}))
	a.NoError(err)
	archivedProjectInstance, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: projectInstance.Name}))
	a.NoError(err)
	a.Equal(v1pb.State_DELETED, archivedProjectInstance.Msg.State)

	// A database-free project instance can still be archived and restored normally.
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: allowMissingName}))
	a.NoError(err)
	archived, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: allowMissingName}))
	a.NoError(err)
	a.Equal(v1pb.State_DELETED, archived.Msg.State)
	restored, err := ctl.instanceServiceClient.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: allowMissingName}))
	a.NoError(err)
	a.Equal(v1pb.State_ACTIVE, restored.Msg.State)
}

func TestBatchUpdateProjectInstanceAllowMissing(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)

	const instanceID = "bot35-batch-allow-missing"
	parent := ctl.project.Name
	name := fmt.Sprintf("%s/instances/%s", parent, instanceID)
	created, err := ctl.instanceServiceClient.BatchUpdateInstances(ctx, connect.NewRequest(&v1pb.BatchUpdateInstancesRequest{
		Parent: &parent,
		Requests: []*v1pb.UpdateInstanceRequest{{
			Instance: func() *v1pb.Instance {
				instance := projectInstanceTestSpec("created by batch allow missing", pg)
				instance.Name = name
				instance.SyncDatabases = &v1pb.SyncDatabases{}
				return instance
			}(),
			UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"title"}},
			AllowMissing: true,
		}},
	}))
	a.NoError(err)
	a.Len(created.Msg.Instances, 1)
	a.Equal(name, created.Msg.Instances[0].Name)
}

//nolint:tparallel // Subtests share one server lifecycle.
func TestProjectInstanceValidation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	for _, parent := range []string{"projects/missing-bot35", "projects/default"} {
		parent := parent
		t.Run(parent, func(t *testing.T) {
			_, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
				Parent:     &parent,
				InstanceId: "bot35-invalid-parent",
				Instance:   &v1pb.Instance{},
			}))
			require.Error(t, err)
			if parent == "projects/default" {
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Equal(t, `invalid_argument: default project "projects/default" cannot own instances`, err.Error())
			} else {
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			}
		})
	}

	deletedProject := createProjectForProjectInstanceTest(ctx, t, ctl, "bot35-deleted-project")
	_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: deletedProject.Name}))
	a.NoError(err)
	_, err = ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		Parent:     &deletedProject.Name,
		InstanceId: "bot35-deleted-parent",
		Instance:   &v1pb.Instance{},
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	parent := ctl.project.Name
	_, err = ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		Parent:                 &parent,
		InitialDatabaseProject: ctl.project.Name,
		InstanceId:             "bot35-conflicting-parent",
		Instance:               &v1pb.Instance{},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}

func createProjectForProjectInstanceTest(ctx context.Context, t *testing.T, ctl *controller, projectID string) *v1pb.Project {
	t.Helper()
	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project:   &v1pb.Project{Title: projectID, AllowSelfApproval: true},
		ProjectId: projectID,
	}))
	require.NoError(t, err)
	return project.Msg
}

func createProjectInstanceTestInstance(ctx context.Context, t *testing.T, ctl *controller, parent *string, instanceID, title string, pg *Container) *v1pb.Instance {
	t.Helper()
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		Parent:     parent,
		InstanceId: instanceID,
		Instance:   projectInstanceTestSpec(title, pg),
	}))
	require.NoError(t, err)
	return instance.Msg
}

func projectInstanceTestSpec(title string, pg *Container) *v1pb.Instance {
	return &v1pb.Instance{
		Title:       title,
		Engine:      v1pb.Engine_POSTGRES,
		Environment: new("environments/prod"),
		Activation:  true,
		DataSources: []*v1pb.DataSource{pg.adminDataSource()},
	}
}

func projectInstanceTestInstanceNames(instances []*v1pb.Instance) []string {
	names := make([]string, 0, len(instances))
	for _, instance := range instances {
		names = append(names, instance.Name)
	}
	return names
}
