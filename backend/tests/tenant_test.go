package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	testTenantNumber = 1
	prodTenantNumber = 1
	testInstanceName = "testInstanceTest"
	prodInstanceName = "testInstanceProd"
)

const baseDirectory = "bbtest"

func TestTenant(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project.
	projectID := generateRandomString("project", 10)
	project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:       fmt.Sprintf("projects/%s", projectID),
			Title:      projectID,
			Key:        projectID,
			TenantMode: v1pb.TenantMode_TENANT_MODE_ENABLED,
		},
		ProjectId: projectID,
	})
	a.NoError(err)

	// Provision instances.
	instanceRootDir := t.TempDir()

	var testInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < testTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", testInstanceName, i))
		a.NoError(err)
		testInstanceDirs = append(testInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		a.NoError(err)
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}

	// Add the provisioned instances.
	var testInstances []*v1pb.Instance
	var prodInstances []*v1pb.Instance
	for i, testInstanceDir := range testInstanceDirs {
		instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance", 10),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
				Engine:      v1pb.Engine_SQLITE,
				Environment: "environments/test",
				Activation:  true,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
			},
		})
		a.NoError(err)
		testInstances = append(testInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance", 10),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", prodInstanceName, i),
				Engine:      v1pb.Engine_SQLITE,
				Environment: "environments/prod",
				Activation:  true,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir, Id: "admin"}},
			},
		})
		a.NoError(err)
		prodInstances = append(prodInstances, instance)
	}

	// Create deployment configuration.
	_, err = ctl.projectServiceClient.UpdateDeploymentConfig(ctx, &v1pb.UpdateDeploymentConfigRequest{
		Config: &v1pb.DeploymentConfig{
			Name:     common.FormatDeploymentConfig(project.Name),
			Schedule: deploySchedule,
		},
	})
	a.NoError(err)

	// Create issues that create databases.
	databaseName := "testTenantSchemaUpdate"
	for i, testInstance := range testInstances {
		err := ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabaseV2(ctx, project, prodInstance, nil, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	resp, err := ctl.databaseServiceClient.ListDatabases(ctx, &v1pb.ListDatabasesRequest{
		Parent: "instances/-",
		Filter: fmt.Sprintf(`project == "%s"`, project.Name),
	})
	a.NoError(err)
	databases := resp.Databases

	var testDatabases []*v1pb.Database
	var prodDatabases []*v1pb.Database
	for _, testInstance := range testInstances {
		for _, database := range databases {
			if strings.HasPrefix(database.Name, testInstance.Name) {
				testDatabases = append(testDatabases, database)
				break
			}
		}
	}
	for _, prodInstance := range prodInstances {
		for _, database := range databases {
			if strings.HasPrefix(database.Name, prodInstance.Name) {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}

	a.Equal(testTenantNumber, len(testDatabases))
	a.Equal(prodTenantNumber, len(prodDatabases))

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "migration statement sheet",
			Content: []byte(migrationStatement1),
		},
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	step := &v1pb.Plan_Step{
		Specs: []*v1pb.Plan_Spec{
			{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Target: fmt.Sprintf("%s/deploymentConfigs/default", project.Name),
						Sheet:  sheet.Name,
						Type:   v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
					},
				},
			},
		},
	}
	_, _, _, err = ctl.changeDatabaseWithConfig(ctx, project, []*v1pb.Plan_Step{step})
	a.NoError(err)

	// Query schema.
	for _, testInstance := range testInstances {
		dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName)})
		a.NoError(err)
		a.Equal(wantBookSchema, dbMetadata.Schema)
	}
	for _, prodInstance := range prodInstances {
		dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Name, databaseName)})
		a.NoError(err)
		a.Equal(wantBookSchema, dbMetadata.Schema)
	}
}
