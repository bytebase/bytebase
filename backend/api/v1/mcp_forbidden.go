package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
	// reasonTakesOverAccount: UpdateUser's password and MFA branches take no
	// proof of the old password. A caller updating itself needs no permission
	// at all, and on self-hosted a caller holding bb.users.update reaches any
	// other end user's password too (user_service.go: the caller != subject
	// branch checks that permission and nothing further). Either way the
	// session ends up holding credentials it can log in with. The whole method
	// is refused, not just those branches — the classification is per method.
	reasonTakesOverAccount = "it can rewrite an account's credentials, which would let the session take that account over"
	// reasonEndsSession: Logout deletes the web refresh token and expires the
	// session cookies. It mints nothing; it destroys the human's own login
	// session, which an agent acting on their behalf has no business doing.
	reasonEndsSession = "it ends the human's own login session"
	// reasonEndsMembership: the workspace-lifecycle pair. Both end in
	// AuthService.switchWorkspaceInternal, which mints a plain workspace token
	// whenever the caller has no refresh cookie — and an MCP session never has
	// one — after having already destroyed the caller's membership.
	reasonEndsMembership = "it destroys the caller's own workspace membership and mints a plain workspace token"
	// reasonMintsCredentialForOthers: the method leaves someone holding a
	// principal the caller is not. Four ways, all of them here:
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
	reasonMintsCredentialForOthers = "it hands someone control of a principal other than the caller, which revoking this session would not take back"
	// reasonForbiddenClass is the fallback for a method annotated FORBIDDEN
	// that has no entry below. Adding the annotation is what denies the
	// method; the table only supplies better wording.
	reasonForbiddenClass = "it is not reachable by an AI agent session"
)

// mcpForbiddenReasons is UX copy, NOT the classification. What a method is
// classified as lives on the RPC itself, as the bytebase.v1.mcp_method_class
// annotation, beside permission / audit / auth_method — one source of truth,
// visible where the RPC is defined, and read here off the AuthContext the auth
// interceptor already resolves. This table only turns that classification into
// a sentence the agent can act on, and a missing row costs wording, never
// enforcement.
var mcpForbiddenReasons = map[string]string{
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

	// Control of a principal that is not the caller. Most rows above run
	// through the caller's own session or account, so revoking that user
	// eventually contains them; these leave a principal behind that no such
	// revocation touches. The line is not perfectly clean on the earlier rows —
	// Signup creates a new principal, and UpdateUser's password branch reaches
	// other end users when the caller holds bb.users.update — but those were
	// classified for what the caller walks away with, and these are classified
	// for what someone else does.
	v1connect.ServiceAccountServiceCreateServiceAccountProcedure:     reasonMintsCredentialForOthers,
	v1connect.ServiceAccountServiceUpdateServiceAccountProcedure:     reasonMintsCredentialForOthers,
	v1connect.WorkspaceServiceRotateDirectorySyncTokenProcedure:      reasonMintsCredentialForOthers,
	v1connect.UserServiceCreateUserProcedure:                         reasonMintsCredentialForOthers,
	v1connect.UserServiceUpdateEmailProcedure:                        reasonMintsCredentialForOthers,
	v1connect.IdentityProviderServiceCreateIdentityProviderProcedure: reasonMintsCredentialForOthers,
	v1connect.IdentityProviderServiceUpdateIdentityProviderProcedure: reasonMintsCredentialForOthers,
	v1connect.IdentityProviderServiceTestIdentityProviderProcedure:   reasonMintsCredentialForOthers,
	v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure: reasonMintsCredentialForOthers,
	v1connect.WorkloadIdentityServiceUpdateWorkloadIdentityProcedure: reasonMintsCredentialForOthers,
	v1connect.SettingServiceTestEmailSettingProcedure:                reasonMintsCredentialForOthers,
	// Two neighbours are deliberately NOT here, and both are closer to this
	// class than anything else left out, so the reasoning is recorded rather
	// than left to be rediscovered:
	//
	// SettingService/UpdateSetting is TestEmailSetting's persisting twin. The
	// same stored-password substitution (value.email.smtp with an empty
	// password keeps the stored one) and the setting it writes is the one
	// resolvePreLoginEmailSetting reads to mail password resets and login
	// codes — so it does not merely leak the SMTP credential once, it owns the
	// credential-delivery channel from then on. It also rewrites the MCP
	// ceiling itself (value.workspace_profile.mcp_capability) and the SSO and
	// sign-in switches. It is out because that second mechanism needs its own
	// sentence, and because one RPC covers a dozen unrelated settings whose
	// agent-legitimacy was never measured here — forbidding all of them on the
	// strength of two is a product decision this change did not earn. BOT-53.
	//
	// The Undelete* family (user, service account, workload identity) restores
	// a principal whose password or key hash survived the soft delete, so an
	// operator's deactivation is undone. It is out because the caller learns
	// and chooses nothing: the credential goes back to whoever already had it,
	// and a second delete takes it away again. Issuing beats re-arming, and
	// this class is about issuing. BOT-54.
}

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
// SwitchWorkspace, Refresh, RequestPasswordReset, ResetPassword,
// TestIdentityProvider and TestEmailSetting: needAudit reads that annotation
// and nothing else, so those six are denied silently until 1b-2 lands the typed
// policy-denial record that bypasses it. The last two are the ones that sting —
// they are the methods that would carry a stored secret to an address the agent
// chose, and their denials are the rows an operator would most want.
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
			procedure := req.Spec().Procedure
			reason, ok := mcpForbiddenReasons[procedure]
			if !ok {
				reason = reasonForbiddenClass
			}
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
				"%s is not available to MCP sessions because %s. Perform this action signed in to the Bytebase console instead",
				procedure, reason))
		}
	})
}
