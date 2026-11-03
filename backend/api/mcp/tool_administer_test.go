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
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// approvalMock is the internal API behind the approval entry: the stored rules,
// the user's open issues, and the update, recorded with its confirmation.
type approvalMock struct {
	mu     sync.Mutex
	stored *v1pb.WorkspaceApprovalSetting
	// openIssues are the issues the PENDING search returns.
	openIssues []string
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
		var issues []map[string]any
		if strings.Contains(request.Filter, `approval_status == "PENDING"`) && strings.Contains(request.Filter, `creator == "users/alice@example.com"`) {
			for _, name := range m.openIssues {
				issues = append(issues, map[string]any{"name": name})
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"issues": issues})
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
	return &mcp.CallToolParams{Arguments: map[string]any{"operationId": administerUpdateSetting, "body": body}}
}

func adminAnswer(choice string) *mcp.ElicitResult {
	return &mcp.ElicitResult{Action: "accept", Content: map[string]any{adminChoiceField: choice}}
}

var (
	testRule = changeRule("Test", "resource.environment_id == 'test'")
	prodRule = changeRule("Prod", "resource.environment_id == 'prod'", "roles/workspaceDBA")
)

func TestAdministerWorkspaceAppliesConfirmedApprovalRules(t *testing.T) {
	mock := &approvalMock{openIssues: []string{"projects/hr/issues/12"}}
	mock.setStored(fallbackRule())
	user := &asker{answer: adminAnswer(adminChoiceApply)}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	body := approvalBody(t, fallbackRule(), testRule, prodRule)

	res := callAdminister(t, session, administerParams(body))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the approval rules for database changes, as alice@example.com?`,
		`Added, no approval: "resource.environment_id == 'test'"`,
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
		"description":"The rules for database changes, in the order an approval check tries them:\n1. \"Test\", no approval: \"resource.environment_id == 'test'\"\n2. \"Prod\", \"Workspace DBA\": \"resource.environment_id == 'prod'\"\nRules for other sources stay as they are.\nEvery approval check from now on uses these rules, including the check after a plan edit.",
		"oneOf":[{"const":"apply","title":"Apply"},{"const":"change","title":"Change it in chat"}],
		"default":"apply"}}}`, string(schema))

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, body, calls[0].body, "the update must be the request the user confirmed")
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure:    v1connect.SettingServiceUpdateSettingProcedure,
		Resource:     approvalSettingName,
		StoredDigest: auth.MCPValueDigest(&v1pb.WorkspaceApprovalSetting{Rules: []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}}),
	}, calls[0].confirmed, "the update must carry the confirmation and the rules the user saw")
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"operationId":"SettingService/UpdateSetting","changed":true,
		"message":"Applied: there are now 2 rules for database changes. Every approval check from now on uses these rules, including the check after a plan edit.\nApproval rules: https://bb.example.com/custom-approval",
		"link":"https://bb.example.com/custom-approval"}`, string(output))
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
	for name, tc := range map[string]struct {
		arguments map[string]any
		want      string
	}{
		"another operation":     {map[string]any{"operationId": "SettingService/ListSettings", "body": map[string]any{}}, "UNSUPPORTED_CHANGE"},
		"another setting":       {map[string]any{"operationId": administerUpdateSetting, "body": map[string]any{"setting": map[string]any{"name": "settings/MCP"}}}, "UNSUPPORTED_CHANGE"},
		"a field it cannot say": {map[string]any{"operationId": administerUpdateSetting, "body": withField}, "the request sets validate_only"},
		"the fallback removed":  {map[string]any{"operationId": administerUpdateSetting, "body": approvalBody(t, prodRule)}, "OTHER_RULES_CHANGED"},
		"too long for the form": {map[string]any{"operationId": administerUpdateSetting, "body": approvalBody(t, tooMany...)}, "CHANGE_TOO_LONG"},
		"two rules, one condition": {map[string]any{"operationId": administerUpdateSetting, "body": approvalBody(t, fallbackRule(), prodRule, changeRule("Again", "resource.environment_id == 'prod'"))},
			"DUPLICATE_CONDITION"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &approvalMock{}
			mock.setStored(fallbackRule())
			user := &asker{answer: adminAnswer(adminChoiceApply)}
			_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callAdminister(t, session, &mcp.CallToolParams{Arguments: tc.arguments})
			require.True(t, res.IsError, resultText(t, res))
			require.Contains(t, resultText(t, res), tc.want)
			require.Zero(t, user.asked.Load())
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerWorkspaceRefusesATooLongChangeWithTheConsolePage(t *testing.T) {
	rules := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
	for _, environment := range []string{"a", "b", "c", "d"} {
		rules = append(rules, changeRule(environment, "resource.environment_id == '"+environment+"'"))
	}
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	res := callAdminister(t, session, administerParams(approvalBody(t, rules...)))
	require.Contains(t, resultText(t, res), "smaller steps")
	require.Contains(t, resultText(t, res), "https://bb.example.com/custom-approval")
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
	require.Contains(t, user.lastParams(t).Message, `Moved to rule 2 of 2: "resource.environment_id == 'prod'"`)
	require.LessOrEqual(t, len(lines), messageLines)
	for _, line := range lines {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
}

func TestAdministerWorkspaceContinuesALongConditionOnTheNextLine(t *testing.T) {
	long := changeRule("HR prod", "resource.environment_id == 'prod' && resource.project_id == 'hr' && resource.database_name == 'hr_prod'", "roles/workspaceDBA")
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), long)))
	lines := strings.Split(user.lastParams(t).Message, "\n")
	require.Len(t, lines, 4)
	for _, line := range lines {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	require.Equal(t, `Added, "Workspace DBA": "resource.environment_id == 'prod' && `+
		`resource.project_id == 'hr' && resource.database_name == 'hr_prod'"`, lines[1]+" "+lines[2])
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
	require.Empty(t, mock.calls())
}

func TestAdministerWorkspaceReportsTheLockedWriteRefusal(t *testing.T) {
	mock := &approvalMock{refuse: "failed_precondition"}
	mock.setStored(fallbackRule())
	_, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: adminAnswer(adminChoiceApply)}).handle})

	res := callAdminister(t, session, administerParams(approvalBody(t, fallbackRule(), prodRule)))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, res), "the approval rules changed after the user was asked")
	require.Len(t, mock.calls(), 1)
}

func TestAdministerWorkspaceRefusesAnExpiredAnswer(t *testing.T) {
	mock := &approvalMock{}
	mock.setStored(fallbackRule())
	s, session := connectToApprovalServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	body := approvalBody(t, fallbackRule(), prodRule)
	change, err := parseApprovalRulesChange(body)
	require.NoError(t, err)
	diff, err := diffApprovalRules(mock.stored.GetRules(), change.request.GetSetting().GetValue().GetWorkspaceApproval().GetRules())
	require.NoError(t, err)
	alice := withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), "ws-test"), "alice@example.com"), "client-A")
	form, err := approvalRulesForm(alice, diff, nil)
	require.NoError(t, err)
	old := s.signConfirmationState(alice, change, approvalRulesDigest(form.message, mock.stored, change.request), time.Now().Add(-11*time.Minute))

	params := administerParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: adminAnswer(adminChoiceApply)}
	params.RequestState = old
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
