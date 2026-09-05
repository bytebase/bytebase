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

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestAdminQueryAffectedRows(t *testing.T) {
	tests := []struct {
		databaseName      string
		prepareStatements string
		query             string
		want              bool
		affectedRows      []*v1pb.QueryResult
	}{
		{
			databaseName:      "Test3",
			prepareStatements: "CREATE TABLE public.tbl(id INT PRIMARY KEY);",
			query:             "INSERT INTO tbl VALUES(1),(2);",
			affectedRows: []*v1pb.QueryResult{
				{
					ColumnNames:     []string{"Affected Rows"},
					ColumnTypeNames: []string{"INT"},
					Rows: []*v1pb.QueryRow{
						{
							Values: []*v1pb.RowValue{
								{Kind: &v1pb.RowValue_Int64Value{Int64Value: 2}},
							},
						},
					},
					Statement: "INSERT INTO tbl VALUES(1),(2);",
					RowsCount: 1,
				},
			},
		},
		{
			databaseName:      "Test4",
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY);",
			query:             "ALTER TABLE tbl ADD COLUMN name VARCHAR(255);",
			affectedRows: []*v1pb.QueryResult{
				{
					ColumnNames:     []string{"Affected Rows"},
					ColumnTypeNames: []string{"INT"},
					Rows: []*v1pb.QueryRow{
						{
							Values: []*v1pb.RowValue{
								{Kind: &v1pb.RowValue_Int64Value{Int64Value: 0}},
							},
						},
					},
					Statement: "ALTER TABLE tbl ADD COLUMN name VARCHAR(255);",
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

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	pgInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "postgres", Password: "root-password", Id: "admin"}},
		},
	}))
	a.NoError(err)
	pgInstance := pgInstanceResp.Msg

	for _, tt := range tests {
		instance := pgInstance
		databaseOwner := "postgres"
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

		err = ctl.changeDatabase(ctx, ctl.project, database, sheet, false)
		a.NoError(err)

		results, err := ctl.adminQuery(ctx, database, tt.query)
		a.NoError(err)

		a.Equal(len(tt.affectedRows), len(results))
		for idx, result := range results {
			a.Equal("", result.Error)
			result.Latency = nil
			diff := cmp.Diff(tt.affectedRows[idx], result, protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
			a.Empty(diff)
		}
	}
}
