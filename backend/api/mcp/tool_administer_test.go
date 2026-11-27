package mcp

import (
	"cmp"
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
	"unicode/utf16"
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
		// The API takes a rule with no condition, and the form must not read
		// two of them as no duplicate: it would call the renamed rule removed.
		"two requested rules with no condition, and a rename": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), prodRule},
			arguments: updateArguments(approvalBody(t, fallbackRule(), changeRule("Renamed", "resource.environment_id == 'prod'", "roles/workspaceDBA"), changeRule("One", ""), changeRule("Two", ""))),
			want:      []string{"DUPLICATE_CONDITION", "two requested rules", "keep one rule per condition"}},
		"two stored rules with no condition": {stored: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), prodRule, changeRule("One", ""), changeRule("Two", "")},
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
// what lets the count line read the removals off the marks. B changes and
// trades places with A: B's line already states a change, so the move is B's
// and A has no line.
func TestAdministerWorkspaceGivesEachChangedRuleOneLine(t *testing.T) {
	a := changeRule("A", "resource.environment_id == 'a'", "roles/projectOwner")
	b := changeRule("B", "resource.environment_id == 'b'", "roles/projectOwner")
	c := changeRule("C", "resource.environment_id == 'c'", "roles/projectOwner")
	renamed := changeRule("B2", "resource.environment_id == 'b'", "roles/workspaceDBA")
	renamed.Template.Description = "now with a description"
	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), a, b, c},
		[]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), renamed, changeRule("D", "resource.environment_id == 'd'"), a}, "")
	require.NoError(t, err)
	require.Equal(t, []string{"changed", "new", "kept"}, diff.marks)
	require.Equal(t, []string{
		"Remove, \"Project owner\": `resource.environment_id == 'c'`",
		"Change, \"Project owner\" to \"Workspace DBA\"; move to rule 1, now ahead of \"A\" (\"Project owner\"); rename to \"B2\"; change description to \"now with a description\": `resource.environment_id == 'b'`",
		"Add rule 2, no approval: `resource.environment_id == 'd'`",
	}, diff.changes)
	require.Equal(t, "Remove 1 rule, add 1 and change 1, listed under Action.", countLine(len(diff.changes), diff.marks, 0))
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
	rules := []string{`1. "A" (kept), "Project owner": ` + "`resource.environment_id == 'a'`", `2. "B" (kept), no approval: ` + "`resource.environment_id == 'b'`", `3. "C" (new), no approval`}
	list := ruleSection{name: "Rules for database changes", header: "The rules for database changes, in the order an approval check tries them:",
		total: len(rules), rule: func(i int) string { return rules[i] }}
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
	require.Equal(t, []string{"Masking rules in all: 1. Their list has no room here."}, sectionRows(ruleSection{name: "Masking rules", header: list.header, total: 1, rule: list.rule}, 2))
}

// The list's rows are built as they are tried: the rules a description shows
// and the first it has no room for. A workspace can store thousands of rules.
func TestSectionRowsBuildsOnlyTheRowsItTries(t *testing.T) {
	built := 0
	list := ruleSection{name: "Masking rules", header: "The masking rules, in the order masking tries them:", total: 5000, rule: func(i int) string {
		built++
		return fmt.Sprintf(`%d. "Rule" (kept), Full mask: every column`, i+1)
	}}
	rows := sectionRows(list, descriptionLines)
	require.Len(t, rows, descriptionLines)
	require.Equal(t, "... and 4992 more rules, tried after these.", rows[descriptionLines-1])
	require.Equal(t, 9, built)
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
	// Staging's line already states a change, so the move is Staging's and
	// one line states both. Staging ends with Prod's approvers, which the line
	// reads from the request.
	require.Equal(t, strings.Join([]string{
		"Change the approval rules for database changes, as alice@example.com?",
		`Change, "Project owner" to "Workspace DBA"; move to rule 1, now ahead`,
		"> of 1 rule with the same approvers: `resource.environment_id ==",
		"> 'staging'`",
	}, "\n"), asked.Message)
	require.True(t, strings.HasPrefix(choiceDescription(t, asked), "Unmatched changes go to the fallback rules: "), choiceDescription(t, asked))
}

// The rule list under the choice can be cut before the rules a move touches,
// so the move's line says who approves on each side of it: it names the one
// rule it passes when that rule has other approvers, and counts one with the
// moved rule's own. Two workspaces that differ in the moved rule's approvers,
// or in those of the rule it passes, get different forms for the same move.
func TestAdministerWorkspaceMoveNamesTheRuleItPasses(t *testing.T) {
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

// passedRule is a requested rule as wantPassedOf reads it: its condition, what
// it gives the changes or columns it matches, whether that is the least a rule
// can give, and the label a move's line names it by.
type passedRule struct {
	condition, gives, label string
	least                   bool
}

// wantPassedOf derives what a move's line says of the rules that
// requested[moved] traded order with, from the rules' places alone: on each
// side a lone rule that gives something else by its label, when that is within
// passedNameWidth, and any other side by its counts, in the words of a rule
// form: for the rules that give the same, the least, and something else.
func wantPassedOf(stored []string, requested []passedRule, moved int, same, least, other string) (clause string, named, counted int) {
	was := map[string]int{}
	for i, condition := range stored {
		was[condition] = i
	}
	place := func(i int) int { return was[requested[i].condition] }
	var sides []string
	for _, behind := range []bool{true, false} {
		label := ""
		total, shared, none := 0, 0, 0
		for i, rule := range requested {
			if i == moved || (i < moved) != behind || (place(i) < place(moved)) == (i < moved) {
				continue
			}
			total++
			switch {
			case rule.gives == requested[moved].gives:
				shared++
				continue
			case rule.least:
				none++
			default:
			}
			label = rule.label
		}
		others := total - shared
		text := ""
		switch {
		case total == 0:
			continue
		case total == 1 && others == 1 && utf8.RuneCountInString(label) <= 48:
			text = label
			named++
		case others == 0:
			text = countRules(shared) + " " + same
		case none == others && shared == 0:
			text = countRules(total) + " " + least
		case none == others:
			text = fmt.Sprintf("%s, %d of them %s", countRules(total), none, least)
		case none > 0:
			text = fmt.Sprintf("%s, %d %s and %d %s", countRules(total), none, least, others-none, other)
		case shared == 0:
			text = countRules(total) + " " + other
		default:
			text = fmt.Sprintf("%s, %d of them %s", countRules(total), others, other)
		}
		if text != label {
			counted++
		}
		sides = append(sides, map[bool]string{true: "behind ", false: "ahead of "}[behind]+text)
	}
	return strings.Join(sides, ", and "), named, counted
}

// wantPassed is wantPassedOf for approval rules: a rule gives its approvers,
// and the least is no approval.
func wantPassed(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule, moved int) (clause string, named, counted int) {
	var conditions []string
	for _, rule := range stored {
		conditions = append(conditions, rule.GetCondition().GetExpression())
	}
	var rules []passedRule
	for i, rule := range requested {
		name := fmt.Sprintf("rule %d", i+1)
		if title := rule.GetTemplate().GetTitle(); title != "" {
			name = fmt.Sprintf("%q", title)
		}
		roles := rule.GetTemplate().GetFlow().GetRoles()
		rules = append(rules, passedRule{condition: rule.GetCondition().GetExpression(), gives: strings.Join(roles, "\n"), least: len(roles) == 0, label: fmt.Sprintf("%s (%s)", name, approvers(rule))})
	}
	return wantPassedOf(conditions, rules, moved, "with the same approvers", "with no approval", "with other approvers")
}

// wantMoveLine derives the change line of requested[moved] as a moved rule
// that changes in no other way, from the two lists alone.
func wantMoveLine(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule, moved int) string {
	passed, _, _ := wantPassed(stored, requested, moved)
	return fmt.Sprintf("Move to rule %d, %s, now %s: %s", moved+1, approvers(requested[moved]), passed, conditionText(requested[moved].GetCondition().GetExpression()))
}

// Reordering changes who approves only for two rules that trade order, and one
// of the two is always a moved rule. So for each moved rule and each side of
// it, the line accounts for every rule it traded order with: it names a lone
// rule with other approvers, and counts any other side, the rules with no
// approval apart. The trades are counted here from the rules' places.
func TestApprovalMoveLinesAccountForEveryPairThatChangedOrder(t *testing.T) {
	random := rand.New(rand.NewPCG(22234, 6))
	flows := [][]string{nil, {"roles/projectOwner"}, {"roles/workspaceDBA"}}
	moveLine := regexp.MustCompile(`^Move to rule (\d+), `)
	moves, both, named, counted := 0, 0, 0, 0
	for range 400 {
		n := 2 + random.IntN(6)
		stored := make([]*v1pb.WorkspaceApprovalSetting_Rule, n)
		for i := range stored {
			stored[i] = changeRule(fmt.Sprintf("R%d", i), fmt.Sprintf("resource.environment_id == 'e%d'", i), flows[random.IntN(len(flows))]...)
		}
		requested := slices.Clone(stored)
		random.Shuffle(n, func(i, j int) { requested[i], requested[j] = requested[j], requested[i] })
		diff, err := diffApprovalRules(stored, requested, "")
		require.NoError(t, err)
		movedPlaces := map[int]bool{}
		for _, line := range diff.changes {
			match := moveLine.FindStringSubmatch(line)
			require.NotNil(t, match, line)
			place, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			movedPlaces[place-1] = true
			require.Equal(t, wantMoveLine(stored, requested, place-1), line)
			passed, names, counts := wantPassed(stored, requested, place-1)
			moves++
			named += names
			counted += counts
			if strings.Contains(passed, "behind ") && strings.Contains(passed, "ahead of ") {
				both++
			}
		}
		// Every rule outside the lines kept its order with every other one.
		for x := range requested {
			for y := range x {
				if slices.Index(stored, requested[x]) < slices.Index(stored, requested[y]) {
					require.True(t, movedPlaces[x] || movedPlaces[y], "rules %d and %d traded order, and neither has a line: %q", y+1, x+1, diff.changes)
				}
			}
		}
	}
	require.Greater(t, moves, 400)
	require.Positive(t, both, "the sweep must reach a rule that is behind one rule and ahead of another")
	require.Greater(t, named, 100)
	require.Greater(t, counted, 400)
}

// A move's line names the one rule it passes on a side when that rule's label,
// its title or place with its approvers, is within passedNameWidth, and counts
// it otherwise. A label is measured in columns, or in characters when they are
// more, since a combining mark takes no column.
func TestApprovalMoveNamesAPassedRuleWithinItsWidth(t *testing.T) {
	const approvers = ` ("Project owner")`
	mover := changeRule("Mover", `resource.project_id == "mover"`, "roles/workspaceDBA")
	for name, tc := range map[string]struct {
		title string
		named bool
	}{
		"48 columns":                       {strings.Repeat("T", 48-2-len(approvers)), true},
		"49 columns":                       {strings.Repeat("T", 49-2-len(approvers)), false},
		"48 columns of wide characters":    {strings.Repeat("\u5bbd", (48-2-len(approvers))/2), true},
		"50 columns of wide characters":    {strings.Repeat("\u5bbd", (50-2-len(approvers))/2), false},
		"48 characters, combining marks":   {"T" + strings.Repeat("\u0301", 47-2-len(approvers)), true},
		"49 characters in fewer columns":   {"T" + strings.Repeat("\u0301", 48-2-len(approvers)), false},
		"no title, so the rule's place":    {"", true},
		"a title of any length is counted": {strings.Repeat("T", 20000), false},
	} {
		// The passed rule's own move takes more rows, so the mover is the moved one.
		passed := changeRule(tc.title, projectsCondition("p", 8), "roles/projectOwner")
		diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), mover, passed}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), passed, mover}, "")
		require.NoError(t, err, name)
		want := "1 rule with other approvers"
		switch {
		case tc.named && tc.title == "":
			want = "rule 1" + approvers
		case tc.named:
			want = fmt.Sprintf("%q", tc.title) + approvers
		default:
		}
		require.Equal(t, []string{"Move to rule 2, \"Workspace DBA\", now behind " + want + ": `resource.project_id == \"mover\"`"}, diff.changes, name)
	}

	// One rule with other approvers among the rules a side passes is counted
	// with them.
	same := changeRule("Same", `resource.project_id == "same"`, "roles/workspaceDBA")
	passed := changeRule("Passed", projectsCondition("p", 8), "roles/projectOwner")
	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), mover, same, passed}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), same, passed, mover}, "")
	require.NoError(t, err)
	require.Equal(t, []string{"Move to rule 3, \"Workspace DBA\", now behind 2 rules, 1 of them with other approvers: `resource.project_id == \"mover\"`"}, diff.changes)
}

// A move that passes many rules counts them, so its line stays short however
// far the rule moves, and says how many of them need no approval. A rule too
// long for any form stays in place: the rules it traded order with are the
// moved ones.
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
		form, err := approvalRulesForm(alice, diff, "", "LINK")
		return form, diff, err
	}
	const finance = `resource.project_id == "finance"`

	whole, diff, err := form(5, finance)
	require.NoError(t, err)
	require.Equal(t, []string{"Move to rule 5, \"Workspace DBA\", now behind 4 rules, 2 with no approval and 1 with other approvers: `" + finance + "`"}, diff.changes)
	require.Contains(t, unwrapMessage(whole.message), diff.changes[0], "the counts leave the move in the message")

	whole, diff, err = form(12, finance)
	require.NoError(t, err)
	require.Equal(t, []string{"Move to rule 12, \"Workspace DBA\", now behind 11 rules, 4 with no approval and 4 with other approvers: `" + finance + "`"}, diff.changes)
	require.Contains(t, unwrapMessage(whole.message), diff.changes[0])

	// A condition the message has no room for: the same line leads the
	// description.
	long := strings.Repeat(`resource.project_id == "p" || `, 14) + "false"
	listed, _, err := form(12, long)
	require.NoError(t, err)
	require.Contains(t, listed.message, "Change 1 rule, listed under Action.")
	require.True(t, strings.HasPrefix(unwrapMessage(listed.description), "Move to rule 12, \"Workspace DBA\", now behind 11 rules, 4 with no approval and 4 with other approvers: `"+long+"`\n"), listed.description)

	// A condition too long for any form: Finance stays in place, and the
	// eleven rules it traded order with each move ahead of it. One form has no
	// room for eleven moves, and steps of a few rules do.
	_, diff, err = form(12, strings.Repeat(`resource.project_id == "p" || `, 30)+"false")
	var refused *toolError
	require.ErrorAs(t, err, &refused)
	require.Equal(t, "CHANGE_TOO_LONG", refused.Code)
	require.Equal(t, 11, diff.moves)
	require.Equal(t, "Move to rule 1, no approval, now ahead of \"Finance\" (\"Workspace DBA\"): `resource.project_id == \"payments-00\"`", diff.changes[0])
	require.Equal(t, "Move to rule 3, \"Workspace DBA\", now ahead of 1 rule with the same approvers: `resource.project_id == \"payments-02\"`", diff.changes[2])
	require.Equal(t, "this change needs 22 lines, more than the 8 one confirmation form has room for under its choice, so the approval rules were not changed", refused.Message)
	require.Contains(t, refused.Suggestion, "make the change in smaller steps, a few rules at a time")
	require.Contains(t, refused.Suggestion, "Or change the rules signed in to the Bytebase console: LINK")
}

// A move's line says how many of the rules it passes need no approval: a rule
// that moves behind one of them loses its approval there.
func TestApprovalMoveCountsTheRulesWithNoApprovalApart(t *testing.T) {
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
		require.Equal(t, []string{fmt.Sprintf("Move to rule %d, %s, %s: `resource.project_id == \"finance\"`", len(passed)+1, approvers(moved), tc.want)}, down.changes)
		up, err := diffApprovalRules(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, passed, []*v1pb.WorkspaceApprovalSetting_Rule{moved}), slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), moved}, passed), "")
		require.NoError(t, err)
		require.Equal(t, []string{fmt.Sprintf("Move to rule 1, %s, %s: `resource.project_id == \"finance\"`", approvers(moved), strings.Replace(tc.want, "now behind", "now ahead of", 1))}, up.changes)
	}
}

// A move behind rules with seven kinds of approvers counts them. The form that
// takes away an approval and the one that does not are still not the same
// form: the count says how many of the rules need no approval.
func TestAdministerWorkspaceMoveTellsARuleWithNoApprovalFromOneWithOtherApprovers(t *testing.T) {
	asked := func(condition string, production ...string) (message, description string) {
		finance := changeRule("Finance", condition, "roles/workspaceDBA")
		var environments []*v1pb.WorkspaceApprovalSetting_Rule
		for i, flow := range [][]string{
			{"roles/projectOwner"}, {"roles/projectDeveloper"}, {"roles/sqlEditorUser"}, {"roles/workspaceAdmin"}, production,
			{"roles/projectOwner", "roles/workspaceDBA"}, {"roles/workspaceAdmin", "roles/projectOwner"},
		} {
			environments = append(environments, changeRule(fmt.Sprintf("Environment %d schema and data changes, all projects", i), fmt.Sprintf(`resource.environment_id == "e%d"`, i), flow...))
		}
		message, description, refusal := askedApprovalChange(t, slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{finance}, environments), slices.Concat(environments, []*v1pb.WorkspaceApprovalSetting_Rule{finance}))
		require.Empty(t, refusal)
		return message, description
	}
	const finance = `resource.project_id == "finance"`
	reviewed, _ := asked(finance, "roles/projectReleaser")
	unapproved, _ := asked(finance)
	require.Contains(t, reviewed, `Move to rule 8, "Workspace DBA", now behind 7 rules with other approvers: `+"`"+finance+"`")
	require.Contains(t, unapproved, `Move to rule 8, "Workspace DBA", now behind 7 rules, 1 with no approval and 6 with other approvers: `+"`"+finance+"`")

	// A condition the message has no room for: the lines under the choice
	// tell the two apart as well.
	long := strings.Repeat(`resource.project_id == "p" || `, 12) + "false"
	message, reviewed := asked(long, "roles/projectReleaser")
	require.Contains(t, message, "Change 1 rule, listed under Action.")
	_, unapproved = asked(long)
	require.Contains(t, reviewed, `Move to rule 8, "Workspace DBA", now behind 7 rules with other approvers: `)
	require.Contains(t, unapproved, `Move to rule 8, "Workspace DBA", now behind 7 rules, 1 with no approval and 6 with other approvers: `)
}

// askedApprovalChange sends requested to a workspace that stores stored, each
// after the fallback rule, and returns the form the user was asked, with its
// lines joined again, or the result when none was.
func askedApprovalChange(t *testing.T, stored, requested []*v1pb.WorkspaceApprovalSetting_Rule) (message, description, refusal string) {
	t.Helper()
	mock := &approvalMock{}
	mock.setStored(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, stored)...)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, administerParams(approvalBody(t, slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, requested)...)))
	if user.asked.Load() == 0 {
		return "", "", resultText(t, res)
	}
	asked := user.lastParams(t)
	return unwrapMessage(asked.Message), unwrapMessage(choiceDescription(t, asked)), ""
}

// projectsCondition matches n projects, in about 32 characters each.
func projectsCondition(prefix string, n int) string {
	var terms []string
	for i := range n {
		terms = append(terms, fmt.Sprintf(`resource.project_id == "%s%02d"`, prefix, i))
	}
	return strings.Join(terms, " || ")
}

// A reorder has more than one description: when two rules trade places,
// either one moved. A form states the move that takes fewer rows, so a swap
// that a form makes, a form swaps back.
func TestAdministerWorkspaceSwapsBackWhatItSwaps(t *testing.T) {
	swap := func(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule) (message, refusal string) {
		message, _, refusal = askedApprovalChange(t, stored, requested)
		return message, refusal
	}
	projects := func(n int) string { return projectsCondition("p", n) }
	production := changeRule("Production", `resource.environment_id == "prod"`, "roles/workspaceDBA")
	for n := 4; n <= 26; n++ {
		many := changeRule("Core projects", projects(n), "roles/projectOwner")
		message, refusal := swap([]*v1pb.WorkspaceApprovalSetting_Rule{production, many}, []*v1pb.WorkspaceApprovalSetting_Rule{many, production})
		require.Empty(t, refusal, "%d projects", n)
		require.Contains(t, message, `Move to rule 2, "Workspace DBA", now behind "Core projects" ("Project owner")`)
		_, refusal = swap([]*v1pb.WorkspaceApprovalSetting_Rule{many, production}, []*v1pb.WorkspaceApprovalSetting_Rule{production, many})
		require.Empty(t, refusal, "%d projects, swapped back", n)
	}

	// The move that takes fewer rows is the one stated, whichever way the
	// two rules trade places, also where both moves fit a form.
	for _, n := range []int{4, 26} {
		core := changeRule("Core projects", projects(n), "roles/projectOwner")
		message, _ := swap([]*v1pb.WorkspaceApprovalSetting_Rule{core, production}, []*v1pb.WorkspaceApprovalSetting_Rule{production, core})
		require.Contains(t, message, `Move to rule 1, "Workspace DBA", now ahead of "Core projects" ("Project owner"): `+"`"+`resource.environment_id == "prod"`+"`", "%d projects", n)
	}
	// Two moves that take the same rows: the earlier rule of the request
	// stays in place, so the rule that ends second is the moved one.
	staging := changeRule("Staging", `resource.environment_id == "test"`, "roles/workspaceDBA")
	message, _ := swap([]*v1pb.WorkspaceApprovalSetting_Rule{staging, production}, []*v1pb.WorkspaceApprovalSetting_Rule{production, staging})
	require.Contains(t, message, `Move to rule 2, "Workspace DBA", now behind 1 rule with the same approvers: `+"`"+`resource.environment_id == "test"`+"`")
	// Two rules that are each too long for any form cannot trade places in
	// one, by either description or in steps.
	long := changeRule("Core projects", projects(26), "roles/projectOwner")
	other := changeRule("Other projects", projectsCondition("q", 26), "roles/workspaceDBA")
	_, refusal := swap([]*v1pb.WorkspaceApprovalSetting_Rule{long, other}, []*v1pb.WorkspaceApprovalSetting_Rule{other, long})
	require.Contains(t, refusal, "CHANGE_TOO_LONG")
	require.Contains(t, refusal, "Move to rule 2, ")
	require.Contains(t, refusal, "Other projects", "the move names the rule it is now behind")
	require.NotContains(t, refusal, "Move to rule 1, ")
	require.Contains(t, refusal, "make that change signed in to the Bytebase console")
	require.NotContains(t, refusal, "smaller steps")
}

// Two swaps in one request, each of a long rule and a short one. A long rule's
// move fits no form, so the two short rules are the moved ones, and their
// lines fit.
func TestAdministerWorkspaceDescribesAReorderByTheMovesThatFit(t *testing.T) {
	core := changeRule("Core projects", projectsCondition("a", 19), "roles/projectOwner")
	production := changeRule("Production", `resource.environment_id == "prod"`, "roles/workspaceDBA")
	staging := changeRule("Staging", `resource.environment_id == "staging"`, "roles/projectReleaser")
	data := changeRule("Data projects", projectsCondition("d", 19), "roles/workspaceAdmin")
	stored := []*v1pb.WorkspaceApprovalSetting_Rule{core, production, staging, data}
	swapped := []*v1pb.WorkspaceApprovalSetting_Rule{production, core, data, staging}

	message, description, refusal := askedApprovalChange(t, stored, swapped)
	require.Empty(t, refusal)
	require.Contains(t, message, "Change 2 rules, listed under Action.")
	require.True(t, strings.HasPrefix(description,
		`Move to rule 1, "Workspace DBA", now ahead of "Core projects" ("Project owner"): `+"`"+`resource.environment_id == "prod"`+"`\n"+
			`Move to rule 4, "Project releaser", now behind "Data projects" ("Workspace admin"): `+"`"+`resource.environment_id == "staging"`+"`\n"), description)

	message, description, refusal = askedApprovalChange(t, swapped, stored)
	require.Empty(t, refusal)
	require.Contains(t, message, "Change 2 rules, listed under Action.")
	require.True(t, strings.HasPrefix(description,
		`Move to rule 2, "Workspace DBA", now behind "Core projects" ("Project owner"): `+"`"+`resource.environment_id == "prod"`+"`\n"+
			`Move to rule 3, "Project releaser", now ahead of "Data projects" ("Workspace admin"): `+"`"+`resource.environment_id == "staging"`+"`\n"), description)

	// The moves that fit can be more than the fewest: a long rule that moves
	// behind three short ones is described by theirs.
	qa := changeRule("QA", `resource.environment_id == "qa"`, "roles/projectOwner")
	message, description, refusal = askedApprovalChange(t,
		[]*v1pb.WorkspaceApprovalSetting_Rule{core, production, staging, qa}, []*v1pb.WorkspaceApprovalSetting_Rule{production, staging, qa, core})
	require.Empty(t, refusal)
	require.Contains(t, message, "Change 3 rules, listed under Action.")
	require.True(t, strings.HasPrefix(description,
		`Move to rule 1, "Workspace DBA", now ahead of "Core projects" ("Project owner"): `+"`"+`resource.environment_id == "prod"`+"`\n"+
			`Move to rule 2, "Project releaser", now ahead of "Core projects" ("Project owner"): `+"`"+`resource.environment_id == "staging"`+"`\n"+
			`Move to rule 3, "Project owner", now ahead of 1 rule with the same approvers: `+"`"+`resource.environment_id == "qa"`+"`\n"), description)

	// A rule whose move fits a form is the one moved when that takes the
	// fewest moves, though each of the three takes fewer rows.
	short := changeRule("Core projects", projectsCondition("a", 3), "roles/projectOwner")
	message, _, refusal = askedApprovalChange(t,
		[]*v1pb.WorkspaceApprovalSetting_Rule{short, production, staging, qa}, []*v1pb.WorkspaceApprovalSetting_Rule{production, staging, qa, short})
	require.Empty(t, refusal)
	require.Contains(t, message, `Move to rule 4, "Project owner", now behind 3 rules, 2 of them with other approvers: `+"`"+projectsCondition("a", 3)+"`")
}

// A reorder too long for one form is sent to smaller steps when steps can make
// it, however many rules moved, and to the console when two rules that are
// each too long for a form trade places. Each step here gets a form.
func TestAdministerWorkspaceAdvisesStepsForAReorderThatStepsCanMake(t *testing.T) {
	roles := []string{"roles/workspaceDBA", "roles/projectOwner", "roles/projectReleaser", "roles/workspaceAdmin", "roles/projectDeveloper", "roles/sqlEditorUser"}
	for _, pairs := range []int{6, 11, 12} {
		var stored, swapped []*v1pb.WorkspaceApprovalSetting_Rule
		for pair := range pairs {
			long := changeRule(fmt.Sprintf("Projects %d", pair), projectsCondition(fmt.Sprintf("g%d-", pair), 19), roles[pair%len(roles)])
			short := changeRule(fmt.Sprintf("Environment %d", pair), fmt.Sprintf(`resource.environment_id == "e%d"`, pair), roles[(pair+1)%len(roles)])
			stored, swapped = append(stored, long, short), append(swapped, short, long)
		}
		for _, request := range [][2][]*v1pb.WorkspaceApprovalSetting_Rule{{stored, swapped}, {swapped, stored}} {
			from, to := request[0], request[1]
			_, _, refusal := askedApprovalChange(t, from, to)
			require.Contains(t, refusal, "CHANGE_TOO_LONG", "%d pairs", pairs)
			require.Contains(t, refusal, fmt.Sprintf("this change needs %d lines, more than the 8 one confirmation form has room for", 2*pairs))
			require.Contains(t, refusal, "make the change in smaller steps, a few rules at a time")
			// The steps: one pair at a time.
			for pair := range pairs {
				step := slices.Clone(from)
				step[2*pair], step[2*pair+1] = to[2*pair], to[2*pair+1]
				message, _, refusal := askedApprovalChange(t, from, step)
				require.Empty(t, refusal, "%d pairs, pair %d alone", pairs, pair)
				require.Contains(t, message, "Move to rule ")
				from = step
			}
			require.Equal(t, to, from)
		}
	}

	// A long rule moves behind five short ones: one form has no room for five
	// moves, and steps of a few rules do.
	long := changeRule("Core projects", projectsCondition("a", 19), "roles/projectOwner")
	var short []*v1pb.WorkspaceApprovalSetting_Rule
	for i := range 5 {
		short = append(short, changeRule(fmt.Sprintf("Environment %d", i), fmt.Sprintf(`resource.environment_id == "e%d"`, i), roles[i%2*2]))
	}
	stored := slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{long}, short)
	_, _, refusal := askedApprovalChange(t, stored, slices.Concat(short, []*v1pb.WorkspaceApprovalSetting_Rule{long}))
	require.Contains(t, refusal, "this change needs 10 lines, more than the 8 one confirmation form has room for")
	require.Contains(t, refusal, "make the change in smaller steps, a few rules at a time")
	message, _, refusal := askedApprovalChange(t, stored, slices.Concat(short[:3], []*v1pb.WorkspaceApprovalSetting_Rule{long}, short[3:]))
	require.Empty(t, refusal)
	require.Contains(t, message, "Change 3 rules, listed under Action.")
	message, _, refusal = askedApprovalChange(t, slices.Concat(short[:3], []*v1pb.WorkspaceApprovalSetting_Rule{long}, short[3:]), slices.Concat(short, []*v1pb.WorkspaceApprovalSetting_Rule{long}))
	require.Empty(t, refusal)
	require.Contains(t, message, "Change 2 rules, listed under Action.")

	// Two rules too long for any form trade places among the swaps: no step
	// can make that one.
	other := changeRule("Other projects", projectsCondition("q", 19), "roles/workspaceDBA")
	_, _, refusal = askedApprovalChange(t, slices.Concat(stored, []*v1pb.WorkspaceApprovalSetting_Rule{other}), slices.Concat(short, []*v1pb.WorkspaceApprovalSetting_Rule{other, long}))
	require.Contains(t, refusal, "CHANGE_TOO_LONG")
	require.Contains(t, refusal, "make that change signed in to the Bytebase console")
	require.NotContains(t, refusal, "smaller steps")
}

// paddedRule returns a rule whose condition is 409 characters and pad more.
func paddedRule(title string, pad int) *v1pb.WorkspaceApprovalSetting_Rule {
	return changeRule(title, projectsCondition(title[:1], 12)+` || resource.project_id == "`+strings.Repeat("m", pad)+`"`, "roles/projectOwner")
}

// ruleWhoseMoveTakes returns the shortest paddedRule whose line takes rows
// under the choice, as the one moved rule of the request that order makes of
// it.
func ruleWhoseMoveTakes(t *testing.T, title string, rows int, order func(rule *v1pb.WorkspaceApprovalSetting_Rule) (stored, requested []*v1pb.WorkspaceApprovalSetting_Rule)) *v1pb.WorkspaceApprovalSetting_Rule {
	t.Helper()
	fallback := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
	for pad := range 200 {
		rule := paddedRule(title, pad)
		stored, requested := order(rule)
		diff, err := diffApprovalRules(slices.Concat(fallback, stored), slices.Concat(fallback, requested), "")
		require.NoError(t, err)
		require.Len(t, diff.changes, 1)
		require.Contains(t, diff.changes[0], rule.GetCondition().GetExpression(), "the padded rule is the moved one")
		if len(wrapLine(diff.changes[0], descriptionLineWidth)) == rows {
			return rule
		}
	}
	require.FailNow(t, "no condition length gives the move these rows", "%s: %d", title, rows)
	return nil
}

// A swap whose move fills the rows a form has for its changes gets a form
// alone. So a request that sends it with one more rule change is sent to
// smaller steps, and each step gets a form. One row more, and no form has room
// for the move: the refusal names the console, whichever change comes with it.
func TestAdministerWorkspaceAdvisesStepsWhenEachChangeFitsAFormAlone(t *testing.T) {
	type rules = []*v1pb.WorkspaceApprovalSetting_Rule
	qa := changeRule("QA", projectsCondition("q", 30), "roles/workspaceDBA")
	staging := changeRule("Staging", `resource.environment_id == "staging"`, "roles/projectOwner")
	reviewed := changeRule("Staging", `resource.environment_id == "staging"`, "roles/workspaceDBA")
	for _, ahead := range []bool{false, true} {
		// The swap of a rule with QA, which is too long for any form as moved:
		// the rule moves behind it, or ahead of it.
		swap := func(rule *v1pb.WorkspaceApprovalSetting_Rule) (stored, requested rules) {
			if ahead {
				return rules{qa, rule}, rules{rule, qa}
			}
			return rules{rule, qa}, rules{qa, rule}
		}
		stored, swapped := swap(ruleWhoseMoveTakes(t, "Core projects", 8, swap))
		for _, tc := range []struct {
			name              string
			stored, requested rules
		}{
			{"and a rule added", stored, slices.Concat(swapped, rules{staging})},
			{"and a rule removed", slices.Concat(stored, rules{staging}), swapped},
			{"and a rule's approvers changed", slices.Concat(stored, rules{staging}), slices.Concat(swapped, rules{reviewed})},
		} {
			_, _, refusal := askedApprovalChange(t, tc.stored, tc.requested)
			require.Contains(t, refusal, "CHANGE_TOO_LONG", "ahead %v, %s", ahead, tc.name)
			require.Contains(t, refusal, "make the change in smaller steps, a few rules at a time", "ahead %v, %s", ahead, tc.name)
			require.NotContains(t, refusal, "do not shorten", "ahead %v, %s", ahead, tc.name)
			// The steps: the swap, then the other change.
			step := slices.Concat(swapped, tc.stored[2:])
			message, description, refusal := askedApprovalChange(t, tc.stored, step)
			require.Empty(t, refusal, "ahead %v, %s: the swap alone", ahead, tc.name)
			require.Contains(t, message, "Change 1 rule, listed under Action.")
			require.Contains(t, description, `"QA" ("Workspace DBA"): `, "the move names the rule it passed")
			_, _, refusal = askedApprovalChange(t, step, tc.requested)
			require.Empty(t, refusal, "ahead %v, %s: the other change alone", ahead, tc.name)
		}

		stored, swapped = swap(ruleWhoseMoveTakes(t, "Core projects", 9, swap))
		for _, request := range [][2]rules{
			{stored, swapped},
			{stored, slices.Concat(swapped, rules{staging})},
			{slices.Concat(stored, rules{staging}), swapped},
		} {
			_, _, refusal := askedApprovalChange(t, request[0], request[1])
			require.Contains(t, refusal, "make that change signed in to the Bytebase console", "ahead %v", ahead)
			require.NotContains(t, refusal, "smaller steps", "ahead %v", ahead)
		}
	}
}

// A rule whose move fills the rows a form has for its changes is the moved
// one. With one row more its move fits no form, so it stays in place and the
// two rules it traded order with move instead.
func TestApprovalReorderLeavesInPlaceARuleWhoseMoveFitsNoForm(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	production := changeRule("Production", `resource.environment_id == "prod"`, "roles/workspaceDBA")
	staging := changeRule("Staging", `resource.environment_id == "staging"`, "roles/projectReleaser")
	diffOf := func(pad int) *approvalRulesDiff {
		core := paddedRule("Core projects", pad)
		diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), core, production, staging}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), production, staging, core}, "")
		require.NoError(t, err)
		return diff
	}
	pad := 0
	for diffOf(pad+1).moves == 1 {
		pad++
		require.Less(t, pad, 200)
	}
	const room = 8
	fits := diffOf(pad)
	require.Len(t, fits.changes, 1)
	require.True(t, strings.HasPrefix(fits.changes[0], `Move to rule 3, "Project owner", now behind 2 rules with other approvers: `), fits.changes[0])
	require.Len(t, wrapLine(fits.changes[0], descriptionLineWidth), room, "the last move that fits takes every row a form has")
	form, err := approvalRulesForm(alice, fits, "", "")
	require.NoError(t, err)
	require.Contains(t, form.message, "Change 1 rule, listed under Action.")

	stays := diffOf(pad + 1)
	require.Equal(t, []string{
		"Move to rule 1, \"Workspace DBA\", now ahead of \"Core projects\" (\"Project owner\"): `resource.environment_id == \"prod\"`",
		"Move to rule 2, \"Project releaser\", now ahead of \"Core projects\" (\"Project owner\"): `resource.environment_id == \"staging\"`",
	}, stays.changes)
	_, err = approvalRulesForm(alice, stays, "", "")
	require.NoError(t, err)
}

// everyDescription calls try with each way to describe a reorder: every set of
// rules that traded order with another whose moves leave the rest in stored
// order. was is each kept rule's stored place, in the order the request has
// them.
func everyDescription(was []int, try func(inOrder []bool)) {
	traded := make([]bool, len(was))
	for x := range was {
		for y := range was {
			traded[x] = traded[x] || x != y && (x < y) != (was[x] < was[y])
		}
	}
	for moved := range 1 << len(was) {
		inOrder := make([]bool, len(was))
		last, valid := -1, true
		for kept := range was {
			if moved>>kept&1 == 1 {
				valid = valid && traded[kept]
				continue
			}
			inOrder[kept] = true
			valid = valid && was[kept] > last
			last = was[kept]
		}
		if valid {
			try(inOrder)
		}
	}
}

// reorderedPlaces returns the places of n rules after a few neighbors trade
// places, or after a shuffle.
func reorderedPlaces(random *rand.Rand, n int) []int {
	if random.IntN(3) == 0 {
		return random.Perm(n)
	}
	places := make([]int, n)
	for i := range places {
		places[i] = i
	}
	for range 1 + random.IntN(4) {
		i := random.IntN(n - 1)
		places[i], places[i+1] = places[i+1], places[i]
	}
	return places
}

// advisesSteps reports whether a rule form's refusal sends the change to
// smaller steps, and not to the console alone.
func advisesSteps(t *testing.T, err error) bool {
	t.Helper()
	var refused *toolError
	require.ErrorAs(t, err, &refused)
	require.Equal(t, "CHANGE_TOO_LONG", refused.Code)
	require.NotContains(t, refused.Message, ", now :", "a rule that traded order with none is not moved")
	return strings.Contains(refused.Suggestion, "smaller steps")
}

// A rule that traded order with no other stays in every common order, so it is
// not priced as moved and gets no label: a workspace can store thousands of
// rules, and a request leaves most of them in place.
func TestKeptRuleLinesPricesOnlyRulesThatTradedOrder(t *testing.T) {
	named := 0
	rule := func(title string, was, position int) keptRule {
		stored := changeRule(title, fmt.Sprintf("resource.project_id == %q", title), "roles/workspaceDBA")
		return keptRule{was: was,
			name: func() string { named++; return fmt.Sprintf("%q (%s)", title, approvers(stored)) },
			line: func(moved bool, passed string) string { return keptRuleLine(stored, stored, moved, position, passed) }}
	}
	// Stored A, B, C, and requested A, C, B.
	inPlace, moved, costs := keptRuleLines([]keptRule{rule("A", 0, 1), rule("C", 2, 2), rule("B", 1, 3)}, 8, approversPassed)
	require.Equal(t, []string{"", "", ""}, inPlace)
	require.Equal(t, []string{
		"",
		"Move to rule 2, \"Workspace DBA\", now ahead of 1 rule with the same approvers: `resource.project_id == \"C\"`",
		"Move to rule 3, \"Workspace DBA\", now behind 1 rule with the same approvers: `resource.project_id == \"B\"`",
	}, moved)
	require.Equal(t, []moveCost{{}, {rows: 2}, {rows: 2}}, costs)
	require.Equal(t, 2, named, "each rule that traded order is named once")
}

// moveTotal is what the moves of one description of a reorder cost together, in
// the order a form weighs them.
type moveTotal struct{ tooLong, moves, rows int }

func (m moveTotal) less(other moveTotal) bool {
	return cmp.Or(cmp.Compare(m.tooLong, other.tooLong), cmp.Compare(m.moves, other.moves), cmp.Compare(m.rows, other.rows)) < 0
}

// add counts one move whose line takes rows, in a form with room rows for its
// changes.
func (m *moveTotal) add(rows, room int) {
	m.moves++
	m.rows += rows
	if rows > room {
		m.tooLong++
	}
}

// A reorder has more than one description: when two rules trade places,
// either one moved. Of every description, a form states one whose moves cost
// the least: the fewest lines too long for any form, then the fewest moves,
// then the fewest rows. Each description's lines are derived here from the
// rules' places. The request gets a form when the stated lines have room, is
// sent to smaller steps when each of them has room alone, and to the console
// otherwise.
func TestApprovalReorderIsStatedByItsCheapestMoves(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	random := rand.New(rand.NewPCG(22234, 12))
	roles := [][]string{nil, {"roles/workspaceDBA"}, {"roles/projectOwner"}, {"roles/projectOwner", "roles/workspaceDBA"}}
	titles := []string{"", "QA", "Production", "Payments platform schema and data changes"}
	// Fallback lines of one, two and three rows.
	staging := &v1pb.WorkspaceApprovalSetting_Rule{
		Condition: &expr.Expr{Expression: `resource.environment_id == "staging"`},
		Template:  &v1pb.ApprovalTemplate{Title: "Staging changes", Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner", "roles/workspaceDBA"}}},
	}
	fallbacks := [][]*v1pb.WorkspaceApprovalSetting_Rule{nil, {fallbackRule()}, {staging, fallbackRule()}}
	rooms := map[int]int{}
	var forms, toSteps, toConsole, moreMoves, fewerRows int
	for range 600 {
		fallback := fallbacks[random.IntN(len(fallbacks))]
		n := 2 + random.IntN(5)
		var stored []*v1pb.WorkspaceApprovalSetting_Rule
		for i := range n {
			// Conditions of every length, so that a column more or less
			// decides a row.
			terms := []int{1, 1, 2, 4, 8, 12, 16, 20}[random.IntN(8)]
			condition := projectsCondition(fmt.Sprintf("r%d-", i), terms) + strings.Repeat(" ", random.IntN(40))
			stored = append(stored, changeRule(titles[random.IntN(len(titles))], condition, roles[random.IntN(len(roles))]...))
		}
		was := reorderedPlaces(random, n)
		var requested []*v1pb.WorkspaceApprovalSetting_Rule
		for _, place := range was {
			requested = append(requested, proto.CloneOf(stored[place]))
		}
		diff, err := diffApprovalRules(slices.Concat(fallback, stored), slices.Concat(fallback, requested), "LINK")
		require.NoError(t, err)
		if diff.moves == 0 {
			continue
		}
		line, _ := fallbackLine(fallback)
		room := descriptionLines - len(wrapLine(line, descriptionLineWidth))
		rooms[room]++

		lines, rows := make([]string, n), make([]int, n)
		for i := range requested {
			lines[i] = wantMoveLine(stored, requested, i)
			rows[i] = len(wrapLine(lines[i], descriptionLineWidth))
		}
		var least, fewest *moveTotal
		everyDescription(was, func(inOrder []bool) {
			total := &moveTotal{}
			for i, in := range inOrder {
				if !in {
					total.add(rows[i], room)
				}
			}
			if least == nil || total.less(*least) {
				least = total
			}
			if fewest == nil || total.moves < fewest.moves || total.moves == fewest.moves && total.rows > fewest.rows {
				fewest = total
			}
		})
		var stated moveTotal
		var want []string
		for i, mark := range diff.marks {
			if mark == "changed" {
				stated.add(rows[i], room)
				want = append(want, lines[i])
			}
		}
		require.Equal(t, want, diff.changes)
		require.Equal(t, *least, stated, "stored %v\nrequested %v", stored, requested)
		require.Equal(t, stated.moves, diff.moves)

		_, err = approvalRulesForm(alice, diff, "", "LINK")
		switch {
		case stated.tooLong > 0:
			require.False(t, advisesSteps(t, err), "stored %v\nrequested %v", stored, requested)
			toConsole++
		case stated.rows > room:
			require.True(t, advisesSteps(t, err), "stored %v\nrequested %v", stored, requested)
			toSteps++
		default:
			require.NoError(t, err, "stored %v\nrequested %v", stored, requested)
			forms++
		}
		if stated.moves > fewest.moves {
			moreMoves++
		}
		if stated.moves == fewest.moves && stated.rows < fewest.rows {
			fewerRows++
		}
	}
	require.Len(t, rooms, 3, "the fallback line takes one, two and three rows: %v", rooms)
	require.Greater(t, forms, 100)
	require.Greater(t, toSteps, 50)
	require.Greater(t, toConsole, 50)
	require.Greater(t, moreMoves, 15, "reorders stated by more moves than the fewest, since a rule too long for a form stays in place")
	require.Greater(t, fewerRows, 50, "reorders whose fewest moves can take more rows than the ones stated")
}

// unweightedLongestCommonOrder is the longest common subsequence alone, and of
// several the one that keeps the earlier elements of b: what longestCommonOrder
// returns when every move costs the same.
func unweightedLongestCommonOrder(a, b []string) []bool {
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

// longestCommonOrder leaves in place a run of rules in stored order, never
// moves a rule that traded order with none, and no other set of moves costs
// less. Of the sets that cost the same it leaves in place the earlier rules of
// the request, so moves that all cost the same leave the unweighted longest
// common subsequence.
func TestLongestCommonOrderLeavesTheCheapestMoves(t *testing.T) {
	random := rand.New(rand.NewPCG(22234, 11))
	untraded, moreMoves, otherRules := 0, 0, 0
	for range 3000 {
		n := 1 + random.IntN(8)
		was := random.Perm(n)
		stored, requested := make([]string, n), make([]string, n)
		for kept, place := range was {
			requested[kept] = fmt.Sprintf("rule %d", place)
			stored[place] = requested[kept]
		}
		costs := make([]moveCost, n)
		for i := range costs {
			costs[i] = moveCost{tooLong: random.IntN(4) == 0, rows: random.IntN(4)}
		}
		traded := tradedOrder(was)
		for x := range was {
			inverted := false
			for y := range was {
				inverted = inverted || x != y && (x < y) != (was[x] < was[y])
			}
			require.Equal(t, inverted, traded[x], "%v, rule %d", was, x)
			if !inverted {
				untraded++
				costs[x] = moveCost{}
			}
		}
		// moved lists the rules a description calls moved, so the description
		// that leaves the earlier rules in place compares greatest.
		moved := func(inOrder []bool) []int {
			var rules []int
			for kept, in := range inOrder {
				if !in {
					rules = append(rules, kept)
				}
			}
			return rules
		}
		total := func(inOrder []bool) moveTotal {
			var total moveTotal
			for kept, in := range inOrder {
				if !in {
					total.moves++
					total.rows += costs[kept].rows
					if costs[kept].tooLong {
						total.tooLong++
					}
				}
			}
			return total
		}
		// Every description, and of the cheapest the one that leaves the
		// earlier rules in place.
		var least *moveTotal
		var earliest []bool
		fewest := n
		everyDescription(was, func(inOrder []bool) {
			found := total(inOrder)
			switch {
			case least == nil || found.less(*least):
				least, earliest = &found, inOrder
			case found == *least && slices.Compare(moved(inOrder), moved(earliest)) > 0:
				earliest = inOrder
			default:
			}
			fewest = min(fewest, found.moves)
		})

		inOrder := longestCommonOrder(stored, requested, costs)
		require.Equal(t, earliest, inOrder, "%v with the costs %v", was, costs)
		last := -1
		for kept := range was {
			if inOrder[kept] {
				require.Greater(t, was[kept], last, "%v: the rules left in place are in stored order", was)
				last = was[kept]
				continue
			}
			require.True(t, traded[kept], "%v: rule %d traded order with none", was, kept)
		}
		require.Equal(t, *least, total(inOrder), "%v with the costs %v", was, costs)
		if least.moves > fewest {
			moreMoves++
		}

		unweighted := unweightedLongestCommonOrder(stored, requested)
		require.Equal(t, unweighted, longestCommonOrder(stored, requested, make([]moveCost, n)), "%v with no costs", was)
		even := make([]moveCost, n)
		for i := range even {
			even[i] = moveCost{tooLong: costs[0].tooLong, rows: costs[0].rows}
		}
		require.Equal(t, unweighted, longestCommonOrder(stored, requested, even), "%v with the same cost for every move", was)
		if !slices.Equal(unweighted, inOrder) {
			otherRules++
		}
	}
	require.Greater(t, untraded, 300)
	require.Greater(t, moreMoves, 100, "the cheapest moves are often more than the fewest")
	require.Greater(t, otherRules, 300, "and often other rules than the longest common subsequence leaves")
}

// A move is priced by what it adds to the rule's line: a line that fits no
// form before anything else, then its rows.
func TestMoveLineCost(t *testing.T) {
	// The longest line of words that takes n rows.
	rows := func(n int) string {
		line := "word"
		for len(wrapLine(line+" word", descriptionLineWidth)) <= n {
			line += " word"
		}
		require.Len(t, wrapLine(line, descriptionLineWidth), n)
		return line
	}
	cost := func(moved, unmoved string, room int) moveCost {
		return moveLineCost(lineRows(moved), lineRows(unmoved), room)
	}
	require.Equal(t, moveCost{rows: 1}, cost("Move to rule 2: a", "", 8))
	require.Equal(t, moveCost{rows: 0}, cost("Change, a to b; move to rule 2: a", "Change, a to b: a", 8))
	require.Equal(t, moveCost{rows: 1}, cost(rows(3)+" word", rows(3), 8))
	require.Equal(t, moveCost{rows: 8}, cost(rows(8), "", 8))
	require.Equal(t, moveCost{rows: 3}, cost(rows(8), rows(5), 8))
	require.Equal(t, moveCost{tooLong: true, rows: 9}, cost(rows(9), "", 8))
	require.Equal(t, moveCost{tooLong: true, rows: 1}, cost(rows(9), rows(8), 8))
	require.Equal(t, moveCost{rows: 9}, cost(rows(9), "", 9))
	require.Equal(t, moveCost{rows: 1}, cost(rows(10), rows(9), 8), "a line too long without the move is too long by any description")
}

// A step is a request of its own: its rules decide which of two rules that
// trade places it calls moved, and which rule that move's line names. So a
// form that lists a move offers no steps, though each of its lines fits a
// message alone, and a form with no move still does.
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
	require.Zero(t, diff.moves)
	require.Len(t, diff.changes, 2)
	require.Contains(t, form.inSteps, "a rule or two at a time")

	form, diff = listed([]*v1pb.WorkspaceApprovalSetting_Rule{finance, prod}, []*v1pb.WorkspaceApprovalSetting_Rule{prod, finance, added})
	require.Equal(t, 1, diff.moves)
	require.Len(t, diff.changes, 2)
	for _, change := range diff.changes {
		require.True(t, alone(change), change)
	}
	require.Empty(t, form.inSteps)

	// Two moves, each of which a message has room for alone: the form lists
	// them and still offers no steps.
	var payments []*v1pb.WorkspaceApprovalSetting_Rule
	for i := range 4 {
		payments = append(payments, changeRule(fmt.Sprintf("Payments platform schema changes %02d", i), fmt.Sprintf(`resource.project_id == "payments-%02d"`, i), [][]string{nil, {"roles/projectOwner"}}[i%2]...))
	}
	stored := slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{finance, ledger}, payments)
	form, diff = listed(stored, slices.Concat(payments, []*v1pb.WorkspaceApprovalSetting_Rule{finance, ledger}))
	require.Equal(t, 2, diff.moves)
	for _, change := range diff.changes {
		require.True(t, alone(change), change)
		require.Contains(t, unwrapMessage(form.description), change)
	}
	require.Empty(t, form.inSteps)
}

// What a move's line says of the rules it passes is short whatever the
// workspace stores: it counts them, and names one only by a label within
// passedNameWidth.
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
	for _, line := range diff.changes {
		require.LessOrEqual(t, len(wrapLine(line, descriptionLineWidth)), 4, line)
	}

	// A rule the move passes alone, with a title or a list of approvers that
	// no line has room for: the move counts it. A combining mark takes no
	// column, so a title or a role of them is long in characters only, and a
	// wide character takes two, so a title of them is long in columns first.
	custom := func(count int) []string {
		var ids []string
		for i := range count {
			ids = append(ids, fmt.Sprintf("roles/custom-approver-%03d", i))
		}
		return ids
	}
	marks := strings.Repeat("\u0301", 20000)
	mover := changeRule("Mover", `resource.project_id == "mover"`, "roles/workspaceDBA")
	for name, passed := range map[string]*v1pb.WorkspaceApprovalSetting_Rule{
		"a long title":             changeRule(strings.Repeat("T", 20000), projectsCondition("p", 8), "roles/projectOwner"),
		"a title of marks":         changeRule("Marked"+marks, projectsCondition("p", 8), "roles/projectOwner"),
		"a role of marks":          changeRule("Marked role", projectsCondition("p", 8), "roles/marked"+marks),
		"a title of wide letters":  changeRule(strings.Repeat("\u5bbd", 400), projectsCondition("p", 8), "roles/projectOwner"),
		"two hundred approvers":    changeRule("Many approvers", projectsCondition("p", 8), custom(200)...),
		"an untitled rule's roles": changeRule("", projectsCondition("p", 8), custom(3)...),
	} {
		diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), mover, passed}, []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule(), passed, mover}, "")
		require.NoError(t, err, name)
		require.Equal(t, []string{"Move to rule 2, \"Workspace DBA\", now behind 1 rule with other approvers: `resource.project_id == \"mover\"`"}, diff.changes, name)
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
			want: []string{"Move to rule 1, \"Workspace DBA\", now ahead of 2 rules, 1 with no approval and 1 with other approvers: `resource.environment_id == 'prod'`"}},
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
// wrapLine's helpers: a row holds width columns, and twice as many UTF-16 code
// units.
func fillRows(line string, width int) int {
	rows, room, units := 1, width, 2*width
	for _, r := range line {
		columns, size := lineColumns(string(r)), codeUnits(string(r))
		if columns > room || size > units {
			rows++
			room, units = width-1, 2*(width-1)
		}
		room, units = room-columns, units-size
	}
	return rows
}

// clientCut is the most UTF-16 code units of a form's message, and of a
// description, that Claude Code 2.1.332 draws. It cuts the rest.
const clientCut = 2000

func codeUnits(text string) int {
	return len(utf16.Encode([]rune(text)))
}

// markedText returns up to most code units of text whose rows hold more code
// units than columns: letters under runs of combining marks, a narrow
// character outside the basic plane, which is two code units, and a wide one.
func markedText(random *rand.Rand, most int) string {
	var text strings.Builder
	for units := random.IntN(most + 1); units > 0; {
		switch random.IntN(6) {
		case 0:
			marks := 1 + random.IntN(60)
			text.WriteString("a" + strings.Repeat("\u0301", marks))
			units -= 1 + marks
		case 1:
			text.WriteString("\U0001D400")
			units -= 2
		case 2:
			text.WriteString("\u540d")
			units--
		case 3:
			text.WriteString(" ")
			units--
		default:
			text.WriteString("e\u0308")
			units -= 2
		}
	}
	return text.String()
}

// Claude Code cuts a form's message, and a description, at 2,000 UTF-16 code
// units, and a combining mark is a unit that takes no column. So the rows a
// form allows must keep both texts within the cut, whatever the request and
// the stored rules hold: a change past the cut would be confirmed unseen.
func TestRuleFormsKeepWithinTheClientsCut(t *testing.T) {
	random := rand.New(rand.NewPCG(22234, 2000))
	alice := withUserEmail(context.Background(), "alice@example.com")
	forms, refused := 0, 0
	check := func(form *adminForm, err error) {
		if err != nil {
			var tooLong *toolError
			require.ErrorAs(t, err, &tooLong)
			require.Equal(t, "CHANGE_TOO_LONG", tooLong.Code)
			refused++
			return
		}
		forms++
		require.LessOrEqual(t, codeUnits(form.message), clientCut, "%q", form.message)
		require.LessOrEqual(t, len(strings.Split(form.message, "\n")), messageLines, "%q", form.message)
		require.LessOrEqual(t, codeUnits(form.description), clientCut, "%q", form.description)
		require.LessOrEqual(t, len(strings.Split(form.description, "\n")), descriptionLines, "%q", form.description)
	}
	text := func(most int) string {
		if random.IntN(3) == 0 {
			return fmt.Sprintf("Plain %d", random.IntN(1000))
		}
		return markedText(random, most)
	}
	types, err := semanticTypesFromSetting([]*v1pb.SemanticTypeSetting_SemanticType{semanticType("marked", markedText(random, 400), nil)})
	require.NoError(t, err)
	for range 600 {
		fallback := fallbackRule()
		fallback.Template.Title = text(3000)
		stored := []*v1pb.WorkspaceApprovalSetting_Rule{fallback}
		var masks []*v1pb.MaskingRulePolicy_MaskingRule
		for i := range random.IntN(7) {
			rule := changeRule(text(300), fmt.Sprintf(`resource.project_id == "p%d%s"`, i, text(40)), [][]string{nil, {"roles/workspaceDBA"}}[random.IntN(2)]...)
			rule.Template.Description = text(200)
			stored = append(stored, rule)
			masks = append(masks, maskingRule(fmt.Sprintf("r%d", i), text(300), fmt.Sprintf(`resource.column_name == "c%d%s"`, i, text(40)), []string{"bb.default", "marked", ""}[random.IntN(3)]))
		}
		requested, requestedMasks := slices.Clone(stored), slices.Clone(masks)
		for range 1 + random.IntN(3) {
			at := 1 + random.IntN(len(stored))
			switch {
			case random.IntN(2) == 0 || at >= len(requested):
				// One rule with a value of any size, among changes of a few
				// units, or a few rules of middling size.
				most := []int{60, 60, 300, 3000}[random.IntN(4)]
				added := changeRule(text(most), fmt.Sprintf(`resource.project_id == "new%d%s"`, random.IntN(1<<30), text(most/10)), "roles/projectOwner")
				added.Template.Description = text(most)
				requested = append(requested, added)
				requestedMasks = append(requestedMasks, maskingRule(fmt.Sprintf("n%d", random.IntN(1<<30)), text(most), fmt.Sprintf(`resource.column_name == "new%d%s"`, random.IntN(1<<30), text(most/10)), "marked"))
			default:
				renamed := proto.CloneOf(requested[at])
				renamed.Template.Title, renamed.Template.Description = text(3000), text(300)
				requested[at] = renamed
				mask := proto.CloneOf(requestedMasks[at-1])
				mask.Condition.Title = text(3000)
				requestedMasks[at-1] = mask
			}
		}
		if len(requested) > 2 && random.IntN(2) == 0 {
			requested[1], requested[len(requested)-1] = requested[len(requested)-1], requested[1]
			requestedMasks[0], requestedMasks[len(requestedMasks)-1] = requestedMasks[len(requestedMasks)-1], requestedMasks[0]
		}
		diff, err := diffApprovalRules(stored, requested, "")
		require.NoError(t, err)
		if len(diff.changes) > 0 {
			check(approvalRulesForm(alice, diff, issuesLine([]string{"hr#12"}, false), ""))
		}
		maskDiff, err := diffMaskingRules(masks, requestedMasks, types, "")
		require.NoError(t, err)
		if len(maskDiff.changes) > 0 {
			check(maskingRulesForm(alice, maskDiff, ""))
		}
	}
	require.Greater(t, forms, 200, "requests of a few marks get a form")
	require.Greater(t, refused, 200, "and a change whose marks pass the rows gets none")
}

// allocated returns the bytes f allocates.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// A title of combining marks takes one row by its columns. Its code units
// decide its rows, so a change after it is never past the client's cut.
func TestAdministerWorkspaceCountsATextOfMarksByItsCodeUnits(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	marks := func(n int) string { return "a" + strings.Repeat("\u0301", n) }
	a := changeRule("Team A", `resource.project_id == "a"`, "roles/workspaceDBA")
	b := changeRule("Team B", `resource.project_id == "b"`, "roles/workspaceDBA")
	c := changeRule("Team C", `resource.project_id == "c"`, "roles/workspaceDBA")
	renamed := func(rule *v1pb.WorkspaceApprovalSetting_Rule, title string) *v1pb.WorkspaceApprovalSetting_Rule {
		rule = proto.CloneOf(rule)
		rule.Template.Title = title
		return rule
	}
	form := func(stored, requested []*v1pb.WorkspaceApprovalSetting_Rule) (*adminForm, error) {
		diff, err := diffApprovalRules(slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, stored), slices.Concat([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, requested), "LINK")
		require.NoError(t, err)
		return approvalRulesForm(alice, diff, "", "LINK")
	}

	// 3,000 marks are one row by their columns and 23 by their code units, so
	// no form states the rename, with or without another change after it.
	open := changeRule("Team B", `resource.project_id == "b"`)
	for _, requested := range [][]*v1pb.WorkspaceApprovalSetting_Rule{{renamed(a, marks(3000)), b}, {renamed(a, marks(3000)), open}} {
		_, err := form([]*v1pb.WorkspaceApprovalSetting_Rule{a, b}, requested)
		var refused *toolError
		require.ErrorAs(t, err, &refused)
		require.Equal(t, "CHANGE_TOO_LONG", refused.Code)
		require.Contains(t, refused.Message, "needs more lines than the 8 one confirmation form has room for under its choice")
	}

	// Three titles of 150 marks take two rows each: the form lists them, and
	// every change is within the cut.
	listed, err := form([]*v1pb.WorkspaceApprovalSetting_Rule{a, b, c}, []*v1pb.WorkspaceApprovalSetting_Rule{renamed(a, marks(150)), renamed(b, marks(150)), renamed(c, marks(150))})
	require.NoError(t, err)
	require.Contains(t, listed.message, "Change 3 rules, listed under Action.")
	require.LessOrEqual(t, codeUnits(listed.description), clientCut)
	for _, project := range []string{"a", "b", "c"} {
		require.Contains(t, unwrapMessage(listed.description), fmt.Sprintf("Rename to %q: `resource.project_id == %q`", marks(150), project))
	}

	// What the workspace stores cannot keep a change from its form: a kept
	// rule and a fallback rule with titles of marks leave the form its rows.
	fallback := fallbackRule()
	fallback.Template.Title = marks(3000)
	kept := renamed(a, marks(3000))
	diff, err := diffApprovalRules([]*v1pb.WorkspaceApprovalSetting_Rule{fallback, kept}, []*v1pb.WorkspaceApprovalSetting_Rule{fallback, kept, b}, "")
	require.NoError(t, err)
	added, err := approvalRulesForm(alice, diff, "", "")
	require.NoError(t, err)
	require.Contains(t, added.message, "Add rule 2, \"Workspace DBA\": `resource.project_id == \"b\"`")
	require.Equal(t, "Unmatched changes go to a fallback rule too long to name here.\nRules for database changes in all: 2. Their list has no room here.", added.description)
}

// A row holds its width in columns and twice its width in UTF-16 code units,
// whichever fills first. A combining mark is one unit, and a character outside
// the basic plane two.
func TestWrapLineFillsARowByItsCodeUnitsToo(t *testing.T) {
	const width = descriptionLineWidth
	marked, astral := "e\u0301", "\U0001D400"
	for name, tc := range map[string]struct {
		line string
		rows []int
	}{
		"as many units as columns":          {strings.Repeat("e", width), []int{width}},
		"one mark on each letter":           {strings.Repeat(marked, width), []int{2 * width}},
		"one mark on each letter, one more": {strings.Repeat(marked, width) + "\u0301", []int{2 * width, 2}},
		"two marks on each letter":          {strings.Repeat(marked+"\u0308", width), []int{2 * width, 1 + width}},
		"marks alone":                       {"e" + strings.Repeat("\u0301", 3*width), []int{2 * width, 1 + width + 1}},
		"characters of two units":           {strings.Repeat(astral, width), []int{2 * width}},
		"characters of two units, marked":   {strings.Repeat(astral+"\u0301", width), []int{2*width - 1, 1 + width + 1}},
	} {
		t.Run(name, func(t *testing.T) {
			var units []int
			for _, row := range wrapLine(tc.line, width) {
				units = append(units, codeUnits(row))
			}
			require.Equal(t, tc.rows, units)
		})
	}
}

// A row that ends at a space must not cost a row: a long quoted value after a
// space would leave the row before it nearly empty, and a form has four.
func TestWrapLineNeverTakesMoreRowsThanFillingThem(t *testing.T) {
	random := rand.New(rand.NewPCG(10598, 22224))
	alphabet := []rune(`abcdefghijklmnopqrstuvwxyz0123456789"',()=&|/.-_@名前` + "\u0301\u0308")
	for n := range 5000 {
		var tokens []string
		for range 1 + random.IntN(12) {
			token := make([]rune, 1+random.IntN(90))
			for i := range token {
				token[i] = alphabet[random.IntN(len(alphabet))]
			}
			tokens = append(tokens, string(token))
			// Every tenth line holds runs of marks and of characters that are
			// two code units, so that a row fills by its units first.
			if n%10 == 0 {
				tokens = append(tokens, markedText(random, 400))
			}
		}
		line := strings.Join(tokens, " ")
		for _, width := range []int{messageLineWidth, 40} {
			rows := wrapLine(line, width)
			require.Len(t, rows, fillRows(line, width), "%q at %d: %q", line, width, rows)
			var joined strings.Builder
			for i, row := range rows {
				require.LessOrEqual(t, lineColumns(row), width, "%q", rows)
				require.LessOrEqual(t, codeUnits(row), 2*width, "%q", rows)
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
