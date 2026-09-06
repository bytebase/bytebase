package v1

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

func TestFailedLoginWithoutWorkspaceIsSkipped(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.AuthContextKey, &common.AuthContext{Audit: true})
	in := NewAuditInterceptor(nil, "test-secret", &config.Profile{})

	rows, err := in.buildAuditRows(ctx, &auditEntry{
		request: &v1pb.LoginRequest{
			Email:    " Unknown@Example.com ",
			Password: "wrong-password",
		},
		method: v1connect.AuthServiceLoginProcedure,
		rerr: connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid email or password"),
		),
	})
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestFailedLoginWithHandlerWorkspaceCreatesSingleAuditRow(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.AuthContextKey, &common.AuthContext{Audit: true})
	in := NewAuditInterceptor(nil, "test-secret", &config.Profile{})

	rows, err := in.buildAuditRows(ctx, &auditEntry{
		request: &v1pb.LoginRequest{
			Email:    " Member@Example.com ",
			Password: "wrong-password",
		},
		method:                  v1connect.AuthServiceLoginProcedure,
		handlerAuditWorkspaceID: auditTestWorkspace,
		rerr: connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid email or password"),
		),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, auditTestWorkspace, rows[0].workspaceID)
	require.Equal(t, common.FormatWorkspace(auditTestWorkspace), rows[0].payload.GetParent())
	require.Equal(t, "member@example.com", rows[0].payload.GetResource())
}

// TestStreamingAuditPersistedBeforeSend pins the streaming audit contract: a
// client must not observe a successful streaming response before the
// corresponding audit entry is durably persisted. Regression test for the
// TestAdminExecuteAuditLog flake where the audit insert raced the
// client-visible response.
func TestStreamingAuditPersistedBeforeSend(t *testing.T) {
	t.Run("audit persisted before response delivered", func(t *testing.T) {
		a := require.New(t)
		recorder := &auditOrderRecorder{}
		interceptor := &AuditInterceptor{
			createAuditLogFunc: func(_ context.Context, _ *auditEntry) error {
				recorder.record("audit")
				return nil
			},
		}
		handler := interceptor.WrapStreamingHandler(func(_ context.Context, conn connect.StreamingHandlerConn) error {
			if err := conn.Receive(&v1pb.AdminExecuteRequest{}); err != nil {
				return err
			}
			return conn.Send(&v1pb.AdminExecuteResponse{})
		})

		ctx := context.WithValue(context.Background(), common.AuthContextKey, &common.AuthContext{Audit: true})
		a.NoError(handler(ctx, &auditRecorderConn{recorder: recorder}))

		a.Equal([]string{"audit", "send"}, recorder.events,
			"audit entry must be durably persisted before the response becomes observable to the client")
	})

	t.Run("audit failure blocks response delivery", func(t *testing.T) {
		a := require.New(t)
		recorder := &auditOrderRecorder{}
		persistErr := errors.New("persist failed")
		interceptor := &AuditInterceptor{
			createAuditLogFunc: func(_ context.Context, _ *auditEntry) error {
				recorder.record("audit")
				return persistErr
			},
		}
		handler := interceptor.WrapStreamingHandler(func(_ context.Context, conn connect.StreamingHandlerConn) error {
			if err := conn.Receive(&v1pb.AdminExecuteRequest{}); err != nil {
				return err
			}
			return conn.Send(&v1pb.AdminExecuteResponse{})
		})

		ctx := context.WithValue(context.Background(), common.AuthContextKey, &common.AuthContext{Audit: true})
		a.ErrorIs(handler(ctx, &auditRecorderConn{recorder: recorder}), persistErr)
		a.Equal([]string{"audit"}, recorder.events,
			"response must not be delivered when the audit entry cannot be persisted")
	})
}

// auditOrderRecorder records the order of audit persistence ("audit") and
// underlying stream delivery ("send") events.
type auditOrderRecorder struct {
	events []string
}

func (r *auditOrderRecorder) record(event string) {
	r.events = append(r.events, event)
}

// auditRecorderConn is a connect.StreamingHandlerConn fake whose Send stands
// in for the moment the response becomes observable to the client.
type auditRecorderConn struct {
	connect.StreamingHandlerConn
	recorder *auditOrderRecorder
}

func (*auditRecorderConn) Spec() connect.Spec {
	return connect.Spec{Procedure: v1connect.SQLServiceAdminExecuteProcedure}
}

func (*auditRecorderConn) Peer() connect.Peer { return connect.Peer{} }

func (*auditRecorderConn) RequestHeader() http.Header { return http.Header{} }

func (*auditRecorderConn) Receive(any) error { return nil }

func (c *auditRecorderConn) Send(any) error {
	c.recorder.record("send")
	return nil
}

func TestLifecycleAuditResource(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		request any
		method  string
		want    string
	}{
		{name: "query", request: &v1pb.QueryRequest{Name: "instances/instance-a/databases/database-a"}, method: "/bytebase.v1.SQLService/Query", want: "instances/instance-a/databases/database-a"},
		{name: "admin execute", request: &v1pb.AdminExecuteRequest{Name: "instances/instance-a/databases/database-a"}, method: "/bytebase.v1.SQLService/AdminExecute", want: "instances/instance-a/databases/database-a"},
		{name: "export", request: &v1pb.ExportRequest{Name: "instances/instance-a/databases/database-a"}, method: "/bytebase.v1.SQLService/Export", want: "instances/instance-a/databases/database-a"},
		{name: "create sheet", request: &v1pb.CreateSheetRequest{Parent: "projects/project-a"}, method: "/bytebase.v1.SheetService/CreateSheet", want: "projects/project-a"},
		{name: "batch create sheets", request: &v1pb.BatchCreateSheetsRequest{Parent: "projects/project-a"}, method: "/bytebase.v1.SheetService/BatchCreateSheets", want: "projects/project-a"},
		{name: "update database", request: &v1pb.UpdateDatabaseRequest{Database: &v1pb.Database{Name: "instances/instance-a/databases/database-a"}}, method: "/bytebase.v1.DatabaseService/UpdateDatabase", want: "instances/instance-a/databases/database-a"},
		{name: "batch update databases", request: &v1pb.BatchUpdateDatabasesRequest{Parent: "instances/instance-a"}, method: "/bytebase.v1.DatabaseService/BatchUpdateDatabases", want: "instances/instance-a"},
		{name: "update database catalog", request: &v1pb.UpdateDatabaseCatalogRequest{Catalog: &v1pb.DatabaseCatalog{Name: "instances/instance-a/databases/database-a/catalog"}}, method: "/bytebase.v1.DatabaseService/UpdateDatabaseCatalog", want: "instances/instance-a/databases/database-a/catalog"},
		{name: "set IAM policy", request: &v1pb.SetIamPolicyRequest{Resource: "projects/project-a"}, method: "/bytebase.v1.ProjectService/SetIamPolicy", want: "projects/project-a"},
		{name: "create user", request: &v1pb.CreateUserRequest{User: &v1pb.User{Name: "users/user@example.com"}}, method: "/bytebase.v1.UserService/CreateUser", want: "users/user@example.com"},
		{name: "update user", request: &v1pb.UpdateUserRequest{User: &v1pb.User{Name: "users/user@example.com"}}, method: "/bytebase.v1.UserService/UpdateUser", want: "users/user@example.com"},
		{name: "login", request: &v1pb.LoginRequest{Email: "user@example.com"}, method: "/bytebase.v1.AuthService/Login", want: "user@example.com"},
		{name: "signup", request: &v1pb.SignupRequest{Email: "user@example.com"}, method: "/bytebase.v1.AuthService/Signup", want: "user@example.com"},
		{name: "exchange token", request: &v1pb.ExchangeTokenRequest{Email: "user@example.com"}, method: "/bytebase.v1.AuthService/ExchangeToken", want: "user@example.com"},
		{name: "request password reset", request: &v1pb.RequestPasswordResetRequest{Email: " User@Example.com "}, method: "/bytebase.v1.AuthService/RequestPasswordReset", want: "user@example.com"},
		{name: "reset password", request: &v1pb.ResetPasswordRequest{Email: " User@Example.com "}, method: "/bytebase.v1.AuthService/ResetPassword", want: "user@example.com"},
		{name: "send email login code", request: &v1pb.SendEmailLoginCodeRequest{Email: " User@Example.com "}, method: "/bytebase.v1.AuthService/SendEmailLoginCode", want: "user@example.com"},
		{name: "create project", request: &v1pb.CreateProjectRequest{ProjectId: "project-a", Project: &v1pb.Project{}}, method: "/bytebase.v1.ProjectService/CreateProject", want: "projects/project-a"},
		{name: "create project ignores nested name", request: &v1pb.CreateProjectRequest{ProjectId: "project-a", Project: &v1pb.Project{Name: "projects/wrong-project"}}, method: "/bytebase.v1.ProjectService/CreateProject", want: "projects/project-a"},
		{name: "update project", request: &v1pb.UpdateProjectRequest{Project: &v1pb.Project{Name: "projects/project-a"}}, method: "/bytebase.v1.ProjectService/UpdateProject", want: "projects/project-a"},
		{name: "delete project", request: &v1pb.DeleteProjectRequest{Name: "projects/project-a"}, method: "/bytebase.v1.ProjectService/DeleteProject", want: "projects/project-a"},
		{name: "undelete project", request: &v1pb.UndeleteProjectRequest{Name: "projects/project-a"}, method: "/bytebase.v1.ProjectService/UndeleteProject", want: "projects/project-a"},
		{name: "run plan checks", request: &v1pb.RunPlanChecksRequest{Name: "projects/project-a/plans/101"}, method: "/bytebase.v1.PlanService/RunPlanChecks", want: "projects/project-a/plans/101"},
		{name: "cancel plan checks", request: &v1pb.CancelPlanCheckRunRequest{Name: "projects/project-a/plans/101/planCheckRun"}, method: "/bytebase.v1.PlanService/CancelPlanCheckRun", want: "projects/project-a/plans/101/planCheckRun"},
		{name: "delete release", request: &v1pb.DeleteReleaseRequest{Name: "projects/project-a/releases/release-a"}, method: "/bytebase.v1.ReleaseService/DeleteRelease", want: "projects/project-a/releases/release-a"},
		{name: "undelete release", request: &v1pb.UndeleteReleaseRequest{Name: "projects/project-a/releases/release-a"}, method: "/bytebase.v1.ReleaseService/UndeleteRelease", want: "projects/project-a/releases/release-a"},
		{name: "create release", request: &v1pb.CreateReleaseRequest{Parent: "projects/project-a"}, method: "/bytebase.v1.ReleaseService/CreateRelease", want: "projects/project-a"},
		{name: "update release", request: &v1pb.UpdateReleaseRequest{Release: &v1pb.Release{Name: "projects/project-a/releases/release-a"}}, method: "/bytebase.v1.ReleaseService/UpdateRelease", want: "projects/project-a/releases/release-a"},
		{name: "create saved query", request: &v1pb.CreateSavedQueryRequest{Parent: "projects/project-a"}, method: "/bytebase.v1.SavedQueryService/CreateSavedQuery", want: "projects/project-a"},
		{name: "delete saved query", request: &v1pb.DeleteSavedQueryRequest{Name: "projects/project-a/savedQueries/101"}, method: "/bytebase.v1.SavedQueryService/DeleteSavedQuery", want: "projects/project-a/savedQueries/101"},
		{name: "batch create revisions", request: &v1pb.BatchCreateRevisionsRequest{Parent: "instances/instance-a/databases/database-a"}, method: "/bytebase.v1.RevisionService/BatchCreateRevisions", want: "instances/instance-a/databases/database-a"},
		{name: "delete revision", request: &v1pb.DeleteRevisionRequest{Name: "instances/instance-a/databases/database-a/revisions/101"}, method: "/bytebase.v1.RevisionService/DeleteRevision", want: "instances/instance-a/databases/database-a/revisions/101"},
		{name: "create workspace instance", request: &v1pb.CreateInstanceRequest{InstanceId: "instance-a", Instance: &v1pb.Instance{Name: "instances/wrong-instance"}}, method: "/bytebase.v1.InstanceService/CreateInstance", want: "instances/instance-a"},
		{name: "create project instance", request: &v1pb.CreateInstanceRequest{Parent: new("projects/project-a"), InstanceId: "instance-a", Instance: &v1pb.Instance{}}, method: "/bytebase.v1.InstanceService/CreateInstance", want: "projects/project-a/instances/instance-a"},
		{name: "update instance", request: &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{Name: "instances/instance-a"}}, method: "/bytebase.v1.InstanceService/UpdateInstance", want: "instances/instance-a"},
		{name: "delete instance", request: &v1pb.DeleteInstanceRequest{Name: "instances/instance-a"}, method: "/bytebase.v1.InstanceService/DeleteInstance", want: "instances/instance-a"},
		{name: "undelete instance", request: &v1pb.UndeleteInstanceRequest{Name: "instances/instance-a"}, method: "/bytebase.v1.InstanceService/UndeleteInstance", want: "instances/instance-a"},
		{name: "add data source", request: &v1pb.AddDataSourceRequest{Name: "instances/instance-a"}, method: "/bytebase.v1.InstanceService/AddDataSource", want: "instances/instance-a"},
		{name: "remove data source", request: &v1pb.RemoveDataSourceRequest{Name: "instances/instance-a"}, method: "/bytebase.v1.InstanceService/RemoveDataSource", want: "instances/instance-a"},
		{name: "update data source", request: &v1pb.UpdateDataSourceRequest{Name: "instances/instance-a"}, method: "/bytebase.v1.InstanceService/UpdateDataSource", want: "instances/instance-a"},
		{name: "update setting", request: &v1pb.UpdateSettingRequest{Setting: &v1pb.Setting{Name: "settings/AI"}}, method: "/bytebase.v1.SettingService/UpdateSetting", want: "settings/AI"},
		{name: "create review config", request: &v1pb.CreateReviewConfigRequest{ReviewConfig: &v1pb.ReviewConfig{Name: "reviewConfigs/config-a"}}, method: "/bytebase.v1.ReviewConfigService/CreateReviewConfig", want: "reviewConfigs/config-a"},
		{name: "update review config", request: &v1pb.UpdateReviewConfigRequest{ReviewConfig: &v1pb.ReviewConfig{Name: "reviewConfigs/config-a"}}, method: "/bytebase.v1.ReviewConfigService/UpdateReviewConfig", want: "reviewConfigs/config-a"},
		{name: "delete review config", request: &v1pb.DeleteReviewConfigRequest{Name: "reviewConfigs/config-a"}, method: "/bytebase.v1.ReviewConfigService/DeleteReviewConfig", want: "reviewConfigs/config-a"},
		{name: "add webhook", request: &v1pb.AddWebhookRequest{Project: "projects/project-a"}, method: v1connect.ProjectServiceAddWebhookProcedure, want: "projects/project-a"},
		{name: "update webhook", request: &v1pb.UpdateWebhookRequest{Webhook: &v1pb.Webhook{Name: "projects/project-a/webhooks/webhook-a"}}, method: v1connect.ProjectServiceUpdateWebhookProcedure, want: "projects/project-a/webhooks/webhook-a"},
		{name: "remove webhook", request: &v1pb.RemoveWebhookRequest{Webhook: &v1pb.Webhook{Name: "projects/project-a/webhooks/webhook-a"}}, method: v1connect.ProjectServiceRemoveWebhookProcedure, want: "projects/project-a/webhooks/webhook-a"},
		{name: "export audit logs", request: &v1pb.ExportAuditLogsRequest{Parent: "projects/project-a"}, method: v1connect.AuditLogServiceExportAuditLogsProcedure, want: "projects/project-a"},
		{name: "prepare sample project instance", request: &v1pb.PrepareSampleProjectInstanceRequest{Parent: "projects/project-a"}, method: "/bytebase.v1.InstanceService/PrepareSampleProjectInstance", want: "projects/project-a"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.want, getRequestResource(test.request, test.method))
		})
	}
}

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

// newRecordingAuditInterceptor builds the interceptor with no store and
// captures the rows each audited call would have written, so a test asserts on
// what the interceptor decided rather than on a database. That the rows reach
// the audit page on a live server is backend/tests' TestMCPAuditProvenance.
func newRecordingAuditInterceptor() (*AuditInterceptor, *[]auditRow) {
	in := NewAuditInterceptor(nil, "test-secret", &config.Profile{})
	rows := &[]auditRow{}
	in.createAuditLogFunc = func(ctx context.Context, e *auditEntry) error {
		built, err := in.buildAuditRows(ctx, e)
		if err != nil {
			return err
		}
		*rows = append(*rows, built...)
		return nil
	}
	return in, rows
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

// rowsByCorrelation returns the captured rows whose MCP delegation carries the
// given correlation ID.
func rowsByCorrelation(rows []auditRow, correlationID string) []*storepb.AuditLog {
	var matched []*storepb.AuditLog
	for _, row := range rows {
		if row.payload.GetMcpDelegation().GetCorrelationId() == correlationID {
			matched = append(matched, row.payload)
		}
	}
	return matched
}

// TestAuditRowCarriesMCPDelegationProvenance pins P1a PR 5b's provenance
// contract: an audited call that arrived with a delegated MCP credential
// (AuthContext.DelegatedGrant non-nil) writes its grant state verbatim onto the
// audit row, empty values preserved as empty; a public-chain call (nil grant)
// writes a row with no MCP fields at all.
func TestAuditRowCarriesMCPDelegationProvenance(t *testing.T) {
	in, rows := newRecordingAuditInterceptor()

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
		matched := rowsByCorrelation(*rows, "corr-full")
		require.Len(t, matched, 1, "an audited internal-chain call must produce exactly one provenance-carrying row")
		got := matched[0].GetMcpDelegation()
		require.Equal(t, "mcp:read-only", got.GetScope())
		require.Equal(t, "https://bb.example.com/mcp", got.GetResource())
		require.Equal(t, "client-A", got.GetClientId())
	})

	t.Run("a legacy empty grant still marks MCP origin, empty stays empty", func(t *testing.T) {
		invoke(t, &common.DelegatedGrant{CorrelationID: "corr-legacy"})
		matched := rowsByCorrelation(*rows, "corr-legacy")
		require.Len(t, matched, 1)
		got := matched[0].GetMcpDelegation()
		require.NotNil(t, got, "presence of the delegation message is the MCP-origin marker, even for empty legacy grants")
		require.Empty(t, got.GetScope(), "an empty grant scope must be recorded empty, never resolved to a label")
		require.Empty(t, got.GetResource())
		require.Empty(t, got.GetClientId())
	})

	t.Run("a public-chain row carries no MCP fields", func(t *testing.T) {
		invoke(t, nil)
		var publicRows int
		for _, row := range *rows {
			if row.payload.GetMcpDelegation() == nil {
				publicRows++
			}
		}
		require.Equal(t, 1, publicRows, "the nil-grant call must produce exactly one row without MCP provenance")
	})
}

// TestAuditParentsDeduplicated pins that an audited call writes ONE row per
// distinct parent. Batch requests repeat the same project resource once per
// item, and since PR 5b routes ACL-denied internal-chain calls through the
// audit interceptor, an unprivileged caller reaches this fan-out — without
// dedup, a single denied batch call naming N items would write N identical
// rows.
func TestAuditParentsDeduplicated(t *testing.T) {
	t.Parallel()
	in, rows := newRecordingAuditInterceptor()

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

	var parents []string
	for _, row := range rowsByCorrelation(*rows, "corr-dedup") {
		parents = append(parents, row.Parent)
	}
	require.ElementsMatch(t,
		[]string{common.FormatProject("proj-batch"), "workspaces/" + auditTestWorkspace},
		parents,
		"one audit row per DISTINCT parent — repeated batch resources must not multiply rows")
}

// TestInternalChainAuditRecordsACLDenial pins PR 5b's denial-audit mechanism:
// with the audit interceptor wrapped OUTSIDE the ACL interceptor (the internal
// MCP chain's order), an ACL denial produces an audit row carrying the
// provenance and the denied status; a method whose annotation opts out of
// auditing stays silent for permitted and denied calls alike. The workspace
// mismatch is decided on the request alone, so no store is needed.
func TestInternalChainAuditRecordsACLDenial(t *testing.T) {
	auditIn, rows := newRecordingAuditInterceptor()
	aclIn := NewACLInterceptor(nil, "test-secret", nil /* iamManager: unreached on these paths */, &config.Profile{})

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

		matched := rowsByCorrelation(*rows, "corr-denied")
		require.Len(t, matched, 1, "an ACL-denied internal-chain call must still produce an audit row")
		row := matched[0]
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
		require.Empty(t, rowsByCorrelation(*rows, "corr-optout"),
			"audit opt-out must behave consistently for permitted and denied calls")
	})

	t.Run("a permitted call is audited exactly once", func(t *testing.T) {
		handlerReached, err := invoke(t, true, "corr-permitted", "workspaces/"+auditTestWorkspace)
		require.NoError(t, err)
		require.True(t, handlerReached)

		matched := rowsByCorrelation(*rows, "corr-permitted")
		require.Len(t, matched, 1)
		require.Nil(t, matched[0].Status, "a permitted call keeps its success status")
	})
}

// TestValidateOnlyAuditSkipAppliesOnlyToSuccess pins the boundary of the
// validate-only skip in createAuditLog.
//
// The skip exists so a dry run — which changes nothing — does not spam the
// audit log. It said nothing about the outcome, so it also swallowed every
// FAILED attempt whose request carries validate_only. That handed an agent an
// unlogged attempt at any forbidden method taking one of those six requests:
// two identical denials of the same method, in the same session, differing only
// in that flag, produced one row between them.
//
// The three cases below are the whole rule: outcome decides, not the flag
// alone.
func TestValidateOnlyAuditSkipAppliesOnlyToSuccess(t *testing.T) {
	in, captured := newRecordingAuditInterceptor()

	// A retarget with the stored password left to ride along — the shape the
	// MCP class refuses. The secrets are here so the redaction assertion below
	// has something to catch.
	const storedPassword = "stored-db-secret"
	const storedKeytab = "stored-keytab-bytes"
	retargetRequest := func(validateOnly bool) *v1pb.UpdateDataSourceRequest {
		return &v1pb.UpdateDataSourceRequest{
			Name: "instances/probe",
			DataSource: &v1pb.DataSource{
				Id:       "admin-ds",
				Host:     "attacker.example.com",
				Password: storedPassword,
				SaslConfig: &v1pb.SASLConfig{
					Mechanism: &v1pb.SASLConfig_KrbConfig{
						KrbConfig: &v1pb.KerberosConfig{Keytab: []byte(storedKeytab)},
					},
				},
			},
			ValidateOnly: validateOnly,
		}
	}

	invoke := func(t *testing.T, correlationID string, request *v1pb.UpdateDataSourceRequest, rerr error) {
		t.Helper()
		authCtx := &common.AuthContext{
			Audit:          true,
			AuthMethod:     common.AuthMethodIAM,
			Resources:      []*common.Resource{{Type: common.ResourceTypeWorkspace, ID: auditTestWorkspace}},
			DelegatedGrant: &common.DelegatedGrant{CorrelationID: correlationID},
		}
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			if rerr != nil {
				return nil, rerr
			}
			return connect.NewResponse(&v1pb.Instance{Name: "instances/probe"}), nil
		}
		req := &specRequest{
			AnyRequest: connect.NewRequest(request),
			procedure:  "/bytebase.v1.InstanceService/UpdateDataSource",
		}
		_, err := in.WrapUnary(next)(newAuditTestContext(authCtx), req)
		if rerr == nil {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}

	t.Run("a denied validate-only call is recorded", func(t *testing.T) {
		denial := connect.NewError(connect.CodePermissionDenied,
			errors.New("InstanceService/UpdateDataSource is not available to MCP sessions"))
		invoke(t, "corr-validate-only-denied", retargetRequest(true), denial)

		rows := rowsByCorrelation(*captured, "corr-validate-only-denied")
		require.Len(t, rows, 1,
			"a refused attempt must be auditable whether or not validate_only was set — "+
				"otherwise the flag is a switch that turns off the record")
		require.Equal(t, int32(connect.CodePermissionDenied), rows[0].GetStatus().GetCode(),
			"the row must carry the denial, not a blank status")

		// The newly-logged row goes through the same marshalAuditPayload
		// redaction as every other UpdateDataSource row — that is why
		// narrowing the skip exposes no field the plain path did not already
		// write. It pins that the path runs on a row the skip used to swallow;
		// which fields the redaction covers is the annotation's job, pinned by
		// TestAuditRedactionCoversEveryAnnotatedField.
		request := rows[0].GetRequest()
		require.NotEmpty(t, request)
		require.NotContains(t, request, storedPassword,
			"the request payload must be redacted before it reaches the audit row")
		// The row is protojson, which renders a bytes field base64, so the
		// keytab has to be looked for in that form — searching for the ASCII
		// literal would pass even with keytab redaction deleted outright.
		require.NotContains(t, request, base64.StdEncoding.EncodeToString([]byte(storedKeytab)),
			"the keytab reaches the row base64-encoded, and must be masked before it does")
		require.Contains(t, request, "attacker.example.com",
			"the destination is what an operator reads the row for — it must survive redaction")
	})

	t.Run("a succeeding validate-only call stays silent", func(t *testing.T) {
		invoke(t, "corr-validate-only-ok", retargetRequest(true), nil)

		require.Empty(t, rowsByCorrelation(*captured, "corr-validate-only-ok"),
			"a dry run that succeeded changed nothing — the original reason for the skip")
	})

	t.Run("a failed validate-only call is recorded, denial or not", func(t *testing.T) {
		// The rule is any failure, not any refusal, and this is the case that
		// pins it: a validate-only connection test that could not reach the
		// host. It is also the bulk of what the change adds, since the
		// instance form dials before every save.
		failedDial := connect.NewError(connect.CodeInvalidArgument,
			errors.New("failed to connect to attacker.example.com: connection refused"))
		invoke(t, "corr-validate-only-dial-failed", retargetRequest(true), failedDial)

		rows := rowsByCorrelation(*captured, "corr-validate-only-dial-failed")
		require.Len(t, rows, 1,
			"keying on a denial code would drop every other rejected attempt — the hole this change closes")
		require.Equal(t, int32(connect.CodeInvalidArgument), rows[0].GetStatus().GetCode())
	})

	t.Run("a denied ordinary call is recorded", func(t *testing.T) {
		denial := connect.NewError(connect.CodePermissionDenied, errors.New("denied"))
		invoke(t, "corr-plain-denied", retargetRequest(false), denial)

		rows := rowsByCorrelation(*captured, "corr-plain-denied")
		require.Len(t, rows, 1, "control: the flag is the only difference between this and the first case")
		require.True(t, strings.Contains(rows[0].GetMethod(), "UpdateDataSource"))
	})
}
