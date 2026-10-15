package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
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

	v1pb.MCPDenialReason_DRIVES_THE_APPROVAL_DECISION: {v1pb.MCPMethodClass_FORBIDDEN, "it can let a change or an access request pass its approval without an approver deciding it"},

	// The other refused class. An EXCLUDED method is out of scope for the modes
	// this release ships, and an admin-capable ceiling could legitimately serve
	// it one day; a FORBIDDEN method breaks the MCP boundary, so no ceiling
	// serves it. A denial that blurred the two would tell an operator the wrong
	// thing about whether asking is worth it. The class on each row is what
	// keeps them apart.
	v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE:   {v1pb.MCPMethodClass_EXCLUDED, "it administers the workspace rather than doing database work, and no MCP mode this release ships covers workspace administration"},
	v1pb.MCPDenialReason_READS_OTHER_USERS_SQL:       {v1pb.MCPMethodClass_EXCLUDED, "it returns SQL that other people wrote, across the workspace or past the sharing that keeps a saved query private"},
	v1pb.MCPDenialReason_OPENS_AN_ADMIN_CONNECTION:   {v1pb.MCPMethodClass_EXCLUDED, "it opens an admin-credentialed connection to the database and returns other sessions' live, unmasked SQL"},
	v1pb.MCPDenialReason_SENDS_DATA_TO_A_THIRD_PARTY: {v1pb.MCPMethodClass_EXCLUDED, "it spends a stored workspace credential to send whatever the caller passes to a third party"},
	v1pb.MCPDenialReason_REDRAWS_GOVERNANCE_BOUNDARY: {v1pb.MCPMethodClass_EXCLUDED, "it changes which rules govern a database or which databases a change that targets a database group runs on, and an agent does not choose the checks on its own changes"},
	// No method carries this one: the three leaks it was written for are
	// redacted on the read path and their eight methods are READ. The row
	// stays so the next read found leaking is one annotation away from a
	// denial that explains itself, rather than two edits away.
	v1pb.MCPDenialReason_RETURNS_A_STORED_SECRET: {v1pb.MCPMethodClass_EXCLUDED, "its response carries a stored secret that the product redacts everywhere else"},
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
// A refusal that turns on a fact the request does not carry, such as whether an
// issue exists, is taken in the handler that loads the fact and keyed on MCP
// origin; TestMCPRefusalsNameThemselvesToTheQueryTool checks each one.
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
			"%s is not available to MCP sessions: no streaming RPC is served on the MCP transport",
			conn.Spec().Procedure))
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
	case v1pb.MCPMethodClass_FORBIDDEN:
		return ctx, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is not available to MCP sessions because %s. %s",
			procedure, denialReason(authCtx.MCPMethodClass, authCtx.MCPDenialReason, reasonForbiddenClass), mcpConsoleWayOut))
	case v1pb.MCPMethodClass_EXCLUDED:
		return ctx, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is served by no MCP capability ceiling because %s. %s",
			procedure, denialReason(authCtx.MCPMethodClass, authCtx.MCPDenialReason, reasonExcludedClass), mcpConsoleWayOut))
	case v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE:
	default:
		return ctx, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s carries no MCP classification, so no MCP session may call it", procedure))
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
			"%s cannot be checked against the MCP capability ceiling: no workspace on the request", procedure))
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
	// nothing reads as an ordinary per-method denial naming the class — the
	// more useful answer at a per-method door than a workspace-level sentence.
	switch verdict := auth.ClassifyMCPCeiling(settings, err); verdict {
	case auth.MCPCeilingServes, auth.MCPCeilingDisabled:
	case auth.MCPCeilingUnavailable:
		slog.Warn("failed to resolve the MCP capability ceiling; refusing the request",
			slog.String("method", procedure), slog.String("workspace", workspaceID), log.BBError(err))
		return nil, false, connect.NewError(connect.CodeUnavailable, errors.Errorf(
			"%s is refused: %s", procedure, verdict.Refusal()))
	default:
		slog.Warn("the MCP capability ceiling refuses this request",
			slog.String("method", procedure), slog.String("workspace", workspaceID), log.BBError(err))
		return nil, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is refused: %s", procedure, verdict.Refusal()))
	}

	served, known := mcpServingClasses[settings.Capability]
	if !known {
		// Unreachable: a capability no mode serves is MCPCeilingUnserved, which
		// the switch above already refused. Kept because the serving table and
		// the classifier are two statements of one rule, and this is what a
		// caller gets if they ever part company.
		return nil, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is refused: %s", procedure, auth.MCPCeilingUnserved.Refusal()))
	}
	if slices.Contains(served, class) {
		return settings, false, nil
	}
	return nil, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"%s is a %v method and this workspace's MCP capability ceiling is %v, which serves %s. "+
			"Ask a workspace admin to raise the MCP ceiling in the workspace settings, "+
			"or perform this action signed in to the Bytebase console instead",
		procedure, class, settings.Capability, describeServedClasses(served)))
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

const mcpConsoleWayOut = "Perform this action signed in to the Bytebase console instead"

// mcpRequestShapeRefusals holds the refusals a per-method class cannot express,
// because the verdict depends on a request field or on what the MCP server
// attached to the request context. Each entry returns the reason and the way
// out, or an empty reason to allow. It lives in the gate so a refusal is
// audited and worded like a class refusal, before the handler runs.
var mcpRequestShapeRefusals = map[string]func(ctx context.Context, msg any) (reason, wayOut string){
	v1connect.IssueServiceCreateIssueProcedure:       refuseGrantIssueCreation,
	v1connect.IssueServiceApproveIssueProcedure:      refuseUnconfirmedIssueDecision,
	v1connect.IssueServiceRejectIssueProcedure:       refuseUnconfirmedIssueDecision,
	v1connect.SheetServiceCreateSheetProcedure:       inTheConsole(refuseMaskedWriteSheet),
	v1connect.SheetServiceBatchCreateSheetsProcedure: inTheConsole(refuseMaskedWriteSheetBatch),
	v1connect.ReleaseServiceCreateReleaseProcedure:   inTheConsole(refuseMaskedWriteRelease),
	v1connect.SQLServiceQueryProcedure:               inTheConsole(refuseMaskedWriteQuery),
	v1connect.SQLServiceExportProcedure:              inTheConsole(refuseMaskedWriteExport),

	v1connect.SavedQueryServiceCreateSavedQueryProcedure: inTheConsole(refuseMaskedWriteSavedQuery),
	v1connect.SavedQueryServiceUpdateSavedQueryProcedure: inTheConsole(refuseMaskedWriteSavedQueryUpdate),
}

func inTheConsole(refuse func(context.Context, any) string) func(context.Context, any) (string, string) {
	return func(ctx context.Context, msg any) (string, string) {
		return refuse(ctx, msg), mcpConsoleWayOut
	}
}

// refuseByRequestShape applies the table above. The context carries what the
// gate resolved and what the MCP server attached, so an entry can key on more
// than the request.
func refuseByRequestShape(ctx context.Context, procedure string, msg any) error {
	refuse, ok := mcpRequestShapeRefusals[procedure]
	if !ok {
		return nil
	}
	reason, wayOut := refuse(ctx, msg)
	if reason == "" {
		return nil
	}
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"%s is not available to MCP sessions for this request because %s. %s", procedure, reason, wayOut))
}

const mcpDecideIssueWayOut = "Use the decide_issue tool, which asks the user to confirm"

// refuseUnconfirmedIssueDecision serves an approval or a rejection only when
// the request context carries that decision as the user confirmed it in the MCP
// client, which decide_issue attaches after asking. call_api reaches the same
// procedures but cannot set a context value, so it is refused here.
func refuseUnconfirmedIssueDecision(ctx context.Context, msg any) (string, string) {
	var want auth.MCPConfirmedDecision
	switch request := msg.(type) {
	case *v1pb.ApproveIssueRequest:
		want = auth.MCPConfirmedDecision{Procedure: v1connect.IssueServiceApproveIssueProcedure, Issue: request.GetName()}
	case *v1pb.RejectIssueRequest:
		want = auth.MCPConfirmedDecision{Procedure: v1connect.IssueServiceRejectIssueProcedure, Issue: request.GetName()}
	default:
		return fmt.Sprintf("its request could not be read as an issue decision (%T)", msg), mcpDecideIssueWayOut
	}
	if confirmed, ok := auth.MCPConfirmedDecisionFromContext(ctx); ok && confirmed.Procedure == want.Procedure && confirmed.Issue == want.Issue {
		return "", ""
	}
	return "an issue is approved or rejected over MCP only after the user confirms that decision in the MCP client, " +
		"and this request carries no such confirmation", mcpDecideIssueWayOut
}

// refuseGrantIssueCreation applies grantIssueRefusal to CreateIssue. UpdateIssue's
// allow_missing creation is guarded in its handler (rejectMCPOriginatedGrantIssue).
func refuseGrantIssueCreation(_ context.Context, msg any) (string, string) {
	request, ok := msg.(*v1pb.CreateIssueRequest)
	if !ok {
		// The table is keyed by procedure, so the request type is fixed. A
		// mismatch is a wiring bug, and a wiring bug on a refusal path fails
		// closed.
		return fmt.Sprintf("its request could not be read as an issue (%T)", msg), mcpConsoleWayOut
	}
	return grantIssueRefusal(request.GetIssue().GetType())
}

// grantIssueRefusal is the issue-type allow-list both MCP creation guards share,
// so a type added later is refused until someone decides otherwise. An unset
// type is left to buildIssueMessage, which rejects it as invalid.
func grantIssueRefusal(issueType v1pb.Issue_Type) (reason, wayOut string) {
	switch issueType {
	case v1pb.Issue_DATABASE_CHANGE, v1pb.Issue_TYPE_UNSPECIFIED:
		return "", ""
	case v1pb.Issue_ACCESS_GRANT:
		reason = fmt.Sprintf("an ACCESS_GRANT issue is filed by %s, not created directly",
			v1connect.AccessGrantServiceCreateAccessGrantProcedure)
		return reason, "Request the access with that method instead"
	default:
		return fmt.Sprintf(
			"a %v issue completes on creation whenever the workspace approval rule produces no template, "+
				"which grants access with no human step", issueType), mcpConsoleWayOut
	}
}
