package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
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
// Message repeats the result's text, since some clients show the model only
// the structured result.
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

It serves one change: the approval rules for database changes. Send SettingService/UpdateSetting with setting.name "settings/WORKSPACE_APPROVAL" and the whole rule list in setting.value.workspaceApproval.rules. Read the stored rules first with call_api SettingService/GetSetting, change only the rules whose source is CHANGE_DATABASE, and send every other rule as stored, in its order. The first rule whose condition matches decides who approves a change.

The tool shows the user what changes and asks them to confirm in a form. It changes the rules only when they accept; declining or cancelling changes nothing. A change too long for the form is refused, so make it in smaller steps. A client that cannot show the form gets the console page instead. Do not ask in chat first: the form is the question.

Changing the approval rules needs bb.settings.set, as in the console. call_api refuses SettingService/UpdateSetting.`

func (s *Server) registerAdministerWorkspaceTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "administer_workspace",
		Description: administerWorkspaceDescription,
	}, s.handleAdministerWorkspace)
}

const (
	administerUpdateSetting = "SettingService/UpdateSetting"
	approvalSettingName     = common.SettingNamePrefix + "WORKSPACE_APPROVAL"

	// The choice field of the form, and its values.
	adminChoiceField  = "choice"
	adminChoiceApply  = "apply"
	adminChoiceChange = "change"

	// approvalRulesApply is the fixed text of the approval form: every rule
	// change says it, so it fills the lines the change leaves.
	approvalRulesApply = "Every approval check from now on uses these rules, including the check after a plan edit."
)

// approvalRulesChange is a request to replace the approval rules.
type approvalRulesChange struct {
	request *v1pb.UpdateSettingRequest
}

func (*approvalRulesChange) confirmedProcedure() string {
	return v1connect.SettingServiceUpdateSettingProcedure
}

func (*approvalRulesChange) confirmedResource() string { return approvalSettingName }

func (s *Server) handleAdministerWorkspace(ctx context.Context, req *mcp.CallToolRequest, input AdministerWorkspaceInput) (*mcp.CallToolResult, *AdministerWorkspaceOutput, error) {
	if input.OperationID != administerUpdateSetting {
		return formatToolError(&toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace does not change %q", input.OperationID),
			Suggestion: "it changes the approval rules for database changes through " + administerUpdateSetting + "; make other changes signed in to the Bytebase console",
		}), nil, nil
	}
	link := s.consoleLink(ctx, "custom-approval")
	change, err := parseApprovalRulesChange(input.Body)
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
	diff, err := diffApprovalRules(stored.GetRules(), change.request.GetSetting().GetValue().GetWorkspaceApproval().GetRules())
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
	form, err := approvalRulesForm(ctx, diff, openIssues)
	if err != nil {
		var te *toolError
		if errors.As(err, &te) {
			te.Suggestion += link
		}
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
	if why := reply.staleReason("the approval rules or this request no longer read as they did when the user was asked"); why != "" {
		return formatToolError(&toolError{
			Code:       "STALE_CONFIRMATION",
			Message:    why + ", so the approval rules were not changed",
			Suggestion: "call administer_workspace again to ask the user about the rules as they are now",
		}), nil, nil
	}

	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure:    change.confirmedProcedure(),
		Resource:     change.confirmedResource(),
		StoredDigest: auth.MCPValueDigest(stored),
	})
	body, err := protojson.Marshal(change.request)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the update")), nil, nil
	}
	var bodyMap map[string]any
	if err := json.Unmarshal(body, &bodyMap); err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the update")), nil, nil
	}
	resp, err := s.apiRequest(confirmedCtx, change.confirmedProcedure(), bodyMap)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to update the approval rules")), nil, nil
	}
	if resp.Status >= http.StatusBadRequest {
		if connectCodeOf(resp.Body) == connect.CodeFailedPrecondition {
			return formatToolError(&toolError{
				Code:       "STALE_CONFIRMATION",
				Message:    "the approval rules changed after the user was asked, so they were not changed",
				Suggestion: "call administer_workspace again to ask the user about the rules as they are now",
			}), nil, nil
		}
		return formatToolError(errors.Errorf("failed to update the approval rules: HTTP %d: %s", resp.Status, parseError(resp.Body))), nil, nil
	}
	text := fmt.Sprintf("Applied: there are now %d rules for database changes. %s\nApproval rules: %s", len(diff.requested), approvalRulesApply, link)
	return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: link}, nil
}

// approvalRequestFields are the UpdateSettingRequest fields the approval
// entry takes. A request that sets any other field is refused before the form,
// since the form could not state it.
var approvalRequestFields = map[string]bool{
	"setting":                                                      true,
	"setting.name":                                                 true,
	"setting.value":                                                true,
	"setting.value.workspace_approval":                             true,
	"setting.value.workspace_approval.rules":                       true,
	"setting.value.workspace_approval.rules.source":                true,
	"setting.value.workspace_approval.rules.condition":             true,
	"setting.value.workspace_approval.rules.condition.expression":  true,
	"setting.value.workspace_approval.rules.condition.title":       true,
	"setting.value.workspace_approval.rules.condition.description": true,
	"setting.value.workspace_approval.rules.condition.location":    true,
	"setting.value.workspace_approval.rules.template":              true,
	"setting.value.workspace_approval.rules.template.title":        true,
	"setting.value.workspace_approval.rules.template.description":  true,
	"setting.value.workspace_approval.rules.template.flow":         true,
	"setting.value.workspace_approval.rules.template.flow.roles":   true,
}

func parseApprovalRulesChange(body map[string]any) (*approvalRulesChange, error) {
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
			Suggestion: "send setting.name and the rule list only",
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
	if setting.GetValue().GetWorkspaceApproval() == nil {
		return &v1pb.WorkspaceApprovalSetting{}, nil
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

// approvalRuleChange is one line of the approval form: how one rule for
// database changes differs between the stored list and the request.
type approvalRuleChange struct {
	line string
}

// approvalRulesDiff is how a request changes the approval rules.
type approvalRulesDiff struct {
	changes []approvalRuleChange
	// requested are the rules for database changes as the request orders them.
	requested []*v1pb.WorkspaceApprovalSetting_Rule
}

// diffApprovalRules compares the requested rule list with the stored one. The
// rules of every other source must come back as stored, in their order, since
// they decide who approves the user's own access and role requests and the
// form does not show them. The rules for database changes are matched by their
// condition, the text an approval check evaluates.
func diffApprovalRules(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule) (*approvalRulesDiff, error) {
	storedOther, storedDB := splitApprovalRules(stored)
	requestedOther, requestedDB := splitApprovalRules(requested)
	if !sameRules(storedOther, requestedOther) {
		return nil, &toolError{
			Code:       "OTHER_RULES_CHANGED",
			Message:    "the request changes approval rules whose source is not CHANGE_DATABASE, and administer_workspace changes only the rules for database changes",
			Suggestion: "send every rule of another source as GetSetting returns it, in its order",
		}
	}
	storedByCondition, err := rulesByCondition(storedDB)
	if err != nil {
		return nil, err
	}
	requestedByCondition, err := rulesByCondition(requestedDB)
	if err != nil {
		return nil, err
	}

	diff := &approvalRulesDiff{requested: requestedDB}
	var storedKept, requestedKept []string
	for _, rule := range storedDB {
		condition := rule.GetCondition().GetExpression()
		if _, kept := requestedByCondition[condition]; kept {
			storedKept = append(storedKept, condition)
			continue
		}
		diff.changes = append(diff.changes, approvalRuleChange{line: fmt.Sprintf("Removed, %s: %q", approvers(rule), condition)})
	}
	for _, rule := range requestedDB {
		condition := rule.GetCondition().GetExpression()
		before, kept := storedByCondition[condition]
		if !kept {
			diff.changes = append(diff.changes, approvalRuleChange{line: fmt.Sprintf("Added, %s: %q", approvers(rule), condition)})
			continue
		}
		requestedKept = append(requestedKept, condition)
		switch {
		case !proto.Equal(before.GetTemplate().GetFlow(), rule.GetTemplate().GetFlow()):
			diff.changes = append(diff.changes, approvalRuleChange{line: fmt.Sprintf("Changed, %s to %s: %q", approvers(before), approvers(rule), condition)})
		case before.GetTemplate().GetTitle() != rule.GetTemplate().GetTitle():
			diff.changes = append(diff.changes, approvalRuleChange{line: fmt.Sprintf("Renamed to %q: %q", rule.GetTemplate().GetTitle(), condition)})
		case !proto.Equal(before, rule):
			diff.changes = append(diff.changes, approvalRuleChange{line: fmt.Sprintf("Changed its description: %q", condition)})
		default:
		}
	}
	// A rule outside the longest common order moved: the first matching rule
	// decides, so a move changes which rule decides some changes.
	inOrder := longestCommonOrder(storedKept, requestedKept)
	for i, condition := range requestedKept {
		if !inOrder[i] {
			position := 1 + indexOfCondition(requestedDB, condition)
			diff.changes = append(diff.changes, approvalRuleChange{line: fmt.Sprintf("Moved to rule %d of %d: %q", position, len(requestedDB), condition)})
		}
	}
	return diff, nil
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

func sameRules(a, b []*v1pb.WorkspaceApprovalSetting_Rule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func rulesByCondition(rules []*v1pb.WorkspaceApprovalSetting_Rule) (map[string]*v1pb.WorkspaceApprovalSetting_Rule, error) {
	byCondition := make(map[string]*v1pb.WorkspaceApprovalSetting_Rule, len(rules))
	for _, rule := range rules {
		condition := rule.GetCondition().GetExpression()
		if _, ok := byCondition[condition]; ok {
			return nil, &toolError{
				Code:       "DUPLICATE_CONDITION",
				Message:    fmt.Sprintf("two rules for database changes have the condition %q, so the form could not tell which one changes", condition),
				Suggestion: "keep one rule per condition, or change the rules signed in to the Bytebase console",
			}
		}
		byCondition[condition] = rule
	}
	return byCondition, nil
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

func indexOfCondition(rules []*v1pb.WorkspaceApprovalSetting_Rule, condition string) int {
	for i, rule := range rules {
		if rule.GetCondition().GetExpression() == condition {
			return i
		}
	}
	return -1
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

// approvalRulesForm builds the approval form. The message states the whole
// change: the question, a line or more per rule change, and the user's issues
// whose approval a plan edit would check again under the new rules. The fixed
// text fills the lines left, and the description holds the full list.
func approvalRulesForm(ctx context.Context, diff *approvalRulesDiff, openIssues []string) (*adminForm, error) {
	question := "Change the approval rules for database changes"
	var changeLines []string
	for _, change := range diff.changes {
		changeLines = append(changeLines, wrapMessageLine(change.line)...)
	}
	var ownWork []string
	if len(openIssues) > 0 {
		ownWork = wrapMessageLine(issuesLine(openIssues))
	}
	message, fixed, ok := fitAdminMessage(ctx, question, changeLines, ownWork, wrapMessageLine(approvalRulesApply))
	if !ok {
		return nil, &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("this change has %d rule changes, more than one confirmation form can show in full, so the approval rules were not changed", len(diff.changes)),
			Suggestion: "make the change in smaller steps, one or two rules at a time, or change the rules signed in to the Bytebase console: ",
		}
	}
	var sb strings.Builder
	sb.WriteString("The rules for database changes, in the order an approval check tries them:")
	for i, rule := range diff.requested {
		fmt.Fprintf(&sb, "\n%d. %s, %s: %q", i+1, ruleTitle(rule), approvers(rule), rule.GetCondition().GetExpression())
	}
	sb.WriteString("\nRules for other sources stay as they are.")
	if len(fixed) > 0 {
		sb.WriteString("\n" + strings.Join(fixed, " "))
	}
	return &adminForm{message: message, description: sb.String()}, nil
}

func ruleTitle(rule *v1pb.WorkspaceApprovalSetting_Rule) string {
	if title := rule.GetTemplate().GetTitle(); title != "" {
		return fmt.Sprintf("%q", title)
	}
	return "untitled"
}

// issuesLine names the user's open issues, as many as one line holds, and
// counts the rest.
func issuesLine(issues []string) string {
	const lead = "Your issues awaiting approval: "
	line := lead
	for i, issue := range issues {
		name := fmt.Sprintf("%q", issue)
		rest := len(issues) - i - 1
		candidate := line + name
		if i > 0 {
			candidate = line + ", " + name
		}
		suffix := "."
		if rest > 0 {
			suffix = fmt.Sprintf(" and %d more.", rest)
		}
		if lineColumns(candidate+suffix) > messageLineWidth && i > 0 {
			return line + fmt.Sprintf(" and %d more.", len(issues)-i)
		}
		line = candidate
	}
	return line + "."
}

// fitAdminMessage lays out a confirmation message in the lines Claude Code
// shows. The question may lose its end, the account's email, as the other
// confirmed tools' questions do, so the question without it must fit a line.
// The change and the user's own work are never cut. Fixed text fills the lines
// left, and the fixed lines that do not fit are returned for the description.
// ok is false when the change does not fit.
func fitAdminMessage(ctx context.Context, question string, change, ownWork, fixed []string) (string, []string, bool) {
	if lineColumns(question+", as ?") > messageLineWidth {
		return "", nil, false
	}
	first := question
	if email := getUserEmail(ctx); email != "" {
		first += ", as " + email
	}
	lines := append([]string{first + "?"}, change...)
	lines = append(lines, ownWork...)
	if len(lines) > messageLines {
		return "", nil, false
	}
	for len(fixed) > 0 && len(lines) < messageLines {
		lines = append(lines, fixed[0])
		fixed = fixed[1:]
	}
	return strings.Join(lines, "\n"), fixed, true
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
		if err := checkAPIResponse(resp, "search issues", "bb.issues.list"); err != nil {
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
	project, rest, ok := strings.Cut(strings.TrimPrefix(name, common.ProjectNamePrefix), "/"+common.IssueNamePrefix)
	if !ok {
		return name
	}
	return project + "#" + rest
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
