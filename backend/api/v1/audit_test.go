package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// TestLogAuditToStdoutFormat is a contract test for the structured JSON fields
// emitted to stdout when audit log stdout is enabled. Operators run commands
// like `docker logs <container> | grep '"log_type":"audit"'` — changes to
// these keys or their values are breaking for downstream log pipelines.
func TestLogAuditToStdoutFormat(t *testing.T) {
	a := require.New(t)

	// Redirect slog default to a JSON handler writing into a buffer so the
	// test can inspect the exact structured payload.
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	p := &storepb.AuditLog{
		Parent:   "workspaces/ws-abc",
		Method:   "/bytebase.v1.AuthService/Login",
		Resource: "alice@example.com",
		Severity: storepb.AuditLog_INFO,
		User:     "users/alice@example.com",
		Request:  `{"email":"alice@example.com","web":true}`,
		Response: `{"user":{"name":"users/alice@example.com","email":"alice@example.com"}}`,
		Status:   &spb.Status{Code: 0, Message: ""},
		Latency:  durationpb.New(123_000_000), // 123ms
		RequestMetadata: &storepb.RequestMetadata{
			CallerIp:                "10.0.1.50",
			CallerSuppliedUserAgent: "TestAgent/1.0",
		},
	}

	logAuditToStdout(context.Background(), p)

	// Every line the handler wrote is an independent JSON object.
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'})
	a.Len(lines, 1, "logAuditToStdout must emit exactly one JSON record")

	var got map[string]any
	a.NoError(json.Unmarshal(lines[0], &got),
		"stdout record must be valid JSON: %s", string(lines[0]))

	// Keys the customer explicitly greps for — these form our public contract.
	// If any of these assertions need to change, it's a breaking change for
	// everyone consuming audit stdout logs.
	a.Equal("audit", got["log_type"], `log_type=="audit" is how operators filter audit lines from application logs`)
	a.Equal("workspaces/ws-abc", got["parent"])
	a.Equal("/bytebase.v1.AuthService/Login", got["method"])
	a.Equal("alice@example.com", got["resource"])
	a.Equal("users/alice@example.com", got["user"])
	a.Equal("INFO", got["severity"])
	a.Equal("10.0.1.50", got["client_ip"])
	a.Equal("TestAgent/1.0", got["user_agent"])
	a.Equal(float64(123), got["latency_ms"], "latency must be exposed in milliseconds")
	a.Equal(`{"email":"alice@example.com","web":true}`, got["request"],
		"request payload must be emitted verbatim (already redacted upstream)")
	a.Equal(`{"user":{"name":"users/alice@example.com","email":"alice@example.com"}}`, got["response"])

	// Status code 0 (OK) should be reported explicitly so downstream pipelines
	// can distinguish successful from failed calls.
	a.Equal(float64(0), got["status_code"])

	// slog standard fields — keep them stable too.
	a.Equal("INFO", got["level"])
	a.Equal("/bytebase.v1.AuthService/Login", got["msg"],
		"slog msg is set to the method so operators can read the log without parsing")

	// Error status round-trip: a failed call must emit status_code and status_message.
	buf.Reset()
	p2 := &storepb.AuditLog{
		Parent:   "workspaces/ws-abc",
		Method:   "/bytebase.v1.AuthService/Login",
		Severity: storepb.AuditLog_INFO,
		Status:   &spb.Status{Code: 16, Message: "invalid email or password"}, // UNAUTHENTICATED
		Latency:  durationpb.New(5_000_000),
	}
	logAuditToStdout(context.Background(), p2)

	var failed map[string]any
	a.NoError(json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &failed))
	a.Equal(float64(16), failed["status_code"], "failed call must carry status_code")
	a.Equal("invalid email or password", failed["status_message"],
		"failed call must carry status_message")

	// MCP-delegated calls carry their provenance in stdout too, so operators
	// tailing container logs can trace an agent session without the API.
	a.NotContains(got, "mcp_correlation_id",
		"non-MCP rows must not emit MCP keys")
	buf.Reset()
	p3 := &storepb.AuditLog{
		Parent:   "workspaces/ws-abc",
		Method:   "/bytebase.v1.SQLService/Query",
		Severity: storepb.AuditLog_INFO,
		McpDelegation: &storepb.MCPDelegation{
			Scope:         "mcp:read-only",
			Resource:      "https://bb.example.com/mcp",
			ClientId:      "client-A",
			CorrelationId: "corr-42",
		},
	}
	logAuditToStdout(context.Background(), p3)

	var delegated map[string]any
	a.NoError(json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &delegated))
	a.Equal(true, delegated["mcp"], `mcp==true is how operators filter MCP-originated rows`)
	a.Equal("corr-42", delegated["mcp_correlation_id"],
		"the session correlation ID is the key operators pivot on")
	a.Equal("mcp:read-only", delegated["mcp_scope"])
	a.Equal("https://bb.example.com/mcp", delegated["mcp_resource"])
	a.Equal("client-A", delegated["mcp_client_id"])
}

// TestAuditRedactsCredentials covers the request/response redactors that
// strip secrets before the audit pipeline serializes payloads. Specifically
// guards against regressions like the Signup password leak and the
// ExchangeToken OIDC/access-token leak that Codex flagged on #20024 — both
// only surfaced once the corresponding audit path was re-enabled by the
// SetAuditWorkspaceID callback.
func TestAuditRedactsCredentials(t *testing.T) {
	a := require.New(t)

	t.Run("LoginRequest redacts password and MFA secrets", func(_ *testing.T) {
		otp := "123456"
		mfa := "mfa-temp-jwt"
		reqStr, err := getRequestString(&v1pb.LoginRequest{
			Email:        "alice@example.com",
			Password:     "hunter2",
			OtpCode:      &otp,
			MfaTempToken: &mfa,
		})
		a.NoError(err)
		a.Contains(reqStr, "alice@example.com", "non-sensitive email stays")
		a.NotContains(reqStr, "hunter2", "plaintext password must not appear")
		a.NotContains(reqStr, "123456", "OTP must not appear")
		a.NotContains(reqStr, "mfa-temp-jwt", "MFA temp token must not appear")
	})

	t.Run("LoginResponse drops token", func(_ *testing.T) {
		respStr, err := getResponseString(&v1pb.LoginResponse{
			Token: "secret-access-token",
			User:  &v1pb.User{Name: "users/alice@example.com"},
		})
		a.NoError(err)
		a.Contains(respStr, "users/alice@example.com", "user info is retained")
		a.NotContains(respStr, "secret-access-token", "access token must not appear")
	})

	t.Run("SignupRequest redacts password", func(_ *testing.T) {
		reqStr, err := getRequestString(&v1pb.SignupRequest{
			Email:    "bob@example.com",
			Password: "signup-password",
			Title:    "bob",
		})
		a.NoError(err)
		a.Contains(reqStr, "bob@example.com")
		a.NotContains(reqStr, "signup-password",
			"plaintext password must not appear in Signup audit")
	})

	t.Run("ExchangeTokenRequest redacts OIDC token", func(_ *testing.T) {
		reqStr, err := getRequestString(&v1pb.ExchangeTokenRequest{
			Token: "oidc.jwt.payload",
			Email: "ci-bot@workload.bytebase.com",
		})
		a.NoError(err)
		a.Contains(reqStr, "ci-bot@workload.bytebase.com",
			"workload email retained for audit correlation")
		a.NotContains(reqStr, "oidc.jwt.payload",
			"external OIDC token must not appear in audit — it can be replayed "+
				"against the original IdP or reveal workload claims")
	})

	t.Run("ExchangeTokenResponse drops issued access token", func(_ *testing.T) {
		respStr, err := getResponseString(&v1pb.ExchangeTokenResponse{
			AccessToken: "issued-bytebase-api-token",
		})
		a.NoError(err)
		a.NotContains(respStr, "issued-bytebase-api-token",
			"issued Bytebase access token must never be logged — anyone with "+
				"audit read access could use it as a valid API token")
	})
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
		{name: "create worksheet", request: &v1pb.CreateWorksheetRequest{Parent: "projects/project-a"}, method: "/bytebase.v1.WorksheetService/CreateWorksheet", want: "projects/project-a"},
		{name: "delete worksheet", request: &v1pb.DeleteWorksheetRequest{Name: "projects/project-a/worksheets/101"}, method: "/bytebase.v1.WorksheetService/DeleteWorksheet", want: "projects/project-a/worksheets/101"},
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
		{name: "test existing identity provider", request: &v1pb.TestIdentityProviderRequest{IdentityProvider: &v1pb.IdentityProvider{Name: "idps/idp-a"}}, method: "/bytebase.v1.IdentityProviderService/TestIdentityProvider", want: "idps/idp-a"},
		{name: "test uncreated identity provider", request: &v1pb.TestIdentityProviderRequest{IdentityProvider: &v1pb.IdentityProvider{}}, method: "/bytebase.v1.IdentityProviderService/TestIdentityProvider", want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, getRequestResource(test.request, test.method))
		})
	}
}
