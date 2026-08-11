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

	for procedure, reason := range mcpForbiddenProcedures {
		t.Run(procedure, func(t *testing.T) {
			dispatched, err := invoke(procedure)
			require.Error(t, err, "a FORBIDDEN method must never reach its handler")
			require.False(t, dispatched, "the denial must happen before dispatch, so no handler side effect can land")
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Contains(t, err.Error(), procedure, "the message must name the method the agent called")
			require.Contains(t, err.Error(), reason, "the message must name why, so the agent can act on it")
		})
	}

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
