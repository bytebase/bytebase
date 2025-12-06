package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCreateDatabaseGroup tests creating database and verify the grouping result.
// For each test case:
// 1. The test provides a number of sqlite instances equal to the number of prepareInstances and creates the specified matchDatabase and unmatchedDatabase in the corresponding instances.
// 2. The database group is then created with the specified expr.
// 3. The results obtained are compared with the results given in prepareInstance and they should be consistent.
func TestCreateDatabaseGroup(t *testing.T) {
	t.Parallel()
	type testCasePrepareInstance struct {
		instanceTitle        string
		matchedDatabasesName map[string]any
	}
	testCases := []struct {
		name                     string
		databaseGroupPlaceholder string
		databaseGroupExpr        string
		prepareInstances         []testCasePrepareInstance
	}{
		{
			name:                     "all-matched-one-instance",
			databaseGroupPlaceholder: "all-matched-one-instance",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_OneInstance",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
				},
			},
		},
		{
			name:                     "partial-matched-one-instance",
			databaseGroupPlaceholder: "partial-matched-one-instance",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_OneInstance",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
				},
			},
		},
		{
			name:                     "all-matched-many-instances",
			databaseGroupPlaceholder: "all-matched-many-instances",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_ManyInstances_01",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
				},
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_ManyInstances_02",
					matchedDatabasesName: map[string]any{
						"employee_02": nil,
						"employee_03": nil,
						"employee_04": nil,
					},
				},
			},
		},
		{
			name:                     "partial-matched-many-instances",
			databaseGroupPlaceholder: "partial-matched-many-instances",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_ManyInstances_01",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
				},
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_ManyInstances_02",
					matchedDatabasesName: map[string]any{
						"employee_02": nil,
						"employee_03": nil,
						"employee_04": nil,
					},
				},
			},
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctl := &controller{}
			ctx := context.Background()
			ctx, err := ctl.StartServerWithExternalPg(ctx)
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			instanceResourceID2InstanceTitle := make(map[string]string)
			for _, prepareInstance := range tc.prepareInstances {
				instanceDir, err := ctl.provisionSQLiteInstance(t.TempDir(), t.Name())
				a.NoError(err)
				instanceResourceID := generateRandomString("instance")
				instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
					InstanceId: instanceResourceID,
					Instance: &v1pb.Instance{
						Title:       prepareInstance.instanceTitle,
						Engine:      v1pb.Engine_SQLITE,
						Environment: stringPtr("environments/prod"),
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
						Activation:  true,
					},
				}))
				a.NoError(err)
				instance := instanceResp.Msg
				instanceResourceID2InstanceTitle[instanceResourceID] = instance.Title
				for preCreateDatabase := range prepareInstance.matchedDatabasesName {
					err = ctl.createDatabase(ctx, ctl.project, instance, nil, preCreateDatabase, "")
					a.NoError(err)
				}
			}
			databaseGroupResp, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
				Parent:          ctl.project.Name,
				DatabaseGroupId: tc.databaseGroupPlaceholder,
				DatabaseGroup: &v1pb.DatabaseGroup{
					Title: tc.databaseGroupPlaceholder,
					DatabaseExpr: &expr.Expr{
						Expression: fmt.Sprintf(`(resource.environment_id == "prod" && (%s))`, tc.databaseGroupExpr),
					},
				},
			}))
			a.NoError(err)
			databaseGroup := databaseGroupResp.Msg
			databaseGroupResp, err = ctl.databaseGroupServiceClient.GetDatabaseGroup(ctx, connect.NewRequest(&v1pb.GetDatabaseGroupRequest{
				Name: databaseGroup.Name,
				View: v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL,
			}))
			a.NoError(err)
			databaseGroup = databaseGroupResp.Msg

			gotInstanceTitleToMatchedDatabases := make(map[string][]string)

			for _, matchedDatabase := range databaseGroup.MatchedDatabases {
				instanceResourceID := strings.Split(matchedDatabase.Name, "/")[1]
				instanceTitle := instanceResourceID2InstanceTitle[instanceResourceID]
				a.NotEmpty(instanceTitle)

				databaseName := strings.Split(matchedDatabase.Name, "/")[3]
				gotInstanceTitleToMatchedDatabases[instanceTitle] = append(gotInstanceTitleToMatchedDatabases[instanceTitle], databaseName)
			}

			for _, prepareInstance := range tc.prepareInstances {
				gotMatchedDatabases := gotInstanceTitleToMatchedDatabases[prepareInstance.instanceTitle]
				a.Equal(len(gotMatchedDatabases), len(prepareInstance.matchedDatabasesName))
				for wantMatchedDatabase := range prepareInstance.matchedDatabasesName {
					a.Contains(gotMatchedDatabases, wantMatchedDatabase)
				}
			}
		})
	}
}
