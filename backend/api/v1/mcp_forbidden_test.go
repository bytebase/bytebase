package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// forbiddenProceduresFromDescriptors reads the FORBIDDEN class straight off
// the compiled descriptors — the same place getAuthContext reads it — so the
// assertions below are against the annotations themselves, not against a Go
// copy of them that could agree with itself while the protos say otherwise.
func forbiddenProceduresFromDescriptors(t *testing.T) map[string]bool {
	t.Helper()
	found := map[string]bool{}
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		services := fd.Services()
		for i := range services.Len() {
			sd := services.Get(i)
			methods := sd.Methods()
			for j := range methods.Len() {
				md := methods.Get(j)
				class, ok := proto.GetExtension(md.Options(), v1pb.E_McpMethodClass).(v1pb.MCPMethodClass)
				require.True(t, ok, "method %s carries a malformed mcp_method_class", md.FullName())
				if class == v1pb.MCPMethodClass_FORBIDDEN {
					found[fmt.Sprintf("/%s/%s", sd.FullName(), md.Name())] = true
				}
			}
		}
		return true
	})
	return found
}

// TestForbiddenClassMembership pins which RPCs are annotated FORBIDDEN.
// Membership is a security decision, so adding or removing one has to be
// deliberate: this list is the second signature on that decision.
func TestForbiddenClassMembership(t *testing.T) {
	want := []string{
		v1connect.AuthServiceLoginProcedure,
		v1connect.AuthServiceSignupProcedure,
		v1connect.AuthServiceExchangeTokenProcedure,
		v1connect.AuthServiceRefreshProcedure,
		v1connect.AuthServiceLogoutProcedure,
		v1connect.AuthServiceRequestPasswordResetProcedure,
		v1connect.AuthServiceResetPasswordProcedure,
		v1connect.AuthServiceSendEmailLoginCodeProcedure,
		v1connect.AuthServiceSwitchWorkspaceProcedure,
		v1connect.UserServiceUpdateUserProcedure,
		v1connect.WorkspaceServiceLeaveWorkspaceProcedure,
		v1connect.WorkspaceServiceDeleteWorkspaceProcedure,
	}
	got := forbiddenProceduresFromDescriptors(t)

	for _, procedure := range want {
		require.True(t, got[procedure], "%s must be annotated mcp_method_class = FORBIDDEN", procedure)
	}
	require.Len(t, got, len(want),
		"an RPC was annotated FORBIDDEN without being added here; membership must be an explicit decision")

	// Every member should also carry wording an agent can act on. A missing
	// row is a UX regression, not a security one — the annotation is what
	// denies — so this is asserted separately from membership.
	for _, procedure := range want {
		require.Contains(t, mcpForbiddenReasons, procedure,
			"%s has no actionable denial reason", procedure)
	}
	require.Len(t, mcpForbiddenReasons, len(want), "a reason row outlived its method")

	// The reason has to describe what the method does, not merely be present:
	// Logout destroys a session rather than issuing a credential, and a denial
	// that says otherwise teaches the next reader something false.
	require.Equal(t, reasonEndsSession, mcpForbiddenReasons[v1connect.AuthServiceLogoutProcedure],
		"Logout mints nothing — it deletes the refresh token and expires the cookies")
	require.Equal(t, reasonMintsCredential, mcpForbiddenReasons[v1connect.AuthServiceLoginProcedure])
	require.Equal(t, reasonResetsCredential, mcpForbiddenReasons[v1connect.AuthServiceResetPasswordProcedure])
	require.Equal(t, reasonTakesOverAccount, mcpForbiddenReasons[v1connect.UserServiceUpdateUserProcedure])
	require.Equal(t, reasonEndsMembership, mcpForbiddenReasons[v1connect.WorkspaceServiceDeleteWorkspaceProcedure])
}

// TestInternalMCPForbiddenInterceptor pins what the interceptor does with a
// classification once the auth interceptor has resolved one.
func TestInternalMCPForbiddenInterceptor(t *testing.T) {
	interceptor := NewInternalMCPForbiddenInterceptor()

	invoke := func(ctx context.Context, procedure string) (bool, error) {
		dispatched := false
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			dispatched = true
			return connect.NewResponse(&v1pb.User{}), nil
		}
		req := &specRequest{
			AnyRequest: connect.NewRequest(&v1pb.GetUserRequest{}),
			procedure:  procedure,
		}
		_, err := interceptor.WrapUnary(next)(ctx, req)
		return dispatched, err
	}
	withClass := func(class v1pb.MCPMethodClass) context.Context {
		return context.WithValue(context.Background(), common.AuthContextKey,
			&common.AuthContext{MCPMethodClass: class})
	}

	for procedure := range forbiddenProceduresFromDescriptors(t) {
		t.Run(procedure, func(t *testing.T) {
			dispatched, err := invoke(withClass(v1pb.MCPMethodClass_FORBIDDEN), procedure)
			require.Error(t, err, "a FORBIDDEN method must never reach its handler")
			require.False(t, dispatched, "the denial must happen before dispatch, so no handler side effect can land")
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Contains(t, err.Error(), procedure, "the message must name the method the agent called")
			require.Contains(t, err.Error(), mcpForbiddenReasons[procedure],
				"the message must name why, so the agent can act on it")
		})
	}

	// Only FORBIDDEN is enforced in this phase: an unclassified method is
	// served exactly as before, and the serving classes 1b-2 will select
	// between are not gated here yet.
	for name, class := range map[string]v1pb.MCPMethodClass{
		"unclassified": v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED,
		"read":         v1pb.MCPMethodClass_READ,
		"write":        v1pb.MCPMethodClass_WRITE,
	} {
		t.Run("dispatched: "+name, func(t *testing.T) {
			dispatched, err := invoke(withClass(class), v1connect.UserServiceGetUserProcedure)
			require.NoError(t, err)
			require.True(t, dispatched)
		})
	}

	t.Run("no auth context fails closed", func(t *testing.T) {
		dispatched, err := invoke(context.Background(), v1connect.AuthServiceLoginProcedure)
		require.Error(t, err, "without a resolved classification the interceptor must not guess")
		require.False(t, dispatched)
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
