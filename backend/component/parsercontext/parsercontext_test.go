package parsercontext

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestBuildGetLinkedDatabaseMetadataFunc(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	links := []*metadatapb.LinkedDatabaseMetadata{
		{Name: "REMOTE", Host: "localhost:1521/FREEPDB1", Username: "SECRET_SCHEMA"},
		{Name: "UNMANAGED", Host: "172.23.0.2:1521/FREEPDB1", Username: "SECRET_SCHEMA"},
		{Name: "RAC", Host: "(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(HOST=rac2)(PORT=1521))(ADDRESS=(HOST=LOCALHOST)(PORT=1521)))(CONNECT_DATA=(SERVICE_NAME=freepdb1)))", Username: "SECRET_SCHEMA"},
		{Name: "ALIAS", Host: "PRODDB", Username: "SECRET_SCHEMA"},
		{Name: "NOUSER", Host: "localhost:1521/FREEPDB1"},
		{Name: "DOMAINED.WORLD", Host: "localhost:1521/FREEPDB1", Username: "SECRET_SCHEMA"},
		{Name: "TWICE", Host: "localhost:1521/FREEPDB1", Username: "SECRET_SCHEMA"},
		{Name: "TWICE", Host: "elsewhere:1521/FREEPDB1", Username: "SECRET_SCHEMA"},
		{Name: "UNSYNCED", Host: "unsynced-host:1521/FREEPDB1", Username: "SECRET_SCHEMA"},
		{Name: "BYSID", Host: "(DESCRIPTION=(ADDRESS=(HOST=localhost)(PORT=1521))(CONNECT_DATA=(SID=FREEPDB1)))", Username: "SECRET_SCHEMA"},
	}
	seedInstance := func(instanceID string, dataSource *storepb.DataSource, databases ...string) {
		t.Helper()
		_, err := s.CreateInstance(ctx, &store.InstanceMessage{
			ResourceID: instanceID,
			Workspace:  "default",
			Metadata: &storepb.Instance{
				Engine:      storepb.Engine_ORACLE,
				DataSources: []*storepb.DataSource{dataSource},
			},
		})
		require.NoError(t, err)
		for _, name := range databases {
			_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
				ProjectID:    "project-a",
				InstanceID:   instanceID,
				DatabaseName: name,
				Metadata:     &storepb.DatabaseMetadata{},
			})
			require.NoError(t, err)
			require.NoError(t, s.UpsertDBSchema(ctx, instanceID, name, &metadatapb.DatabaseSchemaMetadata{
				Name:            name,
				Schemas:         []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{{Name: "SECRET_T"}}}},
				LinkedDatabases: links,
			}, &storepb.DatabaseConfig{}, nil))
		}
	}
	// The instance the statements run against; its data source reaches the loopback links.
	seedInstance("ora-a", &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN, Host: "localhost", Port: "1521", ServiceName: "FREEPDB1"}, "ALLOWED_S", "SECRET_SCHEMA")
	// Instances a substring match would have picked: an empty host is a substring of every
	// link host, and "local" is a substring of "localhost".
	seedInstance("ora-empty-host", &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN, Port: "1521", ServiceName: "FREEPDB1"}, "SECRET_SCHEMA")
	seedInstance("ora-prefix-host", &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN, Host: "local", Port: "1521", ServiceName: "FREEPDB1"}, "SECRET_SCHEMA")
	// Same host, other service: a different database on the same listener.
	seedInstance("ora-other-service", &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN, Host: "localhost", Port: "1521", ServiceName: "OTHERPDB"}, "SECRET_SCHEMA")
	// Same host and spelling, but connecting by SID: only a SID link reaches it.
	seedInstance("ora-sid", &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN, Host: "localhost", Port: "1521", Sid: "FREEPDB1"}, "SECRET_SCHEMA")
	// Reached by the UNSYNCED link, but its database has no synced schema yet.
	seedInstance("ora-unsynced", &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN, Host: "unsynced-host", Port: "1521", ServiceName: "FREEPDB1"})
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{ProjectID: "project-a", InstanceID: "ora-unsynced", DatabaseName: "SECRET_SCHEMA", Metadata: &storepb.DatabaseMetadata{}})
	require.NoError(t, err)

	resolve := BuildGetLinkedDatabaseMetadataFunc(s, storepb.Engine_ORACLE)
	require.Nil(t, BuildGetLinkedDatabaseMetadataFunc(s, storepb.Engine_POSTGRES))

	tests := []struct {
		name         string
		link         string
		schema       string
		wantInstance string
		wantDatabase string
	}{
		{"schema-qualified reference", "REMOTE", "SECRET_SCHEMA", "ora-a", "SECRET_SCHEMA"},
		{"unqualified reference uses the link user", "REMOTE", "", "ora-a", "SECRET_SCHEMA"},
		{"link name case", "remote", "", "ora-a", "SECRET_SCHEMA"},
		{"domained link named without its domain", "DOMAINED", "", "ora-a", "SECRET_SCHEMA"},
		{"RAC descriptor listing the data source among its addresses", "RAC", "", "ora-a", "SECRET_SCHEMA"},
		{"SID descriptor reaches the SID data source, not the service one", "BYSID", "", "ora-sid", "SECRET_SCHEMA"},
		{"schema Bytebase does not track", "REMOTE", "NOT_SYNCED", "", ""},
		{"host no data source reaches", "UNMANAGED", "", "", ""},
		{"TNS alias", "ALIAS", "", "", ""},
		{"no connect user and no schema", "NOUSER", "", "", ""},
		{"defined twice with different targets", "TWICE", "", "", ""},
		{"linked database not synced yet", "UNSYNCED", "", "", ""},
		{"unknown link", "NOSUCHLINK", "", "", ""},
	}
	for _, tc := range tests {
		instanceID, databaseName, meta, err := resolve(ctx, "ora-a", tc.link, tc.schema)
		require.NoError(t, err, tc.name)
		require.Equal(t, tc.wantInstance, instanceID, tc.name)
		require.Equal(t, tc.wantDatabase, databaseName, tc.name)
		if tc.wantInstance == "" {
			require.Nil(t, meta, tc.name)
			continue
		}
		require.NotNil(t, meta, tc.name)
		require.NotNil(t, meta.GetSchemaMetadata("").GetTable("SECRET_T"), tc.name)
	}

	// A second instance whose data source reaches the same listener and service makes the link ambiguous.
	seedInstance("ora-a-twin", &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN, Host: "LOCALHOST", Port: "1521", ServiceName: "freepdb1"}, "SECRET_SCHEMA")
	instanceID, _, meta, err := resolve(ctx, "ora-a", "REMOTE", "")
	require.NoError(t, err)
	require.Empty(t, instanceID)
	require.Nil(t, meta)
}
