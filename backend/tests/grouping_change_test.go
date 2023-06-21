package tests

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// TestCreateDatabaseGroup tests creating database and verify the grouping result.
func TestCreateDatabaseGroups(t *testing.T) {
	testCases := []struct {
		name                     string
		databaseGroupPlaceholder string
		databaseGroupNameExpr    string
		prepareInstances         []struct {
			instanceTitle         string
			matchedDatabasesName  map[string]any
			unmatchedDatabaseName map[string]any
		}
	}{
		{
			name:                     "all-matched-one-instance",
			databaseGroupPlaceholder: "all-matched-one-instance",
			databaseGroupNameExpr:    `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []struct {
				instanceTitle         string
				matchedDatabasesName  map[string]any
				unmatchedDatabaseName map[string]any
			}{
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_OneInstance",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
					unmatchedDatabaseName: map[string]any{},
				},
			},
		},
		{
			name:                     "partial-matched-one-instance",
			databaseGroupPlaceholder: "partial-matched-one-instance",
			databaseGroupNameExpr:    `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []struct {
				instanceTitle         string
				matchedDatabasesName  map[string]any
				unmatchedDatabaseName map[string]any
			}{
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_OneInstance",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
					unmatchedDatabaseName: map[string]any{
						"hello": nil,
						"world": nil,
					},
				},
			},
		},
		{
			name:                     "all-matched-many-instances",
			databaseGroupPlaceholder: "all-matched-many-instances",
			databaseGroupNameExpr:    `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []struct {
				instanceTitle         string
				matchedDatabasesName  map[string]any
				unmatchedDatabaseName map[string]any
			}{
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_ManyInstances_01",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
					unmatchedDatabaseName: map[string]any{},
				},
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_ManyInstances_02",
					matchedDatabasesName: map[string]any{
						"employee_02": nil,
						"employee_03": nil,
						"employee_04": nil,
					},
					unmatchedDatabaseName: map[string]any{},
				},
			},
		},
		{
			name:                     "partial-matched-many-instances",
			databaseGroupPlaceholder: "partial-matched-many-instances",
			databaseGroupNameExpr:    `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []struct {
				instanceTitle         string
				matchedDatabasesName  map[string]any
				unmatchedDatabaseName map[string]any
			}{
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_ManyInstances_01",
					matchedDatabasesName: map[string]any{
						"employee_01": nil,
						"employee_02": nil,
					},
					unmatchedDatabaseName: map[string]any{
						"hello": nil,
						"world": nil,
					},
				},
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_ManyInstances_02",
					matchedDatabasesName: map[string]any{
						"employee_02": nil,
						"employee_03": nil,
						"employee_04": nil,
					},
					unmatchedDatabaseName: map[string]any{
						"hello": nil,
						"world": nil,
					},
				},
			},
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctl := &controller{}
			ctx := context.Background()
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: fake.NewGitLab,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()
			err = ctl.setLicense()
			a.NoError(err)

			prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
			a.NoError(err)

			project, err := ctl.createProject(ctx)
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			instanceResourceID2InstanceTitle := make(map[string]string)
			for _, prepareInstance := range tc.prepareInstances {
				instanceDir, err := ctl.provisionSQLiteInstance(t.TempDir(), t.Name())
				a.NoError(err)
				instanceResourceID := generateRandomString("instance", 10)
				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: instanceResourceID,
					Instance: &v1pb.Instance{
						Title:       prepareInstance.instanceTitle,
						Engine:      v1pb.Engine_SQLITE,
						Environment: prodEnvironment.Name,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
					},
				})
				a.NoError(err)
				instanceResourceID2InstanceTitle[instanceResourceID] = instance.Title
				for preCreateDatabase := range prepareInstance.matchedDatabasesName {
					err = ctl.createDatabase(ctx, projectUID, instance, preCreateDatabase, "", nil /* labelMap */)
				}
				for preCreateDatabase := range prepareInstance.unmatchedDatabaseName {
					err = ctl.createDatabase(ctx, projectUID, instance, preCreateDatabase, "", nil /* labelMap */)
				}
			}
			databaseGroup, err := ctl.projectServiceClient.CreateDatabaseGroup(ctx, &v1pb.CreateDatabaseGroupRequest{
				Parent:          project.Name,
				DatabaseGroupId: tc.databaseGroupPlaceholder,
				DatabaseGroup: &v1pb.DatabaseGroup{
					DatabasePlaceholder: tc.databaseGroupPlaceholder,
					DatabaseExpr: &expr.Expr{
						Expression: fmt.Sprintf(`(resource.environment_name == "environments/prod" && (%s))`, tc.databaseGroupNameExpr),
					},
				},
			})
			a.NoError(err)
			databaseGroup, err = ctl.projectServiceClient.GetDatabaseGroup(ctx, &v1pb.GetDatabaseGroupRequest{
				Name: databaseGroup.Name,
				View: v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL,
			})
			a.NoError(err)

			gotInstanceTitleToMatchedDatabases := make(map[string][]string)
			gotInstanceTitleToUnmatchedDatabases := make(map[string][]string)
			for _, matchedDatabase := range databaseGroup.MatchedDatabases {
				instanceResourceID := strings.Split(matchedDatabase.Name, "/")[1]
				instanceTitle := instanceResourceID2InstanceTitle[instanceResourceID]
				a.NotEmpty(instanceTitle)

				databaseName := strings.Split(matchedDatabase.Name, "/")[3]
				gotInstanceTitleToMatchedDatabases[instanceTitle] = append(gotInstanceTitleToMatchedDatabases[instanceTitle], databaseName)
			}
			for _, unmatchedDatabase := range databaseGroup.UnmatchedDatabases {
				instanceResourceID := strings.Split(unmatchedDatabase.Name, "/")[1]
				instanceTitle := instanceResourceID2InstanceTitle[instanceResourceID]
				a.NotEmpty(instanceTitle)

				databaseName := strings.Split(unmatchedDatabase.Name, "/")[3]
				gotInstanceTitleToUnmatchedDatabases[instanceTitle] = append(gotInstanceTitleToUnmatchedDatabases[instanceTitle], databaseName)
			}

			for _, prepareInstance := range tc.prepareInstances {
				gotMatchedDatabases := gotInstanceTitleToMatchedDatabases[prepareInstance.instanceTitle]
				gotUnmatchedDatabases := gotInstanceTitleToUnmatchedDatabases[prepareInstance.instanceTitle]
				a.Equal(len(gotMatchedDatabases), len(prepareInstance.matchedDatabasesName))
				a.Equal(len(gotUnmatchedDatabases), len(prepareInstance.unmatchedDatabaseName))

				for wantMatchedDatabase := range prepareInstance.matchedDatabasesName {
					a.Contains(gotMatchedDatabases, wantMatchedDatabase)
				}
				for wantUnmatchedDatabase := range prepareInstance.unmatchedDatabaseName {
					a.Contains(gotUnmatchedDatabases, wantUnmatchedDatabase)
				}
			}
		})
	}
}
