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

type createDatabaseGroupInstance struct {
	instanceTitle        string
	matchedDatabasesName map[string]any
}

type createDatabaseGroupCase struct {
	databaseGroupPlaceholder string
	databaseGroupExpr        string
	prepareInstances         []createDatabaseGroupInstance
}

// testCreateDatabaseGroup creates databases and verifies the grouping result:
// 1. It provisions one Postgres instance per prepareInstances entry and creates the listed databases on it.
// 2. It creates the database group with the given expr.
// 3. It compares the matched databases against prepareInstances; they should be consistent.
//
// The cases are top-level tests rather than subtests so that they enter the
// package's parallel queue with everything else. Parallel subtests only get a
// slot once every top-level test has started, which put these 17 s cases at
// the very end of the package.
func testCreateDatabaseGroup(t *testing.T, tc createDatabaseGroupCase) {
	a := require.New(t)
	ctl := &controller{}
	ctx := context.Background()
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer func() {
		_ = ctl.Close(ctx)
	}()

	instanceResourceID2InstanceTitle := make(map[string]string)
	for _, prepareInstance := range tc.prepareInstances {
		pgContainer, err := provisionPgInstance(ctx, t)
		a.NoError(err)
		instanceResourceID := generateRandomString("instance")
		instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
			InstanceId: instanceResourceID,
			Instance: &v1pb.Instance{
				Title:       prepareInstance.instanceTitle,
				Engine:      v1pb.Engine_POSTGRES,
				Environment: new("environments/prod"),
				DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
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
}

func TestCreateDatabaseGroup_AllMatchedOneInstance(t *testing.T) {
	t.Parallel()
	testCreateDatabaseGroup(t, createDatabaseGroupCase{
		databaseGroupPlaceholder: "all-matched-one-instance",
		databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
		prepareInstances: []createDatabaseGroupInstance{
			{
				instanceTitle: "TestCreateDatabaseGroups_AllMatched_OneInstance",
				matchedDatabasesName: map[string]any{
					"employee_01": nil,
					"employee_02": nil,
				},
			},
		},
	})
}

func TestCreateDatabaseGroup_PartialMatchedOneInstance(t *testing.T) {
	t.Parallel()
	testCreateDatabaseGroup(t, createDatabaseGroupCase{
		databaseGroupPlaceholder: "partial-matched-one-instance",
		databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
		prepareInstances: []createDatabaseGroupInstance{
			{
				instanceTitle: "TestCreateDatabaseGroups_PartialMatched_OneInstance",
				matchedDatabasesName: map[string]any{
					"employee_01": nil,
					"employee_02": nil,
				},
			},
		},
	})
}

func TestCreateDatabaseGroup_AllMatchedManyInstances(t *testing.T) {
	t.Parallel()
	testCreateDatabaseGroup(t, createDatabaseGroupCase{
		databaseGroupPlaceholder: "all-matched-many-instances",
		databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
		prepareInstances: []createDatabaseGroupInstance{
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
	})
}

func TestCreateDatabaseGroup_PartialMatchedManyInstances(t *testing.T) {
	t.Parallel()
	testCreateDatabaseGroup(t, createDatabaseGroupCase{
		databaseGroupPlaceholder: "partial-matched-many-instances",
		databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
		prepareInstances: []createDatabaseGroupInstance{
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
	})
}
