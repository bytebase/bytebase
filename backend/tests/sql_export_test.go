package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/alexmullins/zip"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestSQLExport(t *testing.T) {
	tests := []struct {
		databaseName      string
		dbType            storepb.Engine
		prepareStatements string
		query             string
		password          string
		reset             string
		export            string
		want              bool
		affectedRows      []*v1pb.QueryResult
	}{
		{
			databaseName:      "Test1",
			dbType:            storepb.Engine_MYSQL,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY, name VARCHAR(64), gender BIT(1), height BIT(8));",
			query:             "INSERT INTO Test1.tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
			reset:             "DELETE FROM tbl;",
			export:            "SELECT * FROM Test1.tbl;",
			password:          "123",
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
			dbType:            storepb.Engine_POSTGRES,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY, name VARCHAR(64), gender BIT(1), height BIT(8));",
			query:             "INSERT INTO tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
			reset:             "DELETE FROM tbl;",
			export:            "SELECT * FROM tbl;",
			password:          "",
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
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	mysqlPort := getTestPort()
	mysqlStopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer mysqlStopInstance()

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	stopInstance := postgres.SetupTestInstance(pgBinDir, t.TempDir(), pgPort)
	defer stopInstance()

	mysqlInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: "environments/prod",
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
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "root"}},
		},
	})
	a.NoError(err)

	for _, tt := range tests {
		var instance *v1pb.Instance
		databaseOwner := ""
		switch tt.dbType {
		case storepb.Engine_MYSQL:
			instance = mysqlInstance
		case storepb.Engine_POSTGRES:
			instance = pgInstance
			databaseOwner = "root"
		default:
			a.FailNow("unsupported db type")
		}
		err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, tt.databaseName, databaseOwner, nil)
		a.NoError(err)

		database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, tt.databaseName),
		})
		a.NoError(err)

		sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet: &v1pb.Sheet{
				Title:      "prepareStatements",
				Content:    []byte(tt.prepareStatements),
				Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
				Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
				Type:       v1pb.Sheet_TYPE_SQL,
			},
		})
		a.NoError(err)

		err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
		a.NoError(err)

		for _, databaseNameQuery := range []string{tt.databaseName, ""} {
			if databaseNameQuery == "" && tt.dbType != storepb.Engine_MYSQL {
				// not supporting to query SQL when databaseName of PostgreSQL is empty
				continue
			}

			statement := tt.query
			results, err := ctl.adminQuery(ctx, instance, databaseNameQuery, statement)
			a.NoError(err)
			checkResults(a, tt.databaseName, statement, tt.affectedRows, results)

			request := &v1pb.ExportRequest{
				Admin:              true,
				ConnectionDatabase: databaseNameQuery,
				Format:             v1pb.ExportFormat_SQL,
				Limit:              1,
				Name:               instance.Name,
				Statement:          tt.export,
			}
			export, err := ctl.sqlServiceClient.Export(ctx, request)
			a.NoError(err)

			statement = tt.reset
			results, err = ctl.adminQuery(ctx, instance, tt.databaseName, statement)
			a.NoError(err)
			checkResults(a, tt.databaseName, statement, tt.affectedRows, results)

			if tt.password != "" {
				reader := bytes.NewReader(export.Content)
				zipReader, err := zip.NewReader(reader, int64(len(export.Content)))
				a.NoError(err)
				a.Equal(1, len(zipReader.File))

				a.Equal(fmt.Sprintf("export.%s", strings.ToLower(request.Format.String())), zipReader.File[0].Name)
				compressedFile := zipReader.File[0]
				compressedFile.SetPassword(tt.password)
				file, err := compressedFile.Open()
				a.NoError(err)
				content, err := io.ReadAll(file)
				a.NoError(err)
				statement = string(content)
			} else {
				statement = string(export.Content)
			}

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
