package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestProjectInstanceLifecycleAPIGatesArchivedProjectDescendants(t *testing.T) {
	ctx, stores, projectID, instanceID, databaseName := setupProjectInstanceLifecycleAPITest(t)
	licenseService := &instanceLicenseServiceStub{instanceLimit: 10, activatedInstanceLimit: 10}
	databaseLicenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	projectService := NewProjectService(stores, nil, nil)
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
	require.True(t, undeleted.Msg.Activation)
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
	require.True(t, undeleted.Msg.Activation)

	_, err = projectService.UndeleteProject(ctx, connect.NewRequest(&v1pb.UndeleteProjectRequest{Name: projectName}))
	require.NoError(t, err)
	got, err := instanceService.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	require.True(t, got.Msg.Activation)
}

func TestUndeleteProjectInstanceChecksActivationLimit(t *testing.T) {
	ctx, stores, projectID, instanceID, _ := setupProjectInstanceLifecycleAPITest(t)
	licenseService := &instanceLicenseServiceStub{instanceLimit: 10, activatedInstanceLimit: 1}
	instanceService := &InstanceService{store: stores, licenseService: licenseService}
	instanceName := common.FormatProjectInstance(projectID, instanceID)

	_, err := instanceService.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)

	otherProjectID := "project-b"
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "other-project-instance",
		Workspace:  "default",
		ProjectID:  &otherProjectID,
		Metadata: &storepb.Instance{
			Activation: true,
			DataSources: []*storepb.DataSource{{
				Id:   "admin",
				Type: storepb.DataSourceType_ADMIN,
			}},
		},
	})
	require.NoError(t, err)

	_, err = instanceService.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: instanceName}))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	instance, err := getInstanceMessageForLifecycle(ctx, stores, instanceName)
	require.NoError(t, err)
	require.True(t, instance.Deleted)
	require.True(t, instance.Metadata.GetActivation())
}

type instanceLicenseServiceStub struct {
	instanceLimit          int
	activatedInstanceLimit int
}

func (s *instanceLicenseServiceStub) GetActivatedInstanceLimit(context.Context, string) int {
	return s.activatedInstanceLimit
}

func (s *instanceLicenseServiceStub) GetInstanceLimit(context.Context, string) int {
	return s.instanceLimit
}

func (*instanceLicenseServiceStub) IsFeatureEnabledForInstance(context.Context, string, v1pb.PlanFeature, *store.InstanceMessage) error {
	return nil
}

func (s *instanceLicenseServiceStub) IsInstanceEffectivelyActivated(_ context.Context, _ string, instance *store.InstanceMessage) bool {
	return instance.Metadata.GetActivation() || s.instanceLimit <= s.activatedInstanceLimit
}

func (s *instanceLicenseServiceStub) IsUnifiedInstanceLicense(context.Context, string) bool {
	return s.instanceLimit <= s.activatedInstanceLimit
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
