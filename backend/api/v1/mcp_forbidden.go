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
	// reasonTakesOverAccount: UpdateUser's password and MFA branches authorize
	// on caller == subject alone, with no permission check and no proof of the
	// old password, so an MCP session can rewrite its own user's credentials
	// and then log in with them. The whole method is refused, not just those
	// branches — the classification is per method.
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
	// reasonMintsCredentialForOthers: the method puts a credential that belongs
	// to a principal the caller is not within someone's reach — by issuing it
	// (a service key or SCIM token returned in plaintext, a caller-chosen
	// password on a new account), by carrying it out (a stored client secret or
	// SMTP password sent to a caller-supplied host), by choosing what will
	// later be trusted to mint it (an issuer and subject ExchangeToken
	// validates against), or by redirecting where it gets delivered
	// (UpdateEmail rebinds any account to an address the caller picked, and the
	// reset flow mails the code to whatever address the account now carries).
	// Every lever that contains a runaway session acts on the caller's own
	// principal: revoke the OAuth grant, flip the workspace MCP switch,
	// deactivate the human. None of them reaches an identity that was never the
	// caller's, so what these give away survives all three.
	reasonMintsCredentialForOthers = "it gives away a credential for a principal other than the caller, which revoking this session would not reach"
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

	// Credentials for a principal that is not the caller. The rows above end
	// at the caller's own account, so revoking that user contains them; these
	// leave an identity behind that no such revocation touches.
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
// audit annotation, which is all twenty-two currently annotated FORBIDDEN bar
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
