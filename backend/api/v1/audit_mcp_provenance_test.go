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

// TestAuditParentsDeduplicated pins that createAuditLog writes ONE row per
// distinct parent. Batch requests repeat the same project resource once per
// item, and since PR 5b routes ACL-denied internal-chain calls through the
// audit interceptor, an unprivileged caller reaches this fan-out — without
// dedup, a single denied batch call naming N items would write N identical
// rows.
func TestAuditParentsDeduplicated(t *testing.T) {
	st := newAuditLiveStore(t)
	in := NewAuditInterceptor(st, "test-secret", &config.Profile{})

	authCtx := &common.AuthContext{
		Audit:      true,
		AuthMethod: common.AuthMethodIAM,
		Resources: []*common.Resource{
			{Type: common.ResourceTypeProject, ID: "proj-batch"},
			{Type: common.ResourceTypeProject, ID: "proj-batch"},
			{Type: common.ResourceTypeProject, ID: "proj-batch"},
			{Type: common.ResourceTypeWorkspace, ID: auditTestWorkspace},
		},
		DelegatedGrant: &common.DelegatedGrant{CorrelationID: "corr-dedup"},
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

	rows := findRowsByCorrelation(t, st, "corr-dedup")
	var parents []string
	for _, row := range rows {
		parents = append(parents, row.Payload.Parent)
	}
	require.ElementsMatch(t,
		[]string{common.FormatProject("proj-batch"), "workspaces/" + auditTestWorkspace},
		parents,
		"one audit row per DISTINCT parent — repeated batch resources must not multiply rows")
}

// TestAuditRecordsACLDenial pins the rule at the interceptor pair both chains
// now share: with audit wrapped OUTSIDE ACL, an ACL denial produces a row
// carrying the provenance and the denied status, whatever the method's audit
// annotation says. A permitted call on an unannotated method still writes
// nothing — the annotation governs ordinary use, and only that.
//
// This is the mutation check for the mark's ACL half. Break either end of it —
// the common.SetPolicyDenied call inside acl.go's markPolicyDenied helper, or
// the setter registration in audit.go — and the unannotated-denial subtest goes
// red while the permitted ones stay green.
//
// The verdict it drives is the workspace mismatch at acl.go's isolation loop,
// so deleting THAT `return markPolicyDenied` also reddens it. The other two
// ACL sites — the IAM denial and the allow_missing create denial — are not
// reached here. The gate's writer is pinned by
// TestMCPGateDenialIsAuditedWithoutAnAuditAnnotation and the clamp's by
// TestMCPReadOnlyCeilingRefusesAWrite.
func TestAuditRecordsACLDenial(t *testing.T) {
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
		// Both chains' order: audit outside, ACL inside.
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
		require.Equal(t, storepb.AuditLog_WARNING, row.Severity)
	})

	t.Run("a method with no audit annotation records its denial", func(t *testing.T) {
		// The rule decides, not the annotation. ACL marks the verdict and the
		// audit interceptor records it, so the row exists even though needAudit
		// is false for this method.
		_, err := invoke(t, false, "corr-unannotated", "workspaces/other-ws")
		require.Error(t, err)

		rows := findRowsByCorrelation(t, st, "corr-unannotated")
		require.Len(t, rows, 1, "a policy denial is recorded whatever the annotation says")
		require.Equal(t, int32(connect.CodePermissionDenied), rows[0].Payload.Status.Code)
		require.Equal(t, storepb.AuditLog_WARNING, rows[0].Payload.Severity)
	})

	t.Run("a permitted call is audited exactly once", func(t *testing.T) {
		handlerReached, err := invoke(t, true, "corr-permitted", "workspaces/"+auditTestWorkspace)
		require.NoError(t, err)
		require.True(t, handlerReached)

		rows := findRowsByCorrelation(t, st, "corr-permitted")
		require.Len(t, rows, 1)
		require.Nil(t, rows[0].Payload.Status, "a permitted call keeps its success status")
		require.Equal(t, storepb.AuditLog_INFO, rows[0].Payload.Severity,
			"routine traffic is not a refusal")
	})

	t.Run("a permitted call on an unannotated method stays silent", func(t *testing.T) {
		// The other half of the rule: the annotation still governs ordinary
		// use. Only the denial above escapes it.
		handlerReached, err := invoke(t, false, "corr-permitted-unannotated", "workspaces/"+auditTestWorkspace)
		require.NoError(t, err)
		require.True(t, handlerReached)
		require.Empty(t, findRowsByCorrelation(t, st, "corr-permitted-unannotated"))
	})
}
