package tests

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/plugin/db"
	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestSQLExport(t *testing.T) {
	tests := []struct {
		databaseName      string
		dbType            db.Type
		prepareStatements string
		query             string
		reset             string
		export            string
		want              bool
		affectedRows      []*v1pb.QueryResult
	}{
		{
			databaseName:      "Test1",
			dbType:            db.MySQL,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY, name VARCHAR(64), gender BIT(1), height BIT(8));",
			query:             "INSERT INTO Test1.tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
			reset:             "DELETE FROM tbl;",
			export:            "SELECT * FROM Test1.tbl;",
			affectedRows: []*v1pb.QueryResult{
				{
					ColumnNames:     []string{"Affected Rows"},
					ColumnTypeNames: []string{"INT"},
					Rows: []*v1pb.QueryRow{
						{
							Values: []*v1pb.RowValue{
								{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1}},
							},
						},
					},
				},
			},
		},
		{
			databaseName:      "Test2",
			dbType:            db.Postgres,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY, name VARCHAR(64), gender BIT(1), height BIT(8));",
			query:             "INSERT INTO tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
			reset:             "DELETE FROM tbl;",
			export:            "SELECT * FROM tbl;",
			affectedRows: []*v1pb.QueryResult{
				{
					ColumnNames:     []string{"Affected Rows"},
					ColumnTypeNames: []string{"INT"},
					Rows: []*v1pb.QueryRow{
						{
							Values: []*v1pb.RowValue{
								{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1}},
							},
						},
					},
				},
			},
		},
	}

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:                   dataDir,
		vcsProviderCreator:        fake.NewGitLab,
		developmentUseV2Scheduler: true,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	mysqlPort := getTestPort()
	mysqlStopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer mysqlStopInstance()

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	pgStopInstance := postgres.SetupTestInstance(t, pgPort, resourceDir)
	defer pgStopInstance()

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	mysqlInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(mysqlPort), Username: "root", Password: ""}},
		},
	})
	a.NoError(err)

	pgInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "root"}},
		},
	})
	a.NoError(err)

	for _, tt := range tests {
		var instance *v1pb.Instance
		databaseOwner := ""
		switch tt.dbType {
		case db.MySQL:
			instance = mysqlInstance
		case db.Postgres:
			instance = pgInstance
			databaseOwner = "root"
		default:
			a.FailNow("unsupported db type")
		}
		err = ctl.createDatabaseV2(ctx, project, instance, tt.databaseName, databaseOwner, nil)
		a.NoError(err)

		database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, tt.databaseName),
		})
		a.NoError(err)

		sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
			Parent: project.Name,
			Sheet: &v1pb.Sheet{
				Title:      "prepareStatements",
				Content:    []byte(tt.prepareStatements),
				Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
				Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
				Type:       v1pb.Sheet_TYPE_SQL,
			},
		})
		a.NoError(err)

		err = ctl.changeDatabase(ctx, project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
		a.NoError(err)

		for _, databaseNameQuery := range []string{tt.databaseName, ""} {
			if databaseNameQuery == "" && tt.dbType != db.MySQL {
				// not supporting to query SQL when databaseName of PostgreSQL is empty
				continue
			}

			statement := tt.query
			results, err := ctl.adminQuery(ctx, instance, databaseNameQuery, statement)
			a.NoError(err)
			checkResults(a, tt.databaseName, statement, tt.affectedRows, results)

			export, err := ctl.sqlServiceClient.Export(ctx, &v1pb.ExportRequest{
				Admin:              true,
				ConnectionDatabase: databaseNameQuery,
				Format:             v1pb.ExportRequest_SQL,
				Limit:              1,
				Name:               instance.Name,
				Statement:          tt.export,
			})
			a.NoError(err)

			statement = tt.reset
			results, err = ctl.adminQuery(ctx, instance, tt.databaseName, statement)
			a.NoError(err)
			checkResults(a, tt.databaseName, statement, tt.affectedRows, results)

			statement = string(export.Content)
			results, err = ctl.adminQuery(ctx, instance, tt.databaseName, statement)
			a.NoError(err)
			checkResults(a, tt.databaseName, statement, tt.affectedRows, results)

			statement = tt.reset
			results, err = ctl.adminQuery(ctx, instance, tt.databaseName, statement)
			a.NoError(err)
			checkResults(a, tt.databaseName, statement, tt.affectedRows, results)
		}
	}
}

func checkResults(a *require.Assertions, databaseName string, query string, affectedRows []*v1pb.QueryResult, results []*v1pb.QueryResult) {
	a.Equal(len(affectedRows), len(results))
	for idx, result := range results {
		a.Equal("", result.Error, "database %s: %s", databaseName, query)
		result.Latency = nil
		affectedRows[idx].Statement = strings.TrimSuffix(query, ";")
		diff := cmp.Diff(affectedRows[idx], result, protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
		a.Equal("", diff, "database %s: %s", databaseName, query)
	}
}
