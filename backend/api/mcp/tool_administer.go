package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// AdministerWorkspaceInput is the input for the administer_workspace tool.
type AdministerWorkspaceInput struct {
	OperationID string         `json:"operationId"`
	Body        map[string]any `json:"body"`
}

// AdministerWorkspaceOutput is the output for the administer_workspace tool.
// Message repeats the result's text: Claude Code hands the model only the
// structured result when a result has one (BOT-165).
type AdministerWorkspaceOutput struct {
	OperationID string `json:"operationId"`
	Changed     bool   `json:"changed"`
	Message     string `json:"message"`
	Link        string `json:"link"`
}

const administerWorkspaceDescription = `Make a workspace administration change that the user confirms in this client first. It takes a Bytebase API operation and its request body, as search_api describes them.

| Parameter   | Required | Description |
|-------------|----------|-------------|
| operationId | Yes      | The operation, e.g. "SettingService/UpdateSetting" |
| body        | Yes      | The operation's request body |

It serves four changes.

The approval rules for database changes: send SettingService/UpdateSetting with setting.name "settings/WORKSPACE_APPROVAL" and the whole rule list in setting.value.workspaceApproval.rules. Read the stored rules first with call_api SettingService/GetSetting, change only the rules whose source is CHANGE_DATABASE, and send every other rule as stored, in its order. Give each rule you add the source CHANGE_DATABASE. Send a rule's source, condition.expression, template.title, template.description and template.flow.roles; other fields are refused. Name each approver in template.flow.roles by its role name, such as roles/workspaceDBA. An approval check tries the CHANGE_DATABASE rules in order, and the first whose condition matches decides who approves a change. When none matches, the rules with no source, such as the Fallback Rule, decide, and a change that no rule matches needs no approval.

The masking rules: send OrgPolicyService/UpdatePolicy with policy.name "workspaces/{workspace}/policies/masking_rule", policy.type MASKING_RULE and the whole rule list in policy.maskingRulePolicy.rules, with allowMissing true when no rules are stored yet. Read the stored rules first with call_api OrgPolicyService/GetPolicy, and the semantic types with call_api SettingService/GetSetting on settings/SEMANTIC_TYPES; the built-in types bb.default (full mask) and bb.default-partial (partial mask) always exist. Send a rule's id, condition.expression, condition.title and semanticType; other fields are refused, and the tool sets updateMask. Keep each stored rule's id, and give each new rule a new UUID. Masking tries the rules in order, and the first whose condition matches gives a column its semantic type; a column that no rule matches keeps its own.

A PostgreSQL instance in a project: send InstanceService/CreateInstance with parent "projects/{project}", instanceId, and instance with title, engine POSTGRES, environment ("environments/{id}", or left out for none), activation, and one dataSources entry: type ADMIN, host, port, username, password, and optionally database, useSsl, verifyTlsCertificate and authenticationType PASSWORD; the tool sets its id. Other fields are refused, keys that name no field are ignored, and the default project cannot own an instance. Masking skips an instance that is not activated unless every instance is licensed; the create refuses activation true when no license is free, as on the Free plan. When the user accepts, Bytebase tests the connection and adds the instance only if it connects; its databases join the project for good.

A workload identity for CI jobs in one GitHub or GitLab repository: send WorkloadIdentityService/CreateWorkloadIdentity in a project, with the fields search_api's note on it lists. When the user accepts, the tool creates it and grants it the GitOps Service Agent role in the project.

The tool shows the user what changes and asks them to confirm in a form. It makes the change only when they accept; declining or cancelling changes nothing. Do not ask in chat first: the form is the question.

Reading and changing the approval rules needs bb.settings.get and bb.settings.set, as in the console. Reading and changing the masking rules needs bb.policies.getMaskingRulePolicy and bb.policies.updateMaskingRulePolicy, bb.policies.createMaskingRulePolicy for the first save, and bb.settings.get for the semantic types. Adding an instance needs bb.instances.create, and a workload identity bb.workloadIdentities.create, bb.projects.getIamPolicy and bb.projects.setIamPolicy. call_api refuses each of these writes.`

func (s *Server) registerAdministerWorkspaceTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "administer_workspace",
		Description: administerWorkspaceDescription,
	}, s.handleAdministerWorkspace)
}

const (
	// approvalRulesApply is the fixed text of the approval form. It is one line,
	// so it goes in the message whenever a line is left.
	approvalRulesApply = "Approval checks from now on use these rules, also after a plan edit."
)

// adminRules names the rule list an entry changes, for the tool's own text.
type adminRules struct {
	// name is the list as the text names it, such as "the approval rules".
	name string
	// read is the call that reads the stored list.
	read string
	// question starts the form's question, which adds the user and a question
	// mark.
	question string
	// apply is the form's fixed sentence.
	apply string
	// replace is how a step replaces a rule. The added rule goes first, so the
	// rules never lapse between the steps.
	replace string
}

// steps tells the agent how to make a change in several forms of size rules
// each, such as "a few rules".
func (r adminRules) steps(size string) string {
	return "make the change in smaller steps, " + size + " at a time, and keep each rule's condition unchanged: a smaller step changes fewer rules, never a condition. Each step is a form of its own and applies when the user accepts it, so tell the user the change comes in steps, " +
		r.replace + ", and before each step read the rules again with " + r.read + " and build the step from them."
}

var approvalRules = adminRules{
	name:     "the approval rules",
	read:     "call_api SettingService/GetSetting",
	question: "Change the approval rules for database changes",
	apply:    approvalRulesApply,
	replace:  "add a rule before removing the one it replaces",
}

// adminSubject is what an entry changes, in the words confirmAdminChange's
// results use.
type adminSubject interface {
	// unchanged ends a result that changed nothing, such as "the approval
	// rules stay as they are".
	unchanged() string
	// notChanged ends the refusal of a stale answer, such as "the approval
	// rules were not changed".
	notChanged() string
	// revised names what the agent sends after the user chose to change the
	// request in chat.
	revised() string
	// reread is the next step after a stale answer.
	reread() string
	// action names the change on the form's first choice, such as "Save rules".
	action() string
}

func (r adminRules) unchanged() string { return r.name + " stay as they are" }

func (r adminRules) notChanged() string { return r.name + " were not changed" }

func (adminRules) revised() string { return "the revised rules" }

func (adminRules) action() string { return "Save rules" }

// creationSubject is a create in confirmAdminChange's words. nothing says that
// nothing was made, such as "no instance was added", and verb names the create
// on the form, such as "Add instance".
type creationSubject struct{ nothing, verb string }

func (c creationSubject) action() string { return c.verb }

func (c creationSubject) unchanged() string { return c.nothing }

func (c creationSubject) notChanged() string { return c.nothing }

func (creationSubject) revised() string { return "the revised request" }

func (creationSubject) reread() string {
	return "call administer_workspace again to ask the user about the request as it is now"
}

// reread is the next step after the stored rules changed: the request was
// built from the old ones, so sending it again would undo the other change.
func (r adminRules) reread() string {
	return "read the rules again with " + r.read + ", make the user's change to them, and call administer_workspace with the result"
}

var approvalSettingName = common.SettingNamePrefix + v1pb.Setting_WORKSPACE_APPROVAL.String()

// approvalRulesPath is where the rules sit in an UpdateSettingRequest, as
// approvalRequestFields names them.
const approvalRulesPath = "setting.value.workspace_approval.rules"

// approvalRulesChange is a request to replace the approval rules.
type approvalRulesChange struct {
	request *v1pb.UpdateSettingRequest
}

func (*approvalRulesChange) confirmedProcedure() string {
	return v1connect.SettingServiceUpdateSettingProcedure
}

func (*approvalRulesChange) confirmedResource() string { return approvalSettingName }

func (s *Server) handleAdministerWorkspace(ctx context.Context, req *mcp.CallToolRequest, input AdministerWorkspaceInput) (*mcp.CallToolResult, any, error) {
	if endpoint, ok := s.openAPIIndex.GetEndpoint(input.OperationID); ok {
		switch endpoint.Path {
		case v1connect.SettingServiceUpdateSettingProcedure:
			return s.administerApprovalRules(ctx, req, input)
		case v1connect.OrgPolicyServiceUpdatePolicyProcedure:
			return s.administerMaskingRules(ctx, req, input)
		case v1connect.InstanceServiceCreateInstanceProcedure:
			return s.administerInstance(ctx, req, input)
		case v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure:
			return s.administerIdentity(ctx, req, input)
		default:
		}
	}
	return formatToolError(&toolError{
		Code:       "UNSUPPORTED_CHANGE",
		Message:    fmt.Sprintf("administer_workspace does not change %q", input.OperationID),
		Suggestion: "it changes the approval rules for database changes through SettingService/UpdateSetting and the masking rules through OrgPolicyService/UpdatePolicy, adds a PostgreSQL instance to a project through InstanceService/CreateInstance, and creates a workload identity for CI in a project through WorkloadIdentityService/CreateWorkloadIdentity; make other changes with the tool that serves them, or signed in to the Bytebase console",
	}), nil, nil
}

func (s *Server) administerApprovalRules(ctx context.Context, req *mcp.CallToolRequest, input AdministerWorkspaceInput) (*mcp.CallToolResult, any, error) {
	link := s.consoleLink(ctx, "custom-approval")
	change, err := parseApprovalRulesChange(input.Body, link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if !canShowConfirmation(req) {
		return formatToolError(&toolError{
			Code:       "CONFIRMATION_UNAVAILABLE",
			Message:    "this MCP client cannot show the confirmation form administer_workspace needs, so the approval rules were not changed",
			Suggestion: "change them signed in to the Bytebase console: " + link,
		}), nil, nil
	}

	// Read on both rounds: the form describes the rules as stored now, and an
	// answer is acted on only while they read as they did when the user was asked.
	stored, err := s.readApprovalRules(ctx)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	diff, err := diffApprovalRules(stored.GetRules(), change.request.GetSetting().GetValue().GetWorkspaceApproval().GetRules(), link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if len(diff.changes) == 0 {
		text := "Nothing to change: the request holds the approval rules as they are stored.\nApproval rules: " + link
		return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Message: text, Link: link}, nil
	}
	openIssues, more, err := s.issuesAwaitingApproval(ctx)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	form, err := approvalRulesForm(ctx, diff, issuesLine(openIssues, more), link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	digest := approvalRulesDigest(form.message, stored, change.request)

	if result := s.confirmAdminChange(ctx, req, change, digest, form, approvalRules, "the approval rules, the user's issues awaiting approval or this request"); result != nil {
		return result, nil, nil
	}

	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: change.confirmedProcedure(),
		Resource:  change.confirmedResource(),
		Digest:    auth.MCPValueDigest(stored),
	})
	body, err := protojson.Marshal(change.request)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the update")), nil, nil
	}
	resp, err := s.apiRequest(confirmedCtx, change.confirmedProcedure(), json.RawMessage(body))
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to update the approval rules")), nil, nil
	}
	if resp.Status >= http.StatusBadRequest {
		if connectCodeOf(resp.Body) == connect.CodeFailedPrecondition {
			return formatToolError(&toolError{
				Code:       "STALE_CONFIRMATION",
				Message:    "the approval rules changed after the user was asked, so they were not changed",
				Suggestion: approvalRules.reread(),
			}), nil, nil
		}
		return formatToolError(errors.Errorf("failed to update the approval rules: HTTP %d: %s", resp.Status, parseError(resp.Body))), nil, nil
	}
	text := fmt.Sprintf("Applied: the approval rules now hold %s for database changes. %s\nApproval rules: %s", countRules(len(diff.requested)), approvalRulesApply, link)
	return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: link}, nil
}

// approvalRequestFields are the UpdateSettingRequest fields the approval
// entry takes. A request that sets any other field is refused before the form,
// since the form could not state it.
var approvalRequestFields = map[string]bool{
	"setting":                                                     true,
	"setting.name":                                                true,
	"setting.value":                                               true,
	"setting.value.workspace_approval":                            true,
	"setting.value.workspace_approval.rules":                      true,
	"setting.value.workspace_approval.rules.source":               true,
	"setting.value.workspace_approval.rules.condition":            true,
	"setting.value.workspace_approval.rules.condition.expression": true,
	"setting.value.workspace_approval.rules.template":             true,
	"setting.value.workspace_approval.rules.template.title":       true,
	"setting.value.workspace_approval.rules.template.description": true,
	"setting.value.workspace_approval.rules.template.flow":        true,
	"setting.value.workspace_approval.rules.template.flow.roles":  true,
}

func parseApprovalRulesChange(body map[string]any, link string) (*approvalRulesChange, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body could not be read: " + err.Error()}
	}
	request := &v1pb.UpdateSettingRequest{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(encoded, request); err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body is not an UpdateSettingRequest: " + err.Error()}
	}
	if name := request.GetSetting().GetName(); name != approvalSettingName {
		return nil, &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace does not change the setting %q", name),
			Suggestion: "it changes " + approvalSettingName + "; change other settings signed in to the Bytebase console",
		}
	}
	if request.GetSetting().GetValue().GetWorkspaceApproval() == nil {
		return nil, &toolError{
			Code:       "INVALID_REQUEST",
			Message:    "the request has no setting.value.workspaceApproval, so it holds no rule list",
			Suggestion: "send the whole rule list in setting.value.workspaceApproval.rules: the rules GetSetting returns, with the user's change made",
		}
	}
	if field := unlistedField(request.ProtoReflect(), "", approvalRequestFields); field != "" {
		return nil, &toolError{
			Code:       "FIELD_NOT_SUPPORTED",
			Message:    fmt.Sprintf("the request sets %s, which administer_workspace does not change", field),
			Suggestion: "send setting.name and each rule's source, condition.expression, template.title, template.description and template.flow.roles only, or change the rules signed in to the Bytebase console: " + link,
		}
	}
	return &approvalRulesChange{request: request}, nil
}

// unlistedField returns the path of the first field set in m that fields does
// not list, or "".
func unlistedField(m protoreflect.Message, prefix string, fields map[string]bool) string {
	var found string
	m.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		path := string(fd.Name())
		if prefix != "" {
			path = prefix + "." + path
		}
		if !fields[path] {
			found = path
			return false
		}
		if fd.Kind() != protoreflect.MessageKind {
			return true
		}
		switch {
		case fd.IsList():
			list := value.List()
			for i := 0; i < list.Len() && found == ""; i++ {
				found = unlistedField(list.Get(i).Message(), path, fields)
			}
		case fd.IsMap():
			found = path
		default:
			found = unlistedField(value.Message(), path, fields)
		}
		return found == ""
	})
	return found
}

// readApprovalRules reads the stored rules with the user's permissions, as the
// console shows them.
func (s *Server) readApprovalRules(ctx context.Context) (*v1pb.WorkspaceApprovalSetting, error) {
	resp, err := s.apiRequest(ctx, v1connect.SettingServiceGetSettingProcedure, map[string]any{"name": approvalSettingName})
	if err != nil {
		return nil, errors.Wrap(err, "failed to read the approval rules")
	}
	if err := checkAPIResponse(resp, "read the approval rules", "bb.settings.get"); err != nil {
		return nil, err
	}
	setting := &v1pb.Setting{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, setting); err != nil {
		return nil, errors.Wrap(err, "failed to parse the approval rules")
	}
	return setting.GetValue().GetWorkspaceApproval(), nil
}

// approvalRulesDigest binds an answer to the message the user read, the rules
// stored when they read it, and the request.
func approvalRulesDigest(message string, stored *v1pb.WorkspaceApprovalSetting, request *v1pb.UpdateSettingRequest) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(message))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(auth.MCPValueDigest(stored))
	_, _ = h.Write(auth.MCPValueDigest(request))
	return h.Sum(nil)
}

// approvalRulesDiff is how a request changes the approval rules.
type approvalRulesDiff struct {
	// changes are the form's change lines: the removed rules, then the
	// requested rules that differ, in the order an approval check tries them.
	changes []string
	// requested are the rules for database changes as the request orders them,
	// and marks say of each whether it is new, changed or kept as stored.
	requested []*v1pb.WorkspaceApprovalSetting_Rule
	marks     []string
	// fallback are the stored rules with no source, which decide a database
	// change no rule for database changes matches.
	fallback []*v1pb.WorkspaceApprovalSetting_Rule
}

// diffApprovalRules compares the requested rule list with the stored one. The
// rules of every other source must come back as stored, in their order, since
// they decide who approves the user's own access and role requests and the
// form does not show them. The rules for database changes are matched by their
// condition, the text an approval check evaluates.
func diffApprovalRules(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule, link string) (*approvalRulesDiff, error) {
	// The request cannot carry such a field, so the write would drop it unstated.
	for _, rule := range stored {
		if field := unlistedField(rule.ProtoReflect(), approvalRulesPath, approvalRequestFields); field != "" {
			return nil, &toolError{
				Code:       "FIELD_NOT_SUPPORTED",
				Message:    fmt.Sprintf("a stored approval rule sets %s, which the confirmation form cannot show and the write would drop, so the approval rules were not changed", field),
				Suggestion: "change the rules signed in to the Bytebase console: " + link,
			}
		}
	}
	storedOther, storedDB := splitApprovalRules(stored)
	requestedOther, requestedDB := splitApprovalRules(requested)
	if !slices.EqualFunc(storedOther, requestedOther, func(a, b *v1pb.WorkspaceApprovalSetting_Rule) bool { return proto.Equal(a, b) }) {
		return nil, &toolError{
			Code:       "OTHER_RULES_CHANGED",
			Message:    "the request changes approval rules whose source is not CHANGE_DATABASE, and administer_workspace changes only the rules for database changes",
			Suggestion: "give each rule for database changes the source CHANGE_DATABASE, and send every rule of another source as GetSetting returns it, in its order; or change the rules signed in to the Bytebase console: " + link,
		}
	}
	storedByCondition, duplicate := rulesByCondition(storedDB)
	if duplicate != "" {
		return nil, &toolError{
			Code:       "DUPLICATE_CONDITION",
			Message:    fmt.Sprintf("two stored rules for database changes have the condition %q, so the form could not tell which one changes", duplicate),
			Suggestion: "change the rules signed in to the Bytebase console: " + link,
		}
	}
	requestedByCondition, duplicate := rulesByCondition(requestedDB)
	if duplicate != "" {
		return nil, &toolError{
			Code:       "DUPLICATE_CONDITION",
			Message:    fmt.Sprintf("two requested rules for database changes have the condition %q, so the form could not tell which one changes", duplicate),
			Suggestion: "keep one rule per condition",
		}
	}

	diff := &approvalRulesDiff{requested: requestedDB}
	for _, rule := range storedOther {
		if rule.GetSource() == v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED {
			diff.fallback = append(diff.fallback, rule)
		}
	}
	var storedKept, requestedKept []string
	for _, rule := range storedDB {
		condition := rule.GetCondition().GetExpression()
		if _, kept := requestedByCondition[condition]; kept {
			storedKept = append(storedKept, condition)
			continue
		}
		diff.changes = append(diff.changes, fmt.Sprintf("Remove, %s: %s", approvers(rule), conditionText(condition)))
	}
	for _, rule := range requestedDB {
		if condition := rule.GetCondition().GetExpression(); storedByCondition[condition] != nil {
			requestedKept = append(requestedKept, condition)
		}
	}
	// A kept rule outside the longest common order moved: the first matching
	// rule decides, so a move changes which rule decides some changes.
	inOrder := longestCommonOrder(storedKept, requestedKept)
	kept := 0
	for i, rule := range requestedDB {
		condition := rule.GetCondition().GetExpression()
		before := storedByCondition[condition]
		if before == nil {
			if err := approverError(rule); err != nil {
				return nil, err
			}
			// The first matching rule decides, so an added rule's place is part
			// of the change.
			diff.changes = append(diff.changes, fmt.Sprintf("Add rule %d, %s: %s", i+1, approvers(rule), conditionText(condition)))
			diff.marks = append(diff.marks, "new")
			continue
		}
		if !proto.Equal(before.GetTemplate().GetFlow(), rule.GetTemplate().GetFlow()) {
			if err := approverError(rule); err != nil {
				return nil, err
			}
		}
		moved := !inOrder[kept]
		kept++
		if clauses := keptRuleClauses(before, rule, moved, i+1); len(clauses) > 0 {
			line := strings.Join(clauses, "; ")
			diff.changes = append(diff.changes, fmt.Sprintf("%s%s: %s", strings.ToUpper(line[:1]), line[1:], conditionText(condition)))
			diff.marks = append(diff.marks, "changed")
		} else {
			diff.marks = append(diff.marks, "kept")
		}
	}
	return diff, nil
}

// keptRuleClauses states how a kept rule changes, one clause per field the
// form shows: the approvers, the place, the title and the description.
func keptRuleClauses(before, rule *v1pb.WorkspaceApprovalSetting_Rule, moved bool, position int) []string {
	var clauses []string
	if !proto.Equal(before.GetTemplate().GetFlow(), rule.GetTemplate().GetFlow()) {
		clauses = append(clauses, fmt.Sprintf("change, %s to %s", approvers(before), approvers(rule)))
	}
	if moved {
		clauses = append(clauses, fmt.Sprintf("move to rule %d", position))
	}
	if title := rule.GetTemplate().GetTitle(); title != before.GetTemplate().GetTitle() {
		clauses = append(clauses, "rename to "+ruleTitle(rule))
	}
	if description := rule.GetTemplate().GetDescription(); description != before.GetTemplate().GetDescription() {
		if description == "" {
			clauses = append(clauses, "remove description")
		} else {
			clauses = append(clauses, fmt.Sprintf("change description to %q", description))
		}
	}
	return clauses
}

func splitApprovalRules(rules []*v1pb.WorkspaceApprovalSetting_Rule) (other, database []*v1pb.WorkspaceApprovalSetting_Rule) {
	for _, rule := range rules {
		if rule.GetSource() == v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE {
			database = append(database, rule)
		} else {
			other = append(other, rule)
		}
	}
	return other, database
}

// rulesByCondition maps rules by their condition. It returns the first
// condition two rules share, if any.
func rulesByCondition[R interface{ GetCondition() *expr.Expr }](rules []R) (map[string]R, string) {
	byCondition := make(map[string]R, len(rules))
	for _, rule := range rules {
		condition := rule.GetCondition().GetExpression()
		if _, ok := byCondition[condition]; ok {
			return nil, condition
		}
		byCondition[condition] = rule
	}
	return byCondition, ""
}

// longestCommonOrder marks the elements of b that stay in the order a has
// them, by the longest common subsequence; the rest moved.
func longestCommonOrder(a, b []string) []bool {
	lengths := make([][]int, len(a)+1)
	for i := range lengths {
		lengths[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lengths[i][j] = lengths[i+1][j+1] + 1
			} else {
				lengths[i][j] = max(lengths[i+1][j], lengths[i][j+1])
			}
		}
	}
	kept := make([]bool, len(b))
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch {
		case a[i] == b[j]:
			kept[j] = true
			i++
			j++
		case lengths[i+1][j] >= lengths[i][j+1]:
			i++
		default:
			j++
		}
	}
	return kept
}

// approvers names a rule's approval flow: each step's role in order, a
// predefined role by its display name and any other value as written, or "no
// approval" for an empty flow.
func approvers(rule *v1pb.WorkspaceApprovalSetting_Rule) string {
	roles := rule.GetTemplate().GetFlow().GetRoles()
	if len(roles) == 0 {
		return "no approval"
	}
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, fmt.Sprintf("%q", roleTitle(role)))
	}
	return strings.Join(names, " then ")
}

// roleTitle names a predefined role by its display name, as the console does,
// and any other value as written.
func roleTitle(role string) string {
	if roleID, err := common.GetRoleID(role); err == nil {
		if predefined := store.GetPredefinedRole(roleID); predefined != nil {
			return predefined.Name
		}
	}
	return role
}

// conditionText shows a condition between backquotes, as typed, when every
// character prints. strconv.CanBackquote also passes a tab and format
// characters such as U+202E and U+200B, which strconv.IsPrint rejects, so such
// a condition is escaped instead.
func conditionText(condition string) string {
	if strconv.CanBackquote(condition) && !strings.ContainsFunc(condition, func(r rune) bool { return !strconv.IsPrint(r) }) {
		return "`" + condition + "`"
	}
	return strconv.Quote(condition)
}

// approverError refuses a rule whose approvers include a value that is not a
// role name. An approval check matches a step's role by its exact name
// (review.Workflow.canReview), so no one could approve such a step.
func approverError(rule *v1pb.WorkspaceApprovalSetting_Rule) error {
	for _, role := range rule.GetTemplate().GetFlow().GetRoles() {
		if _, err := common.GetRoleID(role); err != nil {
			return &toolError{
				Code:       "INVALID_APPROVER",
				Message:    fmt.Sprintf("the rule for %q names the approver %q, which is not a role name, so no one could approve the changes it matches", rule.GetCondition().GetExpression(), role),
				Suggestion: "name each approver by its role name, such as roles/workspaceDBA or roles/projectOwner",
			}
		}
	}
	return nil
}

// adminForm is the message and the choice description of one confirmation.
type adminForm struct {
	message     string
	description string
	// inSteps is set when the form lists its changes under the choice: what
	// the agent may do when the user wants each one in the message.
	inSteps string
}

// approvalRulesForm builds the approval form. Its effect line names the user's
// issues whose approval a plan edit would check again under the new rules, so
// every step of the change shows it.
func approvalRulesForm(ctx context.Context, diff *approvalRulesDiff, issues string, link string) (*adminForm, error) {
	changes := make([]formChange, 0, len(diff.changes))
	for _, change := range diff.changes {
		changes = append(changes, formChange{text: change, effect: true})
	}
	// A count alone reads the same when approvers are replaced by none, so the
	// count line also says how many new or changed rules end with no approvers.
	unapproved := 0
	for i, rule := range diff.requested {
		if diff.marks[i] != "kept" && len(rule.GetTemplate().GetFlow().GetRoles()) == 0 {
			unapproved++
		}
	}
	line, fallback := fallbackLine(diff.fallback)
	list := ruleSection{before: []string{line}, name: "Rules for database changes"}
	switch {
	case len(diff.requested) == 0 && fallback:
		list.empty = "No rule for database changes is left, so the fallback rules decide."
	case len(diff.requested) == 0:
		list.empty = "No rule for database changes is left."
	default:
		list.header = "The rules for database changes, in the order an approval check tries them:"
		for i, rule := range diff.requested {
			// As on the masking form, every line names its approvers, and a
			// kept rule, which the message never shows, its condition.
			text := fmt.Sprintf("%d. %s (%s), %s", i+1, ruleTitle(rule), diff.marks[i], approvers(rule))
			if diff.marks[i] == "kept" {
				text += ": " + conditionText(rule.GetCondition().GetExpression())
			}
			if description := rule.GetTemplate().GetDescription(); description != "" {
				text += fmt.Sprintf(" (description %q)", description)
			}
			list.rules = append(list.rules, text)
		}
	}
	return approvalRules.form(ctx, changes, countLine(len(changes), diff.marks, unapproved), issues, list, link)
}

// fallbackRows bounds the fallback line. The stored fallback rules are not
// the change's, so the line must leave the change its rows whatever they hold.
const fallbackRows = 3

// fallbackLine names the fallback rules a database change goes to when no rule
// for database changes matches, and reports whether any can match. The
// approval check tries them in order, skips one with no condition and lets
// "true" match every change (review.matchRulesForSource); a change no
// fallback rule matches needs no approval. The line names as many rules as
// fallbackRows hold, from the first tried, and counts the rest.
func fallbackLine(rules []*v1pb.WorkspaceApprovalSetting_Rule) (string, bool) {
	var named []string
	catchAll := false
	for _, rule := range rules {
		condition := rule.GetCondition().GetExpression()
		if condition == "" {
			continue
		}
		text := fmt.Sprintf("%s (%s)", ruleTitle(rule), approvers(rule))
		if condition == "true" {
			catchAll = true
		} else {
			text += " when " + conditionText(condition)
		}
		named = append(named, text)
	}
	if len(named) == 0 {
		return "A change no rule matches needs no approval.", false
	}
	unmatched := ""
	if !catchAll {
		unmatched = " A change no fallback rule matches needs no approval."
	}
	for shown := len(named); shown > 0; shown-- {
		line := "Unmatched changes go to the fallback rules: " + strings.Join(named[:shown], ", ")
		if rest := len(named) - shown; rest > 0 {
			line += fmt.Sprintf(" and %d more", rest)
		}
		if line += "." + unmatched; len(wrapLine(line, descriptionLineWidth)) <= fallbackRows {
			return line, true
		}
	}
	if len(named) == 1 {
		return "Unmatched changes go to a fallback rule too long to name here." + unmatched, true
	}
	return fmt.Sprintf("Unmatched changes go to %d fallback rules too long to name here.", len(named)) + unmatched, true
}

// formChange is one change line of a form. effect marks a change that the
// form's effect is about, so a step that holds it shows the effect too.
type formChange struct {
	text   string
	effect bool
}

// ruleSection is what a rule form's description says of the rules a change
// leaves. Only before counts toward refusing a change, and its builder bounds
// it: empty shows on a removal alone, and a rule a form adds must not be
// refused its removal for it. So empty is one row, which the changes leave
// free unless they take every row.
type ruleSection struct {
	before []string
	// name starts the row that counts a list the description has no room
	// for. The row may follow a line about other rules, and the change lines
	// above it state some of the rules it counts.
	name   string
	header string
	rules  []string
	empty  string
}

// form builds a rule entry's form: the change lines in the message when they
// fit it, and counted there by count and listed under the choice otherwise.
func (r adminRules) form(ctx context.Context, changes []formChange, count, effect string, list ruleSection, link string) (*adminForm, error) {
	if form := r.whole(ctx, r.question, changes, effect, list); form != nil {
		return form, nil
	}
	alone := func(change formChange) bool { return messageHolds(ctx, r.question, change, effect) }
	return r.listed(ctx, changes, count, effect, list, link, !slices.ContainsFunc(changes, func(change formChange) bool { return !alone(change) }))
}

// messageHolds reports whether the message of a form under question has room
// for change by itself.
func messageHolds(ctx context.Context, question string, change formChange, effect string) bool {
	return len(fitMessage(questionLines(ctx, question, ""), []formChange{change}, effect)) <= messageLines
}

// whole returns the form whose message states every change under question, or
// nil when that takes more lines than a form shows.
func (r adminRules) whole(ctx context.Context, question string, changes []formChange, effect string, list ruleSection) *adminForm {
	lines := fitMessage(questionLines(ctx, question, ""), changes, effect)
	if len(lines) > messageLines {
		return nil
	}
	lines, fixed := withFixed(lines, r.apply)
	// With no change to list, the description always has room: the lines
	// before the rule list are bounded (fallbackRows), and the list is cut.
	return &adminForm{message: strings.Join(lines, "\n"), description: describeRules(nil, list, fixed)}
}

// listed returns the form whose message counts the changes and whose
// description lists them first, or refuses a change it has no room for.
// eachAlone says that a message has room for each change by itself, so the
// agent may offer them in steps.
func (r adminRules) listed(ctx context.Context, changes []formChange, count, effect string, list ruleSection, link string, eachAlone bool) (*adminForm, error) {
	lines := append(questionLines(ctx, r.question, ""), wrapMessageLine(count)...)
	// Every step carries the lines before the list, so the changes have the
	// rows those lines leave, and one with no room beside them fits no form.
	room := descriptionLines
	for _, line := range list.before {
		room -= len(wrapLine(line, descriptionLineWidth))
	}
	var texts []string
	needed, tooLong := 0, ""
	affected := false
	for _, change := range changes {
		texts = append(texts, change.text)
		affected = affected || change.effect
		rows := len(wrapLine(change.text, descriptionLineWidth))
		needed += rows
		if tooLong == "" && rows > room {
			tooLong = change.text
		}
	}
	if needed > room {
		return nil, changeTooLong(needed, room, tooLong, r, link)
	}
	if affected && effect != "" {
		lines = append(lines, wrapMessageLine(effect)...)
	}
	lines, fixed := withFixed(lines, r.apply)
	form := &adminForm{message: strings.Join(lines, "\n"), description: describeRules(texts, list, fixed)}
	if eachAlone && len(changes) > 1 {
		form.inSteps = "The form listed the rule changes under its choice, where a client may show one line of them. If the user wants each change in the form's message, " + r.steps("a rule or two")
	}
	return form, nil
}

// withFixed puts the fixed sentence in the message when a line is left, and
// returns it for the description otherwise.
func withFixed(lines []string, fixed string) ([]string, string) {
	if fixed == "" || len(lines) >= messageLines {
		return lines, fixed
	}
	return append(lines, fixed), ""
}

// countLine counts a form's rule changes by kind, for a message that leaves
// them to the description. Each of the changes lines is for one rule, so those
// that marks do not call new or changed remove one. The line never says how
// much of the list a client shows: Claude Code's fullscreen renderer shows one
// row.
func countLine(changes int, marks []string, unapproved int) string {
	added, changed := 0, 0
	for _, mark := range marks {
		switch mark {
		case "new":
			added++
		case "changed":
			changed++
		default:
		}
	}
	var parts []string
	for _, kind := range []struct {
		verb string
		n    int
	}{{"remove", changes - added - changed}, {"add", added}, {"change", changed}} {
		switch {
		case kind.n == 0:
		case len(parts) == 0:
			parts = append(parts, kind.verb+" "+countRules(kind.n))
		default:
			parts = append(parts, fmt.Sprintf("%s %d", kind.verb, kind.n))
		}
	}
	line := parts[0]
	if len(parts) > 1 {
		line = strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
	line = capitalize(line) + ", listed under " + confirmChoiceTitle
	switch unapproved {
	case 0:
		return line + "."
	case 1:
		return line + "; 1 rule will need no approval."
	default:
		return line + fmt.Sprintf("; %d rules will need no approval.", unapproved)
	}
}

// describeRules lays out a rule form's description in descriptionLines rows:
// the changes the message only counts and the lines before the list, which the
// caller has found room for, then as much of the list as fits.
func describeRules(changes []string, list ruleSection, fixed string) string {
	var rows []string
	for _, line := range slices.Concat(changes, list.before) {
		rows = append(rows, wrapLine(line, descriptionLineWidth)...)
	}
	rows = append(rows, sectionRows(list, descriptionLines-len(rows))...)
	if wrapped := wrapLine(fixed, descriptionLineWidth); fixed != "" && len(rows)+len(wrapped) <= descriptionLines {
		rows = append(rows, wrapped...)
	}
	return strings.Join(rows, "\n")
}

// sectionRows returns the header and the rules that fit room rows, from the
// first tried, then a row that counts the rest and says where they sit: a rule
// left out can still be tried before one the change adds. When not even the
// first rule fits, one row counts them all.
func sectionRows(list ruleSection, room int) []string {
	if len(list.rules) == 0 {
		if rows := wrapLine(list.empty, descriptionLineWidth); list.empty != "" && len(rows) <= room {
			return rows
		}
		return nil
	}
	rows := wrapLine(list.header, descriptionLineWidth)
	for i, rule := range list.rules {
		wrapped := wrapLine(rule, descriptionLineWidth)
		left := len(list.rules) - i
		if len(rows)+len(wrapped)+min(left-1, 1) <= room {
			rows = append(rows, wrapped...)
			continue
		}
		switch {
		case i > 0 && left == 1:
			return append(rows, "... and 1 more rule, tried after these.")
		case i > 0:
			return append(rows, fmt.Sprintf("... and %d more rules, tried after these.", left))
		default:
			if count := wrapLine(fmt.Sprintf("%s in all: %d. Their list has no room here.", list.name, left), descriptionLineWidth); len(count) <= room {
				return count
			}
			return nil
		}
	}
	return rows
}

// adminFormDigest binds an answer to the form the user read and to the
// request. What an entry read for the form is bound through the form, which
// states it.
func adminFormDigest(form *adminForm, request proto.Message) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(form.message))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(form.description))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(auth.MCPValueDigest(request))
	return h.Sum(nil)
}

// questionLines wraps a form's question and ends it with rest, the user's
// account and a question mark. Every name in question shows whole: only what
// follows it may run past the width, where Claude Code cuts a row.
func questionLines(ctx context.Context, question, rest string) []string {
	rows := wrapMessageLine(question)
	last := len(rows) - 1
	rows[last] += rest
	if email := getUserEmail(ctx); email != "" {
		rows[last] += ", as " + email
	}
	rows[last] += "?"
	return rows
}

// formLayout is one way to state a request: the question, the lines after it,
// and the description, which leads with what those lines leave out.
type formLayout struct {
	question    string
	changes     []formChange
	description string
}

// fitLayout returns layout's form, or the lines its message needs when they
// are more than a form shows.
func fitLayout(ctx context.Context, layout formLayout) (*adminForm, int) {
	lines := fitMessage(questionLines(ctx, layout.question, ""), layout.changes, "")
	if len(lines) > messageLines {
		return nil, len(lines)
	}
	return &adminForm{message: strings.Join(lines, "\n"), description: layout.description}, 0
}

// fitMessage puts the question, each change line and the change's effect on
// the user into the lines of a message, continuing a long line on the next.
// The effect shows when a change has it.
func fitMessage(question []string, changes []formChange, effect string) []string {
	lines := slices.Clone(question)
	affected := false
	for _, change := range changes {
		lines = append(lines, wrapMessageLine(change.text)...)
		affected = affected || change.effect
	}
	if affected && effect != "" {
		lines = append(lines, wrapMessageLine(effect)...)
	}
	return lines
}

// changeTooLong refuses a change whose lines need more rows than the room a
// form has for them under its choice. Smaller steps help only when every rule
// change fits there alone; tooLong is the first one that does not.
func changeTooLong(needed, room int, tooLong string, rules adminRules, link string) *toolError {
	if tooLong != "" {
		return &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("the rule change %q needs more lines than the %d one confirmation form has room for under its choice, so %s were not changed", tooLong, room, rules.name),
			Suggestion: "do not shorten the rule's condition to make it fit; make that change signed in to the Bytebase console: " + link,
		}
	}
	return &toolError{
		Code:       "CHANGE_TOO_LONG",
		Message:    fmt.Sprintf("this change needs %d lines, more than the %d one confirmation form has room for under its choice, so %s were not changed", needed, room, rules.name),
		Suggestion: rules.steps("a few rules") + " Or change the rules signed in to the Bytebase console: " + link,
	}
}

// confirmAdminChange asks the user to confirm w, or reads their answer. It
// returns the result that ends the call, or nil when the user chose the action and
// what the form showed still reads as it did when they were asked.
func (s *Server) confirmAdminChange(ctx context.Context, req *mcp.CallToolRequest, w confirmableWrite, digest []byte, form *adminForm, subject adminSubject, shown string) *mcp.CallToolResult {
	reply, answered := s.signedReply(ctx, req, w, digest)
	if !answered {
		return s.askToConfirm(ctx, w, digest, form.message, confirmChoice{action: subject.action(), description: form.description})
	}
	switch choice, why := confirmationOutcome(reply.response); choice {
	case confirmChoiceApply:
	case confirmChoiceChange:
		text := fmt.Sprintf("Not changed: %s. Ask them what to change, then call administer_workspace with %s.", why, subject.revised())
		if form.inSteps != "" {
			text += " " + form.inSteps
		}
		return textResult(text)
	default:
		return textResult(fmt.Sprintf("Not changed: %s, so %s. Do not retry unless the user asks.", why, subject.unchanged()))
	}
	if why := reply.staleReason(shown + " no longer read as they did when the user was asked"); why != "" {
		return formatToolError(&toolError{
			Code:       "STALE_CONFIRMATION",
			Message:    why + ", so " + subject.notChanged(),
			Suggestion: subject.reread(),
		})
	}
	return nil
}

func ruleTitle(rule *v1pb.WorkspaceApprovalSetting_Rule) string {
	if title := rule.GetTemplate().GetTitle(); title != "" {
		return fmt.Sprintf("%q", title)
	}
	return "untitled"
}

func countRules(n int) string {
	if n == 1 {
		return "1 rule"
	}
	return fmt.Sprintf("%d rules", n)
}

// issuesLine names the user's open issues awaiting approval, as many as one
// line holds, and counts the rest. A plan edit finds an issue's approval again
// under the rules in force, so the line says so. more means a search stopped at
// its page size, so a count is only a lower bound, and "at least N" never fits
// the line that counts them all, which then drops the count.
func issuesLine(issues []string, more bool) string {
	if len(issues) == 0 {
		return ""
	}
	const recheck, underRules = "Plan edits recheck your ", " under these rules."
	rest := func(n int) string {
		switch {
		case more && n > 0:
			return fmt.Sprintf(" and at least %d more", n)
		case more:
			return " and more"
		case n > 0:
			return fmt.Sprintf(" and %d more", n)
		default:
			return ""
		}
	}
	line := recheck
	for i, issue := range issues {
		name := fmt.Sprintf("%q", issue)
		candidate := line + name
		if i > 0 {
			candidate = line + ", " + name
		}
		if lineColumns(candidate+rest(len(issues)-i-1)+underRules) > messageLineWidth {
			if i > 0 {
				return line + rest(len(issues)-i) + underRules
			}
			counted := fmt.Sprintf("%s%d issues awaiting approval%s", recheck, len(issues), underRules)
			if len(issues) == 1 {
				counted = recheck + "1 issue awaiting approval" + underRules
			}
			if more || lineColumns(counted) > messageLineWidth {
				return recheck + "issues awaiting approval" + underRules
			}
			return counted
		}
		line = candidate
	}
	return line + rest(0) + underRules
}

// continuedLine starts each continuation of a wrapped line. No first row
// starts with it, so no part of a long value can pose as a row of its own. The
// text goes on right after the mark with nothing left out, so a reader can
// tell a break inside a value (">rest") from one before a space ("> rest").
const continuedLine = ">"

// wrapMessageLine breaks line into the rows of a form's message.
func wrapMessageLine(line string) []string {
	return wrapLine(line, messageLineWidth)
}

// wrapLine breaks line into rows of at most width columns, each continuation
// starting with continuedLine. It never takes more rows than filling every row
// would, since a form's rows are few: a row ends before a space, or after a
// separator of a name or a path, only when that costs no row.
func wrapLine(line string, width int) []string {
	var rows []string
	runes := []rune(line)
	prefix := ""
	for lineColumns(prefix+string(runes)) > width {
		full := filledRunes(runes, width-lineColumns(prefix))
		rest := filledRows(runes[full:], width)
		cut := full
		switch space, separator := lastBreaks(runes[:full]); {
		case space > 0 && filledRows(runes[space:], width) <= rest:
			cut = space
		case full < len(runes) && runes[full] == ' ':
			// The row is full, and the space starts the next one.
		case separator > 0 && filledRows(runes[separator:], width) <= rest:
			cut = separator
		default:
		}
		rows = append(rows, prefix+string(runes[:cut]))
		runes = runes[cut:]
		prefix = continuedLine
	}
	return append(rows, prefix+string(runes))
}

// filledRunes returns how many of runes fit in columns, and at least one, so a
// row always takes some.
func filledRunes(runes []rune, columns int) int {
	n := min(columns, len(runes))
	for n > 1 && lineColumns(string(runes[:n])) > columns {
		n--
	}
	return max(n, min(1, len(runes)))
}

// filledRows returns the continuation rows runes take when each is filled.
func filledRows(runes []rune, width int) int {
	rows := 0
	for len(runes) > 0 {
		runes = runes[filledRunes(runes, width-lineColumns(continuedLine)):]
		rows++
	}
	return rows
}

// lastBreaks returns where a row of runes can end early: the index of its last
// space, and the index after its last '/', '.', '-' or '_'. Each is 0 when
// there is none past the row's first rune.
func lastBreaks(runes []rune) (space, separator int) {
	for i := len(runes) - 1; i > 0 && (space == 0 || separator == 0); i-- {
		switch {
		case runes[i] == ' ' && space == 0:
			space = i
		case separator == 0 && strings.ContainsRune("/.-_", runes[i-1]):
			separator = i
		default:
		}
	}
	return space, separator
}

// issuesAwaitingApproval lists the user's open change issues whose approval is
// pending or still being checked, as "project#number". more reports that a
// search had another page, which the tool does not read.
func (s *Server) issuesAwaitingApproval(ctx context.Context) ([]string, bool, error) {
	email := getUserEmail(ctx)
	if email == "" {
		return nil, false, nil
	}
	var names []string
	more := false
	for _, status := range []string{"PENDING", "CHECKING"} {
		filter := fmt.Sprintf(`creator == "%s%s" && status == "OPEN" && type == "DATABASE_CHANGE" && approval_status == "%s"`, common.UserNamePrefix, email, status)
		resp, err := s.apiRequest(ctx, v1connect.IssueServiceSearchIssuesProcedure, map[string]any{
			"parent":   "projects/-",
			"filter":   filter,
			"pageSize": 100,
		})
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to search the user's issues")
		}
		if err := checkAPIResponse(resp, "search issues", "bb.issues.get"); err != nil {
			return nil, false, err
		}
		var found struct {
			Issues []struct {
				Name string `json:"name"`
			} `json:"issues"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := json.Unmarshal(resp.Body, &found); err != nil {
			return nil, false, errors.Wrap(err, "failed to parse the user's issues")
		}
		for _, issue := range found.Issues {
			names = append(names, shortIssueName(issue.Name))
		}
		more = more || found.NextPageToken != ""
	}
	return names, more, nil
}

// shortIssueName turns projects/{project}/issues/{number} into project#number.
func shortIssueName(name string) string {
	project, number, err := common.GetProjectIDIssueUID(name)
	if err != nil {
		return name
	}
	return fmt.Sprintf("%s#%d", project, number)
}

// connectCodeOf reads the code of a Connect error response body.
func connectCodeOf(body []byte) connect.Code {
	var e struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		return connect.CodeUnknown
	}
	var code connect.Code
	if err := code.UnmarshalText([]byte(e.Code)); err != nil {
		return connect.CodeUnknown
	}
	return code
}
