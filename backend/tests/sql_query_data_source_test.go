package tests

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/server"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSQLQueryDataSourceResolution(t *testing.T) {
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
	_, err = pgContainer.db.Exec("DROP ROLE IF EXISTS query_ro")
	a.NoError(err)
	_, err = pgContainer.db.Exec("CREATE ROLE query_ro LOGIN PASSWORD 'query_ro_password'")
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

	const databaseName = "QueryDataSourceResolution"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, "postgres")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE books(id INT PRIMARY KEY, name VARCHAR(64)); INSERT INTO books VALUES (1, 'Bytebase');`)},
	}))
	a.NoError(err)
	err = ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false)
	a.NoError(err)

	grantDB, err := sql.Open("pgx", fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s?sslmode=disable", pgContainer.host, pgContainer.port, databaseName))
	a.NoError(err)
	defer grantDB.Close()
	for _, stmt := range []string{
		fmt.Sprintf("GRANT CONNECT ON DATABASE %q TO query_ro", databaseName),
		"GRANT USAGE ON SCHEMA public TO query_ro",
		"GRANT SELECT ON ALL TABLES IN SCHEMA public TO query_ro",
	} {
		_, err = grantDB.Exec(stmt)
		a.NoError(err)
	}

	queryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT name FROM books;",
	}))
	a.NoError(err)
	a.Len(queryResp.Msg.Results, 1)
	a.Empty(queryResp.Msg.Results[0].Error)

	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Host:     pgContainer.host,
			Port:     pgContainer.port,
			Username: "query_ro",
			Password: "query_ro_password",
		},
	}))
	a.NoError(err)

	queryResp, err = ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT name FROM books;",
	}))
	a.NoError(err)
	a.Len(queryResp.Msg.Results, 1)
	a.Empty(queryResp.Msg.Results[0].Error)
	a.Len(queryResp.Msg.Results[0].Rows, 1)
	a.Equal("Bytebase", queryResp.Msg.Results[0].Rows[0].Values[0].GetStringValue())

	instanceID, err := common.GetInstanceID(instance.Name)
	a.NoError(err)
	stores := getStore(t, ctl.server)
	workspaceID, err := stores.GetWorkspaceID(ctx)
	a.NoError(err)
	a.NotEmpty(workspaceID)
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: common.FormatWorkspace(workspaceID),
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_DATA_QUERY,
			Policy: &v1pb.Policy_QueryDataPolicy{
				QueryDataPolicy: &v1pb.QueryDataPolicy{
					AllowAdminDataSource: true,
				},
			},
		},
	}))
	a.NoError(err)

	queryResp, err = ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "INSERT INTO books VALUES (2, 'Bytebase Admin');",
	}))
	a.NoError(err)
	a.Len(queryResp.Msg.Results, 1)
	a.Empty(queryResp.Msg.Results[0].Error)

	instanceMessage, err := stores.GetInstance(ctx, &store.FindInstanceMessage{Workspace: workspaceID, ResourceID: &instanceID})
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
		Workspace:  workspaceID,
		Metadata:   metadata,
	})
	a.NoError(err)

	_, err = ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT name FROM books;",
	}))
	a.Error(err)
	var connectErr *connect.Error
	a.True(errors.As(err, &connectErr))
	a.Equal(connect.CodeFailedPrecondition, connectErr.Code())
	a.Contains(connectErr.Message(), "multiple read-only data sources")
}

func getStore(t *testing.T, srv *server.Server) *store.Store {
	t.Helper()
	field := reflect.ValueOf(srv).Elem().FieldByName("store")
	stores, ok := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(*store.Store)
	if !ok {
		t.Fatal("failed to access server store")
	}
	return stores
}
