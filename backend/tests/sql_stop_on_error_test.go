package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestSQLQueryStopOnError(t *testing.T) {
	tests := []struct {
		name              string
		databaseName      string
		dbType            storepb.Engine
		prepareStatements string
		query             string
		wantResults       int // Number of successful results before error
		wantError         bool
	}{
		{
			name:              "MySQL - All statements succeed",
			databaseName:      "TestStopOnError1",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE tbl1(id INT PRIMARY KEY, name VARCHAR(64));",
			query:             "INSERT INTO tbl1 VALUES(1, 'Alice'); INSERT INTO tbl1 VALUES(2, 'Bob'); SELECT * FROM tbl1;",
			wantResults:       3, // 2 inserts + 1 select
			wantError:         false,
		},
		{
			name:              "MySQL - Second statement fails",
			databaseName:      "TestStopOnError2",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE tbl2(id INT PRIMARY KEY, name VARCHAR(64));",
			query:             "INSERT INTO tbl2 VALUES(1, 'Alice'); INSERT INTO nonexistent VALUES(2, 'Bob'); INSERT INTO tbl2 VALUES(3, 'Charlie');",
			wantResults:       1, // Only first insert succeeds
			wantError:         true,
		},
		{
			name:              "MySQL - First statement fails",
			databaseName:      "TestStopOnError3",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE tbl3(id INT PRIMARY KEY, name VARCHAR(64));",
			query:             "INSERT INTO nonexistent VALUES(1, 'Alice'); INSERT INTO tbl3 VALUES(2, 'Bob');",
			wantResults:       0, // First statement fails
			wantError:         true,
		},
		{
			name:              "PostgreSQL - All statements succeed",
			databaseName:      "TestStopOnError4",
			dbType:            storepb.Engine_POSTGRES,
			prepareStatements: "CREATE TABLE tbl4(id INT PRIMARY KEY, name VARCHAR(64));",
			query:             "INSERT INTO tbl4 VALUES(1, 'Alice'); INSERT INTO tbl4 VALUES(2, 'Bob'); SELECT * FROM tbl4;",
			wantResults:       3,
			wantError:         false,
		},
		{
			name:              "PostgreSQL - Second statement fails",
			databaseName:      "TestStopOnError5",
			dbType:            storepb.Engine_POSTGRES,
			prepareStatements: "CREATE TABLE tbl5(id INT PRIMARY KEY, name VARCHAR(64));",
			query:             "INSERT INTO tbl5 VALUES(1, 'Alice'); INSERT INTO nonexistent VALUES(2, 'Bob'); INSERT INTO tbl5 VALUES(3, 'Charlie');",
			wantResults:       1,
			wantError:         true,
		},
	}

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	t.Cleanup(func() {
		ctl.Close(ctx)
	})

	mysqlContainer, err := getMySQLContainer(ctx)
	t.Cleanup(func() {
		mysqlContainer.Close(ctx)
	})
	a.NoError(err)

	pgContainer, err := getPgContainer(ctx)
	t.Cleanup(func() {
		pgContainer.Close(ctx)
	})
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
			t.Parallel()
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

			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, tt.databaseName, databaseOwner)
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

			// Execute the query using the Query API (not AdminExecute)
			queryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
				Name:         database.Name,
				Statement:    tt.query,
				DataSourceId: "admin",
			}))

			if tt.wantError {
				// We expect an error, but we should still get partial results
				a.Error(err)
				// The error response might be in the connect error, not in the response
			} else {
				a.NoError(err)
				a.NotNil(queryResp)
				a.Equal(tt.wantResults, len(queryResp.Msg.Results), "expected %d results, got %d", tt.wantResults, len(queryResp.Msg.Results))

				// Verify all results are successful (no errors)
				for i, result := range queryResp.Msg.Results {
					a.Empty(result.Error, "result %d should not have error", i)
				}
			}
		})
	}
}

func TestSQLAdminExecuteStopOnError(t *testing.T) {
	tests := []struct {
		name              string
		databaseName      string
		dbType            storepb.Engine
		prepareStatements string
		query             string
		wantResults       int
		wantError         bool
	}{
		{
			name:              "MySQL AdminExecute - Second statement fails",
			databaseName:      "TestAdminStopOnError1",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE admin_tbl1(id INT PRIMARY KEY, name VARCHAR(64));",
			query:             "INSERT INTO admin_tbl1 VALUES(1, 'Alice'); INSERT INTO nonexistent VALUES(2, 'Bob'); INSERT INTO admin_tbl1 VALUES(3, 'Charlie');",
			wantResults:       1,
			wantError:         true,
		},
		{
			name:              "PostgreSQL AdminExecute - Second statement fails",
			databaseName:      "TestAdminStopOnError2",
			dbType:            storepb.Engine_POSTGRES,
			prepareStatements: "CREATE TABLE admin_tbl2(id INT PRIMARY KEY, name VARCHAR(64));",
			query:             "INSERT INTO admin_tbl2 VALUES(1, 'Alice'); INSERT INTO nonexistent VALUES(2, 'Bob'); INSERT INTO admin_tbl2 VALUES(3, 'Charlie');",
			wantResults:       1,
			wantError:         true,
		},
	}

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	t.Cleanup(func() {
		ctl.Close(ctx)
	})

	mysqlContainer, err := getMySQLContainer(ctx)
	t.Cleanup(func() {
		mysqlContainer.Close(ctx)
	})
	a.NoError(err)

	pgContainer, err := getPgContainer(ctx)
	t.Cleanup(func() {
		pgContainer.Close(ctx)
	})
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
			t.Parallel()
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

			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, tt.databaseName, databaseOwner)
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

			// Use AdminExecute (streaming API)
			// Note: AdminExecute doesn't use queryRetryStopOnError, so this test verifies
			// that the regular behavior is unchanged
			results, err := ctl.adminQuery(ctx, database, tt.query)

			// AdminExecute returns results with errors in the result objects
			a.NoError(err)
			a.NotNil(results)

			// Check that we got some results
			a.GreaterOrEqual(len(results), tt.wantResults)

			// Check if any result has an error
			hasError := false
			for _, result := range results {
				if result.Error != "" {
					hasError = true
					break
				}
			}
			a.Equal(tt.wantError, hasError)
		})
	}
}
