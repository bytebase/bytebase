package v1

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestPreCheckAccessQueryHash(t *testing.T) {
	t.Parallel()
	longSQL := "SELECT '" + strings.Repeat("x", 110_000) + "' AS v"
	for _, tc := range []struct {
		name      string
		approved  string
		statement string
		dropHash  bool
		want      bool
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
				got := service.preCheckAccess(ctx, tc.statement, instance, database, nil, "", requireExport)
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
