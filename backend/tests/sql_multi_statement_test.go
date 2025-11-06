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

// TestMultiStatementSQLExecution tests that multiple SQL statements can be executed
// and returns results for all statements, even if some fail.
func TestMultiStatementSQLExecution(t *testing.T) {
	tests := []struct {
		name              string
		databaseName      string
		dbType            storepb.Engine
		prepareStatements string
		multiStatement    string
		expectedResults   []*v1pb.QueryResult
	}{
		{
			name:              "MySQL - Multiple successful statements",
			databaseName:      "MultiStmtTest1",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE users(id INT PRIMARY KEY, name VARCHAR(64));",
			multiStatement:    "INSERT INTO users (id, name) VALUES(1, 'Alice'); INSERT INTO users (id, name) VALUES(2, 'Bob'); INSERT INTO users (id, name) VALUES(3, 'Charlie');",
			expectedResults: []*v1pb.QueryResult{
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
					Statement: "INSERT INTO users (id, name) VALUES(1, 'Alice')",
					RowsCount: 1,
				},
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
					Statement: "INSERT INTO users (id, name) VALUES(2, 'Bob')",
					RowsCount: 1,
				},
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
					Statement: "INSERT INTO users (id, name) VALUES(3, 'Charlie')",
					RowsCount: 1,
				},
			},
		},
		{
			name:              "PostgreSQL - Multiple successful statements",
			databaseName:      "MultiStmtTest2",
			dbType:            storepb.Engine_POSTGRES,
			prepareStatements: "CREATE TABLE users(id INT PRIMARY KEY, name VARCHAR(64));",
			multiStatement:    "INSERT INTO users (id, name) VALUES(1, 'Alice'); INSERT INTO users (id, name) VALUES(2, 'Bob'); INSERT INTO users (id, name) VALUES(3, 'Charlie');",
			expectedResults: []*v1pb.QueryResult{
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
					Statement: "INSERT INTO users (id, name) VALUES(1, 'Alice')",
					RowsCount: 1,
				},
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
					Statement: "INSERT INTO users (id, name) VALUES(2, 'Bob')",
					RowsCount: 1,
				},
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
					Statement: "INSERT INTO users (id, name) VALUES(3, 'Charlie')",
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
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)

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

			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, tt.databaseName, databaseOwner)
			a.NoError(err)

			databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, tt.databaseName),
			}))
			a.NoError(err)
			database := databaseResp.Msg

			sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
				Parent: ctl.project.Name,
				Sheet: &v1pb.Sheet{
					Title:   "prepareStatements",
					Content: []byte(tt.prepareStatements),
				},
			}))
			a.NoError(err)
			sheet := sheetResp.Msg

			a.NotNil(database.InstanceResource)
			a.Equal(1, len(database.InstanceResource.DataSources))

			err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.MigrationType_DDL)
			a.NoError(err)

			// Execute the multi-statement SQL
			results, err := ctl.adminQuery(ctx, database, tt.multiStatement)
			a.NoError(err)

			// Verify we got results for all statements
			a.Equal(len(tt.expectedResults), len(results), "should return results for all statements")

			// Check each result
			for idx, result := range results {
				a.Equal("", result.Error, "statement %d should not have error", idx)
				result.Latency = nil
				diff := cmp.Diff(tt.expectedResults[idx], result, protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
				a.Empty(diff, "statement %d result mismatch", idx)
			}
		})
	}
}

// TestMultiStatementSQLWithErrors tests that when some statements fail,
// the successful statements still execute and return results.
func TestMultiStatementSQLWithErrors(t *testing.T) {
	tests := []struct {
		name              string
		databaseName      string
		dbType            storepb.Engine
		prepareStatements string
		multiStatement    string
		checkResults      func(t *testing.T, results []*v1pb.QueryResult)
	}{
		{
			name:              "MySQL - One statement fails, others succeed",
			databaseName:      "MultiStmtErrorTest1",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE users(id INT PRIMARY KEY, name VARCHAR(64));",
			// Second statement has duplicate key and will fail
			multiStatement: "INSERT INTO users (id, name) VALUES(1, 'Alice'); INSERT INTO users (id, name) VALUES(1, 'Duplicate'); INSERT INTO users (id, name) VALUES(2, 'Bob');",
			checkResults: func(t *testing.T, results []*v1pb.QueryResult) {
				a := require.New(t)
				a.Equal(3, len(results), "should return results for all 3 statements")

				// First statement succeeds
				a.Equal("", results[0].Error, "first statement should succeed")
				a.Equal(int64(1), results[0].RowsCount, "first statement should affect 1 row")

				// Second statement fails (duplicate key)
				a.NotEqual("", results[1].Error, "second statement should fail with duplicate key error")
				a.Contains(results[1].Error, "Duplicate", "error should mention duplicate")

				// Third statement succeeds despite second failing
				a.Equal("", results[2].Error, "third statement should succeed even though second failed")
				a.Equal(int64(1), results[2].RowsCount, "third statement should affect 1 row")
			},
		},
		{
			name:              "PostgreSQL - One statement fails, others succeed",
			databaseName:      "MultiStmtErrorTest2",
			dbType:            storepb.Engine_POSTGRES,
			prepareStatements: "CREATE TABLE users(id INT PRIMARY KEY, name VARCHAR(64));",
			// Second statement has duplicate key and will fail
			multiStatement: "INSERT INTO users (id, name) VALUES(1, 'Alice'); INSERT INTO users (id, name) VALUES(1, 'Duplicate'); INSERT INTO users (id, name) VALUES(2, 'Bob');",
			checkResults: func(t *testing.T, results []*v1pb.QueryResult) {
				a := require.New(t)
				a.Equal(3, len(results), "should return results for all 3 statements")

				// First statement succeeds
				a.Equal("", results[0].Error, "first statement should succeed")
				a.Equal(int64(1), results[0].RowsCount, "first statement should affect 1 row")

				// Second statement fails (duplicate key)
				a.NotEqual("", results[1].Error, "second statement should fail with duplicate key error")
				a.Contains(results[1].Error, "duplicate", "error should mention duplicate")

				// Third statement succeeds despite second failing
				a.Equal("", results[2].Error, "third statement should succeed even though second failed")
				a.Equal(int64(1), results[2].RowsCount, "third statement should affect 1 row")
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
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)

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

			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, tt.databaseName, databaseOwner)
			a.NoError(err)

			databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, tt.databaseName),
			}))
			a.NoError(err)
			database := databaseResp.Msg

			sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
				Parent: ctl.project.Name,
				Sheet: &v1pb.Sheet{
					Title:   "prepareStatements",
					Content: []byte(tt.prepareStatements),
				},
			}))
			a.NoError(err)
			sheet := sheetResp.Msg

			a.NotNil(database.InstanceResource)
			a.Equal(1, len(database.InstanceResource.DataSources))

			err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.MigrationType_DDL)
			a.NoError(err)

			// Execute the multi-statement SQL
			results, err := ctl.adminQuery(ctx, database, tt.multiStatement)
			a.NoError(err)

			// Use custom check function to verify results
			tt.checkResults(t, results)
		})
	}
}
