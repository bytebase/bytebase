package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestSearchAuditLogsMCPFilters drives the two MCP filters end to end against a
// real Postgres, because both of the things that can be wrong about them are
// invisible to a SQL-shape assertion.
//
// The first is the key spelling: the payload is protojson, so the MCP
// provenance is stored under "mcpDelegation", not the proto field name
// mcp_delegation, and a filter naming the wrong one returns zero rows and no
// error. The second is the escape: the JSONB key-exists operator is a question
// mark, which is also qb's bind placeholder, so a predicate that reaches
// Postgres with one question mark instead of two is a parameter count error
// rather than a query.
func TestSearchAuditLogsMCPFilters(t *testing.T) {
	ctx := context.Background()
	db, s, _ := newTestDB(t)

	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('ws-test')`)
	require.NoError(t, err)

	rows := []*storepb.AuditLog{
		{
			Parent:   "workspaces/ws-test",
			Method:   "/bytebase.v1.SQLService/Query",
			Resource: "instances/i/databases/d",
			McpDelegation: &storepb.MCPDelegation{
				ClientId:      "client-A",
				CorrelationId: "session-one",
			},
		},
		{
			Parent: "workspaces/ws-test",
			Method: "/bytebase.v1.DatabaseService/ListDatabases",
			McpDelegation: &storepb.MCPDelegation{
				ClientId:      "client-A",
				CorrelationId: "session-one",
			},
		},
		{
			Parent: "workspaces/ws-test",
			Method: "/bytebase.v1.SQLService/Query",
			McpDelegation: &storepb.MCPDelegation{
				ClientId:      "client-B",
				CorrelationId: "session-two",
			},
		},
		{
			// A legacy MCP session: the marker is present, every field of it
			// is empty. No field value can stand in for the presence test.
			Parent:        "workspaces/ws-test",
			Method:        "/bytebase.v1.SQLService/Query",
			McpDelegation: &storepb.MCPDelegation{},
		},
		{
			// The console. No marker at all.
			Parent: "workspaces/ws-test",
			Method: "/bytebase.v1.SQLService/Query",
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

	t.Run("mcp == true returns every MCP row and only those", func(t *testing.T) {
		got := search(t, "mcp == true")
		require.Len(t, got, 4, "the four rows carrying the marker, including the empty-field legacy one")
		for _, log := range got {
			require.NotNil(t, log.Payload.GetMcpDelegation(),
				"a row without the marker must not answer the MCP filter")
		}
	})

	t.Run("mcp == false returns the rest", func(t *testing.T) {
		got := search(t, "mcp == false")
		require.Len(t, got, 1)
		require.Nil(t, got[0].Payload.GetMcpDelegation())
	})

	t.Run("the correlation filter returns one session's rows", func(t *testing.T) {
		got := search(t, `mcp_correlation_id == "session-one"`)
		require.Len(t, got, 2)
		for _, log := range got {
			require.Equal(t, "session-one", log.Payload.GetMcpDelegation().GetCorrelationId())
		}
	})

	t.Run("the MCP filters compose with the existing ones", func(t *testing.T) {
		got := search(t, `mcp == true && method == "/bytebase.v1.SQLService/Query"`)
		require.Len(t, got, 3)
	})

	t.Run("an unknown key still errors", func(t *testing.T) {
		_, err := store.GetSearchAuditLogsFilter(`mcp_client_id == "client-A"`)
		require.ErrorContains(t, err, "unknown variable mcp_client_id")
	})

	t.Run("mcp is a boolean, not a string", func(t *testing.T) {
		// Every other boolean filter key in the codebase takes a real CEL
		// boolean and says so on a mismatch; this one must not be the outlier.
		_, err := store.GetSearchAuditLogsFilter(`mcp == "true"`)
		require.ErrorContains(t, err, "expect true or false")
	})
}
