package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

var (
	testTenantNumber = 1
	prodTenantNumber = 1
	testInstanceName = "testInstanceTest"
	prodInstanceName = "testInstanceProd"
)

func TestDatabaseGroup(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project.
	projectID := generateRandomString("project")
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

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
		instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance"),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
				Engine:      v1pb.Engine_SQLITE,
				Environment: stringPtr("environments/test"),
				Activation:  true,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
			},
		}))
		a.NoError(err)
		testInstances = append(testInstances, instanceResp.Msg)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance"),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", prodInstanceName, i),
				Engine:      v1pb.Engine_SQLITE,
				Environment: stringPtr("environments/prod"),
				Activation:  true,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir, Id: "admin"}},
			},
		}))
		a.NoError(err)
		prodInstances = append(prodInstances, instanceResp.Msg)
	}

	// Create issues that create databases.
	databaseName := "testTenantSchemaUpdate"
	for _, testInstance := range testInstances {
		err := ctl.createDatabase(ctx, project, testInstance, nil, databaseName, "")
		a.NoError(err)
	}
	for _, prodInstance := range prodInstances {
		err := ctl.createDatabase(ctx, project, prodInstance, nil, databaseName, "")
		a.NoError(err)
	}

	resp, err := ctl.databaseServiceClient.ListDatabases(ctx, connect.NewRequest(&v1pb.ListDatabasesRequest{
		Parent: project.Name,
	}))
	a.NoError(err)
	databases := resp.Msg.Databases

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

	databaseGroupResp, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
		Parent:          project.Name,
		DatabaseGroupId: "all",
		DatabaseGroup: &v1pb.DatabaseGroup{
			Title:        "all",
			DatabaseExpr: &expr.Expr{Expression: "true"},
		},
	}))
	a.NoError(err)
	databaseGroup := databaseGroupResp.Msg

	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(migrationStatement1),
		},
	}))
	a.NoError(err)
	sheet := sheetResp.Msg

	// Create an issue that updates database schema.
	spec := &v1pb.Plan_Spec{
		Id: uuid.NewString(),
		Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
				Targets:     []string{databaseGroup.Name},
				Sheet:       sheet.Name,
				EnableGhost: false,
			},
		},
	}
	plan, rollout, issue, err := ctl.changeDatabaseWithConfig(ctx, project, spec)
	a.NoError(err)

	// Query schema.
	for _, testInstance := range testInstances {
		dbMetadataResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName)}))
		a.NoError(err)
		a.Equal(wantBookSchema, dbMetadataResp.Msg.Schema)
	}
	for _, prodInstance := range prodInstances {
		dbMetadataResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Name, databaseName)}))
		a.NoError(err)
		a.Equal(wantBookSchema, dbMetadataResp.Msg.Schema)
	}

	// Create another database in the prod environment.
	databaseName2 := "testTenantSchemaUpdate2"
	err = ctl.createDatabase(ctx, project, prodInstances[0], nil, databaseName2, "")
	a.NoError(err)

	resp, err = ctl.databaseServiceClient.ListDatabases(ctx, connect.NewRequest(&v1pb.ListDatabasesRequest{
		Parent: project.Name,
	}))
	a.NoError(err)
	databases = resp.Msg.Databases
	prodDatabases = nil
	for _, prodInstance := range prodInstances {
		for _, database := range databases {
			if strings.HasPrefix(database.Name, prodInstance.Name) {
				prodDatabases = append(prodDatabases, database)
			}
		}
	}
	a.Len(prodDatabases, 2)

	// CreateRollout is now idempotent and will automatically pick up the new database.
	rollout2Resp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: plan.Name,
		Target: nil, // set to nil to create all stages and tasks.
	}))
	a.NoError(err)
	rollout2 := rollout2Resp.Msg
	a.Equal(rollout.Name, rollout2.Name)

	a.Len(rollout.Stages, 2)
	a.Len(rollout2.Stages, 2)
	a.Len(rollout.Stages[1].Tasks, 1)
	a.Len(rollout2.Stages[1].Tasks, 2)
	// The task for databaseName2 should be created.
	a.Equal(rollout2.Stages[1].Tasks[1].Target, fmt.Sprintf("%s/databases/%s", prodInstances[0].Name, databaseName2))

	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	a.NoError(err)
}
