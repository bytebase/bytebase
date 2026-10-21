package v1

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestPreCheckAccessQueryHash(t *testing.T) {
	t.Parallel()
	longSQL := "SELECT '" + strings.Repeat("x", 110_000) + "' AS v"
	for _, tc := range []struct {
		name           string
		approved       string
		statement      string
		dropHash       bool
		grantSchema    string
		grantContainer string
		schema         *string
		container      string
		want           bool
	}{
		{name: "plain", approved: "SELECT 1", statement: "SELECT 1", want: true},
		{name: "boundary whitespace", approved: "\v\fSELECT 1\r\n", statement: " \tSELECT 1\n", want: true},
		{name: "long SQL", approved: longSQL, statement: longSQL, want: true},
		{name: "different query", approved: "SELECT 1", statement: "SELECT 2"},
		{name: "comment newline", approved: "SELECT 1 --\nWHERE false", statement: "SELECT 1 -- WHERE false"},
		{name: "internal whitespace", approved: "SELECT 'a  b'", statement: "SELECT 'a b'"},
		{name: "Unicode boundary", approved: "\u00a0SELECT 1", statement: "SELECT 1"},
		{name: "missing stored hash", approved: "SELECT 1", statement: "SELECT 1", dropHash: true},
		{name: "write statement", approved: "DELETE FROM t", statement: "DELETE FROM t"},
		{name: "grant schema, query without", approved: "SELECT 1", statement: "SELECT 1", grantSchema: "public"},
		{name: "grant schema, query with it", approved: "SELECT 1", statement: "SELECT 1", grantSchema: "public", schema: new("public"), want: true},
		{name: "grant container, query with it", approved: "SELECT 1", statement: "SELECT 1", grantContainer: "employee", container: "employee", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
			ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{Email: "caller@example.com"})
			db, stores, _ := testcontainer.NewMetadataDB(t)
			_, err := db.ExecContext(ctx, `
    INSERT INTO workspace (resource_id) VALUES ('default');
    INSERT INTO project (resource_id, workspace, name, setting)
    VALUES ('grant-project', 'default', 'Grant Project', '{"allowJustInTimeAccess":true}');`)
			require.NoError(t, err)
			expires := time.Now().Add(time.Hour)
			grant, err := stores.CreateAccessGrant(ctx, &store.AccessGrantMessage{
				ProjectID: "grant-project", Creator: "caller@example.com", Status: storepb.AccessGrant_ACTIVE, ExpireTime: &expires,
				Payload: &storepb.AccessGrantPayload{
					Query: tc.approved, Targets: []string{"instances/test/databases/db"}, Export: true, Unmask: true,
					Schema: tc.grantSchema, Container: tc.grantContainer,
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, grant.Payload.QueryHash)
			if tc.dropHash {
				_, err = db.ExecContext(ctx, `UPDATE access_grant SET payload = payload - 'queryHash' WHERE id = $1`, grant.ID)
				require.NoError(t, err)
			}
			service := &SQLService{store: stores}
			instance := &store.InstanceMessage{ResourceID: "test", Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES}}
			database := &store.DatabaseMessage{ProjectID: "grant-project", InstanceID: "test", DatabaseName: "db"}
			for _, requireExport := range []bool{false, true} {
				got := service.preCheckAccess(ctx, tc.statement, instance, database, tc.schema, tc.container, requireExport)
				if tc.want {
					require.NotNil(t, got)
					require.Equal(t, grant.ID, got.ID)
				} else {
					require.Nil(t, got)
				}
			}
		})
	}
}

// TestAccessGrantDoesNotExemptExplain pins the boundary between data access and
// plan access: a grant on the database still lets its holder select there without
// a role, but explaining there needs bb.sql.explain from IAM.
func TestAccessGrantDoesNotExemptExplain(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
    INSERT INTO workspace (resource_id) VALUES ('default');
    INSERT INTO principal (name, email, password_hash) VALUES ('selector', 'selector@example.com', 'unused'), ('explainer', 'explainer@example.com', 'unused');
    INSERT INTO project (resource_id, workspace, name) VALUES ('grant-project', 'default', 'Grant Project');`)
	require.NoError(t, err)
	for _, role := range []struct {
		id   string
		perm permission.Permission
		user string
	}{
		{id: "selector", perm: permission.SQLSelect, user: "selector@example.com"},
		{id: "explainer", perm: permission.SQLExplain, user: "explainer@example.com"},
	} {
		_, err = stores.CreateRole(ctx, &store.RoleMessage{
			ResourceID: role.id, Workspace: "default", Name: role.id,
			Permissions: map[permission.Permission]bool{role.perm: true},
		})
		require.NoError(t, err)
		_, err = stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
			Workspace: "default", Member: common.FormatUserEmail(role.user), Roles: []string{"roles/" + role.id},
		})
		require.NoError(t, err)
	}
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := &SQLService{store: stores, iamManager: iamManager}
	instance := &store.InstanceMessage{ResourceID: "test", Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES}}
	database := &store.DatabaseMessage{ProjectID: "grant-project", InstanceID: "test", DatabaseName: "db"}
	granted := map[string]struct{}{"instances/test/databases/db": {}}
	check := func(email string, isExplain bool) error {
		user := &store.UserMessage{Email: email}
		return service.accessCheckWithGrantedTargets(ctx, instance, database, user, nil, isExplain, nil, granted, "", false)
	}

	require.NoError(t, check("selector@example.com", false), "the grant covers select")
	require.NoError(t, check("nobody@example.com", false), "the grant covers select without any role")
	require.NoError(t, check("explainer@example.com", true), "explain is checked against IAM")
	var denied *queryError
	require.ErrorAs(t, check("selector@example.com", true), &denied, "the grant does not cover explain")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(denied.err))
	require.Equal(t, permission.SQLExplain, denied.permission)
}
