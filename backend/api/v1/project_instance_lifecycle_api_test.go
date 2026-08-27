package v1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sample"
	"github.com/bytebase/bytebase/backend/component/sample/selfhost"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestProjectInstanceLifecycleAPIGatesArchivedProjectDescendants(t *testing.T) {
	ctx, stores, projectID, instanceID, databaseName := setupProjectInstanceLifecycleAPITest(t)
	_, err := stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instanceID,
		Workspace:  "default",
		Metadata: &storepb.Instance{
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	licenseService := newInstanceServiceTestLicenseService(t, stores)
	databaseLicenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	projectService := NewProjectService(stores, nil, nil, nil)
	instanceService := &InstanceService{store: stores, licenseService: licenseService}
	databaseService := NewDatabaseService(stores, nil, nil, nil, databaseLicenseService)
	instanceName := common.FormatProjectInstance(projectID, instanceID)
	projectName := common.FormatProject(projectID)
	databaseResourceName := common.FormatProjectDatabase(projectID, instanceID, databaseName)

	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
		Name:  common.FormatProject("project-b"),
		Purge: true,
	}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	_, err = instanceService.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	undeleted, err := instanceService.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	require.False(t, undeleted.Msg.Activation)
	_, err = instanceService.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceName, Force: true}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName}))
	require.NoError(t, err)

	_, err = instanceService.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceName}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = databaseService.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseResourceName}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = resolveRawResource(ctx, stores, databaseResourceName)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	resources, err := populateRawResources(ctx, stores, &v1pb.UndeleteInstanceRequest{Name: instanceName}, v1connect.InstanceServiceUndeleteInstanceProcedure)
	require.NoError(t, err)
	require.Equal(t, []*common.Resource{{Type: common.ResourceTypeProject, ID: projectID}}, resources)

	var databaseCount int
	require.NoError(t, stores.GetDB().QueryRowContext(ctx, "SELECT COUNT(*) FROM db WHERE instance = $1 AND name = $2", instanceID, databaseName).Scan(&databaseCount))
	require.Equal(t, 1, databaseCount)

	_, err = instanceService.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	undeleted, err = instanceService.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	require.False(t, undeleted.Msg.Activation)

	_, err = projectService.UndeleteProject(ctx, connect.NewRequest(&v1pb.UndeleteProjectRequest{Name: projectName}))
	require.NoError(t, err)
	got, err := instanceService.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	require.False(t, got.Msg.Activation)
}

func TestProjectPurgeCleansSampleBeforeDeletingMetadata(t *testing.T) {
	ctx, stores, projectID, instanceID, _ := setupProjectInstanceLifecycleAPITest(t)
	manager := &sampleManagerStub{}
	manager.projectPurge = func(ctx context.Context, workspaceID, gotProjectID string) error {
		require.Equal(t, "default", workspaceID)
		require.Equal(t, projectID, gotProjectID)
		instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
			Workspace:  workspaceID,
			ProjectID:  &gotProjectID,
			ResourceID: &instanceID,
		})
		require.NoError(t, err)
		require.NotNil(t, instance)
		return nil
	}
	projectService := NewProjectService(stores, nil, nil, manager)
	projectName := common.FormatProject(projectID)

	_, err := projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName}))
	require.NoError(t, err)
	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName, Purge: true}))
	require.NoError(t, err)
	require.Equal(t, []string{projectID}, manager.projectPurgeCalls)

	project, err := stores.GetProject(ctx, &store.FindProjectMessage{
		Workspace:   "default",
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
	require.NoError(t, err)
	require.Nil(t, project)
}

func TestProjectPurgeRemovesSelfHostSample(t *testing.T) {
	ctx, stores, projectID, instanceID, databaseName := setupProjectInstanceLifecycleAPITest(t)
	payload, err := protojson.Marshal(&storepb.SelfHostSampleInstanceSetupPayload{
		Instances: []*storepb.SelfHostSampleInstanceSetupPayload_Instance{{
			InstanceId:   instanceID,
			ProjectId:    &projectID,
			Title:        "Sample Project Instance",
			DatabaseName: databaseName,
			RoleName:     "sample-role",
		}},
	})
	require.NoError(t, err)
	const replicaID = "replica-a"
	_, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: "default",
		ReplicaID:   replicaID,
		Payload:     payload,
	})
	require.NoError(t, err)
	require.True(t, created)
	activated, err := stores.ActivateSampleInstanceSetup(ctx, "default", replicaID, []string{projectID}, time.Now(), nil)
	require.NoError(t, err)
	require.True(t, activated)

	dataRoot := t.TempDir()
	dataDir := filepath.Join(dataRoot, "pgdata-sample-managed", instanceID)
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "sample-data"), []byte("sample"), 0o644))
	manager := selfhost.NewManager(
		stores,
		&config.Profile{DataDir: dataRoot, Port: 8080},
		nil,
		sample.ManagerOptions{ReplicaID: replicaID},
	)
	projectService := NewProjectService(stores, nil, nil, manager)
	projectName := common.FormatProject(projectID)

	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName}))
	require.NoError(t, err)
	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName, Purge: true}))
	require.NoError(t, err)

	_, err = os.Stat(dataDir)
	require.ErrorIs(t, err, os.ErrNotExist)
	setup, err := stores.GetSampleInstanceSetup(ctx, "default")
	require.NoError(t, err)
	require.NotNil(t, setup)
	require.NotNil(t, setup.DeletedAt)
	instances, err := manager.ListInstances(ctx, "default")
	require.NoError(t, err)
	require.Empty(t, instances)
}

func TestUndeleteProjectInstanceChecksActivationLimit(t *testing.T) {
	ctx, stores, projectID, instanceID, _ := setupProjectInstanceLifecycleAPITest(t)
	licenseService := newInstanceServiceTestLicenseService(t, stores)
	instanceService := &InstanceService{store: stores, licenseService: licenseService}
	instanceName := common.FormatProjectInstance(projectID, instanceID)

	_, err := instanceService.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)

	_, err = instanceService.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: instanceName}))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	instance, err := getInstanceMessageForLifecycle(ctx, stores, instanceName)
	require.NoError(t, err)
	require.True(t, instance.Deleted)
	require.True(t, instance.Metadata.GetActivation())
}

func newInstanceServiceTestLicenseService(t *testing.T, stores *store.Store) *enterprise.LicenseService {
	t.Helper()
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	return licenseService
}

func setupProjectInstanceLifecycleAPITest(t *testing.T) (context.Context, *store.Store, string, string, string) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-a', 'default', 'Project A'),
			('project-b', 'default', 'Project B');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	projectID := "project-a"
	instanceID := "project-instance"
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: instanceID,
		Workspace:  "default",
		ProjectID:  &projectID,
		Metadata: &storepb.Instance{
			Activation: true,
			DataSources: []*storepb.DataSource{{
				Id:   "admin",
				Type: storepb.DataSourceType_ADMIN,
			}},
		},
	})
	require.NoError(t, err)

	databaseName := "app"
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		ProjectID:    projectID,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	return ctx, stores, projectID, instanceID, databaseName
}
