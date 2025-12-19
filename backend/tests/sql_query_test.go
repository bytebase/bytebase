package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestSQLAdminQuery(t *testing.T) {
	tests := []struct {
		databaseName      string
		dbType            storepb.Engine
		prepareStatements string
		query             string
		reset             string
		queryResult       []*v1pb.QueryResult
		resetResult       []*v1pb.QueryResult
	}{
		{
			databaseName:      "Test1",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY, name VARCHAR(64), gender BIT(1), height BIT(8));",
			query:             "INSERT INTO Test1.tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
			reset:             "DELETE FROM tbl;",
			queryResult: []*v1pb.QueryResult{
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
					Statement: "INSERT INTO Test1.tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
					RowsCount: 1,
				},
			},
			resetResult: []*v1pb.QueryResult{
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
					RowsCount: 1,
				},
			},
		},
		{
			databaseName:      "Test2",
			dbType:            storepb.Engine_POSTGRES,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY, name VARCHAR(64), gender BIT(1), height BIT(8));",
			query:             "INSERT INTO tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
			reset:             "DELETE FROM tbl;",
			queryResult: []*v1pb.QueryResult{
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
					RowsCount: 1,
				},
			},
			resetResult: []*v1pb.QueryResult{
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
					RowsCount: 1,
				},
			},
		},
	}

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mysqlContainer, err := getMySQLContainer(ctx)
	defer func() {
		mysqlContainer.Close(ctx)
	}()
	a.NoError(err)

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	mysqlInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: mysqlContainer.host, Port: mysqlContainer.port, Username: "root", Password: "root-password", Id: "admin"}},
		},
	}))
	a.NoError(err)
	mysqlInstance := mysqlInstanceResp.Msg

	pgInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "postgres", Password: "root-password", Id: "admin"}},
		},
	}))
	a.NoError(err)
	pgInstance := pgInstanceResp.Msg

	for _, tt := range tests {
		var instance *v1pb.Instance
		databaseOwner := ""
		switch tt.dbType {
		case storepb.Engine_MYSQL:
			instance = mysqlInstance
		case storepb.Engine_POSTGRES:
			instance = pgInstance
			databaseOwner = "postgres"
		default:
			a.FailNow("unsupported db type")
		}
		err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, tt.databaseName, databaseOwner)
		a.NoError(err)

		databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, tt.databaseName),
		}))
		a.NoError(err)
		database := databaseResp.Msg

		sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet: &v1pb.Sheet{
				Content: []byte(tt.prepareStatements),
			},
		}))
		a.NoError(err)
		sheet := sheetResp.Msg

		a.NotNil(database.InstanceResource)
		a.Equal(1, len(database.InstanceResource.DataSources))

		err = ctl.changeDatabase(ctx, ctl.project, database, sheet, false)
		a.NoError(err)

		statement := tt.query
		results, err := ctl.adminQuery(ctx, database, statement)
		a.NoError(err)
		checkResults(a, tt.databaseName, statement, tt.queryResult, results)

		statement = tt.reset
		results, err = ctl.adminQuery(ctx, database, statement)
		a.NoError(err)
		checkResults(a, tt.databaseName, statement, tt.resetResult, results)
	}
}

func checkResults(a *require.Assertions, databaseName string, query string, affectedRows []*v1pb.QueryResult, results []*v1pb.QueryResult) {
	a.Equal(len(affectedRows), len(results))
	for idx, result := range results {
		a.Equal("", result.Error, "database %s: %s", databaseName, query)
		result.Latency = nil
		affectedRows[idx].Statement = query
		diff := cmp.Diff(affectedRows[idx], result, protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
		a.Empty(diff, "database %s: %s", databaseName, query)
	}
}
