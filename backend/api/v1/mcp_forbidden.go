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
	// InstanceService/UpdateDataSource is the same "carry an existing one out"
	// shape as the two Test methods, against the database's credentials rather
	// than Bytebase's own: it merges a partial request onto the STORED,
	// already-decrypted data source, so an update_mask naming only a
	// destination field (host, port, ssh_host, additional_addresses,
	// sasl_config.kdc_host) keeps the stored password, ssl_key and
	// ssh_private_key. With validate_only it dials the caller's host
	// immediately and persists nothing; without it the retarget is written and
	// SyncInstance triggers the connection on demand. No allowlist filters the
	// host on either path. A database user is a principal other than the
	// caller, the same way the SMTP account behind TestEmailSetting is.
	//
	// Its siblings are NOT here and the line is severity, not tidiness:
	// UpdateInstance and BatchUpdateInstances rebuild the data-source list
	// wholesale, so every secret is wiped unless resent — except the Kerberos
	// keytab, which retainStoredKeytabs used to inherit by data-source ID.
	// That narrower, persist-only version of the vector is now closed at the
	// retention rule instead of at reachability: a keytab is not inherited to
	// a destination the caller moved (instance_service_converter.go). It costs
	// an agent no instance management, something no one has yet decided it
	// should lose, and it binds human callers too — a Kerberos host edit now
	// asks for the keytab again, on the console as much as on the API.
	// The keytab is the ONLY secret that rule reaches, because it is the only
	// one those two methods inherit. On UpdateDataSource above, every other
	// stored secret still rides an update_mask that names only a destination,
	// which is why that method is refused here rather than fixed there:
	// requiring a password re-supplied on every host edit is a product call.
	// AddDataSource and CreateInstance also dial on validate_only but build
	// the data source entirely from the request, so they carry no stored
	// secret and are not this class. BOT-57.
	//
	// The Undelete* family (user, service account, workload identity) is
	// deliberately NOT in this group, and it is the nearest thing left out, so
	// the reasoning is recorded rather than left to be rediscovered. It
	// restores a principal whose password or key hash survived the soft
	// delete, so an operator's deactivation is undone. It is out because the
	// caller learns and chooses nothing: the credential goes back to whoever
	// already had it, and a second delete takes it away again. Issuing beats
	// re-arming, and this mechanism is about issuing. BOT-54.
	v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS: "it hands someone control of a principal other than the caller, which revoking this session would not take back",

	// SettingService/UpdateSetting, refused for the boundary it rewrites
	// rather than for any credential it hands out. Three mask paths carry it:
	// value.workspace_profile.mcp_capability IS the MCP ceiling, so a session
	// that reaches it is not bounded by it; value.email.smtp keeps the
	// stored password when the request omits it while accepting a new host,
	// which hands over the relay resolvePreLoginEmailSetting reads to mail
	// password resets and login codes; and value.ai.endpoint does the same to
	// the AI key it never names — the stored api_key survives the mask, and the
	// next AIService/Chat puts it in an auth header to the host just written
	// (ai_service.go). SaaS refuses AI writes outright (setting_service.go), so
	// that third one is a self-hosted vector, and the key it carries out is
	// whatever the operator configured — GEMINI_API_KEY, where it is set, seeds
	// one key into every workspace created after it (getAdditionalWorkspaceSettings).
	// TestEmailSetting is the one-shot version of the second; this is the
	// persisting one. The same method also writes the SSO domain allowlist and
	// the sign-in switches.
	//
	// Classification is per method, so this refuses the whole RPC — including
	// the settings that have nothing to do with any of them. Splitting the
	// handler so ordinary configuration stays reachable to an agent is the
	// follow-up (BOT-53); disallowing first is the deliberate order.
	v1pb.MCPForbiddenReason_REWRITES_SESSION_BOUNDARY: "it rewrites the workspace settings that bound this session, including the switch meant to contain it",

	// The four approval methods. ApproveIssue, RejectIssue and RequestIssue are
	// one handler under three actions (issue_review.go reviewIssue), and that
	// handler records the review decision itself. An agent composes a change; it
	// does not move its own change through the gate. That is the whole claim,
	// and it is deliberately narrower than "an agent only executes approved work",
	// which this classification does NOT deliver: CreatePlan, CreateRollout and
	// BatchRunTasks are all WRITE, and both approval checks on the execution
	// path are guarded on an issue existing, so a plan created without one
	// reaches execution with no approval at all. Whether an issueless rollout
	// belongs in WRITE is the gate PR's decision to take deliberately rather
	// than inherit.
	//
	// RetryIssueApproval is the near miss and is in deliberately. It casts no
	// vote — it re-runs approval-template finding for an issue stuck in
	// CHECKING, and only the issue creator may call it (issue_service.go
	// canRequestIssue) — but it is the other half of the reason's wording
	// rather than an exception to it: on an auto-approved result the same call
	// activates the grant and enqueues the rollout (issue_service.go:789,796),
	// so it moves an issue through the gate without any human acting. What it
	// does not buy is containment of re-derivation in general: UpdatePlan with
	// a specs mask, and UpdateIssue on a label change, both reset
	// ApprovalFindingDone and force the template to be found again against the
	// current workspace rule (component/review/plan.go, metadata.go), and both
	// stay WRITE. That is the intended line — editing a proposal is the agent's
	// job, and re-review after an edit is the system working — not an oversight.
	// Refusing RetryIssueApproval costs an agent the self-service recovery for
	// its own stuck issue; the operator retries from the console.
	v1pb.MCPForbiddenReason_DRIVES_THE_APPROVAL_DECISION: "it works the approval step meant to gate the change, and an agent does not move its own change through that gate",
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
// Only FORBIDDEN is enforced. Every v1 RPC now carries a class and CI rejects
// UNSPECIFIED, so READ, WRITE and EXCLUDED are recorded classifications with no
// request-time reader yet: a method carrying one is served exactly as it is
// today, and the gate that selects between them is a later change. This
// interceptor grows with the annotations rather than with a list kept here.
//
// Sitting inside audit gets the denial a row for most of the annotated
// FORBIDDEN methods, but not all, and the exceptions are worth knowing because
// none of them is visible from the annotations alone. Two mechanisms remain:
//
//   - No audit annotation, so needAudit refuses the row outright:
//     SwitchWorkspace, Refresh, TestIdentityProvider, TestEmailSetting.
//
//   - Audit annotation, but no reachable audit parent: RequestPasswordReset,
//     ResetPassword and SendEmailLoginCode are allow_without_credential, so
//     createAuditLog takes their parent ONLY from what the handler validated
//     via common.SetAuditWorkspaceID and deliberately skips the workspace
//     fallback (handlerValidatedWorkspaceMethod, audit.go) — an unvalidated
//     workspace on an unauthenticated method would let anyone write rows into
//     someone else's. This gate refuses before dispatch, so the handler never
//     runs and no parent is ever set. #21162 gave the first two the audit
//     annotation; it did not give them rows.
//
// A third way is CLOSED and kept here because it is the one a reader would
// otherwise rediscover: createAuditLog used to return on its first statement
// for anything whose validate_only field is set, before it considered the
// denial at all. That silenced any FORBIDDEN method whose request carries the
// field — CreateIdentityProvider, UpdateSetting and UpdateDataSource — and
// validate_only is the variant that leaves no other trace, since it dials the
// caller's host and persists nothing. The skip now applies only when the call
// SUCCEEDED (audit.go), so a denial always records.
//
// The first group is 1b-2's typed policy-denial record. The second needs that
// too, or the audit path to accept the delegated credential's workspace, which
// the internal chain has already verified.
// TestMCPResetFlowDenialsAreSilent pins the second, so the next person to add
// an annotation learns it is not enough from a red test rather than from an
// operator asking where the row went; TestMCPCannotRetargetADataSource now
// asserts the closed third, as an A/B on one flag — two identical denials of
// one method, both auditable. The ones that sting are TestIdentityProvider and
// TestEmailSetting: each would carry a stored secret to an address the agent
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
