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
	"google.golang.org/protobuf/encoding/protojson"

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
	vectors := readAccessGrantQueryHashVectors(t)
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
				Payload: &storepb.AccessGrantPayload{Query: vector.Input, QueryHash: "untrusted", Targets: []string{target}, Export: true},
			})
			require.NoError(t, err)
			require.Equal(t, vector.Hash, grant.Payload.QueryHash)
			filter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q && target == %q && schema == "" && container == "" && status == "ACTIVE" && export == true`, vector.Hash, target))
			require.NoError(t, err)
			grants, err := stores.ListAccessGrants(ctx, &store.FindAccessGrantMessage{Workspace: "default", ProjectID: &project, Creator: &creator, FilterQ: filter})
			require.NoError(t, err)
			require.Len(t, grants, 1)
			require.Equal(t, grant.ID, grants[0].ID)
			require.Equal(t, vector.Hash, grants[0].Payload.QueryHash)
			wrongDigest := sha256.Sum256([]byte(normalized + "v"))
			wrongFilter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q && target == %q`, hex.EncodeToString(wrongDigest[:]), target))
			require.NoError(t, err)
			grants, err = stores.ListAccessGrants(ctx, &store.FindAccessGrantMessage{Workspace: "default", ProjectID: &project, Creator: &creator, FilterQ: wrongFilter})
			require.NoError(t, err)
			require.Empty(t, grants)
			runtimeFind := &store.FindActiveAccessGrantMessage{Workspace: "default", ProjectID: project, Creator: creator, Target: target, Statement: vector.Input, RequireExport: true, ExpireTime: time.Now()}
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

func TestAccessGrantQueryHashUpdates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
 INSERT INTO workspace (resource_id) VALUES ('default');
 INSERT INTO project (resource_id, workspace, name) VALUES ('hash', 'default', 'Hash');
 `)
	require.NoError(t, err)
	grant, err := stores.CreateAccessGrant(ctx, &store.AccessGrantMessage{
		ProjectID: "hash", Creator: "hash@example.com", Status: storepb.AccessGrant_PENDING,
		Payload: &storepb.AccessGrantPayload{Query: "SELECT 0"},
	})
	require.NoError(t, err)
	for _, vector := range readAccessGrantQueryHashVectors(t) {
		oldHash := grant.Payload.QueryHash
		grant.Payload.Query = vector.Input
		grant, err = stores.UpdateAccessGrant(ctx, grant.ID, &store.UpdateAccessGrantMessage{Payload: grant.Payload})
		require.NoError(t, err)
		require.Equal(t, vector.Input, grant.Payload.Query)
		require.Equal(t, vector.Hash, grant.Payload.QueryHash)
		filter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q`, vector.Hash))
		require.NoError(t, err)
		matched, err := stores.GetAccessGrant(ctx, &store.FindAccessGrantMessage{ID: &grant.ID, FilterQ: filter})
		require.NoError(t, err)
		require.NotNil(t, matched)
		if oldHash != vector.Hash {
			filter, err = store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q`, oldHash))
			require.NoError(t, err)
			matched, err = stores.GetAccessGrant(ctx, &store.FindAccessGrantMessage{ID: &grant.ID, FilterQ: filter})
			require.NoError(t, err)
			require.Nil(t, matched)
		}
	}

	wantHash := grant.Payload.QueryHash
	grant.Payload.IssueId = 123
	grant.Payload.QueryHash = "untrusted"
	grant, err = stores.UpdateAccessGrant(ctx, grant.ID, &store.UpdateAccessGrantMessage{Payload: grant.Payload})
	require.NoError(t, err)
	require.Equal(t, wantHash, grant.Payload.QueryHash)
	require.Equal(t, int64(123), grant.Payload.IssueId)
	for _, status := range []storepb.AccessGrant_Status{storepb.AccessGrant_ACTIVE, storepb.AccessGrant_REVOKED} {
		expires := time.Now().Add(time.Hour)
		grant, err = stores.UpdateAccessGrant(ctx, grant.ID, &store.UpdateAccessGrantMessage{Status: &status, ExpireTime: &expires})
		require.NoError(t, err)
		require.Equal(t, wantHash, grant.Payload.QueryHash)
	}

	grant.Payload.Query = ""
	grant, err = stores.UpdateAccessGrant(ctx, grant.ID, &store.UpdateAccessGrantMessage{Payload: grant.Payload})
	require.NoError(t, err)
	require.Empty(t, grant.Payload.QueryHash)
	filter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q`, wantHash))
	require.NoError(t, err)
	matched, err := stores.GetAccessGrant(ctx, &store.FindAccessGrantMessage{ID: &grant.ID, FilterQ: filter})
	require.NoError(t, err)
	require.Nil(t, matched)
}

func TestAccessGrantQueryHashMigration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
 INSERT INTO workspace (resource_id) VALUES ('default');
 INSERT INTO project (resource_id, workspace, name) VALUES ('hash', 'default', 'Hash');
 DROP INDEX idx_access_grant_project_creator_query_hash;
 `)
	require.NoError(t, err)
	vectors := readAccessGrantQueryHashVectors(t)
	timestamp := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	statuses := []string{"PENDING", "ACTIVE", "REVOKED"}
	for i, vector := range vectors {
		payload, err := protojson.Marshal(&storepb.AccessGrantPayload{
			Query: vector.Input, Targets: []string{"instances/hash/databases/db"}, Export: true,
			Schema: "public", Container: "orders", IssueId: 123,
		})
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO access_grant (id, project, creator, status, expire_time, payload, created_at, updated_at)
 VALUES ($1, 'hash', 'hash@example.com', $2, $3, $4, $3, $3)`, vector.Name, statuses[i%len(statuses)], timestamp, payload)
		require.NoError(t, err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO access_grant (id, project, creator, payload)
 VALUES ('missing-query', 'hash', 'hash@example.com', '{"reason":"preserved"}'),
        ('null-query', 'hash', 'hash@example.com', '{"query":null,"reason":"preserved"}')`)
	require.NoError(t, err)
	statement, err := os.ReadFile("../migrator/migration/3.24/0000##access_grant_query_hash.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)
	for i, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			t.Parallel()
			filter, err := store.GetListAccessGrantFilter(fmt.Sprintf(`query_hash == %q`, vector.Hash))
			require.NoError(t, err)
			grant, err := stores.GetAccessGrant(ctx, &store.FindAccessGrantMessage{ID: &vector.Name, FilterQ: filter})
			require.NoError(t, err)
			require.NotNil(t, grant)
			require.Equal(t, vector.Input, grant.Payload.Query)
			require.Equal(t, vector.Hash, grant.Payload.QueryHash)
			require.Equal(t, []string{"instances/hash/databases/db"}, grant.Payload.Targets)
			require.True(t, grant.Payload.Export)
			require.Equal(t, "public", grant.Payload.Schema)
			require.Equal(t, "orders", grant.Payload.Container)
			require.Equal(t, int64(123), grant.Payload.IssueId)
			require.Equal(t, statuses[i%len(statuses)], grant.Status.String())
			require.True(t, timestamp.Equal(*grant.ExpireTime))
			require.True(t, timestamp.Equal(grant.CreatedAt))
			require.True(t, timestamp.Equal(grant.UpdatedAt))
		})
	}
	for _, id := range []string{"missing-query", "null-query"} {
		var payload string
		require.NoError(t, db.QueryRowContext(ctx, `SELECT payload FROM access_grant WHERE id = $1`, id).Scan(&payload))
		want := `{"reason":"preserved"}`
		if id == "null-query" {
			want = `{"query":null,"reason":"preserved"}`
		}
		require.JSONEq(t, want, payload)
	}
}

func readAccessGrantQueryHashVectors(t *testing.T) []struct{ Name, Input, Hash string } {
	t.Helper()
	data, err := os.ReadFile("../../testdata/access_grant_query_hash.json")
	require.NoError(t, err)
	var vectors []struct{ Name, Input, Hash string }
	require.NoError(t, json.Unmarshal(data, &vectors))
	return vectors
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
			runtimeGrants, err := stores.ListActiveAccessGrants(ctx, &store.FindActiveAccessGrantMessage{
				Workspace: "first", ProjectID: project, Creator: creator, Target: target, Statement: query,
				Schema: tc.schema, Container: tc.container, RequireExport: true, ExpireTime: time.Now(),
			})
			require.NoError(t, err)
			require.Equal(t, grants, runtimeGrants)
			if tc.want == "" {
				require.Empty(t, grants)
			} else {
				require.Len(t, grants, 1)
				require.Equal(t, tc.want, grants[0].ID)
			}
		})
	}
}
