package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// The reasons a method is forbidden to an MCP session. Each denial names one,
// so the agent — and the operator reading the audit row — learns why rather
// than just that it was refused.
const (
	// reasonMintsCredential covers the methods that hand back, or reset the
	// secret behind, a plain bb.user.access token. Such a token is not
	// audience-bound to the MCP resource, survives revocation of the OAuth
	// grant, and ignores the workspace MCP kill switch, so obtaining one ends
	// the MCP boundary for good.
	reasonMintsCredential = "it issues or resets login credentials, which would hand the session a token that outlives the MCP grant"
	// reasonTakesOverAccount covers self-service account mutation. UpdateUser's
	// password and MFA branches authorize on caller == subject alone, with no
	// permission check and no proof of the old password, so an MCP session can
	// rewrite its own user's credentials and then log in with them.
	reasonTakesOverAccount = "it can rewrite the account's own credentials, which would let the session take the account over"
	// reasonEndsMembership covers the workspace-lifecycle pair. Both end in
	// AuthService.switchWorkspaceInternal, which mints a plain workspace token
	// whenever the caller has no refresh cookie — and an MCP session never has
	// one — after having already destroyed the caller's membership.
	reasonEndsMembership = "it destroys the caller's own workspace membership and mints a plain workspace token"
)

// mcpForbiddenProcedures is the FORBIDDEN method class: v1 methods an MCP
// session must never reach, whatever the caller's own RBAC says. Every entry
// escapes the MCP boundary rather than merely exercising a permission — a
// human with these permissions uses the console, an agent acting for them does
// not get to.
//
// This hardcoded list is the seam. P1b's method classification (1b-1) types
// every v1 RPC READ / WRITE / FORBIDDEN in one table, and 1b-2's gate replaces
// this lookup with the FORBIDDEN rows of that table in this same interceptor
// slot. Growing the table is the migration; nothing else here has to move.
//
// Handler-level guards for the same methods (rejectMCPOriginatedTokenMint on
// SwitchWorkspace, LeaveWorkspace and DeleteWorkspace) stay in place: they also
// cover an external MCP token replayed against the public chain, which this
// internal-chain interceptor never sees.
var mcpForbiddenProcedures = map[string]string{
	// Credential endpoints. Login and its variants return the token in the
	// response body for a non-web caller; the reset and signup paths set the
	// secret that Login then accepts.
	v1connect.AuthServiceLoginProcedure:                reasonMintsCredential,
	v1connect.AuthServiceSignupProcedure:               reasonMintsCredential,
	v1connect.AuthServiceExchangeTokenProcedure:        reasonMintsCredential,
	v1connect.AuthServiceRefreshProcedure:              reasonMintsCredential,
	v1connect.AuthServiceLogoutProcedure:               reasonMintsCredential,
	v1connect.AuthServiceRequestPasswordResetProcedure: reasonMintsCredential,
	v1connect.AuthServiceResetPasswordProcedure:        reasonMintsCredential,
	v1connect.AuthServiceSendEmailLoginCodeProcedure:   reasonMintsCredential,
	v1connect.AuthServiceSwitchWorkspaceProcedure:      reasonMintsCredential,

	// Self-service account mutation.
	v1connect.UserServiceUpdateUserProcedure: reasonTakesOverAccount,

	// Workspace lifecycle.
	v1connect.WorkspaceServiceLeaveWorkspaceProcedure:  reasonEndsMembership,
	v1connect.WorkspaceServiceDeleteWorkspaceProcedure: reasonEndsMembership,
}

// NewInternalMCPForbiddenInterceptor denies the FORBIDDEN method class before
// dispatch. It belongs to the internal MCP chain only — every request there
// originates at /mcp — and sits inside the audit interceptor so a denial is
// recorded, outside ACL because the class is refused regardless of what RBAC
// would have said.
func NewInternalMCPForbiddenInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			if reason, forbidden := mcpForbiddenProcedures[procedure]; forbidden {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
					"%s is not available to MCP sessions because %s. Perform this action signed in to the Bytebase console instead",
					procedure, reason))
			}
			return next(ctx, req)
		}
	})
}
