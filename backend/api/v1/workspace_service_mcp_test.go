package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestLeaveAndDeleteWorkspaceRefuseMCPCaller pins the BOT-49 handler guards
// where the FORBIDDEN interceptor can no longer reach them. The e2e tests used
// to discriminate on the refusal message, so deleting a handler guard turned
// them red; now the interceptor answers first and those tests stay green
// either way. What still depends on the handler guard is ORDERING: it runs
// ahead of the IAM mutation, so a refused caller is not left stranded outside
// the workspace it was just removed from.
//
// Both services carry a nil store and a nil profile on purpose. The guard has
// to return before either is touched, so a regression panics or answers a
// different code rather than quietly mutating.
func TestLeaveAndDeleteWorkspaceRefuseMCPCaller(t *testing.T) {
	s := &WorkspaceService{authService: &AuthService{secret: "test-secret"}}

	// The internal transport's marker: the AuthContext carries the delegated
	// grant, and its presence alone — no header, no field value — makes the
	// request MCP-originated.
	ctx := context.WithValue(context.Background(), common.AuthContextKey,
		&common.AuthContext{DelegatedGrant: &common.DelegatedGrant{}})
	// A resolved caller, so a regressed guard runs on into the store rather
	// than stopping at the unauthenticated check and returning a code that
	// happens to differ.
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{Email: "demo@example.com"})

	t.Run("LeaveWorkspace", func(t *testing.T) {
		resp, err := s.LeaveWorkspace(ctx, connect.NewRequest(&v1pb.LeaveWorkspaceRequest{
			Name: "workspaces/ws-test",
		}))
		require.Nil(t, resp)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err),
			"the guard must refuse before the IAM bindings are touched")
	})

	t.Run("DeleteWorkspace", func(t *testing.T) {
		resp, err := s.DeleteWorkspace(ctx, connect.NewRequest(&v1pb.DeleteWorkspaceRequest{
			Name: "workspaces/ws-test",
		}))
		require.Nil(t, resp)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err),
			"the guard must refuse ahead of every other check in the handler")
	})
}
