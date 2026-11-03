package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
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

It serves one change: the approval rules for database changes. Send SettingService/UpdateSetting with setting.name "settings/WORKSPACE_APPROVAL" and the whole rule list in setting.value.workspaceApproval.rules. Read the stored rules first with call_api SettingService/GetSetting, change only the rules whose source is CHANGE_DATABASE, and send every other rule as stored, in its order. Give each rule you add the source CHANGE_DATABASE. Send a rule's source, condition.expression, template.title, template.description and template.flow.roles; other fields are refused. An approval check tries the CHANGE_DATABASE rules in order, and the first whose condition matches decides who approves a change. When none matches, the rules with no source, such as the Fallback Rule, decide, and a change that no rule matches needs no approval.

The tool shows the user what changes and asks them to confirm in a form. It changes the rules only when they accept; declining or cancelling changes nothing. A change too long for the form is refused, and the refusal says whether smaller steps help. A client that cannot show the form gets the console page instead. Do not ask in chat first: the form is the question.

Changing the approval rules needs bb.settings.set, as in the console. call_api refuses SettingService/UpdateSetting.`

func (s *Server) registerAdministerWorkspaceTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "administer_workspace",
		Description: administerWorkspaceDescription,
	}, s.handleAdministerWorkspace)
}

const (
	// The choice field of the form, and its values.
	adminChoiceField  = "choice"
	adminChoiceApply  = "apply"
	adminChoiceChange = "change"

	// approvalRulesApply is the fixed text of the approval form. It is one line,
	// so it goes in the message whenever a line is left.
	approvalRulesApply = "Approval checks from now on use these rules, also after a plan edit."

	// rereadApprovalRules is the next step after the stored rules changed: the
	// request was built from the old ones, so sending it again would undo the
	// other change.
	rereadApprovalRules = "read the rules again with call_api SettingService/GetSetting, make the user's change to them, and call administer_workspace with the result"
)

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
	if endpoint, ok := s.openAPIIndex.GetEndpoint(input.OperationID); !ok || endpoint.Path != v1connect.SettingServiceUpdateSettingProcedure {
		return formatToolError(&toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace does not change %q", input.OperationID),
			Suggestion: "it changes the approval rules for database changes through SettingService/UpdateSetting; make other changes with the tool that serves them, or signed in to the Bytebase console",
		}), nil, nil
	}
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
	openIssues, err := s.issuesAwaitingApproval(ctx)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	form, err := approvalRulesForm(ctx, diff, openIssues, link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	digest := approvalRulesDigest(form.message, stored, change.request)

	reply, answered := s.signedReply(ctx, req, change, digest)
	if !answered {
		return s.askWithForm(ctx, change, digest, form.message, map[string]any{
			"type": "object",
			"properties": map[string]any{
				adminChoiceField: map[string]any{
					"type":        "string",
					"title":       "Apply this change?",
					"description": form.description,
					"oneOf": []any{
						map[string]any{"const": adminChoiceApply, "title": "Apply"},
						map[string]any{"const": adminChoiceChange, "title": "Change it in chat"},
					},
					"default": adminChoiceApply,
				},
			},
			"required": []string{adminChoiceField},
		}), nil, nil
	}
	content, accepted, why := acceptedForm(reply.response)
	if !accepted {
		return textResult(fmt.Sprintf("Not changed: %s, so the approval rules stay as they are. Do not retry unless the user asks.", why)), nil, nil
	}
	switch content[adminChoiceField] {
	case adminChoiceApply:
	case adminChoiceChange:
		return textResult("Not changed: the user wants to change this request. Ask them what to change, then call administer_workspace with the revised rules."), nil, nil
	default:
		return textResult("Not changed: the answer chose none of the form's choices, so the approval rules stay as they are."), nil, nil
	}
	if why := reply.staleReason("the approval rules, the user's issues awaiting approval or this request no longer read as they did when the user was asked"); why != "" {
		return formatToolError(&toolError{
			Code:       "STALE_CONFIRMATION",
			Message:    why + ", so the approval rules were not changed",
			Suggestion: rereadApprovalRules,
		}), nil, nil
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
				Suggestion: rereadApprovalRules,
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
	// requested are the rules for database changes as the request orders them.
	requested []*v1pb.WorkspaceApprovalSetting_Rule
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
	var storedKept, requestedKept []string
	for _, rule := range storedDB {
		condition := rule.GetCondition().GetExpression()
		if _, kept := requestedByCondition[condition]; kept {
			storedKept = append(storedKept, condition)
			continue
		}
		diff.changes = append(diff.changes, fmt.Sprintf("Removed, %s: %q", approvers(rule), condition))
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
			// An added rule decides before every rule after it, so its place
			// matters unless it is last.
			if i < len(requestedDB)-1 {
				diff.changes = append(diff.changes, fmt.Sprintf("Added as rule %d, %s: %q", i+1, approvers(rule), condition))
			} else {
				diff.changes = append(diff.changes, fmt.Sprintf("Added, %s: %q", approvers(rule), condition))
			}
			continue
		}
		moved := !inOrder[kept]
		kept++
		if clauses := keptRuleClauses(before, rule, moved, i+1); len(clauses) > 0 {
			line := strings.Join(clauses, "; ")
			diff.changes = append(diff.changes, fmt.Sprintf("%s%s: %q", strings.ToUpper(line[:1]), line[1:], condition))
		}
	}
	return diff, nil
}

// keptRuleClauses states how a kept rule changes, one clause per field the
// form shows: the approvers, the place, the title and the description.
func keptRuleClauses(before, rule *v1pb.WorkspaceApprovalSetting_Rule, moved bool, position int) []string {
	var clauses []string
	if !proto.Equal(before.GetTemplate().GetFlow(), rule.GetTemplate().GetFlow()) {
		clauses = append(clauses, fmt.Sprintf("changed, %s to %s", approvers(before), approvers(rule)))
	}
	if moved {
		clauses = append(clauses, fmt.Sprintf("moved to rule %d", position))
	}
	if title := rule.GetTemplate().GetTitle(); title != before.GetTemplate().GetTitle() {
		clauses = append(clauses, "renamed to "+ruleTitle(rule))
	}
	if description := rule.GetTemplate().GetDescription(); description != before.GetTemplate().GetDescription() {
		if description == "" {
			clauses = append(clauses, "description removed")
		} else {
			clauses = append(clauses, fmt.Sprintf("description changed to %q", description))
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
func rulesByCondition(rules []*v1pb.WorkspaceApprovalSetting_Rule) (map[string]*v1pb.WorkspaceApprovalSetting_Rule, string) {
	byCondition := make(map[string]*v1pb.WorkspaceApprovalSetting_Rule, len(rules))
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

// approvers names a rule's approval flow as the console does: each step's role,
// in order, or "no approval" for an empty flow.
func approvers(rule *v1pb.WorkspaceApprovalSetting_Rule) string {
	roles := rule.GetTemplate().GetFlow().GetRoles()
	if len(roles) == 0 {
		return "no approval"
	}
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		roleID := strings.TrimPrefix(role, common.RolePrefix)
		if predefined := store.GetPredefinedRole(roleID); predefined != nil {
			roleID = predefined.Name
		}
		names = append(names, fmt.Sprintf("%q", roleID))
	}
	return strings.Join(names, " then ")
}

// adminForm is the message and the choice description of one confirmation.
type adminForm struct {
	message     string
	description string
}

// approvalRulesForm builds the approval form in the lines Claude Code shows.
// The message states the whole change: the question, a line or more per rule
// change, and the user's issues whose approval a plan edit would check again
// under the new rules. A change that does not fit is refused, never cut. The
// fixed sentence goes in the message when a line is left, and in the
// description otherwise; the description also lists every rule.
func approvalRulesForm(ctx context.Context, diff *approvalRulesDiff, openIssues []string, link string) (*adminForm, error) {
	question := "Change the approval rules for database changes"
	if email := getUserEmail(ctx); email != "" {
		question += ", as " + email
	}
	var ownWork []string
	if len(openIssues) > 0 {
		ownWork = wrapMessageLine(issuesLine(openIssues))
	}
	lines := []string{question + "?"}
	tooLong := ""
	for _, change := range diff.changes {
		changeLines := wrapMessageLine(change)
		if tooLong == "" && 1+len(changeLines)+len(ownWork) > messageLines {
			tooLong = change
		}
		lines = append(lines, changeLines...)
	}
	lines = append(lines, ownWork...)
	if len(lines) > messageLines {
		return nil, changeTooLong(len(lines), tooLong, link)
	}

	var sb strings.Builder
	if len(diff.requested) == 0 {
		sb.WriteString("No rule for database changes is left, so the fallback rules decide every database change, and a change that none of them matches needs no approval.")
	} else {
		sb.WriteString("The rules for database changes, in the order an approval check tries them:")
		for i, rule := range diff.requested {
			fmt.Fprintf(&sb, "\n%d. %s, %s: %q", i+1, ruleTitle(rule), approvers(rule), rule.GetCondition().GetExpression())
			if description := rule.GetTemplate().GetDescription(); description != "" {
				fmt.Fprintf(&sb, " (description %q)", description)
			}
		}
		sb.WriteString("\nWhen none of them matches, the fallback rules decide, and a change that no rule matches needs no approval.")
	}
	sb.WriteString(" Rules for other sources stay as they are.")
	if len(lines) < messageLines {
		lines = append(lines, approvalRulesApply)
	} else {
		sb.WriteString("\n" + approvalRulesApply)
	}
	return &adminForm{message: strings.Join(lines, "\n"), description: sb.String()}, nil
}

// changeTooLong refuses a change whose message needs more lines than a form
// shows. Smaller steps help only when every rule change fits a form alone;
// tooLong is the first one that does not.
func changeTooLong(needed int, tooLong, link string) *toolError {
	if tooLong != "" {
		return &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("the rule change %q needs more lines than the %d one confirmation form shows, so the approval rules were not changed", tooLong, messageLines),
			Suggestion: "make that change signed in to the Bytebase console: " + link,
		}
	}
	return &toolError{
		Code:       "CHANGE_TOO_LONG",
		Message:    fmt.Sprintf("this change needs %d lines, more than the %d one confirmation form shows, so the approval rules were not changed", needed, messageLines),
		Suggestion: "make the change in smaller steps, a rule or two at a time, or change the rules signed in to the Bytebase console: " + link,
	}
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

// issuesLine names the user's open issues, as many as one line holds, and
// counts the rest. It only counts them when the first name alone does not fit.
func issuesLine(issues []string) string {
	const lead = "Your issues awaiting approval: "
	line := lead
	for i, issue := range issues {
		name := fmt.Sprintf("%q", issue)
		candidate := line + name
		if i > 0 {
			candidate = line + ", " + name
		}
		suffix := "."
		if rest := len(issues) - i - 1; rest > 0 {
			suffix = fmt.Sprintf(" and %d more.", rest)
		}
		if lineColumns(candidate+suffix) > messageLineWidth {
			if i == 0 {
				if len(issues) == 1 {
					return "You have 1 issue awaiting approval."
				}
				return fmt.Sprintf("You have %d issues awaiting approval.", len(issues))
			}
			return line + fmt.Sprintf(" and %d more.", len(issues)-i)
		}
		line = candidate
	}
	return line + "."
}

// wrapMessageLine breaks line into lines of at most messageLineWidth columns,
// at a space where it can.
func wrapMessageLine(line string) []string {
	var lines []string
	for lineColumns(line) > messageLineWidth {
		runes := []rune(line)
		cut := min(messageLineWidth, len(runes))
		for lineColumns(string(runes[:cut])) > messageLineWidth {
			cut--
		}
		if space := strings.LastIndex(string(runes[:cut]), " "); space > 0 {
			cut = len([]rune(string(runes[:cut])[:space]))
			lines = append(lines, string(runes[:cut]))
			line = string(runes[cut+1:])
			continue
		}
		lines = append(lines, string(runes[:cut]))
		line = string(runes[cut:])
	}
	return append(lines, line)
}

// issuesAwaitingApproval lists the user's open change issues whose approval is
// pending or still being checked, as "project#number".
func (s *Server) issuesAwaitingApproval(ctx context.Context) ([]string, error) {
	email := getUserEmail(ctx)
	if email == "" {
		return nil, nil
	}
	var names []string
	for _, status := range []string{"PENDING", "CHECKING"} {
		filter := fmt.Sprintf(`creator == "%s%s" && status == "OPEN" && type == "DATABASE_CHANGE" && approval_status == "%s"`, common.UserNamePrefix, email, status)
		resp, err := s.apiRequest(ctx, v1connect.IssueServiceSearchIssuesProcedure, map[string]any{
			"parent":   "projects/-",
			"filter":   filter,
			"pageSize": 100,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to search the user's issues")
		}
		if err := checkAPIResponse(resp, "search issues", "bb.issues.get"); err != nil {
			return nil, err
		}
		var found struct {
			Issues []struct {
				Name string `json:"name"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(resp.Body, &found); err != nil {
			return nil, errors.Wrap(err, "failed to parse the user's issues")
		}
		for _, issue := range found.Issues {
			names = append(names, shortIssueName(issue.Name))
		}
	}
	return names, nil
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
