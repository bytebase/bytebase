package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// mcpForbiddenReasons is UX copy, NOT the classification and NOT the mapping.
// Both of those live on the RPC itself, as the bytebase.v1.mcp_method_class and
// bytebase.v1.mcp_forbidden_reason annotations, beside permission / audit /
// auth_method — one source of truth, visible where the RPC is defined, and read
// here off the AuthContext the auth interceptor already resolves. This table
// only turns a reason into a sentence the agent can act on, so it has one row
// per mechanism rather than one per method, and a missing row costs wording,
// never enforcement.
//
// Each sentence completes "<procedure> is not available to MCP sessions because
// ___", so it states what the method does rather than merely that it is
// refused: a denial whose stated reason has drifted from the mechanism is worse
// than a bare refusal, because it is the thing the next reader trusts.
var mcpForbiddenReasons = map[v1pb.MCPForbiddenReason]string{
	// The method hands a token back to the caller. For a non-web caller — and
	// an MCP session is always one — finalizeLogin and switchWorkspaceInternal
	// put a plain bb.user.access token in the response body. That token is not
	// audience-bound to the MCP resource, survives revocation of the OAuth
	// grant, and ignores the workspace MCP kill switch, so obtaining one ends
	// the MCP boundary for good.
	v1pb.MCPForbiddenReason_MINTS_CREDENTIAL: "it hands back a login token that would outlive the MCP grant",

	// The method drives the out-of-band reset flow — mailing a reset or login
	// code, or consuming one — that sets or delivers the very secret Login
	// accepts. Denying Login alone would leave the agent holding the credential
	// for the next human login.
	v1pb.MCPForbiddenReason_RESETS_CREDENTIAL: "it drives the credential-reset flow that sets or delivers the secret a login accepts",

	// UpdateUser's password and MFA branches take no proof of the old password.
	// A caller updating itself needs no permission at all, and on self-hosted a
	// caller holding bb.users.update reaches any other end user's password too
	// (user_service.go: the caller != subject branch checks that permission and
	// nothing further). Either way the session ends up holding credentials it
	// can log in with. The whole method is refused, not just those branches —
	// the classification is per method.
	v1pb.MCPForbiddenReason_TAKES_OVER_ACCOUNT: "it can rewrite an account's credentials, which would let the session take that account over",

	// Logout deletes the web refresh token and expires the session cookies. It
	// mints nothing; it destroys the human's own login session, which an agent
	// acting on their behalf has no business doing.
	v1pb.MCPForbiddenReason_ENDS_SESSION: "it ends the human's own login session",

	// The workspace-lifecycle pair. Both end in
	// AuthService.switchWorkspaceInternal, which mints a plain workspace token
	// whenever the caller has no refresh cookie — and an MCP session never has
	// one — after having already destroyed the caller's membership.
	v1pb.MCPForbiddenReason_ENDS_MEMBERSHIP: "it destroys the caller's own workspace membership and mints a plain workspace token",

	// The method leaves someone holding a principal the caller is not. Four
	// ways, all of them annotated:
	//   - issuing the credential outright — a service key or SCIM token
	//     returned in plaintext, a caller-chosen password on a new account;
	//   - carrying an existing one out — a stored client secret or SMTP
	//     password sent to a host the caller named;
	//   - choosing what will later be trusted to mint one — the issuer and
	//     subject ExchangeToken validates against;
	//   - redirecting where one gets delivered — UpdateEmail rebinds any
	//     account to an address the caller picked, and the reset flow mails the
	//     code to whatever address the account then carries.
	// The levers that contain a runaway session all act on the caller's own
	// principal: revoke the OAuth grant, flip the workspace MCP switch,
	// deactivate the human. None of them reaches a principal that was never the
	// caller's, so what these give away outlives all three.
	//
	// Two neighbours are deliberately NOT annotated, and both are closer to
	// this mechanism than anything else left out, so the reasoning is recorded
	// rather than left to be rediscovered:
	//
	// SettingService/UpdateSetting is TestEmailSetting's persisting twin. The
	// same stored-password substitution (value.email.smtp with an empty
	// password keeps the stored one) and the setting it writes is the one
	// resolvePreLoginEmailSetting reads to mail password resets and login
	// codes — so it does not merely leak the SMTP credential once, it owns the
	// credential-delivery channel from then on. It also rewrites the MCP
	// ceiling itself (value.workspace_profile.mcp_capability) and the SSO and
	// sign-in switches. It is out because that second mechanism needs its own
	// reason value, and because one RPC covers a dozen unrelated settings whose
	// agent-legitimacy was never measured here — forbidding all of them on the
	// strength of two is a product decision this change did not earn. BOT-53.
	//
	// The Undelete* family (user, service account, workload identity) restores
	// a principal whose password or key hash survived the soft delete, so an
	// operator's deactivation is undone. It is out because the caller learns
	// and chooses nothing: the credential goes back to whoever already had it,
	// and a second delete takes it away again. Issuing beats re-arming, and
	// this mechanism is about issuing. BOT-54.
	v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS: "it hands someone control of a principal other than the caller, which revoking this session would not take back",
}

// reasonForbiddenClass is the fallback for a method annotated FORBIDDEN whose
// reason is unset or unknown to this build. The class annotation is what denies
// the method; the table only supplies better wording.
const reasonForbiddenClass = "it is not reachable by an AI agent session"

// NewInternalMCPForbiddenInterceptor denies every method annotated
// mcp_method_class = FORBIDDEN before dispatch. It belongs to the internal MCP
// chain only — every request there originates at /mcp — and sits inside the
// audit interceptor, outside ACL: the class is refused regardless of what RBAC
// would have said.
//
// Only FORBIDDEN is enforced. READ and WRITE are the serving classes P1b's
// ceiling modes select between, and until every RPC carries a classification
// an unannotated method is served exactly as before — the rollout is method by
// method, and this interceptor grows with the annotations rather than with a
// list kept here.
//
// Sitting inside audit gets the denial a row for the methods that carry the
// audit annotation, which is all twenty-three currently annotated FORBIDDEN bar
// SwitchWorkspace, Refresh, TestIdentityProvider and TestEmailSetting:
// needAudit reads that annotation and nothing else, so those four are denied
// silently until 1b-2 lands the typed policy-denial record that bypasses it.
// (RequestPasswordReset and ResetPassword were in this set until #21162 gave
// them the audit annotation.) The last two are the ones that sting — they are
// the methods that would carry a stored secret to an address the agent chose,
// and their denials are the rows an operator would most want.
func NewInternalMCPForbiddenInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			authCtx, ok := common.GetAuthContextFromContext(ctx)
			if !ok {
				// The auth interceptor runs first and always sets this. Its
				// absence means the chain was reordered, and guessing which
				// class a method is in is exactly the wrong response.
				return nil, connect.NewError(connect.CodeInternal,
					errors.New("MCP method classification unavailable: no auth context"))
			}
			if authCtx.MCPMethodClass != v1pb.MCPMethodClass_FORBIDDEN {
				return next(ctx, req)
			}
			reason, ok := mcpForbiddenReasons[authCtx.MCPForbiddenReason]
			if !ok {
				reason = reasonForbiddenClass
			}
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
				"%s is not available to MCP sessions because %s. Perform this action signed in to the Bytebase console instead",
				req.Spec().Procedure, reason))
		}
	})
}
