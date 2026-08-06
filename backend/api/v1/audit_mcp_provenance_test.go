package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

const auditTestWorkspace = "ws-audit"

// specRequest overrides a request's Spec so the interceptors see a full
// procedure name — connect.NewRequest alone leaves it empty.
type specRequest struct {
	connect.AnyRequest
	procedure string
}

func (r *specRequest) Spec() connect.Spec {
	return connect.Spec{Procedure: r.procedure}
}

// newAuditLiveStore boots a migrated store with one workspace, so
// CreateAuditLog has a real parent row to write under.
func newAuditLiveStore(t *testing.T) *store.Store {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO workspace (resource_id) VALUES ('%s')`, auditTestWorkspace))
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	st, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	return st
}

func auditTestUser() *store.UserMessage {
	return &store.UserMessage{
		ID:    1,
		Email: "agent-driver@example.com",
		Type:  storepb.PrincipalType_END_USER,
	}
}

func newAuditTestContext(authCtx *common.AuthContext) context.Context {
	ctx := context.WithValue(context.Background(), common.AuthContextKey, authCtx)
	ctx = context.WithValue(ctx, common.UserContextKey, auditTestUser())
	ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, auditTestWorkspace)
	return ctx
}

// findRowsByCorrelation returns the workspace's audit rows whose MCP
// delegation carries the given correlation ID.
func findRowsByCorrelation(t *testing.T, st *store.Store, correlationID string) []*store.AuditLog {
	t.Helper()
	rows, err := st.SearchAuditLogs(context.Background(), &store.AuditLogFind{
		Workspace: auditTestWorkspace,
	})
	require.NoError(t, err)
	var matched []*store.AuditLog
	for _, row := range rows {
		if row.Payload.GetMcpDelegation().GetCorrelationId() == correlationID {
			matched = append(matched, row)
		}
	}
	return matched
}

// TestAuditRowCarriesMCPDelegationProvenance pins P1a PR 5b's provenance
// contract at the store level: an audited call that arrived with a delegated
// MCP credential (AuthContext.DelegatedGrant non-nil) writes its grant state
// verbatim onto the audit row, empty values preserved as empty; a public-chain
// call (nil grant) writes a row with no MCP fields at all.
func TestAuditRowCarriesMCPDelegationProvenance(t *testing.T) {
	st := newAuditLiveStore(t)
	in := NewAuditInterceptor(st, "test-secret", &config.Profile{})

	invoke := func(t *testing.T, grant *common.DelegatedGrant) {
		t.Helper()
		authCtx := &common.AuthContext{
			Audit:          true,
			AuthMethod:     common.AuthMethodIAM,
			Resources:      []*common.Resource{{Type: common.ResourceTypeWorkspace, ID: auditTestWorkspace}},
			DelegatedGrant: grant,
		}
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&v1pb.QueryResponse{}), nil
		}
		req := &specRequest{
			AnyRequest: connect.NewRequest(&v1pb.QueryRequest{Name: "instances/i/databases/d"}),
			procedure:  "/bytebase.v1.SQLService/Query",
		}
		_, err := in.WrapUnary(next)(newAuditTestContext(authCtx), req)
		require.NoError(t, err)
	}

	t.Run("a consented grant is stamped verbatim", func(t *testing.T) {
		invoke(t, &common.DelegatedGrant{
			Scope:         "mcp:read-only",
			Resource:      "https://bb.example.com/mcp",
			ClientID:      "client-A",
			CorrelationID: "corr-full",
		})
		rows := findRowsByCorrelation(t, st, "corr-full")
		require.Len(t, rows, 1, "an audited internal-chain call must produce exactly one provenance-carrying row")
		got := rows[0].Payload.GetMcpDelegation()
		require.Equal(t, "mcp:read-only", got.GetScope())
		require.Equal(t, "https://bb.example.com/mcp", got.GetResource())
		require.Equal(t, "client-A", got.GetClientId())
	})

	t.Run("a legacy empty grant still marks MCP origin, empty stays empty", func(t *testing.T) {
		invoke(t, &common.DelegatedGrant{CorrelationID: "corr-legacy"})
		rows := findRowsByCorrelation(t, st, "corr-legacy")
		require.Len(t, rows, 1)
		got := rows[0].Payload.GetMcpDelegation()
		require.NotNil(t, got, "presence of the delegation message is the MCP-origin marker, even for empty legacy grants")
		require.Empty(t, got.GetScope(), "an empty grant scope must be recorded empty, never resolved to a label")
		require.Empty(t, got.GetResource())
		require.Empty(t, got.GetClientId())
	})

	t.Run("a public-chain row carries no MCP fields", func(t *testing.T) {
		invoke(t, nil)
		rows, err := st.SearchAuditLogs(context.Background(), &store.AuditLogFind{
			Workspace: auditTestWorkspace,
		})
		require.NoError(t, err)
		var publicRows []*store.AuditLog
		for _, row := range rows {
			if row.Payload.GetMcpDelegation() == nil {
				publicRows = append(publicRows, row)
			}
		}
		require.Len(t, publicRows, 1, "the nil-grant call must produce exactly one row without MCP provenance")
	})
}

// TestInternalChainAuditRecordsACLDenial pins PR 5b's denial-audit mechanism:
// with the audit interceptor wrapped OUTSIDE the ACL interceptor (the internal
// MCP chain's order), an ACL denial produces an audit row carrying the
// provenance and the denied status; a method whose annotation opts out of
// auditing stays silent for permitted and denied calls alike.
func TestInternalChainAuditRecordsACLDenial(t *testing.T) {
	st := newAuditLiveStore(t)
	auditIn := NewAuditInterceptor(st, "test-secret", &config.Profile{})
	aclIn := NewACLInterceptor(st, "test-secret", nil /* iamManager: unreached on these paths */, &config.Profile{})

	invoke := func(t *testing.T, audited bool, correlationID, resource string) (handlerReached bool, rerr error) {
		t.Helper()
		authCtx := &common.AuthContext{
			Audit:      audited,
			AuthMethod: common.AuthMethodCustom,
			DelegatedGrant: &common.DelegatedGrant{
				Scope:         "mcp:read-only",
				Resource:      "https://bb.example.com/mcp",
				ClientID:      "client-A",
				CorrelationID: correlationID,
			},
		}
		handler := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			handlerReached = true
			return connect.NewResponse(&v1pb.IamPolicy{}), nil
		}
		// The internal chain's order: audit outside, ACL inside.
		chain := auditIn.WrapUnary(aclIn.WrapUnary(handler))
		req := &specRequest{
			AnyRequest: connect.NewRequest(&v1pb.SetIamPolicyRequest{Resource: resource}),
			procedure:  "/bytebase.v1.WorkspaceService/SetIamPolicy",
		}
		_, rerr = chain(newAuditTestContext(authCtx), req)
		return handlerReached, rerr
	}

	t.Run("an ACL denial produces a provenance-carrying denied row", func(t *testing.T) {
		handlerReached, err := invoke(t, true, "corr-denied", "workspaces/other-ws")
		require.Error(t, err)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		require.False(t, handlerReached, "the denial must come from the ACL interceptor, not the handler")

		rows := findRowsByCorrelation(t, st, "corr-denied")
		require.Len(t, rows, 1, "an ACL-denied internal-chain call must still produce an audit row")
		row := rows[0].Payload
		require.Equal(t, "workspaces/"+auditTestWorkspace, row.Parent,
			"a denied cross-workspace attempt must be audited under the CALLER's workspace, never the foreign one it named")
		require.Equal(t, "/bytebase.v1.WorkspaceService/SetIamPolicy", row.Method)
		require.Equal(t, common.FormatUserEmail(auditTestUser().Email), row.User)
		require.NotNil(t, row.Status, "the row must reflect the denial")
		require.Equal(t, int32(connect.CodePermissionDenied), row.Status.Code)
		require.Equal(t, "mcp:read-only", row.GetMcpDelegation().GetScope())
	})

	t.Run("a method opted out of auditing stays silent for denials too", func(t *testing.T) {
		_, err := invoke(t, false, "corr-optout", "workspaces/other-ws")
		require.Error(t, err)
		require.Empty(t, findRowsByCorrelation(t, st, "corr-optout"),
			"audit opt-out must behave consistently for permitted and denied calls")
	})

	t.Run("a permitted call is audited exactly once", func(t *testing.T) {
		handlerReached, err := invoke(t, true, "corr-permitted", "workspaces/"+auditTestWorkspace)
		require.NoError(t, err)
		require.True(t, handlerReached)

		rows := findRowsByCorrelation(t, st, "corr-permitted")
		require.Len(t, rows, 1)
		require.Nil(t, rows[0].Payload.Status, "a permitted call keeps its success status")
	})
}
