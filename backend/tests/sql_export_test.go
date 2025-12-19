package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestSQLExport(t *testing.T) {
	tests := []struct {
		databaseName     string
		dbType           storepb.Engine
		prepareStatement string
		exportTests      []struct {
			format    v1pb.ExportFormat
			statement string
			password  string
			results   []struct {
				statement string
				content   string
			}
		}
	}{
		{
			databaseName: "Test1",
			dbType:       storepb.Engine_MYSQL,
			prepareStatement: `
			CREATE TABLE tbl(id INT PRIMARY KEY, name VARCHAR(64), gender BIT(1), height BIT(8));
			INSERT INTO Test1.tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');
			`,
			exportTests: []struct {
				format    v1pb.ExportFormat
				statement string
				password  string
				results   []struct {
					statement string
					content   string
				}
			}{
				{
					format:    v1pb.ExportFormat_JSON,
					password:  "123",
					statement: "SELECT * FROM Test1.tbl;",
					results: []struct {
						statement string
						content   string
					}{
						{
							statement: "SELECT * FROM Test1.tbl;",
							content: `[
  {
    "gender": "AA==",
    "height": "fw==",
    "id": 1,
    "name": "Alice"
  }
]`,
						},
					},
				},
				{
					format:    v1pb.ExportFormat_CSV,
					statement: "SELECT * FROM Test1.tbl;",
					results: []struct {
						statement string
						content   string
					}{
						{
							statement: "SELECT * FROM Test1.tbl;",
							content:   "id,name,gender,height\n1,\"Alice\",\"0x00\",\"0x7f\"",
						},
					},
				},
			},
		},
		{
			databaseName:     "Test2",
			dbType:           storepb.Engine_POSTGRES,
			prepareStatement: "CREATE TABLE tbl(id INT PRIMARY KEY);",
			exportTests: []struct {
				format    v1pb.ExportFormat
				statement string
				password  string
				results   []struct {
					statement string
					content   string
				}
			}{
				{
					format: v1pb.ExportFormat_JSON,
					statement: `
					SELECT 1;
					SELECT 2;
					`,
					results: []struct {
						statement string
						content   string
					}{
						{
							statement: "SELECT 1;",
							content: `[
  {
    "?column?": 1
  }
]`,
						},
						{
							statement: "SELECT 2;",
							content: `[
  {
    "?column?": 2
  }
]`,
						},
					},
				},
				{
					format:    v1pb.ExportFormat_CSV,
					statement: "SELECT 1;",
					results: []struct {
						statement string
						content   string
					}{
						{
							statement: "SELECT 1;",
							content:   "?column?\n1",
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

		instanceID, err := common.GetInstanceID(instance.Name)
		a.NoError(err)

		databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, tt.databaseName),
		}))
		a.NoError(err)
		database := databaseResp.Msg

		sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet: &v1pb.Sheet{
				Content: []byte(tt.prepareStatement),
			},
		}))
		a.NoError(err)
		sheet := sheetResp.Msg

		a.NotNil(database.InstanceResource)
		a.Equal(1, len(database.InstanceResource.DataSources))
		dataSource := database.InstanceResource.DataSources[0]

		err = ctl.changeDatabase(ctx, ctl.project, database, sheet, false)
		a.NoError(err)

		for _, exportTest := range tt.exportTests {
			request := &v1pb.ExportRequest{
				Name:         database.Name,
				Format:       exportTest.format,
				Statement:    exportTest.statement,
				Password:     exportTest.password,
				DataSourceId: dataSource.Id,
			}
			exportResp, err := ctl.sqlServiceClient.Export(ctx, connect.NewRequest(request))
			a.NoError(err)
			export := exportResp.Msg

			reader := bytes.NewReader(export.Content)
			zipReader, err := zip.NewReader(reader, int64(len(export.Content)))
			a.NoError(err)

			a.Equal(len(exportTest.results)*2, len(zipReader.File))

			for i, compressedFile := range zipReader.File {
				if exportTest.password != "" {
					compressedFile.SetPassword(exportTest.password)
				}

				expectFilenamePrefix := fmt.Sprintf("%s/%s/statement-%d", instanceID, tt.databaseName, (i/2)+1)
				a.True(strings.HasPrefix(compressedFile.Name, expectFilenamePrefix))
				expectResult := exportTest.results[i/2]

				file, err := compressedFile.Open()
				a.NoError(err)
				content, err := io.ReadAll(file)
				a.NoError(err)
				if compressedFile.Name == fmt.Sprintf("%s.sql", expectFilenamePrefix) {
					a.Equal(strings.TrimSpace(expectResult.statement), strings.TrimSpace(string(content)))
				} else {
					a.Equal(fmt.Sprintf("%s.result.%s", expectFilenamePrefix, strings.ToLower(request.Format.String())), compressedFile.Name)
					a.Equal(expectResult.content, string(content))
				}
			}
		}
	}
}
