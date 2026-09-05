package tests

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSQLExportDataSourceResolution(t *testing.T) {
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

	// A login role for the read-only data source. Its grants have to wait until
	// the table exists, further down: Postgres grants are per object, where
	// MySQL's GRANT SELECT ON *.* covered tables created later.
	_, err = pgContainer.db.Exec("DROP ROLE IF EXISTS export_ro")
	a.NoError(err)
	_, err = pgContainer.db.Exec("CREATE ROLE export_ro LOGIN PASSWORD 'export_ro_password'")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
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
	instance := instanceResp.Msg

	const databaseName = "ExportDataSourceResolution"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, "postgres")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{Content: []byte(`
			CREATE TABLE books(id INT PRIMARY KEY, name VARCHAR(64));
			INSERT INTO books VALUES (1, 'Bytebase');
		`)},
	}))
	a.NoError(err)
	err = ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false)
	a.NoError(err)

	grantDB, err := sql.Open("pgx", fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s?sslmode=disable", pgContainer.host, pgContainer.port, databaseName))
	a.NoError(err)
	defer grantDB.Close()
	for _, stmt := range []string{
		fmt.Sprintf("GRANT CONNECT ON DATABASE %q TO export_ro", databaseName),
		"GRANT USAGE ON SCHEMA public TO export_ro",
		"GRANT SELECT ON ALL TABLES IN SCHEMA public TO export_ro",
	} {
		_, err = grantDB.Exec(stmt)
		a.NoError(err)
	}

	assertExportContent := func() {
		exportResp, err := ctl.sqlServiceClient.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{
			Name:      database.Name,
			Format:    v1pb.ExportFormat_CSV,
			Statement: "SELECT name FROM books;",
		}))
		a.NoError(err)
		a.Equal("name\n\"Bytebase\"", getExportResultContent(t, exportResp.Msg.Content, ".result.csv"))
	}

	assertExportContent()

	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Host:     pgContainer.host,
			Port:     pgContainer.port,
			Username: "export_ro",
			Password: "export_ro_password",
		},
	}))
	a.NoError(err)

	assertExportContent()

	instanceID, err := common.GetInstanceID(instance.Name)
	a.NoError(err)
	stores := getStore(t, ctl.server)
	instanceMessage, err := stores.GetInstance(ctx, &store.FindInstanceMessage{Workspace: common.GetWorkspaceIDFromContext(ctx), ResourceID: &instanceID})
	a.NoError(err)
	metadata := proto.CloneOf(instanceMessage.Metadata)
	var readOnly *storepb.DataSource
	for _, dataSource := range metadata.GetDataSources() {
		if dataSource.GetType() == storepb.DataSourceType_READ_ONLY {
			readOnly = proto.CloneOf(dataSource)
			break
		}
	}
	if readOnly == nil {
		t.Fatal("expected read-only data source")
	}
	readOnly.Id = "readonly-legacy"
	metadata.DataSources = append(metadata.DataSources, readOnly)
	_, err = stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instanceID,
		Workspace:  common.GetWorkspaceIDFromContext(ctx),
		Metadata:   metadata,
	})
	a.NoError(err)

	_, err = ctl.sqlServiceClient.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{
		Name:      database.Name,
		Format:    v1pb.ExportFormat_CSV,
		Statement: "SELECT name FROM books;",
	}))
	a.Error(err)
	var connectErr *connect.Error
	a.True(errors.As(err, &connectErr))
	a.Equal(connect.CodeFailedPrecondition, connectErr.Code())
	a.Contains(connectErr.Message(), "multiple read-only data sources")
}

func getExportResultContent(t *testing.T, export []byte, suffix string) string {
	t.Helper()

	zipReader, err := zip.NewReader(bytes.NewReader(export), int64(len(export)))
	require.NoError(t, err)

	for _, file := range zipReader.File {
		if !strings.HasSuffix(file.Name, suffix) {
			continue
		}
		rc, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		return string(content)
	}

	t.Fatalf("export result with suffix %q not found", suffix)
	return ""
}
