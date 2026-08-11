package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// TestInternalMCPForbiddenInterceptor pins the two halves of the gate: every
// member of the FORBIDDEN class is refused before dispatch with a message that
// names why, and nothing outside the class is touched.
func TestInternalMCPForbiddenInterceptor(t *testing.T) {
	interceptor := NewInternalMCPForbiddenInterceptor()

	invoke := func(procedure string) (bool, error) {
		dispatched := false
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			dispatched = true
			return connect.NewResponse(&v1pb.User{}), nil
		}
		req := &specRequest{
			AnyRequest: connect.NewRequest(&v1pb.GetUserRequest{}),
			procedure:  procedure,
		}
		_, err := interceptor.WrapUnary(next)(context.Background(), req)
		return dispatched, err
	}

	// Spelled out rather than ranged over: iterating mcpForbiddenProcedures
	// would make the test agree with the map by construction, so dropping an
	// entry would drop its own coverage with it. This list is the assertion.
	forbidden := []string{
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
	require.Len(t, mcpForbiddenProcedures, len(forbidden),
		"a method added to or removed from the class must be an explicit decision, made here too")

	for _, procedure := range forbidden {
		t.Run(procedure, func(t *testing.T) {
			reason, listed := mcpForbiddenProcedures[procedure]
			require.True(t, listed, "%s must be in the FORBIDDEN class", procedure)

			dispatched, err := invoke(procedure)
			require.Error(t, err, "a FORBIDDEN method must never reach its handler")
			require.False(t, dispatched, "the denial must happen before dispatch, so no handler side effect can land")
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Contains(t, err.Error(), procedure, "the message must name the method the agent called")
			require.Contains(t, err.Error(), reason, "the message must name why, so the agent can act on it")
		})
	}

	// The reason has to describe what the method does, not merely be present:
	// Logout destroys a session rather than issuing a credential, and a denial
	// that says otherwise teaches the next reader something false.
	require.Equal(t, reasonEndsSession, mcpForbiddenProcedures[v1connect.AuthServiceLogoutProcedure],
		"Logout mints nothing — it deletes the refresh token and expires the cookies")
	require.Equal(t, reasonMintsCredential, mcpForbiddenProcedures[v1connect.AuthServiceLoginProcedure])
	require.Equal(t, reasonResetsCredential, mcpForbiddenProcedures[v1connect.AuthServiceResetPasswordProcedure])
	require.Equal(t, reasonTakesOverAccount, mcpForbiddenProcedures[v1connect.UserServiceUpdateUserProcedure])
	require.Equal(t, reasonEndsMembership, mcpForbiddenProcedures[v1connect.WorkspaceServiceDeleteWorkspaceProcedure])

	t.Run("an unlisted method is dispatched untouched", func(t *testing.T) {
		dispatched, err := invoke(v1connect.UserServiceGetUserProcedure)
		require.NoError(t, err)
		require.True(t, dispatched)
	})

	// Sibling methods on the same services that deliberately stay reachable.
	// The class is credential and account-lifecycle escape, not "anything
	// touching users or workspaces" — pinning these keeps a later widening of
	// the list an explicit decision rather than a drift.
	for _, procedure := range []string{
		v1connect.UserServiceGetCurrentUserProcedure,
		v1connect.UserServiceListUsersProcedure,
		v1connect.WorkspaceServiceGetWorkspaceProcedure,
		v1connect.WorkspaceServiceGetIamPolicyProcedure,
	} {
		t.Run("reachable: "+procedure, func(t *testing.T) {
			dispatched, err := invoke(procedure)
			require.NoError(t, err)
			require.True(t, dispatched)
		})
	}
}
