package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// The payload is protojson, so a predicate naming the proto field instead of
// "federatedIdentity" would return no rows and no error. Only a real Postgres
// shows that.
func TestSearchAuditLogsFederatedSubjectFilter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, s, _ := testcontainer.NewMetadataDB(t)

	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('ws-test')`)
	require.NoError(t, err)

	const (
		actor  = "workloadIdentities/ci@workload.bytebase.com"
		issuer = "https://token.actions.githubusercontent.com"
	)
	rows := []*storepb.AuditLog{
		{
			Parent:            "workspaces/ws-test",
			Method:            "/bytebase.v1.ProjectService/CreateProject",
			User:              actor,
			FederatedIdentity: &storepb.FederatedIdentity{Issuer: issuer, Subject: "repo:my-org/app:ref:refs/heads/main"},
		},
		{
			Parent:            "workspaces/ws-test",
			Method:            "/bytebase.v1.ProjectService/CreateProject",
			User:              actor,
			FederatedIdentity: &storepb.FederatedIdentity{Issuer: issuer, Subject: "repo:my-org/other:ref:refs/heads/main"},
		},
		{
			Parent: "workspaces/ws-test",
			Method: "/bytebase.v1.ProjectService/CreateProject",
			User:   "users/demo@example.com",
		},
	}
	for _, row := range rows {
		require.NoError(t, s.CreateAuditLog(ctx, "ws-test", row))
	}

	search := func(t *testing.T, filter string) []*store.AuditLog {
		t.Helper()
		q, err := store.GetSearchAuditLogsFilter(filter)
		require.NoError(t, err)
		logs, err := s.SearchAuditLogs(ctx, &store.AuditLogFind{Workspace: "ws-test", FilterQ: q})
		require.NoError(t, err)
		return logs
	}

	got := search(t, `federated_subject == "repo:my-org/app:ref:refs/heads/main"`)
	require.Len(t, got, 1, "one identity, two repositories: the filter separates them")
	require.Equal(t, "repo:my-org/app:ref:refs/heads/main", got[0].Payload.GetFederatedIdentity().GetSubject())
	require.Equal(t, issuer, got[0].Payload.GetFederatedIdentity().GetIssuer())

	got = search(t, `actor == "`+actor+`"`)
	require.Len(t, got, 2)
}
