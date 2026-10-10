package v1

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSearchMyAccessGrantsQueryHash(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{Email: "caller@example.com"})
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default'), ('other');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('grant-project', 'default', 'Grant Project'),
			('other-project', 'default', 'Other Project'),
			('foreign-project', 'other', 'Foreign Project');
	`)
	require.NoError(t, err)
	statement := "SELECT '" + strings.Repeat("x", 110_000) + "' AS v"
	expire := time.Now().Add(time.Hour)
	create := func(project, creator, schema string, expiry time.Time) *store.AccessGrantMessage {
		t.Helper()
		grant, err := stores.CreateAccessGrant(ctx, &store.AccessGrantMessage{
			ProjectID: project, Creator: creator, Status: storepb.AccessGrant_ACTIVE, ExpireTime: &expiry,
			Payload: &storepb.AccessGrantPayload{
				Query: "\v\n" + statement + "\r\f", Targets: []string{"instances/test/databases/db"},
				Schema: schema, Export: true,
			},
		})
		require.NoError(t, err)
		return grant
	}
	matched := create("grant-project", "caller@example.com", "public", expire)
	create("grant-project", "other@example.com", "public", expire)
	create("other-project", "caller@example.com", "public", expire)
	create("foreign-project", "caller@example.com", "public", expire)
	create("grant-project", "caller@example.com", "", expire)
	create("grant-project", "caller@example.com", "public", time.Now().Add(-time.Hour))
	service := &AccessGrantService{store: stores}
	filter := fmt.Sprintf(`query_hash == "%x" && schema == "public" && container == "" && target == "instances/test/databases/db" && status == "ACTIVE" && export == true`, sha256.Sum256([]byte(statement)))
	response, err := service.SearchMyAccessGrants(ctx, connect.NewRequest(&v1pb.SearchMyAccessGrantsRequest{
		Parent: "projects/grant-project", Filter: filter, PageSize: 1,
	}))
	require.NoError(t, err)
	require.Len(t, response.Msg.AccessGrants, 1)
	require.Equal(t, "projects/grant-project/accessGrants/"+matched.ID, response.Msg.AccessGrants[0].Name)
	require.Empty(t, response.Msg.NextPageToken)
	response, err = service.SearchMyAccessGrants(ctx, connect.NewRequest(&v1pb.SearchMyAccessGrantsRequest{
		Parent: "projects/foreign-project", Filter: filter,
	}))
	require.NoError(t, err)
	require.Empty(t, response.Msg.AccessGrants)
}

func TestIsReadOnlyStatementForAccessGrantRejectsDocumentEngineWriteStatements(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		engine    storepb.Engine
		statement string
	}{
		{
			name:      "MongoDB DML",
			engine:    storepb.Engine_MONGODB,
			statement: `db.users.insertOne({name: "Bytebase"})`,
		},
		{
			name:      "MongoDB DDL",
			engine:    storepb.Engine_MONGODB,
			statement: `db.createCollection("users")`,
		},
		{
			name:      "Elasticsearch DML",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "POST /users/_doc\n{\"name\":\"Bytebase\"}",
		},
		{
			name:      "Elasticsearch DDL",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "PUT /users",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			readOnly, err := isReadOnlyStatementForAccessGrant(context.Background(), tc.engine, tc.statement)
			require.NoError(t, err)
			require.False(t, readOnly)
		})
	}
}

func TestIsReadOnlyStatementForAccessGrantAllowsDocumentEngineReadStatements(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		engine    storepb.Engine
		statement string
	}{
		{
			name:      "MongoDB read",
			engine:    storepb.Engine_MONGODB,
			statement: `db.users.find({})`,
		},
		{
			name:      "Elasticsearch read",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "GET /users/_search",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			readOnly, err := isReadOnlyStatementForAccessGrant(context.Background(), tc.engine, tc.statement)
			require.NoError(t, err)
			require.True(t, readOnly)
		})
	}
}

func TestIsReadOnlyStatementForAccessGrantRejectsDocumentEngineInvalidStatements(t *testing.T) {
	t.Parallel()
	readOnly, err := isReadOnlyStatementForAccessGrant(context.Background(), storepb.Engine_ELASTICSEARCH, `db.users.find({})`)
	require.Error(t, err)
	require.False(t, readOnly)
}

// TestConvertToAccessGrantPropagatesPayloadFields pins the fix that
// `Reason` (alongside the existing Targets / Query / Unmask / Export)
// must be copied from the store payload onto the v1 message — otherwise
// frontend tooltips and audit displays that depend on the user-typed
// reason silently render empty.
func TestConvertToAccessGrantPropagatesPayloadFields(t *testing.T) {
	t.Parallel()
	expire := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	msg := &store.AccessGrantMessage{
		ProjectID:  "proj",
		ID:         "ag1",
		Creator:    "dev@example.com",
		Status:     storepb.AccessGrant_ACTIVE,
		ExpireTime: &expire,
		Payload: &storepb.AccessGrantPayload{
			Targets:   []string{"instances/inst/databases/db"},
			Query:     "SELECT * FROM t",
			Unmask:    true,
			Export:    true,
			Reason:    "investigating PR-1234",
			Schema:    "APP",
			Container: "orders",
		},
	}

	ag := convertToAccessGrant(msg)

	require.Equal(t, []string{"instances/inst/databases/db"}, ag.Targets)
	require.Equal(t, "SELECT * FROM t", ag.Query)
	require.True(t, ag.Unmask)
	require.True(t, ag.Export)
	require.Equal(t, "investigating PR-1234", ag.Reason)
	require.Equal(t, "APP", ag.Schema)
	require.Equal(t, "orders", ag.Container)
}

// TestConvertToAccessGrantNilPayloadIsSafe guards the `if p := msg.Payload; p != nil`
// branch — a nil payload must not panic and payload-sourced fields stay zero.
func TestConvertToAccessGrantNilPayloadIsSafe(t *testing.T) {
	t.Parallel()
	msg := &store.AccessGrantMessage{
		ProjectID: "proj",
		ID:        "ag1",
		Creator:   "dev@example.com",
		Status:    storepb.AccessGrant_ACTIVE,
		Payload:   nil,
	}

	ag := convertToAccessGrant(msg)

	require.Empty(t, ag.Targets)
	require.Empty(t, ag.Query)
	require.False(t, ag.Unmask)
	require.False(t, ag.Export)
	require.Empty(t, ag.Reason)
	require.Empty(t, ag.Schema)
	require.Empty(t, ag.Container)
}
