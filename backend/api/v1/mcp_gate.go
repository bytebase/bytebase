package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// mcpDenialWording is one reason's sentence and next step, plus the class it
// belongs to. The class is carried here because a reason implies one — a
// mechanism that breaks the MCP boundary is what FORBIDDEN means, a scope
// decision is what EXCLUDED means — and one table saying so is what lets the
// lint check the pairing and the gate refuse to print the wrong half of it.
type mcpDenialWording struct {
	class    v1pb.MCPMethodClass
	sentence string
	nextStep string
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
// An agent reads a denial and relays it to the person it acts for, who trusts
// it over the mechanism, so every row keeps three rules. TestMCPDenialWording
// renders every row through its template and checks what it can:
//
//   - The sentence completes "<procedure> is ... because ___" with what the
//     method can do, at the gate's grain: the whole method, for every caller,
//     argument and resource owner. It says "own" or "the caller's" only where
//     enforcement checks ownership, since a reason naming the worst case reads
//     as permission for the rest.
//   - The next step is one that works for the person reading it: the console
//     runs a masked write unguarded, switching workspace means reauthorizing,
//     and only an approver can approve. mcpDenialNextSteps overrides a row for
//     a method its step would misdirect.
//   - Both use the words of the Access policy page ("MCP access policy",
//     Read-only, Read-write) and never "ceiling", "principal" or an enum name.
var mcpDenialReasons = map[v1pb.MCPDenialReason]mcpDenialWording{
	// The method hands a token back to the caller. For a non-web caller — and
	// an MCP session is always one — finalizeLogin and switchWorkspaceInternal
	// put a plain bb.user.access token in the response body. That token is not
	// audience-bound to the MCP resource, survives revocation of the OAuth
	// grant, and ignores the workspace MCP kill switch, so obtaining one ends
	// the MCP boundary for good.
	v1pb.MCPDenialReason_MINTS_CREDENTIAL: {v1pb.MCPMethodClass_FORBIDDEN,
		"it returns a sign-in credential (a session token, an MFA secret, or recovery codes) that would keep working after this MCP connection is revoked",
		nextStepYourself},

	// The method drives the out-of-band reset flow — mailing a reset or login
	// code, or consuming one — that sets or delivers the very secret Login
	// accepts. Denying Login alone would leave the agent holding the credential
	// for the next human login.
	v1pb.MCPDenialReason_RESETS_CREDENTIAL: {v1pb.MCPMethodClass_FORBIDDEN,
		"it sends or redeems a one-time code or reset link that can sign in to an account or change its credentials",
		nextStepYourself},

	// UpdateUser's password and MFA branches take no proof of the old password.
	// A caller updating itself needs no permission at all, and on self-hosted a
	// caller holding bb.users.update reaches any other end user's password too
	// (user_service.go: the caller != subject branch checks that permission and
	// nothing further). Either way the session ends up holding credentials it
	// can log in with. The whole method is refused, not just those branches —
	// the classification is per method.
	v1pb.MCPDenialReason_TAKES_OVER_ACCOUNT: {v1pb.MCPMethodClass_FORBIDDEN,
		"it can rewrite an account's credentials, which would let the session take that account over",
		nextStepYourself},

	// Logout deletes the web refresh token and expires the session cookies. It
	// mints nothing; it destroys the human's own login session, which an agent
	// acting on their behalf has no business doing. An agent reaching for it
	// most likely wants to disconnect, which is what reauthorize does.
	v1pb.MCPDenialReason_ENDS_SESSION: {v1pb.MCPMethodClass_FORBIDDEN,
		"it signs the user out of their Bytebase web session",
		"To end this MCP connection instead, run the reauthorize tool or remove Bytebase from your MCP client."},

	// The workspace-lifecycle pair. Both end in
	// AuthService.switchWorkspaceInternal, which mints a plain workspace token
	// whenever the caller has no refresh cookie — and an MCP session never has
	// one — after having already destroyed the caller's membership.
	v1pb.MCPDenialReason_ENDS_MEMBERSHIP: {v1pb.MCPMethodClass_FORBIDDEN,
		"it deletes the workspace or removes the user from it, and can return a sign-in token for another workspace that this MCP connection's limits would not cover",
		nextStepYourself},

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
	v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS: {v1pb.MCPMethodClass_FORBIDDEN,
		"it can create, reveal, or redirect a credential for another account, service, or database (a key, token, password, or sign-in trust), and revoking this MCP connection would not take that credential back",
		nextStepConsole},

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
	v1pb.MCPDenialReason_REWRITES_SESSION_BOUNDARY: {v1pb.MCPMethodClass_FORBIDDEN,
		"it changes workspace settings, and some of them (the MCP access policy, sign-in and SSO, the mail server, the AI provider) control what this session can do, so AI agents may not change any workspace setting",
		nextStepConsole},

	// The four approval methods. ApproveIssue and RejectIssue are two actions of
	// one handler (issue_review.go reviewIssue), and that handler records the
	// review decision itself: applyReviewAction requires an approver role via
	// canReview, enforces the self-approval guard, and appends an APPROVED or
	// REJECTED approver (component/review/workflow.go). An agent composes a
	// change; it makes no approval decision on any issue, whoever created it.
	// That is the whole claim,
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
	// its own stuck issue; the issue's creator retries from the console.
	v1pb.MCPDenialReason_DRIVES_THE_APPROVAL_DECISION: {v1pb.MCPMethodClass_FORBIDDEN,
		"it approves, rejects, or re-checks an issue's approval, and AI agents may not make approval decisions on any issue, whoever created it",
		"If you are an approver for this issue, approve or reject it in the Bytebase console."},

	// The other refused class. An EXCLUDED method is out of scope for the modes
	// this release ships and an admin-capable ceiling could legitimately serve
	// it one day, while a FORBIDDEN method never becomes servable — so a denial
	// that blurred the two would tell an operator the wrong thing about whether
	// asking is worth it. The class on each row is what keeps them apart.
	v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE: {v1pb.MCPMethodClass_EXCLUDED,
		"it belongs to workspace administration: members, roles and access, sign-in, instances and projects, policies and data classification, audit logs, settings, and billing",
		nextStepConsole},
	v1pb.MCPDenialReason_READS_OTHER_USERS_SQL: {v1pb.MCPMethodClass_EXCLUDED,
		"it returns SQL that other people wrote, across the workspace or past the sharing that keeps a saved query private",
		"To read your own query history, call QueryHistoryService/SearchQueryHistories; to find saved queries you can open, call SavedQueryService/SearchSavedQueries."},
	v1pb.MCPDenialReason_OPENS_AN_ADMIN_CONNECTION: {v1pb.MCPMethodClass_EXCLUDED,
		"it opens an admin-credentialed connection to the database and returns other sessions' live, unmasked SQL",
		nextStepConsole},
	v1pb.MCPDenialReason_SENDS_DATA_TO_A_THIRD_PARTY: {v1pb.MCPMethodClass_EXCLUDED,
		"it contacts an outside service (the configured AI provider or a webhook endpoint) on the workspace's behalf",
		nextStepConsole},
	// No method carries this one: the three leaks it was written for are
	// redacted on the read path and their eight methods are READ. The row
	// stays so the next read found leaking is one annotation away from a
	// denial that explains itself, rather than two edits away.
	v1pb.MCPDenialReason_RETURNS_A_STORED_SECRET: {v1pb.MCPMethodClass_EXCLUDED,
		"its response carries a stored secret that the product redacts everywhere else",
		nextStepConsole},
}

// The two next steps most rows share. Neither promises the console will do it:
// a person may lack the role there too.
const (
	nextStepConsole  = "If your role allows it, do this in the Bytebase console."
	nextStepYourself = "If you need this, do it yourself in the Bytebase console."
)

// mcpDenialNextSteps replaces a row's next step for a method the row's step
// would misdirect: one with a way through MCP, or one only some people can do.
var mcpDenialNextSteps = map[string]string{
	// The MCP connection is bound to the workspace chosen at consent, so a
	// console switch would not move it; reauthorize runs consent again.
	v1connect.AuthServiceSwitchWorkspaceProcedure: "To use this MCP connection with another workspace, run the reauthorize tool and choose that workspace when you approve access again.",
	// Workload identity is exchanged by a pipeline, not in the console.
	v1connect.AuthServiceExchangeTokenProcedure: "Exchange workload identity tokens from your CI/CD pipeline instead.",
	// Only the issue's creator may retry (canRequestIssue), not an approver.
	v1connect.IssueServiceRetryIssueApprovalProcedure: "The issue's creator can re-run its approval check in the Bytebase console.",
}

// The fallback wording, per class, for a method whose reason is unset, unknown
// to this build, or recorded for the other refused class. The class annotation
// is what denies; the table only supplies a better sentence.
var (
	reasonForbiddenClass = mcpDenialWording{
		class:    v1pb.MCPMethodClass_FORBIDDEN,
		sentence: "it is kept out of reach of AI agents",
		nextStep: nextStepConsole,
	}
	reasonExcludedClass = mcpDenialWording{
		class:    v1pb.MCPMethodClass_EXCLUDED,
		sentence: "it is outside what MCP access covers",
		nextStep: nextStepConsole,
	}
)

// denialWording is the wording for this method's recorded reason, or the
// class's fallback, with any per-method next step applied.
//
// The class is checked here as well as in the lint, and that is the point of
// carrying it on the row. One enum means a reason recorded for the wrong class
// is a value, not a parse error, so the runtime has to decline it: a FORBIDDEN
// method must never explain itself with an exclusion's sentence. A mismatch
// costs wording, the way a missing row does — never enforcement.
func denialWording(procedure string, class v1pb.MCPMethodClass, reason v1pb.MCPDenialReason, fallback mcpDenialWording) mcpDenialWording {
	wording := fallback
	if row, ok := mcpDenialReasons[reason]; ok && row.class == class {
		wording = row
	}
	if nextStep, ok := mcpDenialNextSteps[procedure]; ok {
		wording.nextStep = nextStep
	}
	return wording
}

// classDenial is the refusal for a method no MCP access policy serves. Both
// templates say that no policy helps, so the reader does not ask an admin for
// one, and both keep "not available to MCP sessions", the phrase every gate
// refusal shares.
func classDenial(procedure string, class v1pb.MCPMethodClass, reason v1pb.MCPDenialReason) string {
	if class == v1pb.MCPMethodClass_FORBIDDEN {
		wording := denialWording(procedure, class, reason, reasonForbiddenClass)
		return fmt.Sprintf("%s is not available to MCP sessions, whatever the workspace's MCP access policy, because %s. %s",
			procedure, wording.sentence, wording.nextStep)
	}
	wording := denialWording(procedure, class, reason, reasonExcludedClass)
	return fmt.Sprintf("%s is not available to MCP sessions under any MCP access policy because %s. %s",
		procedure, wording.sentence, wording.nextStep)
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
var mcpServingClasses = map[storepb.MCPSetting_Capability][]v1pb.MCPMethodClass{
	storepb.MCPSetting_DISABLED:   {},
	storepb.MCPSetting_READ_ONLY:  {v1pb.MCPMethodClass_READ},
	storepb.MCPSetting_READ_WRITE: {v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE},
}

// mcpSettingsReader is the whole of what the gate needs from the store: one
// live read of the workspace's MCP settings. *store.Store satisfies it; a test
// supplies its own, which is what lets each ceiling be exercised without a
// database. Named for what it does rather than for what satisfies it, per the
// convention for a single-method interface.
type mcpSettingsReader interface {
	GetMCPSettingsUncached(ctx context.Context, workspace string) (*storepb.MCPSetting, error)
}

// mcpSettingsContextKey carries the MCP settings the gate already resolved, so
// every later enforcement point holds the request against that same read rather
// than a second one the gate could disagree with.
type mcpSettingsContextKey struct{}

func withMCPSettings(ctx context.Context, settings *storepb.MCPSetting) context.Context {
	return context.WithValue(ctx, mcpSettingsContextKey{}, settings)
}

func mcpSettingsFromContext(ctx context.Context) (*storepb.MCPSetting, bool) {
	settings, ok := ctx.Value(mcpSettingsContextKey{}).(*storepb.MCPSetting)
	return settings, ok && settings != nil
}

func mcpSettingsForCurrentWorkspace(ctx context.Context, reader mcpSettingsReader, workspaceID string) (*storepb.MCPSetting, error) {
	if settings, ok := mcpSettingsFromContext(ctx); ok {
		return settings, nil
	}
	return reader.GetMCPSettingsUncached(ctx, workspaceID)
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
// refuses regardless of what RBAC would have said, and the refusal is streamed.
//
// The ceiling is read live, per request, with no caching anywhere in the path
// (store.GetMCPSettingsUncached). An admin tightening the ceiling binds the next
// request of a session already open; work already admitted finishes.
//
// A policy denial is marked (setPermissionDenied), so the audit
// interceptor streams it whatever the method's audit annotation says. That is
// why redaction covers every refused request, not only the audited RPCs.
//
// A ceiling the gate cannot act on splits in two, and the split is the same one
// the /mcp connection gate makes. A stored value this build cannot interpret —
// a mistyped enum name, a wrong-typed row — is a policy refusal: it will never
// succeed on retry, so it answers CodePermissionDenied and is marked, and the
// connection gate answers 403 for it. A read that FAILED is an outage: it
// answers CodeUnavailable, is not marked as a policy denial, and the connection
// gate answers 503, the same way it already answers 503 rather than 401 when it
// cannot resolve the token audience. Both refuse — an unknown policy never
// permits — and neither is allowed to describe itself as the other.
//
// One gap survives, and it is worth knowing because nothing in the annotations
// shows it.
//
//   - Two refusals this gate is the right slot for the CLASS of, but not the
//     right place for the decision, because the fact they turn on is not in the
//     request. Both are guarded at the point where the fact is loaded, and both
//     key on the delegated grant so the console is untouched:
//     rejectMCPOriginatedGrantIssue (issue_service.go), for the issue types
//     that complete into a permission grant, and
//     rejectMCPOriginatedIssuelessRollout (rollout_service.go), for the
//     issueless plan that reaches execution without ever meeting the approval
//     the three FORBIDDEN approval methods exist to protect. Without the
//     second, those three are decorative in the change lane: an agent that
//     cannot clear the gate could skip it instead. The gate keeps the half it
//     can decide — CreateIssue, where the request shape IS the whole story —
//     so the denial still lands before dispatch wherever that is possible.
type internalMCPGateInterceptor struct {
	store mcpSettingsReader
}

// NewInternalMCPGateInterceptor returns the MCP ceiling gate for the internal
// chain. See internalMCPGateInterceptor for what it enforces.
func NewInternalMCPGateInterceptor(stores mcpSettingsReader) connect.Interceptor {
	return &internalMCPGateInterceptor{store: stores}
}

func (in *internalMCPGateInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, policyDenial, err := in.refuse(ctx, req)
		if err == nil {
			return next(ctx, req)
		}
		if policyDenial {
			// The audit interceptor wraps this one and reads the mark when the
			// request comes back out. Only a verdict about the caller is marked —
			// an unreadable ceiling and a broken chain are not policy denials.
			setPermissionDenied(ctx)
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
			"%s is not available to MCP sessions because MCP does not carry streaming calls. %s",
			conn.Spec().Procedure, nextStepConsole))
	}
}

// refuse returns the error the gate refuses this request with, or nil to let it
// through to ACL. The bool reports whether the refusal is a verdict about the
// caller, which is marked; an infrastructure failure is not.
//
// The context it returns carries the MCP settings this request was held
// against, for the enforcement points that read the request's argument rather
// than its method (see mcpSettingsFromContext).
func (in *internalMCPGateInterceptor) refuse(ctx context.Context, req connect.AnyRequest) (context.Context, bool, error) {
	procedure := req.Spec().Procedure
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	if !ok {
		// The auth interceptor runs first and always sets this. Its absence
		// means the chain was reordered, and guessing which class a method is
		// in is exactly the wrong response.
		return ctx, false, connect.NewError(connect.CodeInternal,
			errors.New("MCP method classification unavailable: no auth context"))
	}

	switch authCtx.MCPMethodClass {
	case v1pb.MCPMethodClass_FORBIDDEN, v1pb.MCPMethodClass_EXCLUDED:
		return ctx, true, connect.NewError(connect.CodePermissionDenied,
			errors.New(classDenial(procedure, authCtx.MCPMethodClass, authCtx.MCPDenialReason)))
	case v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE:
	default:
		return ctx, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is not available to MCP sessions because this version of Bytebase has not classified it for MCP access. %s",
			procedure, nextStepConsole))
	}

	settings, policyDenial, err := in.refuseByCeiling(ctx, procedure, authCtx.MCPMethodClass)
	if err != nil {
		return ctx, policyDenial, err
	}
	// Stamped before the request-shape table runs, so an entry that keys on the
	// workspace policy reads the resolution this request was admitted under.
	ctx = withMCPSettings(ctx, settings)
	if err := refuseByRequestShape(ctx, procedure, req.Any()); err != nil {
		return ctx, true, err
	}
	return ctx, false, nil
}

// refuseByCeiling holds the method's class against the workspace's live
// ceiling, and returns the settings it resolved so the request can be held
// against the same read further in.
func (in *internalMCPGateInterceptor) refuseByCeiling(ctx context.Context, procedure string, class v1pb.MCPMethodClass) (*storepb.MCPSetting, bool, error) {
	// The internal auth interceptor puts the delegated credential's workspace
	// on every request it admits, so an empty one means the chain was
	// reordered — a bug in this process, not an outage, and not a verdict
	// about the caller.
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	if workspaceID == "" {
		return nil, false, connect.NewError(connect.CodeInternal, errors.Errorf(
			"%s cannot be checked against the workspace's MCP access policy: no workspace on the request", procedure))
	}
	settings, err := in.store.GetMCPSettingsUncached(ctx, workspaceID)

	// auth.ClassifyMCPCeiling decides, so this gate, the /mcp connection gate,
	// the consent and the token endpoint cannot disagree about a workspace —
	// which is what MCPCeilingVerdict says of itself, and was not true while
	// this door mirrored the split by hand.
	//
	// The agent gets no storage detail either way: the shared sentence names
	// the state and the remedy, never the error. What it does get is the right
	// KIND of answer, because the two failures are opposites for a client. A
	// stored value this build cannot interpret never succeeds on retry — an
	// admin has to rewrite it — so it is a policy refusal and a marked one.
	// A read that failed is an outage: retryable, and not a verdict about the
	// caller.
	//
	// DISABLED is deliberately NOT decided here. It reaches the serving table
	// below, which holds an explicit empty list for it, so a mode that serves
	// nothing is refused on the same path as a method the mode leaves out, and
	// servingDenial words both.
	switch verdict := auth.ClassifyMCPCeiling(settings, err); verdict {
	case auth.MCPCeilingServes, auth.MCPCeilingDisabled:
	case auth.MCPCeilingUnavailable:
		slog.Warn("failed to resolve the MCP capability ceiling; refusing the request",
			slog.String("method", procedure), slog.String("workspace", workspaceID), log.BBError(err))
		return nil, false, connect.NewError(connect.CodeUnavailable, errors.Errorf(
			"%s is not available to MCP sessions right now. %s", procedure, verdict.Refusal()))
	default:
		slog.Warn("the MCP capability ceiling refuses this request",
			slog.String("method", procedure), slog.String("workspace", workspaceID), log.BBError(err))
		return nil, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is not available to MCP sessions. %s", procedure, verdict.Refusal()))
	}

	served, known := mcpServingClasses[settings.Capability]
	if !known {
		// Unreachable: a capability no mode serves is MCPCeilingUnserved, which
		// the switch above already refused. Kept because the serving table and
		// the classifier are two statements of one rule, and this is what a
		// caller gets if they ever part company.
		return nil, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is not available to MCP sessions. %s", procedure, auth.MCPCeilingUnserved.Refusal()))
	}
	if slices.Contains(served, class) {
		return settings, false, nil
	}
	return nil, true, connect.NewError(connect.CodePermissionDenied,
		errors.New(servingDenial(procedure, class, settings.Capability)))
}

// servingDenial is the refusal for a method the workspace's policy could serve
// but does not. Unlike classDenial, asking an admin can help here, so it names
// the policy that would serve the method, in the Access policy page's words.
func servingDenial(procedure string, class v1pb.MCPMethodClass, capability storepb.MCPSetting_Capability) string {
	if capability == storepb.MCPSetting_DISABLED {
		return fmt.Sprintf("%s is not available to MCP sessions. %s", procedure, auth.MCPCeilingDisabled.Refusal())
	}
	if class == v1pb.MCPMethodClass_WRITE && capability == storepb.MCPSetting_READ_ONLY {
		return fmt.Sprintf("%s is not available to MCP sessions in this workspace because it needs Read-write MCP access, "+
			"and the workspace's MCP access policy is Read-only. Ask a workspace admin to switch the policy to Read-write "+
			"under %s, or, if your role allows it, do this in the Bytebase console.", procedure, auth.MCPAccessPolicyLocation)
	}
	// Unreachable while the serving table nests READ inside READ_WRITE; kept
	// well-formed for the table that changes that.
	return fmt.Sprintf("%s is not available to MCP sessions in this workspace because the workspace's MCP access policy "+
		"does not include it. Ask a workspace admin to change the policy under %s, or, if your role allows it, "+
		"do this in the Bytebase console.", procedure, auth.MCPAccessPolicyLocation)
}

// mcpRequestShapeRefusals holds the refusals a per-method class cannot express,
// because what the call does depends on a field of the request rather than on
// which method it is. Each entry returns the reason to refuse for, or "" to
// allow.
//
// This lives inside the gate rather than beside it. The decision is the same
// decision — may this MCP session make this call? — and it has to be taken at
// the same point in the chain, so that the denial is marked the same way, is
// worded the same way, and reaches the caller before any handler side effect
// can land. A second interceptor would duplicate the slot, the message, and the
// audit mark to serve one method, and it would put the exception somewhere the
// next person reading the class rule would not find it.
//
// The table is deliberately small and expected to stay that way. A method that
// needs one is a method whose class annotation is not the whole truth, and the
// better fix is usually to split the RPC.
var mcpRequestShapeRefusals = map[string]requestShapeRule{
	v1connect.IssueServiceCreateIssueProcedure:       {refuseGrantIssueCreation, grantIssueNextStep},
	v1connect.SheetServiceCreateSheetProcedure:       {refuseMaskedWriteSheet, maskedWriteNextStep},
	v1connect.SheetServiceBatchCreateSheetsProcedure: {refuseMaskedWriteSheetBatch, maskedWriteNextStep},
	v1connect.ReleaseServiceCreateReleaseProcedure:   {refuseMaskedWriteRelease, maskedWriteNextStep},
	v1connect.SQLServiceQueryProcedure:               {refuseMaskedWriteQuery, maskedWriteNextStep},
	v1connect.SQLServiceExportProcedure:              {refuseMaskedWriteExport, maskedWriteNextStep},

	v1connect.SavedQueryServiceCreateSavedQueryProcedure: {refuseMaskedWriteSavedQuery, maskedWriteNextStep},
	v1connect.SavedQueryServiceUpdateSavedQueryProcedure: {refuseMaskedWriteSavedQueryUpdate, maskedWriteNextStep},
}

// requestShapeRule is one entry of that table: refuse returns the reason to
// refuse for, or "" to allow, and nextStep is what the refusal offers instead.
type requestShapeRule struct {
	refuse   func(ctx context.Context, msg any) string
	nextStep string
}

// refuseByRequestShape applies the table above. The context carries what the
// gate resolved, so an entry can key on more than the request; none does today.
func refuseByRequestShape(ctx context.Context, procedure string, msg any) error {
	rule, ok := mcpRequestShapeRefusals[procedure]
	if !ok {
		return nil
	}
	reason := rule.refuse(ctx, msg)
	if reason == "" {
		return nil
	}
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"%s is not available to MCP sessions for this request because %s. %s", procedure, reason, rule.nextStep))
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
// An unset type is NOT refused: nothing can be created from it, since
// buildIssueMessage's default arm rejects an unspecified type as an invalid
// argument, so refusing it here would protect nothing and would answer a
// mechanism that does not apply.
//
// UpdateIssue is deliberately NOT in this table, and the reason is the limit of
// what a request shape can decide. allow_missing makes UpdateIssue create the
// issue — but only when it does not already exist, which is not a field of the
// request. An AIP upsert sends the complete resource, so a caller PATCHing an
// existing ROLE_GRANT issue carries type=ROLE_GRANT in a body the handler
// ignores, and refusing on that would refuse an ordinary edit for a mechanism
// that is not running. The creation it really does reach is guarded where the
// creation happens instead: rejectMCPOriginatedGrantIssue in issue_service.go.
func refuseGrantIssueCreation(_ context.Context, msg any) string {
	request, ok := msg.(*v1pb.CreateIssueRequest)
	if !ok {
		// The table is keyed by procedure, so the request type is fixed. A
		// mismatch is a wiring bug, and a wiring bug on a refusal path fails
		// closed.
		return fmt.Sprintf("its request could not be read as an issue (%T)", msg)
	}
	switch request.GetIssue().GetType() {
	case v1pb.Issue_DATABASE_CHANGE, v1pb.Issue_TYPE_UNSPECIFIED:
		return ""
	default:
		return grantIssueRefusal
	}
}

// The grant-issue refusal, shared with rejectMCPOriginatedGrantIssue so the
// gate and the handler state one rule. It names the allow-list rather than the
// type refused, because the allow-list is what is enforced.
const (
	grantIssueRefusal = "AI agents may only create database-change issues: a role or access request grants " +
		"a permission, and it can be granted with no human approval when no approval rule applies"
	grantIssueNextStep = "Request the role or access in the Bytebase console."
)
