package tests

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestAuditLogFormat is both a regression test for the 3.17.0 bug where
// AuthService/Login (and Signup/ExchangeToken) silently dropped audit entries,
// AND a contract test for the shape of audit log entries returned by
// AuditLogService/SearchAuditLogs. Downstream consumers (SIEMs, compliance
// tooling, `docker logs | grep log_type:audit`) depend on this shape being
// stable across releases — changes here are user-visible breaking changes.
func TestAuditLogFormat(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// --- Part 1: Login (workspace-scoped, allow_without_credential) ---
	//
	// Clear the token so the Login call runs without credentials — the exact
	// path that regressed in 3.17.0 (Resources is empty, so the audit loop
	// never fires unless the handler hands us the workspace).
	adminToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = ""

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024bytebase",
	}))
	a.NoError(err)
	workspace := loginResp.Msg.GetUser().GetWorkspace()
	a.NotEmpty(workspace, "login response should carry the user's workspace")

	// Restore the token so the SearchAuditLogs call below authenticates.
	ctl.authInterceptor.token = adminToken

	loginAuditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  workspace,
		Filter:  `method == "/bytebase.v1.AuthService/Login"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	a.NotEmpty(loginAuditLogs.Msg.AuditLogs, "Login must produce an audit entry under the caller's workspace (regression guard for 3.17.0)")

	entry := loginAuditLogs.Msg.AuditLogs[0]
	// Name: "workspaces/{id}/auditLogs/{uid}" — the resource name format is
	// part of the API contract and also the parent users filter on.
	a.Regexp(regexp.MustCompile(`^workspaces/[^/]+/auditLogs/[^/]+$`), entry.Name,
		"audit log name must match workspaces/{id}/auditLogs/{uid}")
	a.True(strings.HasPrefix(entry.Name, workspace+"/auditLogs/"),
		"audit log must be parented under the login workspace")
	a.NotNil(entry.CreateTime, "CreateTime must be set")
	a.Equal("users/demo@example.com", entry.User, "User must be users/{email}")
	a.Equal("/bytebase.v1.AuthService/Login", entry.Method,
		"Method is part of the filter API contract and must be the full procedure name")
	a.Equal(v1pb.AuditLog_INFO, entry.Severity, "successful Login is INFO severity")
	a.Equal("demo@example.com", entry.Resource, "Login's Resource is the login email")
	a.Nil(entry.Status, "successful Login has nil Status (code 0)")
	a.NotNil(entry.Latency, "Latency must be recorded")

	// Request JSON must round-trip back to LoginRequest, have the caller's
	// email, and must NOT contain the plaintext password.
	gotReq := &v1pb.LoginRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(entry.Request), gotReq),
		"Request JSON must be valid protojson for LoginRequest")
	a.Equal("demo@example.com", gotReq.Email)
	a.Empty(gotReq.Password, "password must be redacted in the Request payload")
	a.NotContains(entry.Request, "1024bytebase", "plaintext password must never appear in audit Request")

	// Response JSON must round-trip back to LoginResponse with the user info
	// but NO token (tokens are intentionally dropped from the audit payload).
	gotResp := &v1pb.LoginResponse{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(entry.Response), gotResp),
		"Response JSON must be valid protojson for LoginResponse")
	a.Equal("users/demo@example.com", gotResp.GetUser().GetName())
	a.Equal("demo@example.com", gotResp.GetUser().GetEmail())
	a.Empty(gotResp.Token, "token must be redacted from the Response payload")
	a.NotContains(entry.Response, loginResp.Msg.Token,
		"actual access token must never appear in audit Response")

	// --- Part 2: Signup (workspace-scoped, allow_without_credential) ---
	//
	// The initial `signupAndLogin` in setup already produced a Signup audit
	// entry; just assert it landed with the expected parent and method.
	signupAuditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.AuthService/Signup"`,
	}))
	a.NoError(err)
	a.NotEmpty(signupAuditLogs.Msg.AuditLogs, "Signup must produce an audit entry under the caller's workspace")
	signupEntry := signupAuditLogs.Msg.AuditLogs[0]
	a.True(strings.HasPrefix(signupEntry.Name, workspace+"/auditLogs/"))
	a.Equal("/bytebase.v1.AuthService/Signup", signupEntry.Method)
	a.Equal(v1pb.AuditLog_INFO, signupEntry.Severity)
	a.NotContains(signupEntry.Request, "1024bytebase",
		"plaintext password must never appear in Signup audit Request")

	// --- Part 3: SetIamPolicy (project-scoped, authenticated) ---
	//
	// Project-scoped audit entries must land under projects/{id} (not under
	// the workspace). This is what compliance tooling filters on.
	projectResource := ctl.project.Name // "projects/test-project"

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: projectResource,
	}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectDeveloper",
		Members: []string{"user:demo@example.com"},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:     policy.Etag,
		Policy:   policy,
		Resource: projectResource,
	}))
	a.NoError(err)

	projectAuditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  projectResource,
		Filter:  `method == "/bytebase.v1.ProjectService/SetIamPolicy"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	a.NotEmpty(projectAuditLogs.Msg.AuditLogs,
		"SetIamPolicy must produce an audit entry under projects/{id}")

	projEntry := projectAuditLogs.Msg.AuditLogs[0]
	a.Regexp(regexp.MustCompile(`^projects/[^/]+/auditLogs/[^/]+$`), projEntry.Name,
		"project audit log name must match projects/{id}/auditLogs/{uid}")
	a.True(strings.HasPrefix(projEntry.Name, projectResource+"/auditLogs/"),
		"audit entry must be parented under the target project, not the workspace")
	a.Equal("/bytebase.v1.ProjectService/SetIamPolicy", projEntry.Method)
	a.Equal("users/demo@example.com", projEntry.User)
	a.Equal(v1pb.AuditLog_INFO, projEntry.Severity)
	a.Equal(projectResource, projEntry.Resource,
		"SetIamPolicy's Resource is the target project name")
	a.Nil(projEntry.Status)
	a.NotNil(projEntry.Latency)

	gotSetReq := &v1pb.SetIamPolicyRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(projEntry.Request), gotSetReq),
		"Request JSON must be valid protojson for SetIamPolicyRequest")
	a.Equal(projectResource, gotSetReq.Resource)
	a.NotNil(gotSetReq.Policy, "Policy must round-trip through the audit Request")

	gotIamPolicy := &v1pb.IamPolicy{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(projEntry.Response), gotIamPolicy),
		"Response JSON must be valid protojson for IamPolicy")
	// The updated binding must be visible in the recorded response.
	foundBinding := false
	for _, b := range gotIamPolicy.Bindings {
		if b.Role == "roles/projectDeveloper" {
			for _, m := range b.Members {
				if m == "user:demo@example.com" {
					foundBinding = true
				}
			}
		}
	}
	a.True(foundBinding, "Response JSON must reflect the new IAM binding")

	// A project audit entry must NOT appear when searching under the
	// workspace — verifies parent scoping is strict.
	workspaceSearch, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.ProjectService/SetIamPolicy"`,
	}))
	a.NoError(err)
	a.Empty(workspaceSearch.Msg.AuditLogs,
		"project-scoped SetIamPolicy audit must not leak into the workspace-scoped log stream")
}
