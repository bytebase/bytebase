package tests

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// TestPublicChainAuditsPolicyDenials drives the audit annotation over the
// public chain a browser and the public API use. ALL records every call;
// DENIALS records only the refused ones; and after this change the ACL
// interceptor marks its verdicts, so a refusal on the public chain reaches the
// log exactly as one on the internal MCP chain already did.
//
// It has to be an e2e. The rule is carried by the ORDER of the interceptor
// list in backend/server/grpc_routes.go — audit outside ACL — and only a real
// server builds that list. A package test that composes the two interceptors
// by hand passes whichever order the list actually has.
//
// Before this change every subtest below that expects a row got none: ACL
// returned the denial before the audit interceptor ran, so the same RBAC
// refusal was recorded for an agent on the internal MCP chain and lost for a
// human on the public one.
// The subtests run in order and share one server: each asserts a running row
// count for its method, so neither they nor the parent may be parallel.
func TestPublicChainAuditsPolicyDenials(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspaceResp, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceName := workspaceResp.Msg.Name

	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "policy denial audit",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	instanceName := instanceResp.Msg.Name

	// A plain workspace member: the instance methods below are denied by the
	// ACL interceptor, not by a handler, which is what makes them policy
	// verdicts rather than ordinary failures.
	const memberEmail = "denial-audit-member@example.com"
	const memberPassword = "1024bytebase"
	memberResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "denial audit member",
			Email:    memberEmail,
			Password: memberPassword,
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, memberResp.Msg.Workspace, "user:"+memberEmail, "roles/workspaceMember")
	a.NoError(err)
	memberLogin, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    memberEmail,
		Password: memberPassword,
	}))
	a.NoError(err)

	memberInstances := v1connect.NewInstanceServiceClient(ctl.client, ctl.rootURL,
		connect.WithInterceptors(&authInterceptor{token: memberLogin.Msg.Token}))

	// Rows are read back through the same v1 API an operator uses, so the test
	// fails if the row is written but unreachable.
	rowsFor := func(method string) []*v1pb.AuditLog {
		t.Helper()
		resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:  workspaceName,
			Filter:  `method == "` + method + `"`,
			OrderBy: "create_time desc",
		}))
		require.NoError(t, err)
		return resp.Msg.AuditLogs
	}

	t.Run("a denied call on an audited method is recorded", func(t *testing.T) {
		_, err := memberInstances.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
			Instance:   &v1pb.Instance{Name: instanceName, Title: "renamed by a member"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}))
		require.Error(t, err)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

		rows := rowsFor("/bytebase.v1.InstanceService/UpdateInstance")
		require.Len(t, rows, 1, "the denied update is the only UpdateInstance call in this workspace")
		row := rows[0]
		require.Equal(t, "users/"+memberEmail, row.User)
		require.Equal(t, instanceName, row.Resource,
			"the resource comes from the request, which named it even though the call was refused")
		require.NotNil(t, row.Status)
		require.Equal(t, int32(connect.CodePermissionDenied), row.Status.Code)
		require.Equal(t, v1pb.AuditLog_WARNING, row.Severity,
			"a refused caller must be separable from routine traffic on severity alone")
	})

	t.Run("a denied call on a DENIALS method is recorded", func(t *testing.T) {
		// GetInstance declares DENIALS: refusals recorded, ordinary console
		// reads not. Before the annotation became an enum it could say only
		// "audit everything" or nothing, so this method said nothing and its
		// refusals were invisible on both chains.
		_, err := memberInstances.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
			Name: instanceName,
		}))
		require.Error(t, err)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

		rows := rowsFor("/bytebase.v1.InstanceService/GetInstance")
		require.Len(t, rows, 1)
		require.Equal(t, "users/"+memberEmail, rows[0].User)
		require.Equal(t, v1pb.AuditLog_WARNING, rows[0].Severity)
	})

	t.Run("a permitted call on that same DENIALS method is not", func(t *testing.T) {
		// The half that makes DENIALS worth having. GetInstance is console
		// traffic; recording every success is the volume this mode avoids, and
		// the count stays at the one denial above.
		_, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
			Name: instanceName,
		}))
		require.NoError(t, err)

		require.Len(t, rowsFor("/bytebase.v1.InstanceService/GetInstance"), 1,
			"a permitted call on a DENIALS method must stay unrecorded")
	})

	t.Run("a handler's own IAM verdict is recorded too", func(t *testing.T) {
		// GetSetting is auth_method = CUSTOM, so it clears the ACL interceptor
		// — doIAMPermissionCheck returns true for every non-IAM method — and
		// makes its own verdict in checkSettingPermission. For a CUSTOM method
		// the handler IS the access-control layer, so without the mark there
		// its audit = DENIALS annotation would be false for the caller it just
		// refused. TestLintHandlerIAMVerdictsAreMarked holds the whole
		// population; this proves one of them end to end through a real chain.
		memberSettings := v1connect.NewSettingServiceClient(ctl.client, ctl.rootURL,
			connect.WithInterceptors(&authInterceptor{token: memberLogin.Msg.Token}))
		_, err := memberSettings.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
			Name: "settings/APP_IM",
		}))
		require.Error(t, err)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

		rows := rowsFor("/bytebase.v1.SettingService/GetSetting")
		require.Len(t, rows, 1, "a CUSTOM method's own IAM verdict must reach the audit log")
		require.Equal(t, "users/"+memberEmail, rows[0].User)
		require.Equal(t, v1pb.AuditLog_WARNING, rows[0].Severity)
	})

	t.Run("a rejected credential is not a policy verdict", func(t *testing.T) {
		// Auth sits outside audit and refuses before it. There is no
		// authenticated caller to attribute a verdict to, so a 401 writes
		// nothing — the explicit boundary of the rule.
		bogus := v1connect.NewInstanceServiceClient(ctl.client, ctl.rootURL,
			connect.WithInterceptors(&authInterceptor{token: "not-a-token"}))
		_, err := bogus.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
			Instance:   &v1pb.Instance{Name: instanceName, Title: "renamed by nobody"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}))
		require.Error(t, err)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

		require.Len(t, rowsFor("/bytebase.v1.InstanceService/UpdateInstance"), 1,
			"the count is still the one denial from the first subtest")
	})

	t.Run("a denied method that names no resource files under the workspace", func(t *testing.T) {
		// ListInstances reaches neither arm of getRequestResource: it is not in
		// the explicit type switch, and the reflection fallback returns empty
		// for a List method whose optional parent is unset. The row keeps an
		// empty resource and falls back to the caller's own workspace as its
		// parent. Characterized, not fixed — this is the shape MCP gate
		// denials have had since they started being recorded.
		_, err := memberInstances.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{}))
		require.Error(t, err)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

		rows := rowsFor("/bytebase.v1.InstanceService/ListInstances")
		require.Len(t, rows, 1)
		require.Empty(t, rows[0].Resource, "no field of the request names a resource")
		require.True(t, strings.HasPrefix(rows[0].Name, workspaceName+"/auditLogs/"),
			"the row is filed under the caller's own workspace, not under a project: %s", rows[0].Name)
	})
}
