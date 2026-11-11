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
			procedure, auth.MCPDenialSentence(authCtx.MCPMethodClass, authCtx.MCPDenialReason), mcpConsoleWayOut))
	case v1pb.MCPMethodClass_EXCLUDED:
		return ctx, true, connect.NewError(connect.CodePermissionDenied, errors.Errorf(
			"%s is served by no MCP capability ceiling because %s. %s",
			procedure, auth.MCPDenialSentence(authCtx.MCPMethodClass, authCtx.MCPDenialReason), mcpConsoleWayOut))
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
			"Ask a workspace admin to raise the MCP ceiling under %s, "+
			"or perform this action signed in to the Bytebase console instead",
		procedure, class, settings.Capability, describeServedClasses(served), auth.MCPCeilingLocation))
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
	v1connect.IssueServiceCreateIssueProcedure: refuseGrantIssueCreation,
	v1connect.IssueServiceApproveIssueProcedure: refuseUnconfirmed(v1connect.IssueServiceApproveIssueProcedure,
		approvedIssueName, mcpIssueDecisionUnconfirmed, mcpDecideIssueWayOut),
	v1connect.IssueServiceRejectIssueProcedure: refuseUnconfirmed(v1connect.IssueServiceRejectIssueProcedure,
		rejectedIssueName, mcpIssueDecisionUnconfirmed, mcpDecideIssueWayOut),
	v1connect.ProjectServiceUpdateProjectProcedure: refuseUnconfirmed(v1connect.ProjectServiceUpdateProjectProcedure,
		updatedProjectName, mcpProjectUpdateUnconfirmed, mcpProjectUpdateWayOut),
	v1connect.OrgPolicyServiceUpdatePolicyProcedure: refuseUnconfirmed(v1connect.OrgPolicyServiceUpdatePolicyProcedure,
		updatedPolicyName, mcpPolicyUpdateUnconfirmed, mcpPolicyUpdateWayOut),
	v1connect.SettingServiceUpdateSettingProcedure: refuseUnconfirmed(v1connect.SettingServiceUpdateSettingProcedure,
		updatedSettingName, mcpSettingUpdateUnconfirmed, mcpSettingUpdateWayOut),
	v1connect.InstanceServiceCreateInstanceProcedure: refuseUnconfirmed(v1connect.InstanceServiceCreateInstanceProcedure,
		createdInstanceName, mcpInstanceCreationUnconfirmed, mcpInstanceCreationWayOut),
	v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure: refuseUnconfirmed(v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure,
		createdWorkloadIdentityName, mcpIdentityCreationUnconfirmed, mcpIdentityCreationWayOut),
	v1connect.ProjectServiceSetIamPolicyProcedure: refuseUnconfirmed(v1connect.ProjectServiceSetIamPolicyProcedure,
		projectIamPolicyName, mcpRoleGrantUnconfirmed, mcpRoleGrantWayOut),
	v1connect.SheetServiceCreateSheetProcedure:       inTheConsole(refuseMaskedWriteSheet),
	v1connect.SheetServiceBatchCreateSheetsProcedure: inTheConsole(refuseMaskedWriteSheetBatch),
	v1connect.ReleaseServiceCreateReleaseProcedure:   inTheConsole(refuseMaskedWriteRelease),
	v1connect.SQLServiceQueryProcedure:               inTheConsole(refuseMaskedWriteQuery),
	v1connect.SQLServiceExportProcedure:              inTheConsole(refuseMaskedWriteExport),

	v1connect.SavedQueryServiceCreateSavedQueryProcedure: inTheConsole(refuseMaskedWriteSavedQuery),
	v1connect.SavedQueryServiceUpdateSavedQueryProcedure: inTheConsole(refuseMaskedWriteSavedQueryUpdate),

	v1connect.SettingServiceGetSettingProcedure: inTheConsole(refuseUnreadableSetting),
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

const (
	mcpDecideIssueWayOut        = "Use the decide_issue tool, which asks the user to confirm"
	mcpIssueDecisionUnconfirmed = "an issue is approved or rejected over MCP only after the user confirms that decision in the MCP client, " +
		"and this request carries no such confirmation"
	mcpProjectUpdateUnconfirmed = "a project setting changes over MCP only through set_just_in_time_access, after the user confirms the change in the MCP client, " +
		"and this request carries no such confirmation"
	mcpProjectUpdateWayOut = "To turn just-in-time access on or off, use the set_just_in_time_access tool, which asks the user to confirm. " +
		"Change other project settings signed in to the Bytebase console"
	mcpPolicyUpdateUnconfirmed = "a policy changes over MCP only through grant_masking_exemption, revoke_masking_exemption and administer_workspace, after the user confirms the change in the MCP client, " +
		"and this request carries no such confirmation"
	mcpPolicyUpdateWayOut = "To grant or revoke a masking exemption, use those tools, and to change the masking rules, use the administer_workspace tool; each asks the user to confirm. " +
		"Change other policies signed in to the Bytebase console"
	mcpSettingUpdateUnconfirmed = "a setting changes over MCP only through administer_workspace, after the user confirms the change in the MCP client, " +
		"and this request carries no such confirmation"
	mcpSettingUpdateWayOut = "To change the approval rules for database changes, use the administer_workspace tool, which asks the user to confirm. " +
		"Change other settings signed in to the Bytebase console"
	mcpInstanceCreationUnconfirmed = "an instance is added over MCP only through administer_workspace, after the user confirms it in the MCP client, " +
		"and this request carries no such confirmation"
	mcpInstanceCreationWayOut = "To add a PostgreSQL instance to a project, use the administer_workspace tool, which asks the user to confirm and tests the connection before it adds the instance. " +
		"Add other instances, or test a connection alone, signed in to the Bytebase console"
	mcpIdentityCreationUnconfirmed = "a workload identity is created over MCP only through administer_workspace, after the user confirms it in the MCP client, " +
		"and this request carries no such confirmation"
	mcpIdentityCreationWayOut = "To create a workload identity that GitHub Actions or GitLab CI jobs in one repository sign in as, use the administer_workspace tool, " +
		"which asks the user to confirm and grants the identity the GitOps Service Agent role in its project. Create other workload identities signed in to the Bytebase console"
	mcpRoleGrantUnconfirmed = "a project's IAM policy changes over MCP only when administer_workspace grants a workload identity it creates its role, " +
		"after the user confirms it in the MCP client, and this request carries no such confirmation"
	mcpRoleGrantWayOut = "To create a workload identity for CI with its role, use the administer_workspace tool. " +
		"Change other members, roles and conditions signed in to the Bytebase console"
)

// refuseUnconfirmed serves procedure only when the request context carries a
// confirmation naming that procedure and the resource the request changes. A
// confirmed MCP tool attaches one after asking the user. call_api reaches the
// same procedures but cannot set a context value, so it is refused here.
func refuseUnconfirmed(procedure string, resourceOf func(msg any) (string, bool), reason, wayOut string) func(context.Context, any) (string, string) {
	return func(ctx context.Context, msg any) (string, string) {
		resource, ok := resourceOf(msg)
		if !ok {
			return fmt.Sprintf("its request could not be read (%T)", msg), wayOut
		}
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(ctx); ok && confirmed.Procedure == procedure && confirmed.Resource == resource {
			return "", ""
		}
		return reason, wayOut
	}
}

func updatedSettingName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.UpdateSettingRequest)
	return request.GetSetting().GetName(), ok
}

func approvedIssueName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.ApproveIssueRequest)
	return request.GetName(), ok
}

func rejectedIssueName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.RejectIssueRequest)
	return request.GetName(), ok
}

func updatedProjectName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.UpdateProjectRequest)
	return request.GetProject().GetName(), ok
}

func updatedPolicyName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.UpdatePolicyRequest)
	return request.GetPolicy().GetName(), ok
}

// createdInstanceName names the instance a create adds, as its audit row does.
func createdInstanceName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.CreateInstanceRequest)
	if !ok {
		return "", false
	}
	return getRequestResource(request, v1connect.InstanceServiceCreateInstanceProcedure), true
}

// createdWorkloadIdentityName names the identity a create adds, not its parent,
// so a confirmation covers that one identity. A parent that names no project
// gives a workspace identity's name, and administer_workspace confirms project
// identities only.
func createdWorkloadIdentityName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.CreateWorkloadIdentityRequest)
	if !ok {
		return "", false
	}
	projectID, _ := common.GetProjectID(request.GetParent())
	return common.FormatWorkloadIdentityEmail(common.BuildWorkloadIdentityEmail(request.GetWorkloadIdentityId(), projectID)), true
}

func projectIamPolicyName(msg any) (string, bool) {
	request, ok := msg.(*v1pb.SetIamPolicyRequest)
	return request.GetResource(), ok
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
		return reason, "Request the access with the request_access_grant tool instead"
	case v1pb.Issue_ROLE_GRANT:
		reason = "a ROLE_GRANT issue grants a project role once approved, and MCP sessions do not file role requests, " +
			"whatever the approval rules say"
		return reason, "To run a read-only statement, request an access grant with the request_access_grant tool; " +
			"to get a role, request it signed in to the Bytebase console"
	default:
		return fmt.Sprintf("MCP sessions create DATABASE_CHANGE issues only, not %v", issueType), mcpConsoleWayOut
	}
}

// refuseUnreadableSetting serves GetSetting for the settings
// auth.MCPCanReadSetting names. It parses the name as the handler does, and
// checks no permission: the handler checks the caller's own for each setting.
func refuseUnreadableSetting(_ context.Context, msg any) string {
	request, ok := msg.(*v1pb.GetSettingRequest)
	if !ok {
		return fmt.Sprintf("its request could not be read as a setting read (%T)", msg)
	}
	if name, err := common.GetSettingName(request.GetName()); err == nil {
		if value, ok := v1pb.Setting_SettingName_value[name]; ok && auth.MCPCanReadSetting(v1pb.Setting_SettingName(value)) {
			return ""
		}
	}
	return fmt.Sprintf("%q is not one of the settings an MCP session reads (%s)",
		request.GetName(), strings.Join(auth.MCPReadableSettingNames(), ", "))
}
