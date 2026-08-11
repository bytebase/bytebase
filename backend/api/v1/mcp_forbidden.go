package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// The reasons a method is forbidden to an MCP session. Each denial names one,
// so the agent — and the operator reading the audit row — learns why rather
// than just that it was refused. Each reason states what its methods actually
// do: a denial whose stated reason has drifted from the mechanism is worse
// than a bare refusal, because it is the thing the next reader trusts.
const (
	// reasonMintsCredential: the method hands a token back to the caller. For
	// a non-web caller — and an MCP session is always one — finalizeLogin and
	// switchWorkspaceInternal put a plain bb.user.access token in the response
	// body. That token is not audience-bound to the MCP resource, survives
	// revocation of the OAuth grant, and ignores the workspace MCP kill
	// switch, so obtaining one ends the MCP boundary for good.
	reasonMintsCredential = "it hands back a login token that would outlive the MCP grant"
	// reasonResetsCredential: the method drives the out-of-band reset flow —
	// mailing a reset or login code, or consuming one — that sets or delivers
	// the very secret Login accepts. Denying Login alone would leave the
	// agent holding the credential for the next human login.
	reasonResetsCredential = "it drives the credential-reset flow that sets or delivers the secret a login accepts"
	// reasonTakesOverAccount: UpdateUser's password and MFA branches authorize
	// on caller == subject alone, with no permission check and no proof of the
	// old password, so an MCP session can rewrite its own user's credentials
	// and then log in with them. The whole method is refused, not just those
	// branches — the class is method-keyed (see mcpForbiddenProcedures).
	reasonTakesOverAccount = "it can rewrite the account's own credentials, which would let the session take the account over"
	// reasonEndsSession: Logout deletes the web refresh token and expires the
	// session cookies. It mints nothing; it destroys the human's own login
	// session, which an agent acting on their behalf has no business doing.
	reasonEndsSession = "it ends the human's own login session"
	// reasonEndsMembership: the workspace-lifecycle pair. Both end in
	// AuthService.switchWorkspaceInternal, which mints a plain workspace token
	// whenever the caller has no refresh cookie — and an MCP session never has
	// one — after having already destroyed the caller's membership.
	reasonEndsMembership = "it destroys the caller's own workspace membership and mints a plain workspace token"
)

// mcpForbiddenProcedures holds the credential and account-lifecycle members of
// the FORBIDDEN class: v1 methods an MCP session must never reach, whatever
// the caller's own RBAC says. Every entry escapes the MCP boundary rather than
// merely exercising a permission — a human with these permissions uses the
// console; an agent acting for them does not get to.
//
// It is a SUBSET of the class, deliberately. P1b classifies every v1 RPC
// READ / WRITE / FORBIDDEN (1b-1) and 1b-2's gate replaces this lookup with
// the FORBIDDEN rows of that table, in this same interceptor slot — growing
// the table is the migration; nothing else here has to move. Shipped early
// because these members are the ones an MCP session can reach today with no
// permission at all. Known to be classified FORBIDDEN and NOT yet enforced
// here:
//
//   - the four approval operations (ApproveIssue, RejectIssue, RequestIssue,
//     RetryIssueApproval) — the self-approval guard;
//   - the admin-class credential mints that hand a plaintext bearer straight
//     back in the response body: ServiceAccountService/{Create,Update}
//     ServiceAccount, WorkspaceService/RotateDirectorySyncToken,
//     WorkloadIdentityService and IdentityProviderService writes, and
//     UserService/CreateUser (a caller-chosen password on a fresh principal).
//
// Those are gated on IAM permissions a workspace admin nonetheless holds, so
// they are open until 1b-1 classifies them. Do not read this list as "the
// boundary is closed"; read it as "these twelve are shut".
//
// The classification lives in Go rather than as a proto annotation on purpose:
// the ceiling is private data, with no public vocabulary and nothing an admin
// authors or reads (P1b proposal v2). A method option would publish it in the
// descriptor and OpenAPI surface.
//
// Handler-level guards for three of these (rejectMCPOriginatedTokenMint on
// SwitchWorkspace, LeaveWorkspace and DeleteWorkspace) stay in place as
// defense in depth. Not because an external MCP token could reach them on the
// public chain — checkTokenAudience refuses MCP-provenance bearers there
// during authentication — but because switchWorkspaceInternal is a shared
// mint point that acquired two unguarded callers once already.
var mcpForbiddenProcedures = map[string]string{
	// Methods that put a token in the response body.
	v1connect.AuthServiceLoginProcedure:           reasonMintsCredential,
	v1connect.AuthServiceSignupProcedure:          reasonMintsCredential,
	v1connect.AuthServiceExchangeTokenProcedure:   reasonMintsCredential,
	v1connect.AuthServiceRefreshProcedure:         reasonMintsCredential,
	v1connect.AuthServiceSwitchWorkspaceProcedure: reasonMintsCredential,

	// Methods that set or deliver the secret those tokens are issued against.
	v1connect.AuthServiceRequestPasswordResetProcedure: reasonResetsCredential,
	v1connect.AuthServiceResetPasswordProcedure:        reasonResetsCredential,
	v1connect.AuthServiceSendEmailLoginCodeProcedure:   reasonResetsCredential,

	// Self-service account mutation.
	v1connect.UserServiceUpdateUserProcedure: reasonTakesOverAccount,

	// Session teardown.
	v1connect.AuthServiceLogoutProcedure: reasonEndsSession,

	// Workspace lifecycle.
	v1connect.WorkspaceServiceLeaveWorkspaceProcedure:  reasonEndsMembership,
	v1connect.WorkspaceServiceDeleteWorkspaceProcedure: reasonEndsMembership,
}

// NewInternalMCPForbiddenInterceptor denies mcpForbiddenProcedures before
// dispatch. It belongs to the internal MCP chain only — every request there
// originates at /mcp — and sits inside the audit interceptor, outside ACL:
// the class is refused regardless of what RBAC would have said.
//
// Sitting inside audit gets the denial a row for the methods that carry the
// audit annotation, which is all twelve here bar SwitchWorkspace, Refresh,
// RequestPasswordReset and ResetPassword: needAudit reads that annotation and
// nothing else, so those four are denied silently until 1b-2 lands the typed
// policy-denial record that bypasses it.
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
