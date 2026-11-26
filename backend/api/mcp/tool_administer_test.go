package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
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
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	body := approvalBody(t, fallbackRule(), testRule, prodRule)

	res := callAdminister(t, session, administerParams(body))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the approval rules for database changes, as alice@example.com?`,
		"Add rule 1, no approval: `resource.environment_id == 'test'`",
		"Add rule 2, \"Workspace DBA\": `resource.environment_id == 'prod'`",
		`Plan edits recheck your "hr#12" under these rules.`,
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n") {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	description, err := json.Marshal("Unmatched changes go to the fallback rules: \"Fallback Rule\"\n" +
		"> (\"Project owner\").\n" +
		"The rules for database changes, in the order an approval check\n" +
		"> tries them:\n" +
		"1. \"Test\" (new), no approval\n" +
		"2. \"Prod\" (new), \"Workspace DBA\"\n" +
		approvalRulesApply)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"object","required":["choice"],"properties":{"choice":{"type":"string",
		"title":"Action",
		"description":`+string(description)+`,
		"oneOf":[{"const":"apply","title":"Save rules"},{"const":"change","title":"Change it in chat"}],
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

// The rules as an agent sent them in Claude Code: prod first, and CEL in double
// quotes, which escaped quoting lengthens past a line.
func TestAdministerWorkspaceFitsTheTwoRulesAnAgentWrites(t *testing.T) {
	mock := &approvalMock{pendingIssues: []string{"projects/hr/issues/12"}}
	mock.setStored(fallbackRule())
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	prod := changeRule("Prod changes need DBA", `resource.environment_id == "prod"`, "roles/workspaceDBA")
	test := changeRule("Test changes need no approval", `resource.environment_id == "test"`)

	res := callAdminister(t, session, administerParams(approvalBody(t, prod, test, fallbackRule())))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the approval rules for database changes, as alice@example.com?`,
		"Add rule 1, \"Workspace DBA\": `resource.environment_id == \"prod\"`",
		"Add rule 2, no approval: `resource.environment_id == \"test\"`",
		`Plan edits recheck your "hr#12" under these rules.`,
	}, "\n"), asked.Message)
	require.Contains(t, choiceDescription(t, asked), "1. \"Prod changes need DBA\" (new), \"Workspace DBA\"\n2. \"Test changes need no approval\" (new), no approval\n")
	require.True(t, strings.HasSuffix(choiceDescription(t, asked), "\n"+approvalRulesApply), "a full message leaves the fixed sentence to the description")
	require.Len(t, mock.calls(), 1)
}

// choiceLabels returns the title of a form's choice field and its choices'
// labels.
func choiceLabels(t *testing.T, asked *mcp.ElicitParams) (string, []string) {
	t.Helper()
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	var form struct {
		Properties map[string]struct {
			Title string `json:"title"`
			OneOf []struct {
				Title string `json:"title"`
			} `json:"oneOf"`
		} `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(schema, &form))
	field := form.Properties[confirmChoiceField]
	var labels []string
	for _, choice := range field.OneOf {
		labels = append(labels, choice.Title)
	}
	return field.Title, labels
}

// choiceDescription returns the description of a form's choice field.
func choiceDescription(t *testing.T, asked *mcp.ElicitParams) string {
	t.Helper()
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	var form struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(schema, &form))
	return form.Properties[confirmChoiceField].Description
}

// Claude Code cuts the description of a tool that loads upfront at 4,096
// characters, and before v2.1.296 at 2,048 (code.claude.com/docs/en/mcp).
func TestAdministerWorkspaceDescriptionFitsClaudeCode(t *testing.T) {
	require.LessOrEqual(t, utf8.RuneCountInString(administerWorkspaceDescription), 4096)
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
		"declined":  {&mcp.ElicitResult{Action: "decline"}, "Not changed: the user declined"},
		"cancelled": {&mcp.ElicitResult{Action: "cancel"}, "Not changed: the user cancelled"},
		"changed in chat": {confirmAnswer(confirmChoiceChange),
			"Not changed: the user wants to change this request. Ask them what to change, then call administer_workspace with the revised rules."},
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
		"an unknown choice": confirmAnswer("skip"),
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
			require.Contains(t, resultText(t, res), "Not changed: the answer chose none of the form's choices, so the approval rules stay as they are. Do not retry unless the user asks.")
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
	for i := range descriptionLines + 1 {
		tooMany = append(tooMany, changeRule(fmt.Sprintf("e%d", i), fmt.Sprintf("resource.environment_id == 'e%d'", i)))
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
		"no rule list": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prodRule},
			arguments: updateArguments(map[string]any{"setting": map[string]any{"name": "settings/WORKSPACE_APPROVAL"}}),
			want:      []string{"INVALID_REQUEST", "setting.value.workspaceApproval"}},
		"an added approver that is not a role name": {
			arguments: updateArguments(approvalBody(t, fallbackRule(), changeRule("Prod", "resource.environment_id == 'prod'", "workspaceDBA"))),
			want:      []string{"INVALID_APPROVER", "which is not a role name", "roles/workspaceDBA"}},
		"an approver changed to a bare role ID": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), prodRule},
			arguments: updateArguments(approvalBody(t, fallbackRule(), changeRule("Prod", "resource.environment_id == 'prod'", "workspaceDBA"))),
			want:      []string{"INVALID_APPROVER"}},
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
			user := &asker{answer: confirmAnswer(confirmChoiceApply)}
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

func TestAdministerWorkspaceRefusesAChangeTooLongForTheDescription(t *testing.T) {
	long := changeRule("Long", strings.Repeat("resource.project_id == 'p' || ", 30)+"true")
	several := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
	for i := range descriptionLines + 1 {
		several = append(several, changeRule(fmt.Sprintf("e%d", i), fmt.Sprintf("resource.environment_id == 'e%d'", i)))
	}
	for name, tc := range map[string]struct {
		rules   []*v1pb.WorkspaceApprovalSetting_Rule
		want    []string
		notWant string
	}{
		"more rules than the description has rows": {
			rules: several,
			// The fallback line takes two of the ten rows.
			want: []string{"this change needs 11 lines, more than the 8 one confirmation form has room for under its choice",
				"make the change in smaller steps, a few rules at a time",
				"Each step is a form of its own and applies when the user accepts it", "tell the user the change comes in steps",
				"a smaller step changes fewer rules, never a condition",
				"add a rule before removing the one it replaces", "before each step read the rules again with call_api SettingService/GetSetting"},
		},
		"one rule too long for any form": {
			rules:   []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), long},
			want:    []string{"make that change signed in to the Bytebase console", "do not shorten the rule's condition to make it fit"},
			notWant: "smaller steps",
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &approvalMock{}
			mock.setStored(fallbackRule())
			user := &asker{answer: confirmAnswer(confirmChoiceApply)}
			_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callAdminister(t, session, administerParams(approvalBody(t, tc.rules...)))
			require.True(t, res.IsError, resultText(t, res))
			for _, want := range tc.want {
				require.Contains(t, resultText(t, res), want)
			}
			require.Contains(t, resultText(t, res), "https://bb.example.com/custom-approval")
			if tc.notWant != "" {
				require.NotContains(t, resultText(t, res), tc.notWant)
			}
			require.Zero(t, user.asked.Load())
			require.Empty(t, mock.calls())
		})
	}
}

// The count line names each kind of change, and says how many rules end with
// no approvers: a count alone reads the same when approvers are replaced by
// none.
func TestCountLineNamesEachKindOfChange(t *testing.T) {
	for want, tc := range map[string]struct {
		changes    int
		marks      []string
		unapproved int
	}{
		"Add 1 rule, listed under Action.":                                                           {1, []string{"new"}, 0},
		"Remove 1 rule, listed under Action.":                                                        {1, nil, 0},
		"Change 1 rule, listed under Action.":                                                        {1, []string{"kept", "changed"}, 0},
		"Add 4 rules, listed under Action.":                                                          {4, []string{"new", "new", "kept", "new", "new"}, 0},
		"Remove 1 rule and add 3, listed under Action.":                                              {4, []string{"new", "new", "new"}, 0},
		"Remove 2 rules, add 1 and change 1, listed under Action.":                                   {4, []string{"changed", "kept", "new"}, 0},
		"Change 1 rule, listed under Action; 1 rule will need no approval.":                          {1, []string{"changed"}, 1},
		"Add 2 rules and change 1, listed under Action; 2 rules will need no approval.":              {3, []string{"new", "new", "changed"}, 2},
		"Remove 12 rules, add 34 and change 5, listed under Action; 39 rules will need no approval.": {51, append(slices.Repeat([]string{"new"}, 34), slices.Repeat([]string{"changed"}, 5)...), 39},
	} {
		require.Equal(t, want, countLine(tc.changes, tc.marks, tc.unapproved))
		require.LessOrEqual(t, len(wrapMessageLine(want)), 2, "the count line leaves the question and the effect their lines")
	}
}

// Each rule a change removes, adds or changes gets one change line, which is
// what lets the count line read the removals off the marks.
func TestAdministerWorkspaceGivesEachChangedRuleOneLine(t *testing.T) {
	a := changeRule("A", "resource.environment_id == 'a'", "roles/projectOwner")
	b := changeRule("B", "resource.environment_id == 'b'", "roles/projectOwner")
	c := changeRule("C", "resource.environment_id == 'c'", "roles/projectOwner")
	renamed := changeRule("B2", "resource.environment_id == 'b'", "roles/workspaceDBA")
	renamed.Template.Description = "now with a description"
	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), a, b, c},
		[]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), renamed, changeRule("D", "resource.environment_id == 'd'"), a}, "")
	require.NoError(t, err)
	require.Equal(t, []string{"changed", "new", "changed"}, diff.marks)
	require.Len(t, diff.changes, 4, "one removal, and one line for each new or changed rule: %q", diff.changes)
	require.Equal(t, "Remove 1 rule, add 1 and change 2, listed under Action.", countLine(len(diff.changes), diff.marks, 0))
}

// The fixed sentence is the same on every form, so it takes a row only when
// one is free: in the message first, then under the choice, and nowhere when
// both are full.
func TestAdministerWorkspaceDropsTheFixedSentenceFromAFullForm(t *testing.T) {
	long := func(title, environment string, roles ...string) *v1pb.WorkspaceApprovalSetting_Rule {
		return changeRule(title, `resource.environment_id == "`+environment+`" && resource.project_id == "payments-platform"`, roles...)
	}
	stored := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), long("Prod", "prod", "roles/workspaceDBA"), long("Staging", "staging", "roles/projectOwner")}
	// One rule removed, one changed to no approval and two added: the count
	// line takes two rows, and with the user's issue the message is full.
	requested := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), long("Staging", "staging"), long("Test", "test"), long("Dev", "dev")}
	diff, err := diffApprovalRules(stored, requested, "")
	require.NoError(t, err)
	form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, issuesLine([]string{"hr#12"}, false), "")
	require.NoError(t, err)
	require.Equal(t, strings.Join([]string{
		`Change the approval rules for database changes, as alice@example.com?`,
		`Remove 1 rule, add 2 and change 1, listed under Action; 3 rules will`,
		`> need no approval.`,
		`Plan edits recheck your "hr#12" under these rules.`,
	}, "\n"), form.message)
	rows := strings.Split(form.description, "\n")
	require.Len(t, rows, descriptionLines, "%q", rows)
	require.NotContains(t, form.message+form.description, approvalRulesApply)
	require.Contains(t, unwrapMessage(form.description), `Unmatched changes go to the fallback rules: "Fallback Rule" ("Project owner").`, "the fallback line is never the one that gives way")
}

// The stored fallback rules are not the change's to shorten, so their line is
// bounded: a workspace with many of them still gets a form, with a
// description, and a change still has its rows.
func TestAdministerWorkspaceBoundsTheFallbackLine(t *testing.T) {
	var stored []*v1pb.WorkspaceApprovalSetting_Rule
	for i := range 6 {
		stored = append(stored, &v1pb.WorkspaceApprovalSetting_Rule{
			Condition: &expr.Expr{Expression: fmt.Sprintf(`resource.project_id == "p%d"`, i)},
			Template:  &v1pb.ApprovalTemplate{Title: fmt.Sprintf("Fallback %d", i), Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
		})
	}
	line, matches := fallbackLine(stored)
	require.True(t, matches)
	require.Len(t, wrapLine(line, descriptionLineWidth), fallbackRows)
	require.True(t, strings.HasPrefix(line, `Unmatched changes go to the fallback rules: "Fallback 0" ("Project owner") when `), line)
	require.True(t, strings.HasSuffix(line, "`resource.project_id == \"p0\"` and 5 more. A change no fallback rule matches needs no approval."), line)

	long := &v1pb.WorkspaceApprovalSetting_Rule{Condition: &expr.Expr{Expression: strings.Repeat(`resource.project_id == "p" || `, 12) + "false"},
		Template: &v1pb.ApprovalTemplate{Title: "Long", Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}}}
	line, _ = fallbackLine([]*v1pb.WorkspaceApprovalSetting_Rule{long, fallbackRule()})
	require.Equal(t, "Unmatched changes go to 2 fallback rules too long to name here.", line, "a first rule too long for the rows is counted with the rest")

	alice := withUserEmail(context.Background(), "alice@example.com")
	diff, err := diffApprovalRules(stored, append(slices.Clone(stored), prodRule), "")
	require.NoError(t, err)
	form, err := approvalRulesForm(alice, diff, "", "")
	require.NoError(t, err)
	require.NotContains(t, form.message, "listed under Action", "the message states the change")
	rows := strings.Split(form.description, "\n")
	require.LessOrEqual(t, len(rows), descriptionLines)
	require.Equal(t, strings.Split(strings.Join(wrapLine(line0(stored), descriptionLineWidth), "\n"), "\n"), rows[:fallbackRows], "the description starts with the bounded fallback line")
	require.Contains(t, form.description, `1. "Prod" (new), "Workspace DBA"`)
}

// fallbackLineBefore is the fallback line as it was first built: every rule
// named, then one fewer, until the line fit its rows.
func fallbackLineBefore(rules []*v1pb.WorkspaceApprovalSetting_Rule) (string, bool) {
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

// The line is built from its first rule up and stops once its rows are full,
// whatever the workspace stores. It names the same rules as trying every rule
// and then one fewer did.
func TestFallbackLineStopsOnceItsRowsAreFull(t *testing.T) {
	random := rand.New(rand.NewPCG(22234, 2))
	conditions := []string{"", "true", `resource.project_id == "hr"`, `resource.environment_id == "prod" && resource.project_id == "payments-platform"`}
	flows := [][]string{nil, {"roles/projectOwner"}, {"roles/projectOwner", "roles/workspaceDBA"}}
	counted, cut := 0, 0
	for range 2000 {
		var rules []*v1pb.WorkspaceApprovalSetting_Rule
		for i := range random.IntN(12) {
			condition := conditions[random.IntN(len(conditions))]
			if condition != "" && condition != "true" {
				condition += fmt.Sprintf(` && resource.database_name == "%s%d"`, strings.Repeat("d", random.IntN(60)), i)
			}
			rules = append(rules, &v1pb.WorkspaceApprovalSetting_Rule{
				Condition: &expr.Expr{Expression: condition},
				Template:  &v1pb.ApprovalTemplate{Title: strings.Repeat("t", random.IntN(30)), Flow: &v1pb.ApprovalFlow{Roles: flows[random.IntN(len(flows))]}},
			})
		}
		wantLine, wantMatches := fallbackLineBefore(rules)
		line, matches := fallbackLine(rules)
		require.Equal(t, wantLine, line)
		require.Equal(t, wantMatches, matches)
		switch {
		case strings.Contains(line, "too long to name"):
			counted++
		case strings.Contains(line, " more."):
			cut++
		default:
		}
	}
	require.Positive(t, counted)
	require.Greater(t, cut, 100)

	var stored []*v1pb.WorkspaceApprovalSetting_Rule
	for i := range 200 {
		stored = append(stored, &v1pb.WorkspaceApprovalSetting_Rule{
			Condition: &expr.Expr{Expression: fmt.Sprintf(`resource.project_id == "project-%03d"`, i)},
			Template:  &v1pb.ApprovalTemplate{Title: fmt.Sprintf("Fallback for project-%03d", i), Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
		})
	}
	var line string
	bytes := allocated(func() { line, _ = fallbackLine(stored) })
	require.Less(t, bytes, uint64(1<<20), "the line must not be formatted with every stored rule")
	require.True(t, strings.HasSuffix(line, " and 199 more. A change no fallback rule matches needs no approval."), line)
}

func line0(rules []*v1pb.WorkspaceApprovalSetting_Rule) string {
	line, _ := fallbackLine(rules)
	return line
}

// Every step carries the fallback line, so one rule with no room beside it
// fits no form: the advice is the console, never smaller steps.
func TestAdministerWorkspaceSendsARuleWithNoRoomBesideTheFallbackLineToTheConsole(t *testing.T) {
	rule := changeRule("Long", strings.Repeat(`resource.project_id == "p" || `, 18)+"false", "roles/workspaceDBA")
	change := "Add rule 1, \"Workspace DBA\": " + conditionText(rule.GetCondition().GetExpression())
	require.LessOrEqual(t, len(wrapLine(change, descriptionLineWidth)), descriptionLines, "the rule fits the rows alone")
	line, _ := fallbackLine([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()})
	require.Greater(t, len(wrapLine(change, descriptionLineWidth))+len(wrapLine(line, descriptionLineWidth)), descriptionLines, "and not beside the fallback line")

	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), rule}, "LINK")
	require.NoError(t, err)
	_, err = approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "", "LINK")
	var refused *toolError
	require.ErrorAs(t, err, &refused)
	require.Equal(t, "CHANGE_TOO_LONG", refused.Code)
	// The message counts the rows the fallback line leaves, which the rule
	// passes, and not the ten it fits alone.
	require.True(t, strings.HasPrefix(refused.Message, `the rule change "Add rule 1, \"Workspace DBA\": `), refused.Message)
	require.True(t, strings.HasSuffix(refused.Message, " needs more lines than the 8 one confirmation form has room for under its choice, so the approval rules were not changed"), refused.Message)
	require.Contains(t, refused.Suggestion, "make that change signed in to the Bytebase console: LINK")
	require.NotContains(t, refused.Suggestion, "smaller steps")
}

// When the rows left hold no rule, one row still says how many there are.
func TestSectionRowsCountsAListItCannotShow(t *testing.T) {
	list := ruleSection{name: "Rules for database changes", header: "The rules for database changes, in the order an approval check tries them:",
		rules: []string{`1. "A" (kept), "Project owner": ` + "`resource.environment_id == 'a'`", `2. "B" (kept), no approval: ` + "`resource.environment_id == 'b'`", `3. "C" (new), no approval`}}
	require.Nil(t, sectionRows(list, 0))
	for room := 1; room <= 3; room++ {
		require.Equal(t, []string{"Rules for database changes in all: 3. Their list has no room here."}, sectionRows(list, room), "room %d", room)
	}
	require.Equal(t, []string{
		"The rules for database changes, in the order an approval check",
		"> tries them:",
		`1. "A" (kept), "Project owner": ` + "`resource.environment_id == 'a'`",
		"... and 2 more rules, tried after these.",
	}, sectionRows(list, 4))
	require.Len(t, sectionRows(list, 5), 5)
	require.Equal(t, []string{"Masking rules in all: 1. Their list has no room here."}, sectionRows(ruleSection{name: "Masking rules", header: list.header, rules: list.rules[:1]}, 2))
}

// A change that removes every rule says so while one row is free: the
// sentence takes one row, here the last of the ten.
func TestAdministerWorkspaceSaysNoRuleIsLeftInTheLastRow(t *testing.T) {
	stored := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
	for i := range 7 {
		stored = append(stored, changeRule(fmt.Sprintf("e%d", i), fmt.Sprintf("resource.environment_id == 'e%d'", i), "roles/workspaceDBA"))
	}
	diff, err := diffApprovalRules(stored, stored[:1], "")
	require.NoError(t, err)
	form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "", "")
	require.NoError(t, err)
	require.Contains(t, form.message, "Remove 7 rules, listed under Action.")
	rows := strings.Split(form.description, "\n")
	require.Len(t, rows, descriptionLines)
	require.Equal(t, "No rule for database changes is left, so the fallback rules decide.", rows[descriptionLines-1])
}

// The row that counts a list follows the fallback line, so it names the rules
// it counts: the kept rule and the five the change lines add.
func TestAdministerWorkspaceNamesTheRulesAListHasNoRoomFor(t *testing.T) {
	stored := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), changeRule("Kept", "resource.environment_id == 'kept'", "roles/projectOwner")}
	requested := slices.Clone(stored)
	for i := range 5 {
		requested = append(requested, changeRule(fmt.Sprintf("e%d", i), fmt.Sprintf("resource.environment_id == 'e%d'", i), "roles/workspaceDBA"))
	}
	diff, err := diffApprovalRules(stored, requested, "")
	require.NoError(t, err)
	form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "", "")
	require.NoError(t, err)
	rows := strings.Split(form.description, "\n")
	require.Equal(t, []string{
		"Add rule 6, \"Workspace DBA\": `resource.environment_id == 'e4'`",
		`Unmatched changes go to the fallback rules: "Fallback Rule"`,
		`> ("Project owner").`,
		"Rules for database changes in all: 6. Their list has no room here.",
	}, rows[4:])
}

// A rule a form adds is never refused its removal: the removal's line is no
// longer than the add's, and the sentence only a removal shows gives way.
func TestAdministerWorkspaceRemovesEveryRuleOneFormAdds(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	issues := issuesLine([]string{"hr#12"}, false)
	asked := func(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule) bool {
		diff, err := diffApprovalRules(stored, requested, "")
		require.NoError(t, err)
		_, err = approvalRulesForm(alice, diff, issues, "")
		return err == nil
	}
	added, refused := 0, 0
	for n := 1; n <= 800; n += 3 {
		rule := changeRule("Long", `resource.environment_id == "prod" && resource.database_name == "`+strings.Repeat("d", n)+`"`, "roles/workspaceDBA")
		if !asked([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), rule}) {
			refused++
			continue
		}
		added++
		require.True(t, asked([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), rule}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}), "remove a rule of %d", n)
	}
	require.Positive(t, added)
	require.Positive(t, refused, "the sweep must pass the longest rule a form states")
}

// Approvers replaced by none keep the number of changes. The form that lists
// its changes under the choice still says so in the message.
func TestAdministerWorkspaceCountsARuleThatEndsWithNoApproval(t *testing.T) {
	long := func(title string, roles ...string) *v1pb.WorkspaceApprovalSetting_Rule {
		return changeRule(title, `resource.environment_id == "prod" && resource.project_id == "hr" && resource.database_name == "hr_prod" && resource.table_name == "salary"`, roles...)
	}
	staging := changeRule("Staging", `resource.environment_id == "staging"`, "roles/projectOwner")
	ask := func(t *testing.T, requested ...*v1pb.WorkspaceApprovalSetting_Rule) *mcp.ElicitParams {
		t.Helper()
		mock := &approvalMock{pendingIssues: []string{"projects/hr/issues/12"}}
		mock.setStored(fallbackRule(), long("HR salary", "roles/workspaceDBA"), staging)
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		callAdminister(t, session, administerParams(approvalBody(t, append([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, requested...)...)))
		return user.lastParams(t)
	}

	toOwner := ask(t, long("HR salary", "roles/projectOwner"), staging)
	require.Equal(t, strings.Join([]string{
		`Change the approval rules for database changes, as alice@example.com?`,
		`Change 1 rule, listed under Action.`,
		`Plan edits recheck your "hr#12" under these rules.`,
		approvalRulesApply,
	}, "\n"), toOwner.Message)

	toNone := ask(t, long("HR salary"), staging)
	require.Equal(t, `Change 1 rule, listed under Action; 1 rule will need no approval.`, strings.Split(toNone.Message, "\n")[1])
	description := strings.Split(unwrapMessage(choiceDescription(t, toNone)), "\n")
	require.Equal(t, "Change, \"Workspace DBA\" to no approval: `resource.environment_id == \"prod\" && resource.project_id == \"hr\" && resource.database_name == \"hr_prod\" && resource.table_name == \"salary\"`", description[0],
		"the change leads the description, where fullscreen shows one row")
	require.Equal(t, `Unmatched changes go to the fallback rules: "Fallback Rule" ("Project owner").`, description[1])
	require.LessOrEqual(t, len(strings.Split(choiceDescription(t, toNone), "\n")), descriptionLines)
}

func TestAdministerWorkspaceStatesMovedAndChangedRules(t *testing.T) {
	staging := changeRule("Staging", "resource.environment_id == 'staging'", "roles/projectOwner")
	mock := &approvalMock{}
	mock.setStored(fallbackRule(), prodRule, staging)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	stagingWithDBA := changeRule("Staging", "resource.environment_id == 'staging'", "roles/workspaceDBA")

	callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), stagingWithDBA, prodRule)))
	asked := user.lastParams(t)
	// The move says what it is now behind, so the two lines need five message
	// rows and lead the description. Staging ends with Prod's approvers, which
	// the line reads from the request.
	require.Equal(t, strings.Join([]string{
		"Change the approval rules for database changes, as alice@example.com?",
		"Change 2 rules, listed under Action.",
		approvalRulesApply,
	}, "\n"), asked.Message)
	require.True(t, strings.HasPrefix(choiceDescription(t, asked), strings.Join([]string{
		`Change, "Project owner" to "Workspace DBA":`,
		"> `resource.environment_id == 'staging'`",
		`Move to rule 2, "Workspace DBA", now behind 1 rule with the same`,
		"> approvers: `resource.environment_id == 'prod'`",
	}, "\n")+"\n"), choiceDescription(t, asked))
}

// The rule list under the choice can be cut before the rules a move touches,
// so the move's line says who approves on each side of it: it names a passed
// rule with other approvers and counts those with the moved rule's own. Two
// workspaces that differ in the moved rule's approvers, or in those of the rule
// it passes, get different forms for the same move.
func TestAdministerWorkspaceMoveNamesTheRulesItPasses(t *testing.T) {
	message := func(production []string, finance ...string) string {
		kept := []*v1pb.WorkspaceApprovalSetting_Rule{
			changeRule("Staging schema changes", `resource.environment_id == "staging" && resource.project_id == "payments-platform"`, "roles/projectOwner"),
			changeRule("Development", `resource.environment_id == "dev"`),
			changeRule("Reporting warehouse", `resource.project_id == "reporting-warehouse" && resource.environment_id == "prod"`, "roles/projectOwner", "roles/workspaceDBA"),
			changeRule("Sandbox", `resource.project_id == "sandbox"`),
		}
		fin := changeRule("Finance", `resource.project_id == "finance"`, finance...)
		prod := changeRule("Production", `resource.environment_id == "prod"`, production...)
		mock := &approvalMock{}
		mock.setStored(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, kept, []*v1pb.WorkspaceApprovalSetting_Rule{fin, prod})...)
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		res := callAdminister(t, session, administerParams(approvalBody(t, slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, kept, []*v1pb.WorkspaceApprovalSetting_Rule{prod, fin})...)))
		require.False(t, res.IsError, resultText(t, res))
		asked := user.lastParams(t)
		require.Contains(t, choiceDescription(t, asked), "... and 4 more rules, tried after these.", "the list is cut before rules 5 and 6")
		return asked.Message
	}
	dba := []string{"roles/workspaceDBA"}
	require.Equal(t, strings.Join([]string{
		"Change the approval rules for database changes, as alice@example.com?",
		`Move to rule 6, "Workspace DBA", now behind "Production" (no`,
		"> approval): `resource.project_id == \"finance\"`",
		approvalRulesApply,
	}, "\n"), message(nil, dba...))
	require.Contains(t, message(nil), "Move to rule 6, no approval, now behind 1 rule with the same\n> approvers:")
	require.Contains(t, message(dba, dba...), `Move to rule 6, "Workspace DBA", now behind 1 rule with the same`)
	require.Contains(t, message([]string{"roles/projectOwner"}, dba...), `Move to rule 6, "Workspace DBA", now behind "Production" ("Project`)
}

// Reordering changes who approves only for two rules that trade order, and one
// of the two is always a moved rule. So for each moved rule and each side of
// it, the line accounts for every rule it traded order with: it names those
// with other approvers, or counts them by their approvers, and counts those
// with its own. The trades are counted here from the rules' places.
func TestApprovalMoveLinesNameEveryPairThatChangedOrder(t *testing.T) {
	random := rand.New(rand.NewPCG(22234, 6))
	flows := [][]string{nil, {"roles/projectOwner"}, {"roles/workspaceDBA"}}
	moveLine := regexp.MustCompile(`^Move to rule (\d+), (.*?), now (.*): ` + "`")
	title := regexp.MustCompile(`"(R\d)" \(([^()]*)\)`)
	same := regexp.MustCompile(`(\d+)(?: rules?)? with the same approvers`)
	group := regexp.MustCompile(`(\d+)(?: rules?)? with ("[^:,]*"|no approval)`)
	// sides splits what follows "now" into its behind and ahead parts.
	sides := func(text string) map[string]string {
		behind, ahead, _ := strings.Cut(strings.TrimPrefix(text, "behind "), "ahead of ")
		if !strings.HasPrefix(text, "behind ") {
			behind, ahead = "", strings.TrimPrefix(text, "ahead of ")
		}
		return map[string]string{"behind": strings.TrimSuffix(behind, ", and "), "ahead of": ahead}
	}
	count := func(match []string) int {
		if match == nil {
			return 0
		}
		n, err := strconv.Atoi(match[1])
		require.NoError(t, err)
		return n
	}
	moves, both, named, counted := 0, 0, 0, 0
	for range 400 {
		n := 2 + random.IntN(6)
		stored := make([]*v1pb.WorkspaceApprovalSetting_Rule, n)
		was := map[string]int{}
		for i := range stored {
			stored[i] = changeRule(fmt.Sprintf("R%d", i), fmt.Sprintf("resource.environment_id == 'e%d'", i), flows[random.IntN(len(flows))]...)
			was[stored[i].GetTemplate().GetTitle()] = i
		}
		requested := slices.Clone(stored)
		random.Shuffle(n, func(i, j int) { requested[i], requested[j] = requested[j], requested[i] })
		diff, err := diffApprovalRules(stored, requested, "")
		require.NoError(t, err)
		require.Len(t, diff.shorter[0], len(diff.changes))
		require.Len(t, diff.shorter[1], len(diff.changes))
		now := map[string]int{}
		for i, rule := range requested {
			now[rule.GetTemplate().GetTitle()] = i
		}
		for i, line := range diff.changes {
			match := moveLine.FindStringSubmatch(line)
			require.NotNil(t, match, line)
			place, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			moved := requested[place-1]
			require.Equal(t, approvers(moved), match[2], line)
			countedMatch := moveLine.FindStringSubmatch(diff.shorter[0][i])
			require.NotNil(t, countedMatch, diff.shorter[0][i])
			require.Equal(t, match[1:3], countedMatch[1:3], "both lines are for the same move")
			moves++
			if strings.Contains(match[3], "behind ") && strings.Contains(match[3], "ahead of ") {
				both++
			}
			for side, text := range sides(match[3]) {
				// What the places say: the rules that traded order with the
				// moved one on this side, by whether they share its approvers.
				var others []string
				shared := 0
				byApprovers := map[string]int{}
				for _, other := range requested {
					x, y := moved.GetTemplate().GetTitle(), other.GetTemplate().GetTitle()
					traded := x != y && (was[x] < was[y]) != (now[x] < now[y])
					if !traded || (side == "behind") != (now[x] > now[y]) {
						continue
					}
					if slices.Equal(other.GetTemplate().GetFlow().GetRoles(), moved.GetTemplate().GetFlow().GetRoles()) {
						shared++
						continue
					}
					others = append(others, fmt.Sprintf("%q (%s)", y, approvers(other)))
					byApprovers[approvers(other)]++
				}
				var names []string
				for _, found := range title.FindAllStringSubmatch(text, -1) {
					names = append(names, fmt.Sprintf("%q (%s)", found[1], found[2]))
				}
				require.Equal(t, others, names, "%s, in %q", side, line)
				require.Equal(t, shared, count(same.FindStringSubmatch(text)), "%s, in %q", side, line)
				named += len(names)

				countedText := sides(countedMatch[3])[side]
				require.Empty(t, title.FindAllStringSubmatch(countedText, -1), "the counted line names no rule: %q", diff.shorter[0][i])
				require.Equal(t, shared, count(same.FindStringSubmatch(countedText)), "%s, in %q", side, diff.shorter[0][i])
				groups := map[string]int{}
				for _, found := range group.FindAllStringSubmatch(same.ReplaceAllString(countedText, ""), -1) {
					groups[found[2]] += count(found)
				}
				if len(byApprovers) == 0 {
					require.Empty(t, groups, diff.shorter[0][i])
				} else {
					require.Equal(t, byApprovers, groups, "%s, in %q", side, diff.shorter[0][i])
				}
				counted += len(groups)

				terse := sides(moveLine.FindStringSubmatch(diff.shorter[1][i])[3])[side]
				want := ""
				none := byApprovers["no approval"]
				switch total := len(others) + shared; {
				case total == 0:
				case len(others) == 0:
					want = countRules(shared) + " with the same approvers"
				case none == len(others) && shared == 0:
					want = countRules(total) + " with no approval"
				case none == len(others):
					want = fmt.Sprintf("%s, %d of them with no approval", countRules(total), none)
				case none > 0:
					want = fmt.Sprintf("%s, %d with no approval and %d with other approvers", countRules(total), none, len(others)-none)
				case shared == 0:
					want = countRules(total) + " with other approvers"
				default:
					want = fmt.Sprintf("%s, %d of them with other approvers", countRules(total), len(others))
				}
				require.Equal(t, want, terse, "%s, in %q", side, diff.shorter[1][i])
			}
		}
		// Every rule outside the lines kept its order with every other one.
		movedTitles := map[string]bool{}
		for _, line := range diff.changes {
			place, _ := strconv.Atoi(moveLine.FindStringSubmatch(line)[1])
			movedTitles[requested[place-1].GetTemplate().GetTitle()] = true
		}
		for x := range was {
			for y := range was {
				if x != y && (was[x] < was[y]) != (now[x] < now[y]) {
					require.True(t, movedTitles[x] || movedTitles[y], "%q and %q traded order, and neither has a line: %q", x, y, diff.changes)
				}
			}
		}
	}
	require.Greater(t, moves, 400)
	require.Positive(t, both, "the sweep must reach a rule that is behind one rule and ahead of another")
	require.Greater(t, named, 400)
	require.Greater(t, counted, 400)
}

// A move that passes many rules with other approvers has no room for their
// names. It counts them by their approvers, so the clause about them stays
// short.
func TestAdministerWorkspaceCountsTheRulesAFarMovePasses(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	flows := [][]string{nil, {"roles/projectOwner"}, {"roles/workspaceDBA"}}
	form := func(n int, condition string) (*adminForm, *approvalRulesDiff, error) {
		stored := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), changeRule("Finance", condition, "roles/workspaceDBA")}
		for i := range n - 1 {
			stored = append(stored, changeRule(fmt.Sprintf("Payments platform schema changes %02d", i), fmt.Sprintf(`resource.project_id == "payments-%02d"`, i), flows[i%len(flows)]...))
		}
		requested := slices.Concat(stored[:1], stored[2:], stored[1:2])
		diff, err := diffApprovalRules(stored, requested, "LINK")
		require.NoError(t, err)
		require.Len(t, diff.changes, 1)
		form, err := approvalRulesForm(alice, diff, "", "LINK")
		return form, diff, err
	}
	const finance = `resource.project_id == "finance"`

	// Three rules with other approvers: their names fit under the choice.
	listed, diff, err := form(5, finance)
	require.NoError(t, err)
	require.Contains(t, listed.message, "Change 1 rule, listed under Action.")
	require.True(t, strings.HasPrefix(unwrapMessage(listed.description), diff.changes[0]+"\n"), listed.description)
	require.True(t, strings.HasPrefix(diff.changes[0], `Move to rule 5, "Workspace DBA", now behind "Payments platform schema changes 00" (no approval), `), diff.changes[0])
	require.Contains(t, diff.changes[0], " and 1 rule with the same approvers: `")

	// Eleven rules: the names have no room in any form, and the counts do.
	countedForm, diff, err := form(12, finance)
	require.NoError(t, err)
	require.Contains(t, diff.changes[0], `"Payments platform schema changes 10" ("Project owner")`, "the line with names is the first one tried")
	require.NotContains(t, countedForm.message, "Payments platform schema changes", "the message names no passed rule")
	require.Equal(t, "Move to rule 12, \"Workspace DBA\", now behind 11 rules: 4 with no approval, 4 with \"Project owner\", 3 with the same approvers: `"+finance+"`", diff.shorter[0][0])
	require.Contains(t, unwrapMessage(countedForm.message), diff.shorter[0][0], "and the counted line fits the message")

	// A condition that leaves the counts by approvers no room: the line only
	// counts the rules.
	long := strings.Repeat(`resource.project_id == "p" || `, 14) + "false"
	terse, diff, err := form(12, long)
	require.NoError(t, err)
	require.Equal(t, "Move to rule 12, \"Workspace DBA\", now behind 11 rules, 4 with no approval and 4 with other approvers: `"+long+"`", diff.shorter[1][0])
	require.Greater(t, len(wrapLine(diff.shorter[0][0], descriptionLineWidth)), descriptionLines-2, "the counts by approvers pass the rows the fallback line leaves")
	require.True(t, strings.HasPrefix(unwrapMessage(terse.description), diff.shorter[1][0]+"\n"), terse.description)

	// A condition too long for any form is still refused, with its shortest line.
	_, diff, err = form(12, strings.Repeat(`resource.project_id == "p" || `, 30)+"false")
	var refused *toolError
	require.ErrorAs(t, err, &refused)
	require.Equal(t, "CHANGE_TOO_LONG", refused.Code)
	require.Contains(t, refused.Message, strconv.Quote(diff.shorter[1][0]))
	require.Contains(t, refused.Suggestion, "make that change signed in to the Bytebase console: LINK")
}

// The shortest line of a move still says how many of the rules it passes need
// no approval: a rule that moves behind one of them loses its approval there.
func TestApprovalMoveShortestLineCountsTheRulesWithNoApproval(t *testing.T) {
	dba, owner := []string{"roles/workspaceDBA"}, []string{"roles/projectOwner"}
	for _, tc := range []struct {
		moved            []string
		none, other, dba int
		want             string
	}{
		{dba, 2, 7, 3, "now behind 12 rules, 2 with no approval and 7 with other approvers"},
		{dba, 1, 6, 0, "now behind 7 rules, 1 with no approval and 6 with other approvers"},
		{dba, 3, 0, 0, "now behind 3 rules with no approval"},
		{dba, 0, 9, 3, "now behind 12 rules, 9 of them with other approvers"},
		{dba, 0, 7, 0, "now behind 7 rules with other approvers"},
		{dba, 2, 0, 3, "now behind 5 rules, 2 of them with no approval"},
		{dba, 0, 0, 4, "now behind 4 rules with the same approvers"},
		// With no approval of its own, the moved rule has the same approvers as
		// the rules with none.
		{nil, 3, 2, 0, "now behind 5 rules, 2 of them with other approvers"},
	} {
		moved := changeRule("Finance", `resource.project_id == "finance"`, tc.moved...)
		var passed []*v1pb.WorkspaceApprovalSetting_Rule
		for kind, flow := range [][]string{nil, owner, dba} {
			for i := range []int{tc.none, tc.other, tc.dba}[kind] {
				passed = append(passed, changeRule(fmt.Sprintf("Rule %d-%d", kind, i), fmt.Sprintf(`resource.project_id == "p%d-%d"`, kind, i), flow...))
			}
		}
		down, err := diffApprovalRules(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), moved}, passed), slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, passed, []*v1pb.WorkspaceApprovalSetting_Rule{moved}), "")
		require.NoError(t, err)
		require.Len(t, down.shorter[1], 1)
		require.Equal(t, fmt.Sprintf("Move to rule %d, %s, %s: `resource.project_id == \"finance\"`", len(passed)+1, approvers(moved), tc.want), down.shorter[1][0])
		up, err := diffApprovalRules(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, passed, []*v1pb.WorkspaceApprovalSetting_Rule{moved}), slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), moved}, passed), "")
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("Move to rule 1, %s, %s: `resource.project_id == \"finance\"`", approvers(moved), strings.Replace(tc.want, "now behind", "now ahead of", 1)), up.shorter[1][0])
	}
}

// A move behind rules with seven kinds of approvers has no room for their
// names and room for its counts by approvers. So the form that takes away an
// approval and the one that does not are not the same form.
func TestAdministerWorkspaceMoveCountsEveryKindOfApproversItHasRowsFor(t *testing.T) {
	description := func(condition string, production ...string) (string, string) {
		finance := changeRule("Finance", condition, "roles/workspaceDBA")
		var environments []*v1pb.WorkspaceApprovalSetting_Rule
		for i, flow := range [][]string{
			{"roles/projectOwner"}, {"roles/projectDeveloper"}, {"roles/sqlEditorUser"}, {"roles/workspaceAdmin"}, production,
			{"roles/projectOwner", "roles/workspaceDBA"}, {"roles/workspaceAdmin", "roles/projectOwner"},
		} {
			environments = append(environments, changeRule(fmt.Sprintf("Environment %d schema and data changes, all projects", i), fmt.Sprintf(`resource.environment_id == "e%d"`, i), flow...))
		}
		mock := &approvalMock{}
		mock.setStored(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), finance}, environments)...)
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		res := callAdminister(t, session, administerParams(approvalBody(t, slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, environments, []*v1pb.WorkspaceApprovalSetting_Rule{finance})...)))
		require.False(t, res.IsError, resultText(t, res))
		asked := user.lastParams(t)
		return asked.Message, unwrapMessage(choiceDescription(t, asked))
	}
	const finance = `resource.project_id == "finance"`
	message, reviewed := description(finance, "roles/projectReleaser")
	require.Contains(t, message, "Change 1 rule, listed under Action.")
	_, unapproved := description(finance)
	require.Contains(t, reviewed, `Move to rule 8, "Workspace DBA", now behind 7 rules: 1 with "Project owner", 1 with "Project developer", 1 with "SQL Editor User", 1 with "Workspace admin", 1 with "Project releaser", 1 with "Project owner" then "Workspace DBA", 1 with "Workspace admin" then "Project owner": `+"`"+finance+"`")
	require.Contains(t, unapproved, `1 with "Workspace admin", 1 with no approval, 1 with "Project owner" then "Workspace DBA"`)

	// A condition that leaves the counts by approvers no room: the shortest
	// line still tells the two apart.
	long := strings.Repeat(`resource.project_id == "p" || `, 12) + "false"
	_, reviewed = description(long, "roles/projectReleaser")
	_, unapproved = description(long)
	require.Contains(t, reviewed, `Move to rule 8, "Workspace DBA", now behind 7 rules with other approvers: `)
	require.Contains(t, unapproved, `Move to rule 8, "Workspace DBA", now behind 7 rules, 1 with no approval and 6 with other approvers: `)
}

// A reorder can be read by more than one set of moved rules, and its reverse
// reads them the other way round. So before a form is refused the other
// reading is tried: a swap that a form makes, a form swaps back.
func TestAdministerWorkspaceSwapsBackWhatItSwaps(t *testing.T) {
	swap := func(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule) (message, refusal string) {
		mock := &approvalMock{}
		mock.setStored(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, stored)...)
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		res := callAdminister(t, session, administerParams(approvalBody(t, slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, requested)...)))
		if user.asked.Load() == 0 {
			return "", resultText(t, res)
		}
		return unwrapMessage(user.lastParams(t).Message), ""
	}
	projects := func(n int) string {
		var terms []string
		for i := range n {
			terms = append(terms, fmt.Sprintf(`resource.project_id == "p%02d"`, i))
		}
		return strings.Join(terms, " || ")
	}
	production := changeRule("Production", `resource.environment_id == "prod"`, "roles/workspaceDBA")
	for n := 4; n <= 26; n++ {
		many := changeRule("Core projects", projects(n), "roles/projectOwner")
		message, refusal := swap([]*v1pb.WorkspaceApprovalSetting_Rule{production, many}, []*v1pb.WorkspaceApprovalSetting_Rule{many, production})
		require.Empty(t, refusal, "%d projects", n)
		require.Contains(t, message, `Move to rule 2, "Workspace DBA", now behind "Core projects" ("Project owner")`)
		_, refusal = swap([]*v1pb.WorkspaceApprovalSetting_Rule{many, production}, []*v1pb.WorkspaceApprovalSetting_Rule{production, many})
		require.Empty(t, refusal, "%d projects, swapped back", n)
	}

	// The first reading stands wherever it gets a form: the rule that ends
	// second is the moved one.
	short := changeRule("Core projects", projects(4), "roles/projectOwner")
	message, _ := swap([]*v1pb.WorkspaceApprovalSetting_Rule{short, production}, []*v1pb.WorkspaceApprovalSetting_Rule{production, short})
	require.Contains(t, message, `Move to rule 2, "Project owner", now behind "Production" ("Workspace DBA"): `+"`"+projects(4)+"`")
	// A rule too long for any form is not the moved one in the second reading.
	long := changeRule("Core projects", projects(26), "roles/projectOwner")
	message, _ = swap([]*v1pb.WorkspaceApprovalSetting_Rule{long, production}, []*v1pb.WorkspaceApprovalSetting_Rule{production, long})
	require.Contains(t, message, `Move to rule 1, "Workspace DBA", now ahead of "Core projects" ("Project owner"): `+"`"+`resource.environment_id == "prod"`+"`")
	// With no reading that fits, the refusal is the first reading's.
	other := changeRule("Other projects", strings.ReplaceAll(projects(26), `"p`, `"q`), "roles/workspaceDBA")
	_, refusal := swap([]*v1pb.WorkspaceApprovalSetting_Rule{long, other}, []*v1pb.WorkspaceApprovalSetting_Rule{other, long})
	require.Contains(t, refusal, "CHANGE_TOO_LONG")
	require.Contains(t, refusal, "Move to rule 2, ")
	require.Contains(t, refusal, "now behind 1 rule with other approvers")
	require.NotContains(t, refusal, "Move to rule 1, ")
}

// The second reading of a reorder is the first reading of its reverse, and it
// is worth a form only when it differs and its moves could fit one.
func TestMovesReadBothWays(t *testing.T) {
	random := rand.New(rand.NewPCG(22234, 9))
	kept := func(elements []string, inOrder []bool) []string {
		var names []string
		for i, name := range elements {
			if inOrder[i] {
				names = append(names, name)
			}
		}
		slices.Sort(names)
		return names
	}
	differ := 0
	for range 400 {
		n := 2 + random.IntN(7)
		a := make([]string, n)
		for i := range a {
			a[i] = fmt.Sprintf("r%d", i)
		}
		b := slices.Clone(a)
		random.Shuffle(n, func(i, j int) { b[i], b[j] = b[j], b[i] })
		first, second := longestCommonOrders(a, b)
		back, backSecond := longestCommonOrders(b, a)
		require.Equal(t, kept(b, second), kept(a, back), "%v to %v", a, b)
		require.Equal(t, kept(b, first), kept(a, backSecond), "%v to %v", a, b)
		require.Len(t, kept(b, second), len(kept(b, first)), "both readings are as long")
		for _, inOrder := range [][]bool{first, second} {
			var order []string
			for i, name := range b {
				if inOrder[i] {
					order = append(order, name)
				}
			}
			require.True(t, slices.IsSorted(order), "the kept rules are in a's order: %v", order)
		}
		if !slices.Equal(first, second) {
			differ++
		}
	}
	require.Greater(t, differ, 100)

	rules := func(n int) []string {
		var names []string
		for i := range n {
			names = append(names, fmt.Sprintf("r%02d", i))
		}
		return names
	}
	reversed := func(n int) []string {
		names := rules(n)
		slices.Reverse(names)
		return names
	}
	_, another := movesRead(rules(2), reversed(2), false)
	require.True(t, another, "a swap has two readings")
	_, another = movesRead(rules(2), reversed(2), true)
	require.False(t, another, "the second reading has no other")
	_, another = movesRead(rules(4), append(rules(4)[1:], "r00"), false)
	require.False(t, another, "a move across the list has one reading")
	_, another = movesRead(rules(descriptionLines+1), reversed(descriptionLines+1), false)
	require.True(t, another, "ten moves can take a row each")
	_, another = movesRead(rules(descriptionLines+2), reversed(descriptionLines+2), false)
	require.False(t, another, "eleven moves fit no form by either reading")
}

// A step is a request of its own, and its rules decide which rules a move's
// line names there. So a form that lists a move offers no steps, though each
// of its lines fits a message alone, and a form with no move still does.
func TestAdministerWorkspaceOffersNoStepsForAMove(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	const issues = "Plan edits recheck your \"hr#101\" under these rules."
	listed := func(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule) (*adminForm, *approvalRulesDiff) {
		diff, err := diffApprovalRules(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, stored), slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, requested), "")
		require.NoError(t, err)
		form, err := approvalRulesForm(alice, diff, issues, "")
		require.NoError(t, err)
		require.Contains(t, form.message, "listed under Action")
		return form, diff
	}
	alone := func(change string) bool {
		return messageHolds(alice, approvalRules.question, formChange{text: change, effect: true}, issues)
	}
	finance := changeRule("Finance", `resource.project_id == "finance"`, "roles/workspaceDBA")
	ledger := changeRule("Ledger", `resource.project_id == "ledger"`, "roles/workspaceDBA")
	prod := changeRule("Production", `resource.environment_id == "prod"`)
	added := changeRule("Staging", `resource.environment_id == "staging"`, "roles/projectOwner")

	form, diff := listed([]*v1pb.WorkspaceApprovalSetting_Rule{finance, prod}, []*v1pb.WorkspaceApprovalSetting_Rule{
		changeRule("Finance", `resource.project_id == "finance"`, "roles/projectOwner"), prod, added,
	})
	require.False(t, diff.moved)
	require.Len(t, diff.changes, 2)
	require.Contains(t, form.inSteps, "a rule or two at a time")

	form, diff = listed([]*v1pb.WorkspaceApprovalSetting_Rule{finance, prod}, []*v1pb.WorkspaceApprovalSetting_Rule{prod, finance, added})
	require.True(t, diff.moved)
	require.Len(t, diff.changes, 2)
	for _, change := range diff.changes {
		require.True(t, alone(change), change)
	}
	require.Empty(t, form.inSteps)

	// Two moves that this form states by counts, since their lines with names
	// need more rows than it has. Each count fits a message alone, and a step
	// would still list its move, by its names, under the choice.
	var payments []*v1pb.WorkspaceApprovalSetting_Rule
	for i := range 4 {
		payments = append(payments, changeRule(fmt.Sprintf("Payments platform schema changes %02d", i), fmt.Sprintf(`resource.project_id == "payments-%02d"`, i), [][]string{nil, {"roles/projectOwner"}}[i%2]...))
	}
	stored := slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{finance, ledger}, payments)
	form, diff = listed(stored, slices.Concat(payments, []*v1pb.WorkspaceApprovalSetting_Rule{finance, ledger}))
	require.Contains(t, unwrapMessage(form.description), diff.shorter[0][0])
	require.True(t, alone(diff.shorter[0][0]))
	require.Empty(t, form.inSteps)
	form, diff = listed(stored, slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{ledger}, payments, []*v1pb.WorkspaceApprovalSetting_Rule{finance}))
	require.Len(t, diff.changes, 1)
	require.Contains(t, unwrapMessage(form.description), diff.changes[0])
}

// What a move's line says of the rules it passes is short whatever the
// workspace stores: it names no rule when the names could not fit a
// description, and counts kinds of approvers apart only while their texts
// could.
func TestApprovalMoveLineIsBoundedForAnyStoredRules(t *testing.T) {
	roles := []string{"roles/projectOwner", "roles/workspaceDBA", "roles/workspaceAdmin", "roles/projectDeveloper", "roles/sqlEditorUser", "roles/projectReleaser", "roles/gitopsServiceAgent", "roles/projectViewer"}
	stored := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
	for i := range 400 {
		// Every rule gets its own list of approvers.
		flow := []string{roles[i%len(roles)], roles[(i/len(roles))%len(roles)], roles[(i/len(roles)/len(roles))%len(roles)]}
		stored = append(stored, changeRule(fmt.Sprintf("Rule %03d", i), fmt.Sprintf(`resource.project_id == "p%03d"`, i), flow...))
	}
	requested := slices.Clone(stored)
	slices.Reverse(requested[1:])
	diff, err := diffApprovalRules(stored, requested, "")
	require.NoError(t, err)
	require.Len(t, diff.changes, 399)
	for i, line := range diff.changes {
		// Each side's names, and its counts by approvers, stop at about one
		// description's worth of text.
		for _, text := range []string{line, diff.shorter[0][i]} {
			require.LessOrEqual(t, lineColumns(text), 3*descriptionLines*descriptionLineWidth, "%.200s", text)
		}
	}
	last := len(diff.changes) - 1
	require.Equal(t, diff.shorter[0][last], diff.changes[last], "names that stopped short are not shown as the whole list")
	require.Equal(t, "Move to rule 400, \"Project owner\" then \"Project owner\" then \"Project owner\", now behind 399 rules with other approvers: `resource.project_id == \"p000\"`", diff.shorter[0][last])

	// Rules that fifty moves pass, with a title or lists of approvers that a
	// description has no room for, alone or together: no line of those moves
	// carries one, and a rule with a short name after them is not named alone.
	custom := func(first, count int) []string {
		var ids []string
		for i := range count {
			ids = append(ids, fmt.Sprintf("roles/custom-approver-%03d", first+i))
		}
		return ids
	}
	passed := []*v1pb.WorkspaceApprovalSetting_Rule{
		changeRule(strings.Repeat("T", 20000), `resource.project_id == "long-title"`, "roles/projectOwner"),
		changeRule("Many approvers", `resource.project_id == "many"`, custom(0, 200)...),
		changeRule("Short", `resource.project_id == "short"`, "roles/projectOwner"),
	}
	for i := range 6 {
		passed = append(passed, changeRule(fmt.Sprintf("Ten approvers %d", i), fmt.Sprintf(`resource.project_id == "ten-%d"`, i), custom(200+10*i, 10)...))
	}
	var movers []*v1pb.WorkspaceApprovalSetting_Rule
	for i := range 50 {
		movers = append(movers, changeRule(fmt.Sprintf("Rule %02d", i), fmt.Sprintf(`resource.project_id == "p%02d"`, i), "roles/workspaceDBA"))
	}
	reversed := slices.Clone(movers)
	slices.Reverse(reversed)
	// The lists of two have one rule with other approvers and a rule with the
	// movers' approvers, which keeps the two in the common order. A combining
	// mark takes no column, so a title or a role of them is long in characters
	// only, and a wide character takes two, so a title of them is long in
	// columns first.
	anchor := changeRule("Anchor", `resource.project_id == "anchor"`, "roles/workspaceDBA")
	marks := strings.Repeat("\u0301", 20000)
	for _, kept := range []struct {
		rules  []*v1pb.WorkspaceApprovalSetting_Rule
		others int
	}{
		{[]*v1pb.WorkspaceApprovalSetting_Rule{passed[0], anchor}, 1},
		{[]*v1pb.WorkspaceApprovalSetting_Rule{changeRule("Marked"+marks, `resource.project_id == "marked"`, "roles/projectOwner"), anchor}, 1},
		{[]*v1pb.WorkspaceApprovalSetting_Rule{changeRule("Marked role", `resource.project_id == "marked-role"`, "roles/marked"+marks), anchor}, 1},
		{[]*v1pb.WorkspaceApprovalSetting_Rule{changeRule(strings.Repeat("\u5bbd", 400), `resource.project_id == "wide"`, "roles/projectOwner"), anchor}, 1},
		{passed[:3], 3},
		{passed[3:], 6},
	} {
		before := slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, kept.rules, movers)
		moves, err := diffApprovalRules(before, slices.Concat(before[:1], reversed, kept.rules), "")
		require.NoError(t, err)
		require.Len(t, moves.changes, len(movers))
		for i, line := range moves.changes {
			// Reversed, mover i is ahead of the movers after it, which have its
			// approvers, and of every kept rule.
			total := len(movers) - 1 - i + len(kept.rules)
			ahead := fmt.Sprintf("ahead of %d rules, %d of them with other approvers", total, kept.others)
			if total == kept.others {
				ahead = fmt.Sprintf("ahead of %d rules with other approvers", total)
			}
			require.Equal(t, moves.shorter[0][i], line, "names that stopped short give way to the counts")
			require.Contains(t, moves.shorter[1][i], ahead)
			for _, text := range []string{line, moves.shorter[1][i]} {
				require.LessOrEqual(t, lineColumns(text), descriptionLines*descriptionLineWidth, "%.300s", text)
				require.LessOrEqual(t, utf8.RuneCountInString(text), descriptionLines*descriptionLineWidth, "%.300s", text)
			}
		}
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
	untitled := changeRule("", "resource.environment_id == 'x'", "roles/projectOwner")
	for name, tc := range map[string]struct {
		stored, requested []*v1pb.WorkspaceApprovalSetting_Rule
		want              []string
	}{
		"removed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: nil,
			want: []string{"Remove, \"Workspace DBA\": `resource.environment_id == 'prod'`"}},
		"renamed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(func(r *v1pb.WorkspaceApprovalSetting_Rule) { r.Template.Title = "Production" })},
			want:      []string{"Rename to \"Production\": `resource.environment_id == 'prod'`"}},
		"description changed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{described},
			want: []string{"Change description to \"Production\": `resource.environment_id == 'prod'`"}},
		"description removed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{described}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)},
			want: []string{"Remove description: `resource.environment_id == 'prod'`"}},
		"approvers and title changed": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(func(r *v1pb.WorkspaceApprovalSetting_Rule) {
				r.Template.Title = "Production"
				r.Template.Flow.Roles = nil
			})},
			want: []string{"Change, \"Workspace DBA\" to no approval; rename to \"Production\": `resource.environment_id == 'prod'`"}},
		"added ahead of a kept rule": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{dev, prod(nil)},
			want: []string{"Add rule 1, no approval: `resource.environment_id != 'dev'`"}},
		"added after the kept rules": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil)}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil), dev},
			want: []string{"Add rule 2, no approval: `resource.environment_id != 'dev'`"}},
		"moved behind a kept rule": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil), dev}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{dev, prod(nil)},
			want: []string{"Move to rule 2, \"Workspace DBA\", now behind \"Dev\" (no approval): `resource.environment_id == 'prod'`"}},
		"moved, with other approvers": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil), dev},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{dev, prod(func(r *v1pb.WorkspaceApprovalSetting_Rule) { r.Template.Flow.Roles = nil })},
			want:      []string{"Change, \"Workspace DBA\" to no approval; move to rule 2, now behind 1 rule with the same approvers: `resource.environment_id == 'prod'`"}},
		"moved behind a rule with no title": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil), untitled}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{untitled, prod(nil)},
			want: []string{"Move to rule 2, \"Workspace DBA\", now behind rule 1 (\"Project owner\"): `resource.environment_id == 'prod'`"}},
		"moved ahead of two kept rules": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{dev, untitled, prod(nil)}, requested: []*v1pb.WorkspaceApprovalSetting_Rule{prod(nil), dev, untitled},
			want: []string{"Move to rule 1, \"Workspace DBA\", now ahead of \"Dev\" (no approval) and rule 3 (\"Project owner\"): `resource.environment_id == 'prod'`"}},
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
	require.True(t, strings.HasPrefix(lines[2], continuedLine), lines[2])
	require.Equal(t, "Add rule 1, \"Workspace DBA\": `resource.environment_id == 'prod' && "+
		"resource.project_id == 'hr' && resource.database_name == 'hr_prod'`", lines[1]+strings.TrimPrefix(lines[2], continuedLine))
	require.Equal(t, approvalRulesApply, lines[3], "the fixed sentence goes whole in the line that is left")
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.NotContains(t, string(schema), approvalRulesApply)
}

// With the mark, this rule's second line takes 69 of the 70 columns, so a mark
// two columns wider wraps it to a third line, and beside an issue line the form
// is refused.
func TestAdministerWorkspaceFitsALongRuleBesideAnIssue(t *testing.T) {
	long := changeRule("HR prod", "resource.environment_id == 'prod' && resource.project_id == 'hr' && resource.database_name == 'hr_prod'", "roles/workspaceDBA")
	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), long}, "")
	require.NoError(t, err)
	form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, issuesLine([]string{"hr#12"}, false), "")
	require.NoError(t, err)
	lines := strings.Split(form.message, "\n")
	require.Len(t, lines, 4)
	require.True(t, strings.HasPrefix(lines[2], continuedLine), lines[2])
	require.Equal(t, `Plan edits recheck your "hr#12" under these rules.`, lines[3])
}

// Only the masking entry states its effect in the question: the approval
// effect names the user's issues, which leave a question no room. A rule too
// long to share the message with that line goes under the choice.
func TestAdministerWorkspaceKeepsTheIssueLineOutOfTheQuestion(t *testing.T) {
	long := changeRule("HR salary", `resource.environment_id == "prod" && resource.project_id == "hr" && resource.database_name == "hr_prod" && resource.table_name == "salary"`, "roles/workspaceDBA")
	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), long}, "")
	require.NoError(t, err)
	alice := withUserEmail(context.Background(), "alice@example.com")
	form, err := approvalRulesForm(alice, diff, "", "")
	require.NoError(t, err)
	require.NotContains(t, form.message, "listed under Action", "with no issue line, the message states the rule")
	form, err = approvalRulesForm(alice, diff, issuesLine([]string{"hr#12"}, false), "")
	require.NoError(t, err)
	require.Equal(t, strings.Join([]string{
		`Change the approval rules for database changes, as alice@example.com?`,
		`Add 1 rule, listed under Action.`,
		`Plan edits recheck your "hr#12" under these rules.`,
		approvalRulesApply,
	}, "\n"), form.message)
	require.True(t, strings.HasPrefix(form.description, "Add rule 1, \"Workspace DBA\": `resource.environment_id == \"prod\" &&"), form.description)
}

func TestAdministerWorkspaceNamesTheUserIssuesAwaitingApproval(t *testing.T) {
	for name, tc := range map[string]struct {
		morePending bool
		want        string
	}{
		"every issue read": {want: `Plan edits recheck your "hr#12" and 3 more under these rules.`},
		"a search with another page": {morePending: true,
			want: `Plan edits recheck your "hr#12" and at least 3 more under these rules.`},
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
			asked := user.lastParams(t)
			require.Contains(t, asked.Message, tc.want)
			lines := strings.Split(asked.Message, "\n")
			require.Len(t, lines, 4)
			require.Equal(t, approvalRulesApply, lines[3], "a free line takes the fixed sentence beside the issue line")
			require.NotContains(t, choiceDescription(t, asked), approvalRulesApply)
		})
	}
}

func TestApproversNameEachRoleAsWritten(t *testing.T) {
	for roles, want := range map[string]string{
		"roles/workspaceDBA":                    `"Workspace DBA"`,
		"workspaceDBA":                          `"workspaceDBA"`,
		"roles/custom-dba":                      `"roles/custom-dba"`,
		"roles/workspaceDBA,roles/projectOwner": `"Workspace DBA" then "Project owner"`,
		"":                                      "no approval",
	} {
		rule := changeRule("Prod", "resource.environment_id == 'prod'")
		if roles != "" {
			rule.Template.Flow.Roles = strings.Split(roles, ",")
		}
		require.Equal(t, want, approvers(rule), roles)
	}
}

func TestConditionTextShowsAConditionAsTypedOnlyWhenEveryCharacterPrints(t *testing.T) {
	for condition, want := range map[string]string{
		`resource.environment_id == "prod"`: "`resource.environment_id == \"prod\"`",
		"resource.environment_id == 'prod'": "`resource.environment_id == 'prod'`",
		`resource.environment_id == "環境"`:   "`resource.environment_id == \"環境\"`",
		"resource.environment_id == `prod`": `"resource.environment_id == ` + "`prod`" + `"`,
		"a\tb":                              `"a\tb"`,
		"a\nb":                              `"a\nb"`,
		"pr\u202eod":                        `"pr\u202eod"`,
		"a\u200bb":                          `"a\u200bb"`,
		"a\u00a0b":                          `"a\u00a0b"`,
		"a\xffb":                            `"a\xffb"`,
	} {
		require.Equal(t, want, conditionText(condition), "%q", condition)
	}
}

func TestAdministerWorkspaceEscapesAConditionThatDoesNotPrintAtEverySite(t *testing.T) {
	removed := changeRule("Removed", "a\tb", "roles/workspaceDBA")
	kept := changeRule("Kept", "x\u200by", "roles/projectOwner")
	keptWithDBA := changeRule("Kept", "x\u200by", "roles/workspaceDBA")
	added := changeRule("Added", "pr\u202eod")
	// The console writes a long condition on several lines.
	console := changeRule("Console", "resource.environment_id == \"prod\" &&\nresource.project_id == \"hr\"")
	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), removed, kept},
		[]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), keptWithDBA, added, console}, "")
	require.NoError(t, err)
	require.Equal(t, []string{
		`Remove, "Workspace DBA": "a\tb"`,
		`Change, "Project owner" to "Workspace DBA": "x\u200by"`,
		`Add rule 2, no approval: "pr\u202eod"`,
		`Add rule 3, no approval: "resource.environment_id == \"prod\" &&\nresource.project_id == \"hr\""`,
	}, diff.changes)

	oddFallback := &v1pb.WorkspaceApprovalSetting_Rule{Condition: &expr.Expr{Expression: "p\u200bq"},
		Template: &v1pb.ApprovalTemplate{Title: "Odd", Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}}}
	diff, err = diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{oddFallback, kept}, []*v1pb.WorkspaceApprovalSetting_Rule{oddFallback, kept, added}, "")
	require.NoError(t, err)
	form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "", "")
	require.NoError(t, err)
	require.Contains(t, form.message, `Add rule 2, no approval: "pr\u202eod"`)
	require.Contains(t, unwrapMessage(form.description), `"Odd" ("Project owner") when "p\u200bq"`)
	require.Contains(t, form.description, `1. "Kept" (kept), "Project owner": "x\u200by"`)
	require.False(t, strings.ContainsAny(form.message+form.description, "\t\u200b\u202e"))
}

// A long condition continues after continuedLine, so a string literal inside it
// cannot wrap into a line that reads as one of the form's own.
func TestAdministerWorkspaceMarksEveryContinuationLine(t *testing.T) {
	for name, tc := range map[string]struct {
		condition, forged string
	}{
		"an issue line": {
			condition: `resource.environment_id == "test" || 'x Plan edits recheck your "hr#12" under these rules.` + strings.Repeat(" ", 25) + `'.size() > 0`,
			forged:    `Plan edits recheck your "hr#12" under these rules.`,
		},
		"a change line in double quotes": {
			condition: `resource.environment_id == 'test' || '''xxx Add rule 2, "Workspace DBA": "resource.environment_id == 'prod'" yyy''' != ''`,
			forged:    `Add rule 2, "Workspace DBA": "resource.environment_id == 'prod'"`,
		},
		"a change line in backquotes": {
			condition: "resource.environment_id == 'test' ||\t''' Add rule 2, no approval: `resource.environment_id == 'dev'` yyy''' != ''",
			forged:    "Add rule 2, no approval: `resource.environment_id == 'dev'`",
		},
	} {
		t.Run(name, func(t *testing.T) {
			diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()},
				[]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), changeRule("t", tc.condition)}, "")
			require.NoError(t, err)
			form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "", "")
			require.NoError(t, err)
			require.Contains(t, form.message, tc.forged, "the fixture must wrap its forged line into the message")
			lines := strings.Split(form.message, "\n")
			require.True(t, strings.HasPrefix(lines[1], "Add rule 1, no approval: "), lines[1])
			for _, line := range lines[2:] {
				require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
				if first, _ := utf8.DecodeRuneInString(line); line != approvalRulesApply {
					require.False(t, unicode.IsLetter(first), "%q poses as a line of its own", line)
				}
			}
		})
	}
}

// fillRows counts the rows of line when every row is filled, without
// wrapLine's helpers.
func fillRows(line string, width int) int {
	rows, room := 1, width
	for _, r := range line {
		columns := lineColumns(string(r))
		if columns > room {
			rows++
			room = width - lineColumns(continuedLine)
		}
		room -= columns
	}
	return rows
}

// allocated returns the bytes f allocates.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// A row that ends at a space must not cost a row: a long quoted value after a
// space would leave the row before it nearly empty, and a form has four.
func TestWrapLineNeverTakesMoreRowsThanFillingThem(t *testing.T) {
	random := rand.New(rand.NewPCG(10598, 22224))
	alphabet := []rune(`abcdefghijklmnopqrstuvwxyz0123456789"',()=&|/.-_@名前` + "\u0301\u0308")
	for range 5000 {
		var tokens []string
		for range 1 + random.IntN(12) {
			token := make([]rune, 1+random.IntN(90))
			for i := range token {
				token[i] = alphabet[random.IntN(len(alphabet))]
			}
			tokens = append(tokens, string(token))
		}
		line := strings.Join(tokens, " ")
		for _, width := range []int{messageLineWidth, 40} {
			rows := wrapLine(line, width)
			require.Len(t, rows, fillRows(line, width), "%q at %d: %q", line, width, rows)
			var joined strings.Builder
			for i, row := range rows {
				require.LessOrEqual(t, lineColumns(row), width, "%q", rows)
				if i > 0 {
					require.True(t, strings.HasPrefix(row, continuedLine), "%q", rows)
					row = strings.TrimPrefix(row, continuedLine)
				}
				joined.WriteString(row)
			}
			require.Equal(t, line, joined.String(), "a break leaves nothing out")
		}
	}
}

// A line that the last-space rule already wrapped in the fewest rows, with
// every break at a space, keeps its rows. So a form changes only where that
// rule cost it a row, or cut a value at the width: such a row may now end at a
// separator of a name, and its continuation starts ">" with no space.
func TestWrapLineKeepsEveryBreakAtASpaceThatCostNoRow(t *testing.T) {
	// before is the wrap as it was: every row ended at its last space, the
	// space was dropped, and a continuation started with "> ". atSpaces
	// reports that no row had to be cut at the width.
	before := func(line string) (rows []string, atSpaces bool) {
		prefix := ""
		atSpaces = true
		for lineColumns(prefix+line) > messageLineWidth {
			width := messageLineWidth - lineColumns(prefix)
			runes := []rune(line)
			cut := min(width, len(runes))
			for lineColumns(string(runes[:cut])) > width {
				cut--
			}
			next := cut
			if space := strings.LastIndex(string(runes[:cut]), " "); space > 0 {
				cut = len([]rune(string(runes[:cut])[:space]))
				next = cut + 1
			} else {
				atSpaces = false
			}
			rows = append(rows, prefix+string(runes[:cut]))
			line = string(runes[next:])
			prefix = "> "
		}
		return append(rows, prefix+line), atSpaces
	}
	random := rand.New(rand.NewPCG(11, 2))
	alphabet := []rune(`abcdefghijklmnopqrstuvwxyz0123456789"'=&|/.-_`)
	same, fewer := 0, 0
	for range 5000 {
		var tokens []string
		for range 2 + random.IntN(20) {
			token := make([]rune, 1+random.IntN(1+random.IntN(80)))
			for i := range token {
				token[i] = alphabet[random.IntN(len(alphabet))]
			}
			tokens = append(tokens, string(token))
		}
		line := strings.Join(tokens, " ")
		was, atSpaces := before(line)
		now := wrapMessageLine(line)
		require.LessOrEqual(t, len(now), len(was), line)
		switch {
		case len(now) > descriptionLines:
			// No form shows such a line, and it is filled.
		case len(now) < len(was):
			fewer++
		case atSpaces:
			same++
			// The space the old wrap dropped after "> " now follows ">", so
			// the rows are equal as text.
			require.Equal(t, was, now, line)
		default:
		}
	}
	require.Greater(t, same, 1000)
	require.Greater(t, fewer, 100)
}

// A line of more rows than a form's description is in no form, only counted.
// It is filled in one pass, since the rule that picks each break measures the
// rest of the line again for every row.
func TestWrapLineFillsALineNoFormShows(t *testing.T) {
	words := func(n int) string { return strings.TrimSuffix(strings.Repeat("abcde ", n), " ") }
	shown := wrapLine(words(105), descriptionLineWidth)
	require.Len(t, shown, descriptionLines)
	require.Equal(t, words(11), shown[0], "a line a form can show ends its rows at spaces")
	counted := wrapLine(words(115), descriptionLineWidth)
	require.Len(t, counted, descriptionLines+1)
	require.Equal(t, words(11)+" ab", counted[0], "one row more, and the rows are filled")

	long := strings.Repeat("名前 e\u0301te\u0301 payments-ledger.example.com ", 4000)
	var rows []string
	bytes := allocated(func() { rows = wrapLine(long, descriptionLineWidth) })
	require.Less(t, bytes, uint64(64*len(long)), "the wrap must not copy the rest of the line for every row")
	require.Len(t, rows, fillRows(long, descriptionLineWidth))
	var joined strings.Builder
	for i, row := range rows {
		require.LessOrEqual(t, lineColumns(row), descriptionLineWidth)
		if i > 0 {
			require.True(t, strings.HasPrefix(row, continuedLine), row)
			row = strings.TrimPrefix(row, continuedLine)
		}
		joined.WriteString(row)
	}
	require.Equal(t, long, joined.String(), "a break leaves nothing out")
}

// A combining mark takes no column, so a text takes the same rows with its
// accents precomposed or as marks, and no row starts with a mark.
func TestWrapLineCountsACombiningMarkWithItsCharacter(t *testing.T) {
	for _, repeat := range []int{4, 30} {
		composed := norm.NFC.String(strings.Repeat("vérifié à la clôture, côté données agrégées équipe ", repeat))
		decomposed := norm.NFD.String(composed)
		require.Greater(t, utf8.RuneCountInString(decomposed), utf8.RuneCountInString(composed))
		for _, width := range []int{messageLineWidth, descriptionLineWidth, 40} {
			rows := wrapLine(decomposed, width)
			require.Len(t, rows, len(wrapLine(composed, width)), "%d columns at %d", lineColumns(composed), width)
			for _, row := range rows {
				require.LessOrEqual(t, lineColumns(row), width, row)
				first, _ := utf8.DecodeRuneInString(strings.TrimPrefix(row, continuedLine))
				require.False(t, unicode.Is(unicode.Mn, first), "%q starts with a mark", row)
			}
		}
	}
}

func TestWrapLineBreaksWhereALineReadsBest(t *testing.T) {
	for name, tc := range map[string]struct {
		line string
		want []string
	}{
		"at a space, as before, when that costs no row": {
			`Host "prod-payments-db.c1abcdefghij.us-east-1.rds.amazonaws.com", port 5432, user "bytebase", TLS required and verified`,
			[]string{
				`Host "prod-payments-db.c1abcdefghij.us-east-1.rds.amazonaws.com",`,
				`> port 5432, user "bytebase", TLS required and verified`,
			},
		},
		"after a dot of a host when the space before the host would cost a row": {
			`Host "payments-ledger-prod.cluster-c1abcdefghij.us-east-1.rds.amazonaws.com", port 5432, user "bytebase", TLS required and verified`,
			[]string{
				`Host "payments-ledger-prod.cluster-c1abcdefghij.us-east-1.rds.`,
				`>amazonaws.com", port 5432, user "bytebase", TLS required and verified`,
			},
		},
		"after a dash of a path": {
			`Any GitLab CI job in "acme/platform/data/payments-ledger/schema-migrations" signs in as it.`,
			[]string{
				`Any GitLab CI job in "acme/platform/data/payments-ledger/schema-`,
				`>migrations" signs in as it.`,
			},
		},
		"before the space that follows a full row": {
			strings.Repeat("a", 70) + " " + strings.Repeat("b", 10),
			[]string{strings.Repeat("a", 70), "> " + strings.Repeat("b", 10)},
		},
		"before a value's own space, which the next row keeps, and at the width": {
			`Column "` + strings.Repeat("c", 60) + ` name" and "` + strings.Repeat("d", 80) + `"`,
			[]string{
				`Column "` + strings.Repeat("c", 60),
				`> name" and "` + strings.Repeat("d", 57),
				`>` + strings.Repeat("d", 23) + `"`,
			},
		},
		"a line that fits": {"Environment \"prod\".", []string{`Environment "prod".`}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, wrapMessageLine(tc.line))
		})
	}
}

func TestIssuesLineCountsNamesThatDoNotFit(t *testing.T) {
	long := "projects/" + strings.Repeat("p", 40) + "/issues/1"
	require.Equal(t, "Plan edits recheck your 1 issue awaiting approval under these rules.", issuesLine([]string{shortIssueName(long)}, false))
	require.Equal(t, "Plan edits recheck your 2 issues awaiting approval under these rules.", issuesLine([]string{shortIssueName(long), "hr#1"}, false))
	require.Equal(t, `Plan edits recheck your "hr#1" under these rules.`, issuesLine([]string{"hr#1"}, false))
	require.Empty(t, issuesLine(nil, false))
	require.Equal(t, "Plan edits recheck your issues awaiting approval under these rules.", issuesLine([]string{shortIssueName(long), "hr#1"}, true))
	require.Equal(t, `Plan edits recheck your "hr#1" and more under these rules.`, issuesLine([]string{"hr#1"}, true))
	// A three-digit count does not fit with its number.
	hundred := slices.Repeat([]string{shortIssueName(long)}, 150)
	require.Equal(t, "Plan edits recheck your issues awaiting approval under these rules.", issuesLine(hundred, false))
}

func TestIssuesLineFitsOneLineForEveryNameAndCount(t *testing.T) {
	for length := 1; length <= 60; length++ {
		for _, count := range []int{1, 2, 3, 5, 9, 10, 99, 100, 200} {
			for _, more := range []bool{false, true} {
				names := make([]string, count)
				for i := range names {
					names[i] = fmt.Sprintf("%s#%d", strings.Repeat("p", length), i+1)
				}
				line := issuesLine(names, more)
				require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
				require.True(t, strings.HasPrefix(line, "Plan edits recheck your ") && strings.HasSuffix(line, " under these rules."), line)
			}
		}
	}
}

func TestAdministerWorkspaceRefusesAnAnswerToRulesThatChanged(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	// Another admin adds a rule while the form is open.
	user.whileAsked = func() { mock.setStored(fallbackRule(), testRule) }
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, res), approvalRules.reread())
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
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
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
	// Another admin renames the rule the request removes. A Remove line names
	// no title, so the message stays the same.
	mock.setStored(fallbackRule(), changeRule("New", "resource.environment_id == 'x'"))

	params := administerParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

func TestAdministerWorkspaceDescribesTheRulesLeft(t *testing.T) {
	described := changeRule("Prod", "resource.environment_id == 'prod'", "roles/workspaceDBA")
	described.Template.Description = "Production"
	hrFallback := &v1pb.WorkspaceApprovalSetting_Rule{Condition: &expr.Expr{Expression: `resource.project_id == "hr"`},
		Template: &v1pb.ApprovalTemplate{Title: "HR", Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}}}
	emptyFallback := &v1pb.WorkspaceApprovalSetting_Rule{Condition: &expr.Expr{},
		Template: &v1pb.ApprovalTemplate{Title: "Empty", Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/workspaceDBA"}}}}
	for name, tc := range map[string]struct {
		stored, requested []*v1pb.WorkspaceApprovalSetting_Rule
		want              string
	}{
		"an added rule's description": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), described},
			want:      "1. \"Prod\" (new), \"Workspace DBA\" (description \"Production\")",
		},
		"a kept rule": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), described},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), described, changeRule("Test", "resource.environment_id == 'test'")},
			want:      "1. \"Prod\" (kept), \"Workspace DBA\": `resource.environment_id == 'prod'` (description \"Production\")\n2. \"Test\" (new), no approval",
		},
		"no rule left": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), prodRule},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()},
			want: "Unmatched changes go to the fallback rules: \"Fallback Rule\" (\"Project owner\").\n" +
				"No rule for database changes is left, so the fallback rules decide.",
		},
		"no fallback rule": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{prodRule},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{},
			want:      "A change no rule matches needs no approval.\nNo rule for database changes is left.",
		},
		"a fallback rule with a condition": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{hrFallback},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{hrFallback, described},
			want: "Unmatched changes go to the fallback rules: \"HR\" (\"Project owner\") when `resource.project_id == \"hr\"`. " +
				"A change no fallback rule matches needs no approval.\n",
		},
		"a moved rule names its approvers": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), described, changeRule("Test", "resource.environment_id == 'test'")},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), changeRule("Test", "resource.environment_id == 'test'"), described},
			want:      "2. \"Prod\" (changed), \"Workspace DBA\" (description \"Production\")",
		},
		"a fallback rule with no condition, which never matches": {
			stored:    []*v1pb.WorkspaceApprovalSetting_Rule{emptyFallback, fallbackRule()},
			requested: []*v1pb.WorkspaceApprovalSetting_Rule{emptyFallback, fallbackRule(), described},
			want:      "Unmatched changes go to the fallback rules: \"Fallback Rule\" (\"Project owner\").\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			diff, err := diffApprovalRules(tc.stored, tc.requested, "")
			require.NoError(t, err)
			form, err := approvalRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "", "")
			require.NoError(t, err)
			require.Contains(t, unwrapMessage(form.description), tc.want)
		})
	}
}

func TestAdministerWorkspaceReportsTheLockedWriteRefusal(t *testing.T) {
	mock := &approvalMock{refuse: "failed_precondition"}
	mock.setStored(fallbackRule())
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})

	res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, res), "the approval rules changed after the user was asked")
	require.Contains(t, resultText(t, res), approvalRules.reread())
	require.Len(t, mock.calls(), 1)
}

func TestAdministerWorkspaceRefusesAnExpiredAnswer(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	s, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	body := approvalBody(t, fallbackRule(), prodRule)

	params := administerParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
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
	requireEveryFieldClassified(t, (&v1pb.UpdateSettingRequest{}).ProtoReflect().Descriptor(), "", approvalRequestFields, refused)
}

// requireEveryFieldClassified fails for a field under md that an entry neither
// takes nor refuses. An entry serves one member of a oneof; the others are
// other settings or policies, so the walk skips them.
func requireEveryFieldClassified(t *testing.T, md protoreflect.MessageDescriptor, prefix string, taken, refused map[string]bool) {
	t.Helper()
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		path := string(fd.Name())
		if prefix != "" {
			path = prefix + "." + path
		}
		if oneof := fd.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() && !taken[path] {
			continue
		}
		if refused[path] {
			continue
		}
		require.True(t, taken[path], "%s is neither taken nor refused", path)
		if fd.Kind() == protoreflect.MessageKind && !fd.IsMap() {
			requireEveryFieldClassified(t, fd.Message(), path, taken, refused)
		}
	}
}
