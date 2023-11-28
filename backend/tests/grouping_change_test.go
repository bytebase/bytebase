package tests

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/tests/fake"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// TestCreateDatabaseGroup tests creating database and verify the grouping result.
// For each test case:
// 1. The test provides a number of sqlite instances equal to the number of prepareInstances and creates the specified matchDatabase and unmatchedDatabase in the corresponding instances.
// 2. The database group is then created with the specified expr.
// 3. The results obtained are compared with the results given in prepareInstance and they should be consistent.
func TestCreateDatabaseGroup(t *testing.T) {
	type testCasePrepareInstance struct {
		instanceTitle         string
		matchedDatabasesName  map[string]any
		unmatchedDatabaseName map[string]any
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
					unmatchedDatabaseName: map[string]any{},
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
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []testCasePrepareInstance{
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
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			prepareInstances: []testCasePrepareInstance{
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
			t.Parallel()
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
						Environment: "environments/prod",
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
						Activation:  true,
					},
				})
				a.NoError(err)
				instanceResourceID2InstanceTitle[instanceResourceID] = instance.Title
				for preCreateDatabase := range prepareInstance.matchedDatabasesName {
					err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, preCreateDatabase, "", nil /* labelMap */)
					a.NoError(err)
				}
				for preCreateDatabase := range prepareInstance.unmatchedDatabaseName {
					err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, preCreateDatabase, "", nil /* labelMap */)
					a.NoError(err)
				}
			}
			databaseGroup, err := ctl.projectServiceClient.CreateDatabaseGroup(ctx, &v1pb.CreateDatabaseGroupRequest{
				Parent:          ctl.project.Name,
				DatabaseGroupId: tc.databaseGroupPlaceholder,
				DatabaseGroup: &v1pb.DatabaseGroup{
					DatabasePlaceholder: tc.databaseGroupPlaceholder,
					DatabaseExpr: &expr.Expr{
						Expression: fmt.Sprintf(`(resource.environment_name == "environments/prod" && (%s))`, tc.databaseGroupExpr),
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

// TestCreateTableGroup tests create table group and verify the grouping result.
// For each test case:
// 1. The test provides a number of sqlite instances equal to the number of prepareInstances, creates the specified matchedDatabase and the specified matchedTable and unmatchedTable.
// 2. The table group and database group is then created with the specified expr.
// 3. The results obtained are compared with the results given in prepareInstances and they should be consistent.
// NOTE, matched tables and unmatched tables are all in the matched database, so FOCUS THE UNMATCHED DATABASE IS MEANINGLESS IN THIS TEST(i.e. this test should not contains the unmatched databases in test case and logic).
func TestCreateTableGroup(t *testing.T) {
	type tableNames struct {
		matched   []string
		unmatched []string
	}
	type testCasePrepareInstance struct {
		instanceTitle               string
		matchDatabasesNameTableName map[string]tableNames
	}
	testCases := []struct {
		name                     string
		databaseGroupPlaceholder string
		databaseGroupExpr        string
		tableGroupPlaceholder    string
		tableGroupExpr           string
		prepareInstances         []testCasePrepareInstance
	}{
		{
			name:                     "all-matched-one-instance",
			databaseGroupPlaceholder: "all-matched-one-instance",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			tableGroupPlaceholder:    `all-matched-one-instance-salary`,
			tableGroupExpr:           `(resource.table_name.startsWith("salary_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_OneInstance",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_01": {
							matched: []string{"salary_01", "salary_02"},
						},
						"employee_02": {
							matched: []string{"salary_03", "salary_04"},
						},
					},
				},
			},
		},
		{
			name:                     "partial-matched-one-instance",
			databaseGroupPlaceholder: "partial-matched-one-instance",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			tableGroupPlaceholder:    `salary`,
			tableGroupExpr:           `(resource.table_name.startsWith("salary_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_OneInstance",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_01": {
							matched:   []string{"salary_01", "salary_02"},
							unmatched: []string{"person_01", "person_02"},
						},
						"employee_02": {
							matched:   []string{"salary_03", "salary_04"},
							unmatched: []string{"person_03", "person_04"},
						},
						"employee_03": {
							unmatched: []string{"person_05", "person_06"},
						},
					},
				},
			},
		},
		{
			name:                     "all-matched-many-instances",
			databaseGroupPlaceholder: "all-matched-many-instances",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			tableGroupPlaceholder:    `salary`,
			tableGroupExpr:           `(resource.table_name.startsWith("salary_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_ManyInstances_01",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_01": {
							matched: []string{"salary_01", "salary_02"},
						},
						"employee_02": {
							matched: []string{"salary_03", "salary_04"},
						},
					},
				},
				{
					instanceTitle: "TestCreateDatabaseGroups_AllMatched_ManyInstances_02",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_03": {
							matched: []string{"salary_05", "salary_06"},
						},
						"employee_04": {
							matched: []string{"salary_07", "salary_08"},
						},
						"employee_05": {
							matched: []string{"salary_09", "salary_10"},
						},
					},
				},
			},
		},
		{
			name:                     "partial-matched-many-instances",
			databaseGroupPlaceholder: "partial-matched-many-instances",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			tableGroupPlaceholder:    `salary`,
			tableGroupExpr:           `(resource.table_name.startsWith("salary_"))`,
			prepareInstances: []testCasePrepareInstance{
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_ManyInstances_01",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_01": {
							matched:   []string{"salary_01", "salary_02"},
							unmatched: []string{"person_01", "person_02"},
						},
						"employee_02": {
							matched:   []string{"salary_03", "salary_04"},
							unmatched: []string{"person_03", "person_04"},
						},
					},
				},
				{
					instanceTitle: "TestCreateDatabaseGroups_PartialMatched_ManyInstances_02",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_03": {
							matched:   []string{"salary_05", "salary_06"},
							unmatched: []string{"person_05", "person_06"},
						},
						"employee_04": {
							matched:   []string{"salary_07", "salary_08"},
							unmatched: []string{"person_07", "person_08"},
						},
						"employee_05": {
							matched:   []string{"salary_09", "salary_10"},
							unmatched: []string{"person_09", "person_10"},
						},
						"employee_06": {
							unmatched: []string{"person_11", "person_12"},
						},
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: fake.NewGitLab,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

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
						Environment: "environments/prod",
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
						Activation:  true,
					},
				})
				a.NoError(err)
				instanceResourceID2InstanceTitle[instanceResourceID] = instance.Title
				for preCreateDatabase := range prepareInstance.matchDatabasesNameTableName {
					err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, preCreateDatabase, "", nil /* labelMap */)
					a.NoError(err)
					dbDriver, err := db.Open(ctx, storepb.Engine_SQLITE, db.DriverConfig{}, db.ConnectionConfig{
						Host:     instanceDir,
						Database: preCreateDatabase,
					}, db.ConnectionContext{})
					a.NoError(err)

					for _, preCreateTable := range prepareInstance.matchDatabasesNameTableName[preCreateDatabase].matched {
						_, err = dbDriver.Execute(ctx, fmt.Sprintf(`CREATE TABLE %s (id INT);`, preCreateTable), false, db.ExecuteOptions{})
						a.NoError(err)
					}
					for _, preCreateTable := range prepareInstance.matchDatabasesNameTableName[preCreateDatabase].unmatched {
						_, err = dbDriver.Execute(ctx, fmt.Sprintf(`CREATE TABLE %s (id INT);`, preCreateTable), false, db.ExecuteOptions{})
						a.NoError(err)
					}
					err = dbDriver.Close(ctx)
					a.NoError(err)

					_, err = ctl.databaseServiceClient.SyncDatabase(ctx, &v1pb.SyncDatabaseRequest{
						Name: fmt.Sprintf("%s/databases/%s", instance.Name, preCreateDatabase),
					})
					a.NoError(err)
				}
			}
			databaseGroup, err := ctl.projectServiceClient.CreateDatabaseGroup(ctx, &v1pb.CreateDatabaseGroupRequest{
				Parent:          ctl.project.Name,
				DatabaseGroupId: tc.databaseGroupPlaceholder,
				DatabaseGroup: &v1pb.DatabaseGroup{
					DatabasePlaceholder: tc.databaseGroupPlaceholder,
					DatabaseExpr: &expr.Expr{
						Expression: fmt.Sprintf(`(resource.environment_name == "environments/prod" && (%s))`, tc.databaseGroupExpr),
					},
				},
			})
			a.NoError(err)
			databaseGroup, err = ctl.projectServiceClient.GetDatabaseGroup(ctx, &v1pb.GetDatabaseGroupRequest{
				Name: databaseGroup.Name,
				View: v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL,
			})
			a.NoError(err)

			gotInstanceTitleToMatchedDatabasesTables := make(map[string]map[string]*tableNames)

			tableGroup, err := ctl.projectServiceClient.CreateSchemaGroup(ctx, &v1pb.CreateSchemaGroupRequest{
				Parent:        databaseGroup.Name,
				SchemaGroupId: tc.tableGroupPlaceholder,
				SchemaGroup: &v1pb.SchemaGroup{
					TableExpr: &expr.Expr{
						Expression: tc.tableGroupExpr,
					},
					TablePlaceholder: tc.tableGroupPlaceholder,
				},
			})
			a.NoError(err)
			tableGroup, err = ctl.projectServiceClient.GetSchemaGroup(ctx, &v1pb.GetSchemaGroupRequest{
				Name: tableGroup.Name,
				View: v1pb.SchemaGroupView_SCHEMA_GROUP_VIEW_FULL,
			})
			a.NoError(err)
			for _, matchedTable := range tableGroup.MatchedTables {
				instanceResourceID := strings.Split(matchedTable.Database, "/")[1]
				instanceTitle := instanceResourceID2InstanceTitle[instanceResourceID]
				a.NotEmpty(instanceTitle)

				databaseName := strings.Split(matchedTable.Database, "/")[3]
				tableName := matchedTable.Table
				if _, ok := gotInstanceTitleToMatchedDatabasesTables[instanceTitle]; !ok {
					gotInstanceTitleToMatchedDatabasesTables[instanceTitle] = make(map[string]*tableNames)
				}
				if _, ok := gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName]; !ok {
					gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName] = &tableNames{}
				}
				gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName].matched = append(gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName].matched, tableName)
			}
			for _, unmatchedTable := range tableGroup.UnmatchedTables {
				instanceResourceID := strings.Split(unmatchedTable.Database, "/")[1]
				instanceTitle := instanceResourceID2InstanceTitle[instanceResourceID]
				a.NotEmpty(instanceTitle)

				databaseName := strings.Split(unmatchedTable.Database, "/")[3]
				tableName := unmatchedTable.Table
				if _, ok := gotInstanceTitleToMatchedDatabasesTables[instanceTitle]; !ok {
					gotInstanceTitleToMatchedDatabasesTables[instanceTitle] = make(map[string]*tableNames)
				}
				if _, ok := gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName]; !ok {
					gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName] = &tableNames{}
				}
				gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName].unmatched = append(gotInstanceTitleToMatchedDatabasesTables[instanceTitle][databaseName].unmatched, tableName)
			}

			for _, prepareInstance := range tc.prepareInstances {
				gotMatchedDatabases := gotInstanceTitleToMatchedDatabasesTables[prepareInstance.instanceTitle]
				a.Equal(len(gotMatchedDatabases), len(prepareInstance.matchDatabasesNameTableName))

				for wantMatchDatabaseName, wantTableNames := range prepareInstance.matchDatabasesNameTableName {
					a.Contains(gotMatchedDatabases, wantMatchDatabaseName)
					gotTableNames := gotMatchedDatabases[wantMatchDatabaseName]
					a.Equal(len(wantTableNames.matched), len(gotTableNames.matched))
					a.Equal(len(wantTableNames.unmatched), len(gotTableNames.unmatched))
					for _, wantMatchTableName := range wantTableNames.matched {
						a.Contains(gotTableNames.matched, wantMatchTableName)
					}
					for _, wantUnmatchedTableName := range wantTableNames.unmatched {
						a.Contains(gotTableNames.unmatched, wantUnmatchedTableName)
					}
				}
			}
		})
	}
}

func TestCreateGroupingChangeIssue(t *testing.T) {
	type tableNames struct {
		matched   []string
		unmatched []string
	}
	type testCasePrepareInstance struct {
		instanceID                     string
		matchDatabasesNameTableName    map[string]tableNames
		unmatchedDatabaseNameTableName map[string][]string
		wantDatabaseTaskStatement      map[string][]string
	}
	type tableGroupMetaData struct {
		tableGroupPlaceholder string
		tableGroupExpr        string
	}
	testCases := []struct {
		name                     string
		databaseGroupPlaceholder string
		databaseGroupExpr        string
		statement                string
		tableGroupsMetaData      []tableGroupMetaData
		prepareInstances         []testCasePrepareInstance
	}{
		{
			name:                     "simple statement",
			databaseGroupPlaceholder: "employee",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			statement:                "ALTER TABLE salary ADD COLUMN num INT;",
			tableGroupsMetaData: []tableGroupMetaData{
				{
					tableGroupPlaceholder: "salary",
					tableGroupExpr:        `(resource.table_name.startsWith("salary_"))`,
				},
			},
			prepareInstances: []testCasePrepareInstance{
				{
					instanceID: "simplestatement",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_01": {
							matched: []string{"salary_01", "salary_02"},
						},
						"employee_02": {
							matched: []string{"salary_03", "salary_04"},
						},
					},
					unmatchedDatabaseNameTableName: map[string][]string{
						"blog": {"comments"},
					},
					wantDatabaseTaskStatement: map[string][]string{
						"employee_01": {
							"ALTER TABLE salary_01 ADD COLUMN num INT;\n",
							"ALTER TABLE salary_02 ADD COLUMN num INT;\n",
						},
						"employee_02": {
							"ALTER TABLE salary_03 ADD COLUMN num INT;\n",
							"ALTER TABLE salary_04 ADD COLUMN num INT;\n",
						},
					},
				},
			},
		},
		{
			name:                     "complex statement",
			databaseGroupPlaceholder: "employee",
			databaseGroupExpr:        `(resource.database_name.startsWith("employee_"))`,
			statement: `ALTER TABLE salary ADD COLUMN num INT;
CREATE INDEX salary_num_idx ON salary (num);
CREATE TABLE singleton(id INT);
ALTER TABLE person ADD COLUMN name VARCHAR(30);
ALTER TABLE partpartially ADD COLUMN num INT;
ALTER TABLE singleton ADD COLUMN num INT;`,
			tableGroupsMetaData: []tableGroupMetaData{
				{
					tableGroupPlaceholder: "salary",
					tableGroupExpr:        `(resource.table_name.startsWith("salary_"))`,
				},
				{
					tableGroupPlaceholder: "person",
					tableGroupExpr:        `(resource.table_name.startsWith("person_"))`,
				},
				{
					tableGroupPlaceholder: "partpartially",
					tableGroupExpr:        `(resource.table_name.startsWith("part_partially_"))`,
				},
			},
			prepareInstances: []testCasePrepareInstance{
				{
					instanceID: "complexstatement-instance01",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_01": {
							matched: []string{"salary_01", "salary_02", "person_01", "person_02", "part_partially_01"},
						},
						"employee_02": {
							matched: []string{"salary_03", "salary_04", "person_03", "person_04"},
						},
					},
					unmatchedDatabaseNameTableName: map[string][]string{
						"blog": {"comments", "article"},
					},
					wantDatabaseTaskStatement: map[string][]string{
						"employee_01": {
							"\nALTER TABLE part_partially_01 ADD COLUMN num INT;\n",
							"\nALTER TABLE person_01 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE person_02 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE singleton ADD COLUMN num INT;\n",
							"\nCREATE TABLE singleton(id INT);\n",
							"ALTER TABLE salary_01 ADD COLUMN num INT;\n\nCREATE INDEX salary_01_num_idx ON salary_01 (num);\n",
							"ALTER TABLE salary_02 ADD COLUMN num INT;\n\nCREATE INDEX salary_02_num_idx ON salary_02 (num);\n",
						},
						"employee_02": {
							"\nALTER TABLE person_03 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE person_04 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE singleton ADD COLUMN num INT;\n",
							"\nCREATE TABLE singleton(id INT);\n",
							"ALTER TABLE salary_03 ADD COLUMN num INT;\n\nCREATE INDEX salary_03_num_idx ON salary_03 (num);\n",
							"ALTER TABLE salary_04 ADD COLUMN num INT;\n\nCREATE INDEX salary_04_num_idx ON salary_04 (num);\n",
						},
					},
				},
				{
					instanceID: "complexstatement-instance02",
					matchDatabasesNameTableName: map[string]tableNames{
						"employee_03": {
							matched: []string{"salary_05", "salary_06", "person_05", "person_06"},
						},
						"employee_04": {
							matched: []string{"salary_07", "salary_08", "person_07", "person_08", "part_partially_02"},
						},
					},
					unmatchedDatabaseNameTableName: map[string][]string{
						"blog": {"comments", "article"},
					},
					wantDatabaseTaskStatement: map[string][]string{
						"employee_03": {
							"\nALTER TABLE person_05 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE person_06 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE singleton ADD COLUMN num INT;\n",
							"\nCREATE TABLE singleton(id INT);\n",
							"ALTER TABLE salary_05 ADD COLUMN num INT;\n\nCREATE INDEX salary_05_num_idx ON salary_05 (num);\n",
							"ALTER TABLE salary_06 ADD COLUMN num INT;\n\nCREATE INDEX salary_06_num_idx ON salary_06 (num);\n",
						},
						"employee_04": {
							"\nALTER TABLE part_partially_02 ADD COLUMN num INT;\n",
							"\nALTER TABLE person_07 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE person_08 ADD COLUMN name VARCHAR(30);\n",
							"\nALTER TABLE singleton ADD COLUMN num INT;\n",
							"\nCREATE TABLE singleton(id INT);\n",
							"ALTER TABLE salary_07 ADD COLUMN num INT;\n\nCREATE INDEX salary_07_num_idx ON salary_07 (num);\n",
							"ALTER TABLE salary_08 ADD COLUMN num INT;\n\nCREATE INDEX salary_08_num_idx ON salary_08 (num);\n",
						},
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: fake.NewGitLab,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			instanceResourceID2InstanceTitle := make(map[string]string)
			var stopInstances []func()

			for _, prepareInstance := range tc.prepareInstances {
				mysqlPort := getTestPort()
				stopInstance := mysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
				stopInstances = append(stopInstances, stopInstance)

				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: prepareInstance.instanceID,
					Instance: &v1pb.Instance{
						Title:       prepareInstance.instanceID,
						Engine:      v1pb.Engine_MYSQL,
						Environment: "environments/prod",
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(mysqlPort), Username: "root", Password: ""}},
						Activation:  true,
					},
				})
				a.NoError(err)
				instanceResourceID2InstanceTitle[prepareInstance.instanceID] = instance.Title

				for preCreateDatabase := range prepareInstance.matchDatabasesNameTableName {
					err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, preCreateDatabase, "", nil /* labelMap */)
					a.NoError(err)
					dbDriver, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/%s", mysqlPort, preCreateDatabase))
					a.NoError(err)

					for _, preCreateTable := range prepareInstance.matchDatabasesNameTableName[preCreateDatabase].matched {
						_, err = dbDriver.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s (id INT);`, preCreateTable))
						a.NoError(err)
					}
					for _, preCreateTable := range prepareInstance.matchDatabasesNameTableName[preCreateDatabase].unmatched {
						_, err = dbDriver.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s (id INT);`, preCreateTable))
						a.NoError(err)
					}
					err = dbDriver.Close()
					a.NoError(err)

					_, err = ctl.databaseServiceClient.SyncDatabase(ctx, &v1pb.SyncDatabaseRequest{
						Name: fmt.Sprintf("%s/databases/%s", instance.Name, preCreateDatabase),
					})
					a.NoError(err)
				}
				for preCreateDatabase, preCreateTables := range prepareInstance.unmatchedDatabaseNameTableName {
					err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, preCreateDatabase, "", nil /* labelMap */)
					a.NoError(err)
					dbDriver, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/%s", mysqlPort, preCreateDatabase))
					a.NoError(err)

					for _, preCreateTable := range preCreateTables {
						_, err = dbDriver.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s (id INT);`, preCreateTable))
						a.NoError(err)
					}
					err = dbDriver.Close()
					a.NoError(err)

					_, err = ctl.databaseServiceClient.SyncDatabase(ctx, &v1pb.SyncDatabaseRequest{
						Name: fmt.Sprintf("%s/databases/%s", instance.Name, preCreateDatabase),
					})
					a.NoError(err)
				}
			}

			defer func() {
				for _, stopInstance := range stopInstances {
					stopInstance()
				}
			}()

			databaseGroup, err := ctl.projectServiceClient.CreateDatabaseGroup(ctx, &v1pb.CreateDatabaseGroupRequest{
				Parent:          ctl.project.Name,
				DatabaseGroupId: tc.databaseGroupPlaceholder,
				DatabaseGroup: &v1pb.DatabaseGroup{
					DatabasePlaceholder: tc.databaseGroupPlaceholder,
					DatabaseExpr: &expr.Expr{
						Expression: fmt.Sprintf(`(resource.environment_name == "environments/prod" && (%s))`, tc.databaseGroupExpr),
					},
				},
			})
			a.NoError(err)

			databaseGroup, err = ctl.projectServiceClient.GetDatabaseGroup(ctx, &v1pb.GetDatabaseGroupRequest{
				Name: databaseGroup.Name,
				View: v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL,
			})
			a.NoError(err)

			for _, tableGroupMetaData := range tc.tableGroupsMetaData {
				_, err := ctl.projectServiceClient.CreateSchemaGroup(ctx, &v1pb.CreateSchemaGroupRequest{
					Parent:        databaseGroup.Name,
					SchemaGroupId: tableGroupMetaData.tableGroupPlaceholder,
					SchemaGroup: &v1pb.SchemaGroup{
						TableExpr: &expr.Expr{
							Expression: tableGroupMetaData.tableGroupExpr,
						},
						TablePlaceholder: tableGroupMetaData.tableGroupPlaceholder,
					},
				})
				a.NoError(err)
			}

			sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
				Parent: ctl.project.Name,
				Sheet: &v1pb.Sheet{
					Title:      "migration statement sheet",
					Content:    []byte(tc.statement),
					Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
					Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
					Type:       v1pb.Sheet_TYPE_SQL,
				},
			})
			a.NoError(err)
			_, rollout, _, err := ctl.changeDatabaseWithConfig(ctx, ctl.project, []*v1pb.Plan_Step{
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
								ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
									Target: databaseGroup.Name,
									Sheet:  sheet.Name,
									Type:   v1pb.Plan_ChangeDatabaseConfig_DATA,
								},
							},
						},
					},
				},
			})
			a.NoError(err)

			gotDatabaseToTaskStatement := make(map[string][]string)
			for _, task := range rollout.Stages[0].Tasks {
				var sheetName string
				switch task.Type {
				case v1pb.Task_DATABASE_SCHEMA_UPDATE:
					sheetName = task.GetDatabaseSchemaUpdate().Sheet
				case v1pb.Task_DATABASE_DATA_UPDATE:
					sheetName = task.GetDatabaseDataUpdate().Sheet
				}
				if sheetName != "" {
					statementSheet, err := ctl.sheetServiceClient.GetSheet(ctx, &v1pb.GetSheetRequest{Name: sheetName})
					a.NoError(err)
					gotDatabaseToTaskStatement[task.Target] = append(gotDatabaseToTaskStatement[task.Target], string(statementSheet.Content))
				}
			}

			for _, prepareInstance := range tc.prepareInstances {
				for wantDatabaseName, wantDatabaseStatements := range prepareInstance.wantDatabaseTaskStatement {
					database := common.FormatDatabase(prepareInstance.instanceID, wantDatabaseName)
					gotDatabaseStatements := gotDatabaseToTaskStatement[database]
					slices.Sort(gotDatabaseStatements)
					a.Equal(wantDatabaseStatements, gotDatabaseStatements)
				}
			}
		})
	}
}
