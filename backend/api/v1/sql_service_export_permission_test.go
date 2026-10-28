package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

// exportAccessFixture is a MySQL database db whose users hold different mixes
// of read, export and write permissions on it.
type exportAccessFixture struct {
	service  *SQLService
	instance *store.InstanceMessage
	database *store.DatabaseMessage
}

func newExportAccessFixture(ctx context.Context, t *testing.T) *exportAccessFixture {
	t.Helper()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
    INSERT INTO workspace (resource_id) VALUES ('default');
    INSERT INTO principal (name, email, password_hash) VALUES
      ('reader', 'reader@example.com', 'unused'),
      ('exporter', 'exporter@example.com', 'unused'),
      ('table-exporter', 'table-exporter@example.com', 'unused'),
      ('explain-exporter', 'explain-exporter@example.com', 'unused'),
      ('writer', 'writer@example.com', 'unused'),
      ('grantee', 'grantee@example.com', 'unused');
    INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'A');`)
	require.NoError(t, err)
	for _, role := range []struct {
		id    string
		perms map[permission.Permission]bool
	}{
		{id: "reader", perms: map[permission.Permission]bool{permission.SQLSelect: true, permission.SQLInfo: true, permission.SQLExplain: true}},
		{id: "exporter", perms: map[permission.Permission]bool{permission.SQLExport: true}},
		{id: "writer", perms: map[permission.Permission]bool{permission.SQLDml: true}},
	} {
		_, err = stores.CreateRole(ctx, &store.RoleMessage{
			ResourceID: role.id, Workspace: "default", Name: role.id, Permissions: role.perms,
		})
		require.NoError(t, err)
	}
	bindings := []*storepb.Binding{
		{Role: "roles/reader", Members: []string{common.FormatUserEmail("reader@example.com"), common.FormatUserEmail("explain-exporter@example.com")}},
		{Role: "roles/exporter", Members: []string{common.FormatUserEmail("exporter@example.com"), common.FormatUserEmail("explain-exporter@example.com")}},
		{
			Role:      "roles/exporter",
			Members:   []string{common.FormatUserEmail("table-exporter@example.com")},
			Condition: &expr.Expr{Expression: `resource.database == "instances/test/databases/db" && resource.table_name in ["t1"]`},
		},
		{Role: "roles/writer", Members: []string{common.FormatUserEmail("writer@example.com")}},
	}
	_, err = stores.PatchIamPolicy(ctx, "default", storepb.Policy_PROJECT, common.FormatProject("project-a"), func(policy *storepb.IamPolicy) error {
		policy.Bindings = append(policy.Bindings, bindings...)
		return nil
	})
	require.NoError(t, err)
	instance, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "test", Workspace: "default",
		Metadata: &storepb.Instance{Engine: storepb.Engine_MYSQL, Activation: true, DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}}},
	})
	require.NoError(t, err)
	database, err := stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "test", DatabaseName: "db", ProjectID: "project-a", Metadata: &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	return &exportAccessFixture{
		service:  &SQLService{store: stores, iamManager: iamManager},
		instance: instance,
		database: database,
	}
}

func (f *exportAccessFixture) check(ctx context.Context, email string, export bool, grant *store.AccessGrantMessage, statement string, spans ...*parserbase.QuerySpan) error {
	user := &store.UserMessage{Email: email}
	return f.service.accessCheckWithGrant(grant, export)(ctx, f.instance, f.database, user, spans, false, []parserbase.Statement{{Text: statement}}, "", false)
}

func readsTable(database, table string) *parserbase.QuerySpan {
	return &parserbase.QuerySpan{Type: parserbase.Select, SourceColumns: parserbase.SourceColumnSet{{Database: database, Table: table}: true}}
}

func requireDenied(t *testing.T, err error, perm permission.Permission, resources ...string) {
	t.Helper()
	var denied *queryError
	require.ErrorAs(t, err, &denied)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(denied.err))
	require.Equal(t, perm, denied.permission)
	require.Equal(t, resources, denied.resources)
}

func TestExportRequiresExportPermission(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	f := newExportAccessFixture(ctx, t)
	const (
		db       = "instances/test/databases/db"
		t1       = db + "/tables/t1"
		t2       = db + "/tables/t2"
		query    = false
		export   = true
		selectT1 = "SELECT * FROM t1"
	)

	t.Run("select alone does not export", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, f.check(ctx, "reader@example.com", query, nil, selectT1, readsTable("db", "t1")))
		requireDenied(t, f.check(ctx, "reader@example.com", export, nil, selectT1, readsTable("db", "t1")), permission.SQLExport, t1)
	})

	t.Run("export scoped to one table", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, f.check(ctx, "table-exporter@example.com", export, nil, selectT1, readsTable("db", "t1")))
		requireDenied(t, f.check(ctx, "table-exporter@example.com", export, nil, "SELECT * FROM t2", readsTable("db", "t2")), permission.SQLExport, t2)
		requireDenied(t, f.check(ctx, "table-exporter@example.com", query, nil, selectT1, readsTable("db", "t1")), permission.SQLSelect, t1)
	})

	t.Run("info schema and engines without spans", func(t *testing.T) {
		t.Parallel()
		info := &parserbase.QuerySpan{Type: parserbase.SelectInfoSchema}
		require.NoError(t, f.check(ctx, "reader@example.com", query, nil, "SHOW TABLES", info))
		requireDenied(t, f.check(ctx, "reader@example.com", export, nil, "SHOW TABLES", info), permission.SQLExport, db)
		require.NoError(t, f.check(ctx, "exporter@example.com", export, nil, "SHOW TABLES", info))

		require.NoError(t, f.check(ctx, "reader@example.com", query, nil, selectT1))
		requireDenied(t, f.check(ctx, "reader@example.com", export, nil, selectT1), permission.SQLExport, db)
		require.NoError(t, f.check(ctx, "exporter@example.com", export, nil, selectT1))
	})

	t.Run("explain keeps bb.sql.explain", func(t *testing.T) {
		t.Parallel()
		plan := &parserbase.QuerySpan{Type: parserbase.Explain, SourceColumns: parserbase.SourceColumnSet{{Database: "db", Table: "t1"}: true}}
		require.NoError(t, f.check(ctx, "reader@example.com", export, nil, "EXPLAIN SELECT * FROM t1", plan))
		requireDenied(t, f.check(ctx, "exporter@example.com", export, nil, "EXPLAIN SELECT * FROM t1", plan), permission.SQLExplain, db)

		analyze := readsTable("db", "t1")
		analyze.ExplainAnalyze = true
		requireDenied(t, f.check(ctx, "reader@example.com", export, nil, "EXPLAIN ANALYZE SELECT * FROM t1", analyze), permission.SQLExport, t1)
		require.NoError(t, f.check(ctx, "explain-exporter@example.com", export, nil, "EXPLAIN ANALYZE SELECT * FROM t1", analyze))
	})

	t.Run("export grant", func(t *testing.T) {
		t.Parallel()
		grant := &store.AccessGrantMessage{Payload: &storepb.AccessGrantPayload{Targets: []string{db}, Export: true}}
		requireDenied(t, f.check(ctx, "grantee@example.com", export, nil, selectT1, readsTable("db", "t1")), permission.SQLExport, t1)
		require.NoError(t, f.check(ctx, "grantee@example.com", export, grant, selectT1, readsTable("db", "t1")))
		require.NoError(t, f.check(ctx, "grantee@example.com", export, grant, selectT1))
		requireDenied(t, f.check(ctx, "grantee@example.com", export, grant, "SELECT * FROM other.t1", readsTable("other", "t1")), permission.SQLExport, "instances/test/databases/other/tables/t1")
		require.NoError(t, f.check(ctx, "grantee@example.com", query, grant, selectT1, readsTable("db", "t1")))
	})

	t.Run("DML keeps bb.sql.dml", func(t *testing.T) {
		t.Parallel()
		const insert = "INSERT INTO t1 VALUES (1)"
		write := &parserbase.QuerySpan{Type: parserbase.DML}
		requireDenied(t, f.check(ctx, "exporter@example.com", export, nil, insert, write), permission.SQLDml, db)
		require.NoError(t, f.check(ctx, "writer@example.com", export, nil, insert, write))
		grant := &store.AccessGrantMessage{Payload: &storepb.AccessGrantPayload{Targets: []string{db}, Export: true}}
		requireDenied(t, f.check(ctx, "grantee@example.com", export, grant, insert, write), permission.SQLDml, db)
	})
}
