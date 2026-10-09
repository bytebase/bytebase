package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestAccessGrantQueryHash(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
 INSERT INTO workspace (resource_id) VALUES ('default');
 INSERT INTO project (resource_id, workspace, name) VALUES ('hash', 'default', 'Hash');
 `)
	require.NoError(t, err)
	data, err := os.ReadFile("../../testdata/access_grant_query_hash.json")
	require.NoError(t, err)
	var vectors []struct {
		Name  string
		Input string
		Hash  string
	}
	require.NoError(t, json.Unmarshal(data, &vectors))
	expires := time.Now().Add(time.Hour)
	project, creator := "hash", "hash@example.com"
	for i, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			t.Parallel()
			normalized := strings.Trim(vector.Input, " \t\n\r\v\f")
			digest := sha256.Sum256([]byte(normalized))
			require.Equal(t, vector.Hash, hex.EncodeToString(digest[:]))
			target := fmt.Sprintf("instances/hash/databases/db%d", i)
			grant, err := stores.CreateAccessGrant(ctx, &store.AccessGrantMessage{
				ProjectID: project, Creator: creator, Status: storepb.AccessGrant_ACTIVE, ExpireTime: &expires,
				Payload: &storepb.AccessGrantPayload{Query: vector.Input, Targets: []string{target}, Export: true},
			})
			require.NoError(t, err)
			filter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q && target == %q && schema == "" && container == "" && status == "ACTIVE" && export == true`, vector.Hash, target))
			require.NoError(t, err)
			grants, err := stores.ListAccessGrants(ctx, &store.FindAccessGrantMessage{Workspace: "default", ProjectID: &project, Creator: &creator, FilterQ: filter})
			require.NoError(t, err)
			require.Len(t, grants, 1)
			require.Equal(t, grant.ID, grants[0].ID)
			wrongDigest := sha256.Sum256([]byte(normalized + "v"))
			wrongFilter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q && target == %q`, hex.EncodeToString(wrongDigest[:]), target))
			require.NoError(t, err)
			grants, err = stores.ListAccessGrants(ctx, &store.FindAccessGrantMessage{Workspace: "default", ProjectID: &project, Creator: &creator, FilterQ: wrongFilter})
			require.NoError(t, err)
			require.Empty(t, grants)
			runtimeFind := &store.FindActiveAccessGrantMessage{Workspace: "default", ProjectID: project, Creator: creator, Target: target, Statement: normalized, RequireExport: true, ExpireTime: time.Now()}
			grants, err = stores.ListActiveAccessGrants(ctx, runtimeFind)
			require.NoError(t, err)
			require.Len(t, grants, 1)
			require.Equal(t, grant.ID, grants[0].ID)
			runtimeFind.Statement = normalized + "v"
			grants, err = stores.ListActiveAccessGrants(ctx, runtimeFind)
			require.NoError(t, err)
			require.Empty(t, grants)
			if len(vector.Input) < 100000 {
				encoded, err := json.Marshal(vector.Input)
				require.NoError(t, err)
				filter, err := store.GetListAccessGrantFilter("query == " + string(encoded))
				require.NoError(t, err)
				grants, err = stores.ListAccessGrants(ctx, &store.FindAccessGrantMessage{ID: &grant.ID, FilterQ: filter})
				require.NoError(t, err)
				require.Len(t, grants, 1)
			}
		})
	}
}

func TestAccessGrantHashDiscoveryScope(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
 INSERT INTO workspace (resource_id) VALUES ('first'), ('second');
 INSERT INTO project (resource_id, workspace, name) VALUES ('first', 'first', 'First'), ('other', 'first', 'Other'), ('second', 'second', 'Second');
 `)
	require.NoError(t, err)
	project, creator := "first", "alice@example.com"
	expires, expired := time.Now().Add(time.Hour), time.Now().Add(-time.Hour)
	query := "SELECT 1"
	digest := sha256.Sum256([]byte(query))
	hash := hex.EncodeToString(digest[:])
	create := func(projectID, user, schema, container, target string, status storepb.AccessGrant_Status, expiry time.Time, export bool) string {
		grant, err := stores.CreateAccessGrant(ctx, &store.AccessGrantMessage{ProjectID: projectID, Creator: user, Status: status, ExpireTime: &expiry, Payload: &storepb.AccessGrantPayload{Query: query, Schema: schema, Container: container, Targets: []string{target}, Export: export}})
		require.NoError(t, err)
		return grant.ID
	}
	target := "instances/prod/databases/app"
	emptyID := create(project, creator, "", "", target, storepb.AccessGrant_ACTIVE, expires, true)
	scopedID := create(project, creator, "public", "orders", target, storepb.AccessGrant_ACTIVE, expires, true)
	create("other", creator, "", "", target, storepb.AccessGrant_ACTIVE, expires, true)
	create("second", creator, "", "", target, storepb.AccessGrant_ACTIVE, expires, true)
	create(project, "bob@example.com", "", "", target, storepb.AccessGrant_ACTIVE, expires, true)
	create(project, creator, "", "", "instances/other/databases/app", storepb.AccessGrant_ACTIVE, expires, true)
	create(project, creator, "", "", target, storepb.AccessGrant_REVOKED, expires, true)
	create(project, creator, "", "", target, storepb.AccessGrant_ACTIVE, expired, true)
	create(project, creator, "", "", target, storepb.AccessGrant_ACTIVE, expires, false)
	for _, tc := range []struct{ schema, container, want string }{{"", "", emptyID}, {"public", "orders", scopedID}, {"public", "", ""}, {"", "orders", ""}} {
		t.Run(tc.schema+"/"+tc.container, func(t *testing.T) {
			t.Parallel()
			filter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q && target == %q && schema == %q && container == %q && status == "ACTIVE" && export == true`, hash, target, tc.schema, tc.container))
			require.NoError(t, err)
			grants, err := stores.ListAccessGrants(ctx, &store.FindAccessGrantMessage{Workspace: "first", ProjectID: &project, Creator: &creator, FilterQ: filter})
			require.NoError(t, err)
			if tc.want == "" {
				require.Empty(t, grants)
			} else {
				require.Len(t, grants, 1)
				require.Equal(t, tc.want, grants[0].ID)
			}
		})
	}
}
