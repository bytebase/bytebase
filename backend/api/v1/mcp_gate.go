package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// mcpDenialWording is one reason's sentence plus the class it belongs to. The
// class is carried here because a reason implies one — a mechanism that breaks
// the MCP boundary is what FORBIDDEN means, a scope decision is what EXCLUDED
// means — and one table saying so is what lets the lint check the pairing and
// the gate refuse to print the wrong half of it.
type mcpDenialWording struct {
	class    v1pb.MCPMethodClass
	sentence string
}

// mcpDenialReasons is UX copy, NOT the classification and NOT the mapping.
// Both of those live on the RPC itself, as the bytebase.v1.mcp_method_class and
// bytebase.v1.mcp_denial_reason annotations, beside permission / audit /
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
var mcpDenialReasons = map[v1pb.MCPDenialReason]mcpDenialWording{
	// The method hands a token back to the caller. For a non-web caller — and
	// an MCP session is always one — finalizeLogin and switchWorkspaceInternal
	// put a plain bb.user.access token in the response body. That token is not
	// audience-bound to the MCP resource, survives revocation of the OAuth
	// grant, and ignores the workspace MCP kill switch, so obtaining one ends
	// the MCP boundary for good.
	v1pb.MCPDenialReason_MINTS_CREDENTIAL: {v1pb.MCPMethodClass_FORBIDDEN, "it hands back a login token that would outlive the MCP grant"},

	// The method drives the out-of-band reset flow — mailing a reset or login
	// code, or consuming one — that sets or delivers the very secret Login
	// accepts. Denying Login alone would leave the agent holding the credential
	// for the next human login.
	v1pb.MCPDenialReason_RESETS_CREDENTIAL: {v1pb.MCPMethodClass_FORBIDDEN, "it drives the credential-reset flow that sets or delivers the secret a login accepts"},

	// UpdateUser's password and MFA branches take no proof of the old password.
	// A caller updating itself needs no permission at all, and on self-hosted a
	// caller holding bb.users.update reaches any other end user's password too
	// (user_service.go: the caller != subject branch checks that permission and
	// nothing further). Either way the session ends up holding credentials it
	// can log in with. The whole method is refused, not just those branches —
	// the classification is per method.
	v1pb.MCPDenialReason_TAKES_OVER_ACCOUNT: {v1pb.MCPMethodClass_FORBIDDEN, "it can rewrite an account's credentials, which would let the session take that account over"},

	// Logout deletes the web refresh token and expires the session cookies. It
	// mints nothing; it destroys the human's own login session, which an agent
	// acting on their behalf has no business doing.
	v1pb.MCPDenialReason_ENDS_SESSION: {v1pb.MCPMethodClass_FORBIDDEN, "it ends the human's own login session"},

	// The workspace-lifecycle pair. Both end in
	// AuthService.switchWorkspaceInternal, which mints a plain workspace token
	// whenever the caller has no refresh cookie — and an MCP session never has
	// one — after having already destroyed the caller's membership.
	v1pb.MCPDenialReason_ENDS_MEMBERSHIP: {v1pb.MCPMethodClass_FORBIDDEN, "it destroys the caller's own workspace membership and mints a plain workspace token"},

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
	v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS: {v1pb.MCPMethodClass_FORBIDDEN, "it hands someone control of a principal other than the caller, which revoking this session would not take back"},

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
	v1pb.MCPDenialReason_REWRITES_SESSION_BOUNDARY: {v1pb.MCPMethodClass_FORBIDDEN, "it rewrites the workspace settings that bound this session, including the switch meant to contain it"},

	// The four approval methods. ApproveIssue and RejectIssue are two actions of
	// one handler (issue_review.go reviewIssue), and that handler records the
	// review decision itself: applyReviewAction requires an approver role via
	// canReview, enforces the self-approval guard, and appends an APPROVED or
	// REJECTED approver (component/review/workflow.go). An agent composes a
	// change; it does not move its own change through the gate. That is the
	// whole claim,
	// and it is deliberately narrower than "an agent only executes approved work",
	// which this classification does NOT deliver: CreatePlan, CreateRollout and
	// BatchRunTasks are all WRITE, and both approval checks on the execution
	// path are guarded on an issue existing, so a plan created without one
	// reaches execution with no approval at all. Whether an issueless rollout
	// belongs in WRITE is the gate PR's decision to take deliberately rather
	// than inherit.
	//
	// RequestIssue is the third action of the same handler and is deliberately
	// NOT here — it is WRITE. Spec §1b-1 named four approval methods; that line
	// grouped by RPC family rather than by mechanism, and the mechanism does not
	// agree. This action requires the issue to be already rejected, requires the
	// actor to be the issue CREATOR, never calls canReview, and records no
	// decision: it strips the REJECTED approvers and returns the issue to
	// PENDING for a fresh human decision. It cannot approve anything, so
	// refusing it protects nothing the other two do not, and it costs the
	// propose-fix-resubmit loop, since while a rejection stands Approve and
	// Reject both hard-fail and this is the only exit from that state. The
	// reasoning is on the RPC in issue_service.proto. Raised by the Codex
	// review; Vincent took the call.
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
	v1pb.MCPDenialReason_DRIVES_THE_APPROVAL_DECISION: {v1pb.MCPMethodClass_FORBIDDEN, "it works the approval step meant to gate the change, and an agent does not move its own change through that gate"},

	// The other refused class. An EXCLUDED method is out of scope for the modes
	// this release ships and an admin-capable ceiling could legitimately serve
	// it one day, while a FORBIDDEN method never becomes servable — so a denial
	// that blurred the two would tell an operator the wrong thing about whether
	// asking is worth it. The class on each row is what keeps them apart.
	v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE:   {v1pb.MCPMethodClass_EXCLUDED, "it administers the workspace rather than doing database work, and no MCP mode this release ships covers workspace administration"},
	v1pb.MCPDenialReason_READS_OTHER_USERS_SQL:       {v1pb.MCPMethodClass_EXCLUDED, "it returns SQL that other people wrote, across the workspace or past the sharing that keeps a saved query private"},
	v1pb.MCPDenialReason_OPENS_AN_ADMIN_CONNECTION:   {v1pb.MCPMethodClass_EXCLUDED, "it opens an admin-credentialed connection to the database and returns other sessions' live, unmasked SQL"},
	v1pb.MCPDenialReason_SENDS_DATA_TO_A_THIRD_PARTY: {v1pb.MCPMethodClass_EXCLUDED, "it spends a stored workspace credential to send whatever the caller passes to a third party"},
	v1pb.MCPDenialReason_RETURNS_A_STORED_SECRET:     {v1pb.MCPMethodClass_EXCLUDED, "its response carries a stored secret that the product redacts everywhere else"},
}

// The fallback wording, per class, for a method whose reason is unset, unknown
// to this build, or recorded for the other refused class. The class annotation
// is what denies; the table only supplies a better sentence.
const (
	reasonForbiddenClass = "it is not reachable by an AI agent session"
	reasonExcludedClass  = "no MCP mode this release ships serves it"
)

// denialReason is the sentence for this method's recorded reason, or the
// class's fallback.
//
// The class is checked here as well as in the lint, and that is the point of
// carrying it on the row. One enum means a reason recorded for the wrong class
// is a value, not a parse error, so the runtime has to decline it: a FORBIDDEN
// method must never explain itself with an exclusion's sentence. A mismatch
// costs wording, the way a missing row does — never enforcement.
func denialReason(class v1pb.MCPMethodClass, reason v1pb.MCPDenialReason, fallback string) string {
	if wording, ok := mcpDenialReasons[reason]; ok && wording.class == class {
		return wording.sentence
	}
	return fallback
}

// mcpServingClasses is the ceiling: which method classes each stored capability
// serves. It is the whole of what the gate evaluates against the classification,
// and the lint in mcp_gate_test.go holds the annotations against this same
// variable rather than a copy — two copies would let the lint stay green while
// the runtime rules drifted away from it.
//
// It is keyed on the STORE enum because that is what a workspace's setting row
// holds and what the gate reads back; TestMCPCapabilityEnumsAgree pins the v1
// enum the settings API writes against it, so the two cannot drift apart.
//
// DISABLED serves nothing. It is here rather than omitted because a missing key
// and an empty list mean different things to the gate: an empty list is a mode
// that decided to serve nothing, a missing key is a mode nobody decided about,
// and only the first may reach a caller as an ordinary denial.
var mcpServingClasses = map[storepb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
	storepb.WorkspaceProfileSetting_DISABLED:   {},
	storepb.WorkspaceProfileSetting_READ_ONLY:  {v1pb.MCPMethodClass_READ},
	storepb.WorkspaceProfileSetting_READ_WRITE: {v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE},
}

// mcpCeilingStore is the whole of what the gate needs from the store: one live
// read of the workspace's ceiling. *store.Store satisfies it; a test supplies
// its own, which is what lets a READ_ONLY ceiling be exercised at all before
// 1b-3 lets one connect.
type mcpCeilingStore interface {
	GetMCPCapabilityUncached(ctx context.Context, workspace string) (storepb.WorkspaceProfileSetting_MCPCapability, error)
}

// internalMCPGateInterceptor refuses, before dispatch, every request an MCP
// session may not make. The rule is one line — effective = ceiling ∩ RBAC — and
// this interceptor is the ceiling half: it never grants anything, and ACL runs
// after it exactly as before, so a caller still needs the permission for
// whatever the ceiling lets through.
//
// The ceiling admits READ under a read-only workspace ceiling and above,
// READ and WRITE under read-write, and neither EXCLUDED nor FORBIDDEN under any
// ceiling. A method carrying no classification is refused too: CI rejects an
// unannotated RPC, so reaching that arm means the build was never linted, and
// guessing on behalf of an unclassified method is how a new RPC ships
// reachable.
//
// It belongs to the internal MCP chain only — every request there originates at
// /mcp — and sits inside the audit interceptor, outside ACL: the ceiling
// refuses regardless of what RBAC would have said, and the refusal is recorded.
//
// The ceiling is read live, per request, with no caching anywhere in the path
// (store.GetMCPCapabilityUncached). An admin tightening the ceiling binds the
// next request of a session already open; work already admitted finishes.
//
// A policy denial is recorded whatever the method's audit annotation says: the
// gate marks the outcome and the audit interceptor records it (see
// common.SetMCPPolicyDenied). 47 of the 121 refused methods carry no audit
// annotation — the 4 FORBIDDEN ones that were silent before this gate
// (Refresh, SwitchWorkspace, TestIdentityProvider, TestEmailSetting) plus 43
// EXCLUDED ones — and TestEmailSetting and TestIdentityProvider are the rows an
// operator would most want, since each would have carried a stored secret to an
// address the agent chose. Recording requests that were never recorded is why
// getRequestString grew redactors in the same change (audit.go).
//
// A ceiling the gate cannot act on splits in two, and the split is the same one
// the /mcp connection gate makes. A stored value this build cannot interpret —
// a mistyped enum name, a wrong-typed row — is a policy refusal: it will never
// succeed on retry, so it answers CodePermissionDenied and is audited, and the
// connection gate answers 403 for it. A read that FAILED is an outage: it
// answers CodeUnavailable, is not marked as a policy denial, and the connection
// gate answers 503, the same way it already answers 503 rather than 401 when it
// cannot resolve the token audience. Both refuse — an unknown policy never
// permits — and neither is allowed to describe itself as the other.
//
// Two gaps survive this PR, and both are worth knowing because nothing in the
// annotations shows either.
//
//   - RequestPasswordReset, ResetPassword and SendEmailLoginCode are
//     allow_without_credential, so createAuditLog takes their audit parent ONLY
//     from what the handler announced (handlerValidatedWorkspaceMethod,
//     audit.go) — an unvalidated workspace on an unauthenticated method would
//     let anyone write rows into someone else's. This gate refuses before
//     dispatch, so the handler never runs and no parent is ever set. Their
//     denials are still silent, TestMCPResetFlowDenialsAreSilent pins that they
//     are, and closing it means letting the audit path trust the workspace of
//     the delegated credential the internal chain already verified.
//
//   - The approval gate contains who APPROVES a change, not who executes one
//     without an approval to move past. ApproveIssue, RejectIssue and
//     RetryIssueApproval are FORBIDDEN, but CreatePlan, CreateRollout and
//     BatchRunTasks are WRITE, and the approval check on the execution path is
//     guarded on an issue existing (`project.Setting.RequireIssueApproval &&
//     issueN != nil`, rollout_service.go), so a plan created without an issue
//     reaches execution with no approval at all. 1b-1 left the call to this PR
//     and this PR takes it deliberately: they stay WRITE. Composing and running
//     a change IS the agent's job, the route is no wider than it was before
//     this gate existed, and the gate cannot express the refusal anyway —
//     "the plan has no issue" is not a field of either request, so a
//     request-shape check cannot see it and only a classification change or a
//     handler-level guard can. Both are outside this PR, and the second is the
//     one that would also bind the console. Recorded so the next reader knows
//     it was decided rather than missed.
type internalMCPGateInterceptor struct {
	store mcpCeilingStore
}

// NewInternalMCPGateInterceptor returns the MCP ceiling gate for the internal
// chain. See internalMCPGateInterceptor for what it enforces.
func NewInternalMCPGateInterceptor(stores mcpCeilingStore) connect.Interceptor {
	return &internalMCPGateInterceptor{store: stores}
}

func (in *internalMCPGateInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		policyDenial, err := in.refuse(ctx, req)
		if err == nil {
			return next(ctx, req)
		}
		if policyDenial {
			// Record the refusal before returning it: the audit interceptor
			// wraps this one and reads the mark when the request comes back
			// out. Only a verdict about the caller is marked — an unreadable
			// ceiling and a broken chain are not policy denials.
			common.SetMCPPolicyDenied(ctx)
		}
		return nil, err
	}
}

// WrapStreamingClient is a pass-through: the gate guards inbound handlers.
func (*internalMCPGateInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// WrapStreamingHandler refuses outright. The gate's rule is per method and its
// classification vocabulary covers streaming RPCs too, but a streaming handler
// gets no request message, so the request-shape half of the rule cannot run —
// and the one streaming v1 RPC, SQLService/AdminExecute, is EXCLUDED anyway
// (it opens an admin connection to the customer's database). Refusing every
// stream keeps "every class is enforced at the gate" true without depending on
// the auth interceptor, which also refuses streams on this chain, continuing to
// do so.
func (*internalMCPGateInterceptor) WrapStreamingHandler(connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(_ context.Context, conn connect.StreamingHandlerConn) error {
		return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is not available to MCP sessions: no streaming RPC is served on the MCP transport",
			conn.Spec().Procedure))
	}
}

// refuse returns the error the gate refuses this request with, or nil to let it
// through to ACL. The bool reports whether the refusal is a verdict about the
// caller, which is an audited outcome; an infrastructure failure is not.
func (in *internalMCPGateInterceptor) refuse(ctx context.Context, req connect.AnyRequest) (bool, error) {
	procedure := req.Spec().Procedure
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	if !ok {
		// The auth interceptor runs first and always sets this. Its absence
		// means the chain was reordered, and guessing which class a method is
		// in is exactly the wrong response.
		return false, connect.NewError(connect.CodeInternal,
			errors.New("MCP method classification unavailable: no auth context"))
	}

	switch authCtx.MCPMethodClass {
	case v1pb.MCPMethodClass_FORBIDDEN:
		return true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is not available to MCP sessions because %s. Perform this action signed in to the Bytebase console instead",
			procedure, denialReason(authCtx.MCPMethodClass, authCtx.MCPDenialReason, reasonForbiddenClass)))
	case v1pb.MCPMethodClass_EXCLUDED:
		return true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is served by no MCP capability ceiling because %s. Perform this action signed in to the Bytebase console instead",
			procedure, denialReason(authCtx.MCPMethodClass, authCtx.MCPDenialReason, reasonExcludedClass)))
	case v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE:
	default:
		return true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s carries no MCP classification, so no MCP session may call it", procedure))
	}

	if policyDenial, err := in.refuseByCeiling(ctx, procedure, authCtx.MCPMethodClass); err != nil {
		return policyDenial, err
	}
	if err := refuseByRequestShape(procedure, req.Any()); err != nil {
		return true, err
	}
	return false, nil
}

// refuseByCeiling holds the method's class against the workspace's live
// ceiling.
func (in *internalMCPGateInterceptor) refuseByCeiling(ctx context.Context, procedure string, class v1pb.MCPMethodClass) (bool, error) {
	// The internal auth interceptor puts the delegated credential's workspace
	// on every request it admits, so an empty one means the chain was
	// reordered — a bug in this process, not an outage, and not a verdict
	// about the caller.
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	if workspaceID == "" {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf(
			"%s cannot be checked against the MCP capability ceiling: no workspace on the request", procedure))
	}
	// The store applies the backward-compatible default for a workspace that
	// never set a ceiling and reports everything it cannot make sense of as an
	// error.
	ceiling, err := in.store.GetMCPCapabilityUncached(ctx, workspaceID)
	if err != nil {
		// The agent gets no detail either way: the error text can carry the
		// workspace's storage state. What it does get is the right KIND of
		// answer, because the two failures are opposites for a client. A
		// stored value this build cannot interpret will never succeed on
		// retry — an admin has to rewrite it — so it is a policy refusal and
		// an audited one. A read that failed is an outage, so it is
		// retryable and is not a verdict about the caller. The /mcp
		// connection gate splits the same two the same way.
		slog.Warn("failed to resolve the MCP capability ceiling; refusing the request",
			slog.String("method", procedure), slog.String("workspace", workspaceID), log.BBError(err))
		if errors.Is(err, store.ErrMCPCapabilityUnreadable) {
			return true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
				"%s is refused: this workspace's stored MCP capability ceiling is not one this build understands, "+
					"so the request fails closed. Ask a workspace admin to set the MCP ceiling again in the workspace settings",
				procedure))
		}
		return false, connect.NewError(connect.CodeUnavailable, errors.Errorf(
			"%s is refused: this workspace's MCP capability ceiling could not be read, so the request fails closed. "+
				"Retry shortly; if it persists, a workspace admin must check the workspace MCP settings",
			procedure))
	}
	served, known := mcpServingClasses[ceiling]
	if !known {
		return true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is refused: this workspace's stored MCP capability ceiling %v is not one this build serves. "+
				"Ask a workspace admin to set the MCP ceiling to a supported value in the workspace settings",
			procedure, ceiling))
	}
	if slices.Contains(served, class) {
		return false, nil
	}
	return true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"%s is a %v method and this workspace's MCP capability ceiling is %v, which serves %s. "+
			"Ask a workspace admin to raise the MCP ceiling in the workspace settings, "+
			"or perform this action signed in to the Bytebase console instead",
		procedure, class, ceiling, describeServedClasses(served)))
}

// describeServedClasses renders a mode's serving list for a denial message.
func describeServedClasses(served []v1pb.MCPMethodClass) string {
	if len(served) == 0 {
		return "no method"
	}
	names := make([]string, 0, len(served))
	for _, class := range served {
		names = append(names, class.String())
	}
	return strings.Join(names, " and ") + " methods"
}

// mcpRequestShapeRefusals holds the refusals a per-method class cannot express,
// because what the call does depends on a field of the request rather than on
// which method it is. Each entry returns the reason to refuse for, or "" to
// allow.
//
// This lives inside the gate rather than beside it. The decision is the same
// decision — may this MCP session make this call? — and it has to be taken at
// the same point in the chain, so that the denial is recorded the same way, is
// worded the same way, and reaches the caller before any handler side effect
// can land. A second interceptor would duplicate the slot, the message, and the
// audit mark to serve one method, and it would put the exception somewhere the
// next person reading the class rule would not find it.
//
// The table is deliberately small and expected to stay that way. A method that
// needs one is a method whose class annotation is not the whole truth, and the
// better fix is usually to split the RPC.
var mcpRequestShapeRefusals = map[string]func(msg any) string{
	v1connect.IssueServiceCreateIssueProcedure: refuseGrantIssueCreation,
	v1connect.IssueServiceUpdateIssueProcedure: refuseGrantIssueCreation,
}

// refuseByRequestShape applies the table above.
func refuseByRequestShape(procedure string, msg any) error {
	refuse, ok := mcpRequestShapeRefusals[procedure]
	if !ok {
		return nil
	}
	reason := refuse(msg)
	if reason == "" {
		return nil
	}
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"%s is not available to MCP sessions for this request because %s. "+
			"Perform this action signed in to the Bytebase console instead", procedure, reason))
}

// refuseGrantIssueCreation is the CreateIssue carve-out. CreateIssue is WRITE
// for the database-change issue it exists for, and an agent composing a change
// is the whole point of the MCP surface. A ROLE_GRANT issue is a different
// method wearing the same name: it completes on creation whenever the workspace
// approval rule produces no template, and completing it writes the project IAM
// binding for whichever grantee the request names — which is
// ProjectService/SetIamPolicy, EXCLUDED for exactly that outcome. The session
// ends up granting access with no human step.
//
// It is an allow-list of the type the class covers, not a deny-list of the ones
// that reach past it. A deny-list would silently admit the next issue type
// somebody adds, and adding an issue type is not where anyone would think to
// re-read the MCP ceiling. ACCESS_GRANT is refused by that allow-list and never
// reaches the handler in the first place: buildIssueMessage has no arm for it,
// so the type is an invalid argument there, and those issues are written by
// AccessGrantService/CreateAccessGrant, which is EXCLUDED.
//
// UpdateIssue is here because allow_missing makes it the same method: on an
// issue that does not exist it calls CreateIssue with the request's own issue
// (issue_service.go). Refusing only CreateIssue would leave the carve-out a
// one-line detour.
//
// An unset type is NOT refused, on either method, and that is the one place the
// allow-list gives ground deliberately. Nothing can be created from it —
// buildIssueMessage's default arm rejects an unspecified type as an invalid
// argument — so refusing it protects nothing, while refusing it would break the
// ordinary AIP upsert: UpdateIssue takes no `type` update path, so a caller
// setting allow_missing on an issue that DOES exist sends no type at all, and
// would meet a denial whose stated mechanism has nothing to do with its call.
func refuseGrantIssueCreation(msg any) string {
	var issue *v1pb.Issue
	switch m := msg.(type) {
	case *v1pb.CreateIssueRequest:
		issue = m.GetIssue()
	case *v1pb.UpdateIssueRequest:
		if !m.GetAllowMissing() {
			return ""
		}
		issue = m.GetIssue()
	default:
		// The table is keyed by procedure, so the request type is fixed. A
		// mismatch is a wiring bug, and a wiring bug on a refusal path fails
		// closed.
		return fmt.Sprintf("its request could not be read as an issue (%T)", msg)
	}
	switch issue.GetType() {
	case v1pb.Issue_DATABASE_CHANGE, v1pb.Issue_TYPE_UNSPECIFIED:
		return ""
	default:
		return fmt.Sprintf(
			"a %v issue completes on creation whenever the workspace approval rule produces no template, "+
				"which grants access with no human step", issue.GetType())
	}
}
