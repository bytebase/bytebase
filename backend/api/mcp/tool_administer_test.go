package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const administerUpdateSetting = "SettingService/UpdateSetting"

// approvalMock is the internal API behind the approval entry: the stored rules,
// the user's open issues, and the update, recorded with its confirmation.
type approvalMock struct {
	mu     sync.Mutex
	stored *v1pb.WorkspaceApprovalSetting
	// pendingIssues and checkingIssues are the issues the search returns for
	// alice's open change issues in each approval status.
	pendingIssues  []string
	checkingIssues []string
	// morePending, when set, makes the PENDING search answer with a next page.
	morePending bool
	// refuse, when set, is the Connect code the update answers with.
	refuse  string
	reads   int
	updates []approvalCall
}

type approvalCall struct {
	body      map[string]any
	confirmed *auth.MCPConfirmedDecision
}

func (m *approvalMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.URL.Path {
	case v1connect.SettingServiceGetSettingProcedure:
		m.reads++
		body, _ := protojson.Marshal(&v1pb.Setting{Name: approvalSettingName, Value: &v1pb.SettingValue{
			Value: &v1pb.SettingValue_WorkspaceApproval{WorkspaceApproval: m.stored},
		}})
		_, _ = w.Write(body)
	case v1connect.IssueServiceSearchIssuesProcedure:
		var request struct {
			Filter string `json:"filter"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		var names []string
		nextPageToken := ""
		if strings.Contains(request.Filter, `creator == "users/alice@example.com"`) &&
			strings.Contains(request.Filter, `status == "OPEN"`) && strings.Contains(request.Filter, `type == "DATABASE_CHANGE"`) {
			switch {
			case strings.Contains(request.Filter, `approval_status == "PENDING"`):
				names = m.pendingIssues
				if m.morePending {
					nextPageToken = "next"
				}
			case strings.Contains(request.Filter, `approval_status == "CHECKING"`):
				names = m.checkingIssues
			default:
			}
		}
		issues := []map[string]any{}
		for _, name := range names {
			issues = append(issues, map[string]any{"name": name})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"issues": issues, "nextPageToken": nextPageToken})
	case v1connect.SettingServiceUpdateSettingProcedure:
		call := approvalCall{}
		_ = json.NewDecoder(r.Body).Decode(&call.body)
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(r.Context()); ok {
			call.confirmed = &confirmed
		}
		m.updates = append(m.updates, call)
		if m.refuse != "" {
			w.WriteHeader(http.StatusPreconditionFailed)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": m.refuse, "message": "the approval rules changed"})
			return
		}
		_, _ = w.Write([]byte(`{"name":"settings/WORKSPACE_APPROVAL"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *approvalMock) calls() []approvalCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]approvalCall(nil), m.updates...)
}

func (m *approvalMock) setStored(rules ...*v1pb.WorkspaceApprovalSetting_Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stored = &v1pb.WorkspaceApprovalSetting{Rules: rules}
}

func connectToApprovalServer(t *testing.T, mock *approvalMock, opts *mcp.ClientOptions) (*Server, *mcp.ClientSession) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	return s, connectLegacy(t, ts.URL+mcpResourcePath, token, opts)
}

func fallbackRule() *v1pb.WorkspaceApprovalSetting_Rule {
	return &v1pb.WorkspaceApprovalSetting_Rule{
		Condition: &expr.Expr{Expression: "true"},
		Template:  &v1pb.ApprovalTemplate{Title: "Fallback Rule", Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
	}
}

func roleRequestRule(roles ...string) *v1pb.WorkspaceApprovalSetting_Rule {
	return &v1pb.WorkspaceApprovalSetting_Rule{
		Source:    v1pb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE,
		Condition: &expr.Expr{Expression: "true"},
		Template:  &v1pb.ApprovalTemplate{Title: "Role requests", Flow: &v1pb.ApprovalFlow{Roles: roles}},
	}
}

func changeRule(title, condition string, roles ...string) *v1pb.WorkspaceApprovalSetting_Rule {
	return &v1pb.WorkspaceApprovalSetting_Rule{
		Source:    v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
		Condition: &expr.Expr{Expression: condition},
		Template:  &v1pb.ApprovalTemplate{Title: title, Flow: &v1pb.ApprovalFlow{Roles: roles}},
	}
}

func approvalBody(t *testing.T, rules ...*v1pb.WorkspaceApprovalSetting_Rule) map[string]any {
	t.Helper()
	encoded, err := protojson.Marshal(&v1pb.UpdateSettingRequest{Setting: &v1pb.Setting{
		Name:  approvalSettingName,
		Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_WorkspaceApproval{WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{Rules: rules}}},
	}})
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(encoded, &body))
	return body
}

func callAdminister(t *testing.T, session *mcp.ClientSession, params *mcp.CallToolParams) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	params.Name = "administer_workspace"
	res, err := session.CallTool(ctx, params)
	require.NoError(t, err)
	return res
}

func administerParams(body map[string]any) *mcp.CallToolParams {
	return &mcp.CallToolParams{Arguments: updateArguments(body)}
}

func updateArguments(body map[string]any) map[string]any {
	return map[string]any{"operationId": administerUpdateSetting, "body": body}
}

func adminAnswer(choice string) *mcp.ElicitResult {
	return &mcp.ElicitResult{Action: "accept", Content: map[string]any{adminChoiceField: choice}}
}

// signedApprovalState signs the state the tool would have signed when it asked
// about body, as alice, at askedAt.
func signedApprovalState(t *testing.T, s *Server, stored *v1pb.WorkspaceApprovalSetting, body map[string]any, askedAt time.Time) string {
	t.Helper()
	change, err := parseApprovalRulesChange(body, "")
	require.NoError(t, err)
	diff, err := diffApprovalRules(stored.GetRules(), change.request.GetSetting().GetValue().GetWorkspaceApproval().GetRules(), "")
	require.NoError(t, err)
	alice := withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), "ws-test"), "alice@example.com"), "client-A")
	form, err := approvalRulesForm(alice, diff, "", "")
	require.NoError(t, err)
	return s.signConfirmationState(alice, change, approvalRulesDigest(form.message, stored, change.request), askedAt)
}

var (
	testRule = changeRule("Test", "resource.environment_id == 'test'")
	prodRule = changeRule("Prod", "resource.environment_id == 'prod'", "roles/workspaceDBA")
)

func TestAdministerWorkspaceAppliesConfirmedApprovalRules(t *testing.T) {
	mock := &approvalMock{pendingIssues: []string{"projects/hr/issues/12"}}
	mock.setStored(fallbackRule())
	user := &asker{answer: adminAnswer(adminChoiceApply)}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	body := approvalBody(t, fallbackRule(), testRule, prodRule)

	res := callAdminister(t, session, administerParams(body))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the approval rules for database changes, as alice@example.com?`,
		`Added as rule 1, no approval: "resource.environment_id == 'test'"`,
		`Added, "Workspace DBA": "resource.environment_id == 'prod'"`,
		`Your issues awaiting approval: "hr#12".`,
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n") {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"object","required":["choice"],"properties":{"choice":{"type":"string",
		"title":"Apply this change?",
		"description":"The rules for database changes, in the order an approval check tries them:\n1. \"Test\", no approval: \"resource.environment_id == 'test'\"\n2. \"Prod\", \"Workspace DBA\": \"resource.environment_id == 'prod'\"\nWhen none of them matches, the fallback rules decide, and a change that no rule matches needs no approval. Rules for other sources stay as they are.\nApproval checks from now on use these rules, also after a plan edit.",
		"oneOf":[{"const":"apply","title":"Apply"},{"const":"change","title":"Change it in chat"}],
		"default":"apply"}}}`, string(schema))

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, body, calls[0].body, "the update must be the request the user confirmed")
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure: v1connect.SettingServiceUpdateSettingProcedure,
		Resource:  approvalSettingName,
		Digest:    auth.MCPValueDigest(&v1pb.WorkspaceApprovalSetting{Rules: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}}),
	}, calls[0].confirmed, "the update must carry the confirmation and the rules the user saw")
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"operationId":"SettingService/UpdateSetting","changed":true,
		"message":"Applied: the approval rules now hold 2 rules for database changes. Approval checks from now on use these rules, also after a plan edit.\nApproval rules: https://bb.example.com/custom-approval",
		"link":"https://bb.example.com/custom-approval"}`, string(output))
}

func TestAdministerWorkspaceTakesTheFullOperationID(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, &mcp.CallToolParams{Arguments: map[string]any{
		"operationId": "bytebase.v1.SettingService.UpdateSetting",
		"body":        approvalBody(t, fallbackRule(), prodRule),
	}})
	require.Contains(t, resultText(t, res), "Not changed: the user declined")
	require.Equal(t, int32(1), user.asked.Load())
}

func TestAdministerWorkspaceChangesNothingUnlessApplied(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":        {&mcp.ElicitResult{Action: "decline"}, "Not changed: the user declined"},
		"cancelled":       {&mcp.ElicitResult{Action: "cancel"}, "Not changed: the user cancelled"},
		"changed in chat": {adminAnswer(adminChoiceChange), "Not changed: the user wants to change this request"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &approvalMock{}
			mock.setStored(fallbackRule())
			_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: tc.answer}).handle})
			res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
			require.Contains(t, resultText(t, res), tc.want)
			require.Nil(t, res.StructuredContent, "the text is the whole result")
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerWorkspaceChangesNothingForAnAnswerWithNoChoice(t *testing.T) {
	for name, answer := range map[string]*mcp.ElicitResult{
		"an unknown choice": adminAnswer("skip"),
		"no choice":         {Action: "accept", Content: map[string]any{}},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &approvalMock{}
			mock.setStored(fallbackRule())
			s, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
			body := approvalBody(t, fallbackRule(), prodRule)
			params := administerParams(body)
			params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: answer}
			params.RequestState = signedApprovalState(t, s, mock.stored, body, time.Now())
			res := callAdminister(t, session, params)
			require.Contains(t, resultText(t, res), "Not changed: the answer chose none of the form's choices")
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerWorkspaceRefusesAClientThatCannotConfirm(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	_, session := connectToApprovalServer(t, mock, nil)
	res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "CONFIRMATION_UNAVAILABLE")
	require.Contains(t, resultText(t, res), "https://bb.example.com/custom-approval")
	require.Nil(t, res.StructuredContent)
	require.Zero(t, mock.reads, "a client that cannot show the form is refused before anything is read")
	require.Empty(t, mock.calls())
}

func TestAdministerWorkspaceRefusesBeforeAsking(t *testing.T) {
	tooMany := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
	for _, environment := range []string{"a", "b", "c", "d"} {
		tooMany = append(tooMany, changeRule(environment, "resource.environment_id == '"+environment+"'"))
	}
	withField := approvalBody(t, fallbackRule(), prodRule)
	withField["validateOnly"] = true
	annotated := changeRule("Prod", "resource.environment_id == 'prod'", "roles/workspaceDBA")
	annotated.Condition.Title = "prod"
	annotatedRoleRule := roleRequestRule("roles/projectOwner")
	annotatedRoleRule.Condition.Title = "roles"
	editedRoleRule := roleRequestRule("roles/workspaceDBA")
	for name, tc := range map[string]struct {
		stored    []*v1pb.WorkspaceApprovalSetting_Rule
		arguments map[string]any
		want      []string
	}{
		"another operation": {arguments: map[string]any{"operationId": "SettingService/ListSettings", "body": map[string]any{}},
			want: []string{"UNSUPPORTED_CHANGE", "with the tool that serves them"}},
		"another setting": {arguments: map[string]any{"operationId": administerUpdateSetting, "body": map[string]any{"setting": map[string]any{"name": "settings/MCP"}}},
			want: []string{"UNSUPPORTED_CHANGE"}},
		"a field it cannot say": {arguments: updateArguments(withField),
			want: []string{"the request sets validate_only"}},
		"a condition annotation": {arguments: updateArguments(approvalBody(t, fallbackRule(), annotated)),
			want: []string{"the request sets setting.value.workspace_approval.rules.condition.title", "https://bb.example.com/custom-approval"}},
		"a stored condition annotation": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), annotated},
			arguments: updateArguments(approvalBody(t, fallbackRule(), prodRule, testRule)),
			want:      []string{"a stored approval rule sets setting.value.workspace_approval.rules.condition.title", "https://bb.example.com/custom-approval"}},
		"a stored annotation on another source's rule": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), annotatedRoleRule},
			arguments: updateArguments(approvalBody(t, fallbackRule(), roleRequestRule("roles/projectOwner"), prodRule)),
			want:      []string{"FIELD_NOT_SUPPORTED", "a stored approval rule sets", "https://bb.example.com/custom-approval"}},
		"the fallback removed": {arguments: updateArguments(approvalBody(t, prodRule)),
			want: []string{"OTHER_RULES_CHANGED", "the source CHANGE_DATABASE", "https://bb.example.com/custom-approval"}},
		"another source's rule edited in place": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), roleRequestRule("roles/projectOwner")},
			arguments: updateArguments(approvalBody(t, fallbackRule(), editedRoleRule, prodRule)),
			want:      []string{"OTHER_RULES_CHANGED"}},
		"two rules of other sources swapped": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), roleRequestRule("roles/projectOwner")},
			arguments: updateArguments(approvalBody(t, roleRequestRule("roles/projectOwner"), fallbackRule(), prodRule)),
			want:      []string{"OTHER_RULES_CHANGED"}},
		"too long for the form": {arguments: updateArguments(approvalBody(t, tooMany...)),
			want: []string{"CHANGE_TOO_LONG"}},
		"two requested rules, one condition": {arguments: updateArguments(approvalBody(t, fallbackRule(), prodRule, changeRule("Again", "resource.environment_id == 'prod'"))),
			want: []string{"DUPLICATE_CONDITION", "keep one rule per condition"}},
		"two stored rules, one condition": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), prodRule, changeRule("Again", "resource.environment_id == 'prod'")},
			arguments: updateArguments(approvalBody(t, fallbackRule(), prodRule)),
			want:      []string{"DUPLICATE_CONDITION", "two stored rules", "https://bb.example.com/custom-approval"}},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &approvalMock{}
			stored := tc.stored
			if stored == nil {
				stored = []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
			}
			mock.setStored(stored...)
			user := &asker{answer: adminAnswer(adminChoiceApply)}
			_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callAdminister(t, session, &mcp.CallToolParams{Arguments: tc.arguments})
			require.True(t, res.IsError, resultText(t, res))
			for _, want := range tc.want {
				require.Contains(t, resultText(t, res), want)
			}
			require.Nil(t, res.StructuredContent, "the text is the whole result")
			require.Zero(t, user.asked.Load())
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerWorkspaceRefusesATooLongChangeWithTheConsolePage(t *testing.T) {
	long := changeRule("Long", strings.Repeat("resource.project_id == 'p' || ", 9)+"true")
	for name, tc := range map[string]struct {
		rules   []*v1pb.WorkspaceApprovalSetting_Rule
		want    string
		notWant string
	}{
		"several short rules": {
			rules: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), changeRule("a", "resource.environment_id == 'a'"),
				changeRule("b", "resource.environment_id == 'b'"), changeRule("c", "resource.environment_id == 'c'"), changeRule("d", "resource.environment_id == 'd'")},
			want: "this change needs 5 lines, more than the 4 one confirmation form shows",
		},
		"one rule too long for any form": {
			rules:   []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), long},
			want:    "make that change signed in to the Bytebase console",
			notWant: "smaller steps",
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &approvalMock{}
			mock.setStored(fallbackRule())
			_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
			res := callAdminister(t, session, administerParams(approvalBody(t, tc.rules...)))
			require.Contains(t, resultText(t, res), tc.want)
			require.Contains(t, resultText(t, res), "https://bb.example.com/custom-approval")
			if tc.notWant != "" {
				require.NotContains(t, resultText(t, res), tc.notWant)
			}
		})
	}
}

func TestAdministerWorkspaceStatesMovedAndChangedRules(t *testing.T) {
	staging := changeRule("Staging", "resource.environment_id == 'staging'", "roles/projectOwner")
	mock := &approvalMock{}
	mock.setStored(fallbackRule(), prodRule, staging)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	stagingWithDBA := changeRule("Staging", "resource.environment_id == 'staging'", "roles/workspaceDBA")

	callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), stagingWithDBA, prodRule)))
	lines := strings.Split(user.lastParams(t).Message, "\n")
	require.Contains(t, user.lastParams(t).Message, `Changed, "Project owner" to "Workspace DBA":`)
	require.Contains(t, user.lastParams(t).Message, `Moved to rule 2: "resource.environment_id == 'prod'"`)
	require.LessOrEqual(t, len(lines), messageLines)
	for _, line := range lines {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
}

func TestAdministerWorkspaceStatesEachKindOfChange(t *testing.T) {
	prod := func(change func(*v1pb.WorkspaceApprovalSetting_Rule)) *v1pb.WorkspaceApprovalSetting_Rule {
		rule := changeRule("Prod", "resource.environment_id == 'prod'", "roles/workspaceDBA")
		if change != nil {
			change(rule)
		}
		return rule
	}
	described := prod(func(r *v1pb.WorkspaceApprovalSetting_Rule) { r.Template.Description = "Production" })
	dev := changeRule("Dev", "resource.environment_id != 'dev'")
	for name, tc := range map[string]struct {
		stored, requested []*v1pb.WorkspaceApprovalSetting_Rule
		want              []string
	}{
		"removed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: nil,
			want: []string{`Removed, "Workspace DBA": "resource.environment_id == 'prod'"`}},
		"renamed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(func(r *v1pb.WorkspaceApprovalSetting_Rule) { r.Template.Title = "Production" })},
			want:      []string{`Renamed to "Production": "resource.environment_id == 'prod'"`}},
		"description changed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{described},
			want: []string{`Description changed to "Production": "resource.environment_id == 'prod'"`}},
		"description removed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{described}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)},
			want: []string{`Description removed: "resource.environment_id == 'prod'"`}},
		"approvers and title changed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(func(r *v1pb.WorkspaceApprovalSetting_Rule) {
				r.Template.Title = "Production"
				r.Template.Flow.Roles = nil
			})},
			want: []string{`Changed, "Workspace DBA" to no approval; renamed to "Production": "resource.environment_id == 'prod'"`}},
		"added ahead of a kept rule": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{dev, prod(nil)},
			want: []string{`Added as rule 1, no approval: "resource.environment_id != 'dev'"`}},
		"added after the kept rules": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil), dev},
			want: []string{`Added, no approval: "resource.environment_id != 'dev'"`}},
	} {
		t.Run(name, func(t *testing.T) {
			diff, err := diffApprovalRules(append([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, tc.stored...),
				append([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, tc.requested...), "")
			require.NoError(t, err)
			require.Equal(t, tc.want, diff.changes)
		})
	}
}

// TestAdministerWorkspaceStatesEveryRuleFieldItTakes changes each rule field
// the approval entry takes, on a rule that is otherwise kept. Each change must
// get a line or be refused, so a field added to approvalRequestFields cannot
// reach the write unstated.
func TestAdministerWorkspaceStatesEveryRuleFieldItTakes(t *testing.T) {
	for path := range approvalRequestFields {
		field, ok := strings.CutPrefix(path, approvalRulesPath+".")
		if !ok {
			continue
		}
		t.Run(field, func(t *testing.T) {
			stored := changeRule("Prod", "resource.environment_id == 'prod'", "roles/workspaceDBA")
			changed := proto.CloneOf(stored)
			m := changed.ProtoReflect()
			names := strings.Split(field, ".")
			for _, name := range names[:len(names)-1] {
				m = m.Mutable(m.Descriptor().Fields().ByName(protoreflect.Name(name))).Message()
			}
			fd := m.Descriptor().Fields().ByName(protoreflect.Name(names[len(names)-1]))
			switch {
			case fd.Kind() == protoreflect.MessageKind:
				return
			case fd.IsList():
				m.Mutable(fd).List().Append(protoreflect.ValueOfString("roles/projectOwner"))
			case fd.Kind() == protoreflect.EnumKind:
				m.Set(fd, protoreflect.ValueOfEnum(v1pb.WorkspaceApprovalSetting_Rule_CREATE_DATABASE.Number()))
			default:
				m.Set(fd, protoreflect.ValueOfString(m.Get(fd).String()+" changed"))
			}
			diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), stored}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), changed}, "")
			if err == nil {
				require.NotEmpty(t, diff.changes, "a change to %s must get a line", field)
			}
		})
	}
}

func TestAdministerWorkspaceContinuesALongConditionOnTheNextLine(t *testing.T) {
	long := changeRule("HR prod", "resource.environment_id == 'prod' && resource.project_id == 'hr' && resource.database_name == 'hr_prod'", "roles/workspaceDBA")
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), long)))
	asked := user.lastParams(t)
	lines := strings.Split(asked.Message, "\n")
	require.Len(t, lines, 4)
	for _, line := range lines {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	require.Equal(t, `Added, "Workspace DBA": "resource.environment_id == 'prod' && `+
		`resource.project_id == 'hr' && resource.database_name == 'hr_prod'"`, lines[1]+" "+lines[2])
	require.Equal(t, approvalRulesApply, lines[3], "the fixed sentence goes whole in the line that is left")
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.NotContains(t, string(schema), approvalRulesApply)
}

func TestAdministerWorkspaceNamesTheUserIssuesAwaitingApproval(t *testing.T) {
	for name, tc := range map[string]struct {
		morePending bool
		want        string
	}{
		"every issue read": {want: `Your issues awaiting approval: "hr#12", "finance#7" and 2 more.`},
		"a search with another page": {morePending: true,
			want: `Your issues awaiting approval: "hr#12" and at least 3 more.`},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &approvalMock{
				pendingIssues:  []string{"projects/hr/issues/12"},
				checkingIssues: []string{"projects/finance/issues/7", "projects/marketing-analytics/issues/1234", "projects/ops/issues/5"},
				morePending:    tc.morePending,
			}
			mock.setStored(fallbackRule())
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

			callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
			require.Contains(t, user.lastParams(t).Message, tc.want)
		})
	}
}

func TestIssuesLineCountsNamesThatDoNotFit(t *testing.T) {
	long := "projects/" + strings.Repeat("p", 40) + "/issues/1"
	require.Equal(t, "You have 1 issue awaiting approval.", issuesLine([]string{shortIssueName(long)}, false))
	require.Equal(t, "You have 2 issues awaiting approval.", issuesLine([]string{shortIssueName(long), "hr#1"}, false))
	require.Equal(t, `Your issues awaiting approval: "hr#1".`, issuesLine([]string{"hr#1"}, false))
	require.Empty(t, issuesLine(nil, false))
	// A search that stopped at its page size gives a lower bound.
	require.Equal(t, "You have at least 2 issues awaiting approval.", issuesLine([]string{shortIssueName(long), "hr#1"}, true))
	require.Equal(t, `Your issues awaiting approval: "hr#1" and more.`, issuesLine([]string{"hr#1"}, true))
}

func TestAdministerWorkspaceRefusesAnAnswerToRulesThatChanged(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	user := &asker{answer: adminAnswer(adminChoiceApply)}
	// Another admin adds a rule while the form is open.
	user.whileAsked = func() { mock.setStored(fallbackRule(), testRule) }
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, res), rereadApprovalRules)
	require.Empty(t, mock.calls())
}

func TestAdministerWorkspaceRefusesAnAnswerToAnotherRequest(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	s, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	state := signedApprovalState(t, s, mock.stored, approvalBody(t, fallbackRule(), prodRule), time.Now())
	// The same message: an added rule's title is not in it.
	renamed := changeRule("Production", "resource.environment_id == 'prod'", "roles/workspaceDBA")

	params := administerParams(approvalBody(t, fallbackRule(), renamed))
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: adminAnswer(adminChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

func TestAdministerWorkspaceRefusesAnAnswerOverStoredRulesThatChanged(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule(), changeRule("Old", "resource.environment_id == 'x'"))
	s, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	body := approvalBody(t, fallbackRule())
	state := signedApprovalState(t, s, mock.stored, body, time.Now())
	// Another admin renames the rule the request removes. A Removed line names
	// no title, so the message stays the same.
	mock.setStored(fallbackRule(), changeRule("New", "resource.environment_id == 'x'"))

	params := administerParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: adminAnswer(adminChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

func TestAdministerWorkspaceDescribesTheRulesLeft(t *testing.T) {
	described := changeRule("Prod", "resource.environment_id == 'prod'", "roles/workspaceDBA")
	described.Template.Description = "Production"
	for name, tc := range map[string]struct {
		stored, requested []*v1pb.WorkspaceApprovalSetting_Rule
		want              string
	}{
		"an added rule's description": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), described},
			want:      `1. "Prod", "Workspace DBA": "resource.environment_id == 'prod'" (description "Production")`,
		},
		"no rule left": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), prodRule},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()},
			want:      "No rule for database changes is left, so the fallback rules decide every database change, and a change that none of them matches needs no approval.",
		},
	} {
		t.Run(name, func(t *testing.T) {
			diff, err := diffApprovalRules(tc.stored, tc.requested, "")
			require.NoError(t, err)
			form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "", "")
			require.NoError(t, err)
			require.Contains(t, form.description, tc.want)
		})
	}
}

func TestAdministerWorkspaceReportsTheLockedWriteRefusal(t *testing.T) {
	mock := &approvalMock{refuse: "failed_precondition"}
	mock.setStored(fallbackRule())
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: adminAnswer(adminChoiceApply)}).handle})

	res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, res), "the approval rules changed after the user was asked")
	require.Contains(t, resultText(t, res), rereadApprovalRules)
	require.Len(t, mock.calls(), 1)
}

func TestAdministerWorkspaceRefusesAnExpiredAnswer(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	s, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	body := approvalBody(t, fallbackRule(), prodRule)

	params := administerParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: adminAnswer(adminChoiceApply)}
	params.RequestState = signedApprovalState(t, s, mock.stored, body, time.Now().Add(-11*time.Minute))
	res := callAdminister(t, session, params)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "the user was asked more than 10 minutes ago")
	require.Empty(t, mock.calls())
}

func TestAdministerWorkspaceAsksNothingWhenNothingChanges(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule(), prodRule)
	user := &asker{}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "Nothing to change")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
}

// TestAdministerWorkspaceClassifiesEveryApprovalRequestField walks the request
// message the approval entry serves. A field the entry neither takes nor names
// here as refused fails it, so a new field stays refused until someone decides
// how its form states it.
func TestAdministerWorkspaceClassifiesEveryApprovalRequestField(t *testing.T) {
	refused := map[string]bool{
		"allow_missing": true,
		"validate_only": true,
		"update_mask":   true,
		"setting.value.workspace_approval.rules.condition.title":       true,
		"setting.value.workspace_approval.rules.condition.description": true,
		"setting.value.workspace_approval.rules.condition.location":    true,
	}
	var walk func(protoreflect.MessageDescriptor, string)
	walk = func(md protoreflect.MessageDescriptor, prefix string) {
		fields := md.Fields()
		for i := 0; i < fields.Len(); i++ {
			fd := fields.Get(i)
			path := string(fd.Name())
			if prefix != "" {
				path = prefix + "." + path
			}
			if oneof := fd.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() && !approvalRequestFields[path] {
				// The entry serves one member of a oneof; the others are other settings.
				continue
			}
			if refused[path] {
				continue
			}
			require.True(t, approvalRequestFields[path], "%s is neither taken nor refused by the approval entry", path)
			if fd.Kind() == protoreflect.MessageKind && !fd.IsMap() {
				walk(fd.Message(), path)
			}
		}
	}
	walk((&v1pb.UpdateSettingRequest{}).ProtoReflect().Descriptor(), "")
}
