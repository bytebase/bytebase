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
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const (
	administerUpdatePolicy = "OrgPolicyService/UpdatePolicy"
	maskingPolicyName      = "workspaces/ws-test/policies/masking_rule"
	// journeyCondition is the rule journey 3's agent writes: CEL in double
	// quotes, as the console's rule editor saves it too.
	journeyCondition = `resource.environment_id == "prod" && resource.column_name in ["birth_date", "gender", "amount"]`
)

// maskingMock is the internal API behind the masking entry: the stored rules,
// the stored semantic types, and the update, recorded with its confirmation.
type maskingMock struct {
	mu sync.Mutex
	// stored is nil for a workspace that never saved masking rules.
	stored *v1pb.MaskingRulePolicy
	// types is nil for a workspace that stores no semantic type of its own.
	types   []*v1pb.SemanticTypeSetting_SemanticType
	updates []approvalCall
}

func (m *maskingMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m.mu.Lock()
	defer m.mu.Unlock()
	notFound := func() {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"not_found","message":"not found"}`))
	}
	switch r.URL.Path {
	case v1connect.OrgPolicyServiceGetPolicyProcedure:
		if m.stored == nil {
			notFound()
			return
		}
		body, _ := protojson.Marshal(&v1pb.Policy{Name: maskingPolicyName, Type: v1pb.PolicyType_MASKING_RULE,
			Policy: &v1pb.Policy_MaskingRulePolicy{MaskingRulePolicy: m.stored}})
		_, _ = w.Write(body)
	case v1connect.SettingServiceGetSettingProcedure:
		if m.types == nil {
			notFound()
			return
		}
		body, _ := protojson.Marshal(&v1pb.Setting{Name: common.SettingNamePrefix + v1pb.Setting_SEMANTIC_TYPES.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_SemanticType{SemanticType: &v1pb.SemanticTypeSetting{Types: m.types}}}})
		_, _ = w.Write(body)
	case v1connect.OrgPolicyServiceUpdatePolicyProcedure:
		call := approvalCall{}
		_ = json.NewDecoder(r.Body).Decode(&call.body)
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(r.Context()); ok {
			call.confirmed = &confirmed
		}
		m.updates = append(m.updates, call)
		_, _ = w.Write([]byte(`{"name":"` + maskingPolicyName + `"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *maskingMock) calls() []approvalCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]approvalCall(nil), m.updates...)
}

func (m *maskingMock) setStored(rules ...*v1pb.MaskingRulePolicy_MaskingRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stored = &v1pb.MaskingRulePolicy{Rules: rules}
}

func connectToMaskingServer(t *testing.T, mock *maskingMock, opts *mcp.ClientOptions) (*Server, *mcp.ClientSession) {
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

// signedMaskingState is the request state the masking entry signs when it asks
// about body over the stored rules, on a workspace with only the built-in
// semantic types.
func signedMaskingState(t *testing.T, s *Server, stored []*v1pb.MaskingRulePolicy_MaskingRule, body map[string]any, askedAt time.Time) string {
	t.Helper()
	change, err := parseMaskingRulesChange(body, "ws-test", "")
	require.NoError(t, err)
	types, err := semanticTypesFromSetting(nil)
	require.NoError(t, err)
	diff, err := diffMaskingRules(stored, change.request.GetPolicy().GetMaskingRulePolicy().GetRules(), types, "")
	require.NoError(t, err)
	alice := withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), "ws-test"), "alice@example.com"), "client-A")
	form, err := maskingRulesForm(alice, diff, "")
	require.NoError(t, err)
	return s.signConfirmationState(alice, change, maskingRulesDigest(form, change.request), askedAt)
}

func maskingRule(id, title, condition, semanticType string) *v1pb.MaskingRulePolicy_MaskingRule {
	return &v1pb.MaskingRulePolicy_MaskingRule{Id: id, Condition: &expr.Expr{Expression: condition, Title: title}, SemanticType: semanticType}
}

func maskingRequest(allowMissing bool, rules ...*v1pb.MaskingRulePolicy_MaskingRule) *v1pb.UpdatePolicyRequest {
	return &v1pb.UpdatePolicyRequest{
		Policy: &v1pb.Policy{Name: maskingPolicyName, Type: v1pb.PolicyType_MASKING_RULE,
			Policy: &v1pb.Policy_MaskingRulePolicy{MaskingRulePolicy: &v1pb.MaskingRulePolicy{Rules: rules}}},
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"masking_rule_policy"}},
		AllowMissing: allowMissing,
	}
}

func maskingBody(t *testing.T, request *v1pb.UpdatePolicyRequest) map[string]any {
	t.Helper()
	encoded, err := protojson.Marshal(request)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(encoded, &body))
	return body
}

func maskingParams(body map[string]any) *mcp.CallToolParams {
	return &mcp.CallToolParams{Arguments: map[string]any{"operationId": administerUpdatePolicy, "body": body}}
}

func semanticType(id, title string, algorithm *v1pb.Algorithm) *v1pb.SemanticTypeSetting_SemanticType {
	return &v1pb.SemanticTypeSetting_SemanticType{Id: id, Title: title, Algorithm: algorithm}
}

func TestAdministerMaskingRulesAsksForTheJourneyRuleOnAWorkspaceWithNone(t *testing.T) {
	mock := &maskingMock{}
	user := &asker{answer: adminAnswer(adminChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	body := maskingBody(t, maskingRequest(true, maskingRule("7d3c1a52-1f6e-4a5b-9d0e-2b8f4c6a1e09", "Personal data in prod", journeyCondition, "bb.default")))

	res := callAdminister(t, session, maskingParams(body))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		"Add rule 1, \"Default\" (Full mask): `resource.environment_id == \"prod\"",
		"> && resource.column_name in [\"birth_date\", \"gender\", \"amount\"]`",
		maskingRulesApply,
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n") {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	require.Equal(t, "The masking rules, in the order masking tries them:\n"+
		"1. \"Personal data in prod\", \"Default\" (Full mask): `"+journeyCondition+"`", choiceDescription(t, asked))

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, body, calls[0].body, "the update must be the request the user confirmed")
	require.Equal(t, &auth.MCPConfirmedDecision{Procedure: v1connect.OrgPolicyServiceUpdatePolicyProcedure, Resource: maskingPolicyName},
		calls[0].confirmed, "the update must carry the confirmation for this policy")
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"operationId":"OrgPolicyService/UpdatePolicy","changed":true,
		"message":"Applied: the masking rules now hold 1 rule. `+maskingRulesApply+`\nMasking rules: https://bb.example.com/global-masking",
		"link":"https://bb.example.com/global-masking"}`, string(output))
}

// An added rule can take a column from a later rule or from the column's own
// type, so any type but a full mask can unmask data.
func TestAdministerMaskingRulesStatesWhatEachTypeDoes(t *testing.T) {
	fullMask := &v1pb.Algorithm{Mask: &v1pb.Algorithm_FullMask_{FullMask: &v1pb.Algorithm_FullMask{Substitution: "*"}}}
	md5 := &v1pb.Algorithm{Mask: &v1pb.Algorithm_Md5Mask{Md5Mask: &v1pb.Algorithm_MD5Mask{}}}
	noSlice := &v1pb.Algorithm{Mask: &v1pb.Algorithm_RangeMask_{RangeMask: &v1pb.Algorithm_RangeMask{}}}
	types, err := semanticTypesFromSetting([]*v1pb.SemanticTypeSetting_SemanticType{
		semanticType("email", "Email", md5),
		semanticType("plain", "Plain", nil),
		semanticType("empty-range", "Empty range", noSlice),
		semanticType("redacted", "Redacted", fullMask),
		// A stored copy of a built-in type is ignored, as masking ignores it.
		semanticType("bb.default", "Stored default", nil),
	})
	require.NoError(t, err)
	for id, want := range map[string]struct {
		phrase  string
		unmasks bool
	}{
		"bb.default":         {`"Default" (Full mask)`, false},
		"redacted":           {`"Redacted" (Full mask)`, false},
		"bb.default-partial": {`"Default Partial" (Partial mask)`, true},
		"email":              {`"Email" (Hash (MD5))`, true},
		"plain":              {`"Plain" (masks nothing)`, true},
		"empty-range":        {`"Empty range" (masks nothing)`, true},
		"bb.typo":            {`"bb.typo" (no such type, masks nothing)`, true},
		"":                   {"no type (each column's own type)", true},
	} {
		diff, err := diffMaskingRules(nil, []*v1pb.MaskingRulePolicy_MaskingRule{maskingRule("r1", "", `resource.environment_id == "prod"`, id)}, types, "")
		require.NoError(t, err, id)
		require.Equal(t, []formChange{{"Add rule 1, " + want.phrase + ": `resource.environment_id == \"prod\"`", want.unmasks}}, diff.changes, id)
	}
}

func TestAdministerMaskingRulesStatesEachKindOfChange(t *testing.T) {
	types, err := semanticTypesFromSetting(nil)
	require.NoError(t, err)
	prod := maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default")
	test := maskingRule("r2", "Test", `resource.environment_id == "test"`, "bb.default-partial")
	with := func(rule *v1pb.MaskingRulePolicy_MaskingRule, change func(*v1pb.MaskingRulePolicy_MaskingRule)) *v1pb.MaskingRulePolicy_MaskingRule {
		changed := proto.CloneOf(rule)
		change(changed)
		return changed
	}
	rules := func(rules ...*v1pb.MaskingRulePolicy_MaskingRule) []*v1pb.MaskingRulePolicy_MaskingRule { return rules }
	for name, tc := range map[string]struct {
		stored, requested []*v1pb.MaskingRulePolicy_MaskingRule
		want              []formChange
	}{
		"removed": {stored: rules(prod), requested: nil,
			want: []formChange{{"Remove, \"Default\" (Full mask): `resource.environment_id == \"prod\"`", true}}},
		"a full mask added before the kept rules": {stored: rules(test), requested: rules(prod, test),
			want: []formChange{{"Add rule 1, \"Default\" (Full mask): `resource.environment_id == \"prod\"`", false}}},
		"a partial mask added after the kept rules": {stored: rules(prod), requested: rules(prod, test),
			want: []formChange{{"Add rule 2, \"Default Partial\" (Partial mask): `resource.environment_id == \"test\"`", true}}},
		"moved": {stored: rules(prod, test), requested: rules(test, prod),
			want: []formChange{{"Move to rule 2: `resource.environment_id == \"prod\"`", true}}},
		"given a partial mask": {stored: rules(prod),
			requested: rules(with(prod, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.SemanticType = "bb.default-partial" })),
			want:      []formChange{{"Change type, \"Default\" (Full mask) to \"Default Partial\" (Partial mask): `resource.environment_id == \"prod\"`", true}}},
		"given a full mask": {stored: rules(test),
			requested: rules(with(test, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.SemanticType = "bb.default" })),
			want:      []formChange{{"Change type, \"Default Partial\" (Partial mask) to \"Default\" (Full mask): `resource.environment_id == \"test\"`", false}}},
		"renamed": {stored: rules(prod),
			requested: rules(with(prod, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.Condition.Title = "Production" })),
			want:      []formChange{{"Rename to \"Production\": `resource.environment_id == \"prod\"`", false}}},
		"renamed to no title": {stored: rules(prod),
			requested: rules(with(prod, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.Condition.Title = "" })),
			want:      []formChange{{"Rename to untitled: `resource.environment_id == \"prod\"`", false}}},
		"given a full mask, moved and renamed": {stored: rules(test, prod),
			requested: rules(prod, with(test, func(r *v1pb.MaskingRulePolicy_MaskingRule) {
				r.SemanticType = "bb.default"
				r.Condition.Title = "Testing"
			})),
			want: []formChange{{"Change type, \"Default Partial\" (Partial mask) to \"Default\" (Full mask); move to rule 2; rename to \"Testing\": `resource.environment_id == \"test\"`", true}}},
		"condition changed": {stored: rules(prod),
			requested: rules(maskingRule("r3", "Prod", `resource.environment_id == "production"`, "bb.default")),
			want: []formChange{
				{"Remove, \"Default\" (Full mask): `resource.environment_id == \"prod\"`", true},
				{"Add rule 1, \"Default\" (Full mask): `resource.environment_id == \"production\"`", false},
			}},
	} {
		t.Run(name, func(t *testing.T) {
			diff, err := diffMaskingRules(tc.stored, tc.requested, types, "")
			require.NoError(t, err)
			require.Equal(t, tc.want, diff.changes)
		})
	}
}

// A change that can unmask data says so in the message: masking applies to the
// user's own queries too.
func TestAdministerMaskingRulesSaysAChangeCanUnmaskData(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored(maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default"))
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false))))
	require.Contains(t, resultText(t, res), "Not changed: the user declined")
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		"Remove, \"Default\" (Full mask): `resource.environment_id == \"prod\"`",
		maskingRulesUnmask,
		maskingRulesApply,
	}, "\n"), user.lastParams(t).Message)
	require.Equal(t, "No masking rule is left, so each column keeps its own semantic type.", choiceDescription(t, user.lastParams(t)))
	require.Empty(t, mock.calls())
}

// With the question, two change lines and the unmask line, the message is full,
// so the fixed sentence ends the description.
func TestAdministerMaskingRulesLeavesTheFixedSentenceToAFullDescription(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored(maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default"))
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false,
		maskingRule("r2", "Test", `resource.environment_id == "test"`, "bb.default")))))
	require.Contains(t, resultText(t, res), "Not changed: the user declined")
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		"Remove, \"Default\" (Full mask): `resource.environment_id == \"prod\"`",
		"Add rule 1, \"Default\" (Full mask): `resource.environment_id == \"test\"`",
		maskingRulesUnmask,
	}, "\n"), user.lastParams(t).Message)
	require.Equal(t, "The masking rules, in the order masking tries them:\n"+
		"1. \"Test\", \"Default\" (Full mask): `resource.environment_id == \"test\"`\n"+maskingRulesApply,
		choiceDescription(t, user.lastParams(t)))
}

func TestAdministerMaskingRulesRefusesBeforeAsking(t *testing.T) {
	prod := maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default")
	withPolicy := func(change func(*v1pb.UpdatePolicyRequest)) *v1pb.UpdatePolicyRequest {
		request := maskingRequest(true, proto.CloneOf(prod))
		change(request)
		return request
	}
	described := maskingRule("r9", "Stored", `resource.environment_id == "test"`, "bb.default")
	described.Condition.Description = "set through the API"
	located := maskingRule("r9", "Stored", `resource.environment_id == "test"`, "bb.default")
	located.Condition.Location = "set through the API"
	for name, tc := range map[string]struct {
		stored  []*v1pb.MaskingRulePolicy_MaskingRule
		none    bool
		request *v1pb.UpdatePolicyRequest
		want    string
	}{
		"another policy": {request: withPolicy(func(r *v1pb.UpdatePolicyRequest) { r.Policy.Name = "workspaces/ws-test/policies/masking_exemption" }),
			want: "UNSUPPORTED_CHANGE"},
		"another workspace": {request: withPolicy(func(r *v1pb.UpdatePolicyRequest) { r.Policy.Name = "workspaces/ws-other/policies/masking_rule" }),
			want: "UNSUPPORTED_CHANGE"},
		"the policy type in capitals": {request: withPolicy(func(r *v1pb.UpdatePolicyRequest) { r.Policy.Name = "workspaces/ws-test/policies/MASKING_RULE" }),
			want: "UNSUPPORTED_CHANGE"},
		"another policy type": {request: withPolicy(func(r *v1pb.UpdatePolicyRequest) { r.Policy.Type = v1pb.PolicyType_MASKING_EXEMPTION }),
			want: "UNSUPPORTED_CHANGE"},
		"no rule list": {request: withPolicy(func(r *v1pb.UpdatePolicyRequest) { r.Policy.Policy = nil }),
			want: "no policy.maskingRulePolicy"},
		"another update mask": {request: withPolicy(func(r *v1pb.UpdatePolicyRequest) { r.UpdateMask.Paths = []string{"enforce"} }),
			want: "updateMask"},
		"inherit from parent": {request: withPolicy(func(r *v1pb.UpdatePolicyRequest) { r.Policy.InheritFromParent = true }),
			want: "policy.inherit_from_parent"},
		"a condition description": {request: maskingRequest(true, described),
			want: "policy.masking_rule_policy.rules.condition.description"},
		"a stored condition description": {stored: []*v1pb.MaskingRulePolicy_MaskingRule{described}, request: maskingRequest(true, prod),
			want: "a stored masking rule sets policy.masking_rule_policy.rules.condition.description"},
		"a condition location": {request: maskingRequest(true, located),
			want: "policy.masking_rule_policy.rules.condition.location"},
		"a stored condition location": {stored: []*v1pb.MaskingRulePolicy_MaskingRule{located}, request: maskingRequest(true, prod),
			want: "a stored masking rule sets policy.masking_rule_policy.rules.condition.location"},
		"no id": {request: maskingRequest(true, maskingRule("", "Prod", `resource.environment_id == "prod"`, "bb.default")),
			want: "rule 1 has no id"},
		"an id twice": {request: maskingRequest(true, prod, maskingRule("r1", "Test", `resource.environment_id == "test"`, "bb.default")),
			want: "two rules have the id"},
		"a kept rule with a new id": {stored: []*v1pb.MaskingRulePolicy_MaskingRule{prod},
			request: maskingRequest(true, maskingRule("r2", "Prod", `resource.environment_id == "prod"`, "bb.default")),
			want:    "the stored rule for that condition has the id"},
		"no condition": {request: maskingRequest(true, &v1pb.MaskingRulePolicy_MaskingRule{Id: "r1", SemanticType: "bb.default"}),
			want: "rule 1 has no condition"},
		"an invalid condition": {request: maskingRequest(true, maskingRule("r1", "", `resource.environment == "prod"`, "bb.default")),
			want: "is not a masking rule condition"},
		"one condition twice": {request: maskingRequest(true, prod, maskingRule("r2", "Again", `resource.environment_id == "prod"`, "bb.default")),
			want: "two requested masking rules have the condition"},
		"one stored condition twice": {stored: []*v1pb.MaskingRulePolicy_MaskingRule{prod, maskingRule("r2", "Again", `resource.environment_id == "prod"`, "bb.default")},
			request: maskingRequest(true, prod),
			want:    "two stored masking rules have the condition"},
		"no stored policy and no allowMissing": {none: true, request: maskingRequest(false, prod),
			want: "allowMissing"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &maskingMock{}
			if !tc.none {
				mock.setStored(tc.stored...)
			}
			user := &asker{answer: adminAnswer(adminChoiceApply)}
			_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callAdminister(t, session, maskingParams(maskingBody(t, tc.request)))
			require.True(t, res.IsError, resultText(t, res))
			require.Contains(t, resultText(t, res), tc.want)
			require.Zero(t, user.asked.Load(), "a refused change is refused before the user is asked")
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerMaskingRulesRefusesAnAnswerOverRulesThatChanged(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored(maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default"))
	user := &asker{answer: adminAnswer(adminChoiceApply)}
	// Another admin renames the stored rule while the form is open.
	user.whileAsked = func() {
		mock.setStored(maskingRule("r1", "Production", `resource.environment_id == "prod"`, "bb.default"))
	}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false,
		maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default"),
		maskingRule("r2", "Test", `resource.environment_id == "test"`, "bb.default")))))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, res), maskingRules.reread())
	require.Empty(t, mock.calls())
}

// The form states what each rule's type does, so an answer over a type that
// changed is refused even when only the description reads differently.
func TestAdministerMaskingRulesRefusesAnAnswerOverTypesThatChanged(t *testing.T) {
	md5 := &v1pb.Algorithm{Mask: &v1pb.Algorithm_Md5Mask{Md5Mask: &v1pb.Algorithm_MD5Mask{}}}
	mock := &maskingMock{types: []*v1pb.SemanticTypeSetting_SemanticType{semanticType("email", "Email", md5)}}
	kept := maskingRule("r1", "Emails", `resource.column_name == "email"`, "email")
	mock.setStored(kept)
	user := &asker{answer: adminAnswer(adminChoiceApply)}
	// Another admin removes the email type's algorithm while the form is open.
	user.whileAsked = func() {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		mock.types = []*v1pb.SemanticTypeSetting_SemanticType{semanticType("email", "Email", nil)}
	}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false,
		proto.CloneOf(kept), maskingRule("r2", "Prod", `resource.environment_id == "prod"`, "bb.default")))))
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

func TestAdministerMaskingRulesRefusesAnAnswerToAnotherRequest(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored()
	s, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	state := signedMaskingState(t, s, nil, maskingBody(t, maskingRequest(false,
		maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default"))), time.Now())
	// The same form: an added rule's id is not in it.
	params := maskingParams(maskingBody(t, maskingRequest(false,
		maskingRule("r2", "Prod", `resource.environment_id == "prod"`, "bb.default"))))
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: adminAnswer(adminChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

func TestAdministerMaskingRulesRefusesAChangeTooLongForTheForm(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored()
	var rules []*v1pb.MaskingRulePolicy_MaskingRule
	for _, environment := range []string{"a", "b", "c", "d"} {
		rules = append(rules, maskingRule("r-"+environment, "", `resource.environment_id == "`+environment+`"`, "bb.default"))
	}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, rules...))))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "this change needs 5 lines, more than the 4 one confirmation form shows, so the masking rules were not changed")
	require.Contains(t, resultText(t, res), "read the rules again with call_api OrgPolicyService/GetPolicy")
	require.Contains(t, resultText(t, res), "https://bb.example.com/global-masking")
}

// The unmask line belongs to the removal, so the long added full mask fits a
// form of its own, and smaller steps help.
func TestAdministerMaskingRulesSuggestsStepsForAChangeThatFitsAFormAlone(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored(maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default"))
	condition := `resource.environment_id == "prod" && resource.column_name in ["birth_date", "gender", "amount", "salary", "address", "phone"]`
	added := maskingRule("r2", "Personal data in prod", condition, "bb.default")
	require.Len(t, wrapMessageLine("Add rule 1, \"Default\" (Full mask): "+conditionText(condition)), 3, "the added rule takes three lines")
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, added))))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "this change needs 6 lines")
	require.Contains(t, resultText(t, res), "make the change in smaller steps")
	require.Empty(t, mock.calls())
}

func TestAdministerMaskingRulesAsksNothingWhenNothingChanges(t *testing.T) {
	mock := &maskingMock{}
	prod := maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default")
	mock.setStored(prod)
	user := &asker{answer: adminAnswer(adminChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, proto.CloneOf(prod)))))
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "Nothing to change")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
}

// The agent may leave the update mask out, or write its one path as the API
// names it or in the JSON form, under either name of the field.
func TestAdministerMaskingRulesSetsTheUpdateMask(t *testing.T) {
	for _, key := range []string{"updateMask", "update_mask"} {
		for _, mask := range []any{"masking_rule_policy", "maskingRulePolicy", nil} {
			body := maskingBody(t, maskingRequest(true, maskingRule("r1", "", `resource.environment_id == "prod"`, "bb.default")))
			delete(body, "updateMask")
			if mask != nil {
				body[key] = mask
			}
			change, err := parseMaskingRulesChange(body, "ws-test", "")
			require.NoError(t, err, "%s: %v", key, mask)
			require.Equal(t, []string{"masking_rule_policy"}, change.request.GetUpdateMask().GetPaths(), "%s: %v", key, mask)
		}
	}
}

// An agent may send the policy as GetPolicy returns it, with policy.enforce and
// policy.resource_type, which the write ignores.
func TestAdministerMaskingRulesTakesThePolicyAsGetPolicyReturnsIt(t *testing.T) {
	request := maskingRequest(true, maskingRule("r1", "", `resource.environment_id == "prod"`, "bb.default"))
	request.Policy.Enforce = true
	request.Policy.ResourceType = v1pb.PolicyResourceType_WORKSPACE
	_, err := parseMaskingRulesChange(maskingBody(t, request), "ws-test", "")
	require.NoError(t, err)
}

func TestStoreSemanticTypeKeepsEveryAlgorithmField(t *testing.T) {
	for name, algorithm := range map[string]*v1pb.Algorithm{
		"full":  {Mask: &v1pb.Algorithm_FullMask_{FullMask: &v1pb.Algorithm_FullMask{Substitution: "#"}}},
		"range": {Mask: &v1pb.Algorithm_RangeMask_{RangeMask: &v1pb.Algorithm_RangeMask{Slices: []*v1pb.Algorithm_RangeMask_Slice{{Start: 1, End: 3, Substitution: "*"}}}}},
		"md5":   {Mask: &v1pb.Algorithm_Md5Mask{Md5Mask: &v1pb.Algorithm_MD5Mask{Salt: "s"}}},
		"outer": {Mask: &v1pb.Algorithm_InnerOuterMask_{InnerOuterMask: &v1pb.Algorithm_InnerOuterMask{PrefixLen: 1, SuffixLen: 2, Type: v1pb.Algorithm_InnerOuterMask_OUTER, Substitution: "*"}}},
	} {
		stored, err := storeSemanticType(semanticType("t", "T", algorithm))
		require.NoError(t, err, name)
		encodedAPI, err := protojson.Marshal(algorithm)
		require.NoError(t, err)
		encodedStore, err := protojson.Marshal(stored.GetAlgorithm())
		require.NoError(t, err)
		require.JSONEq(t, string(encodedAPI), string(encodedStore), name)
		require.NotNil(t, stored.GetAlgorithm().GetMask(), name)
	}
}

func TestAdministerMaskingRulesClassifiesEveryRequestField(t *testing.T) {
	requireEveryFieldClassified(t, (&v1pb.UpdatePolicyRequest{}).ProtoReflect().Descriptor(), "", maskingRequestFields, map[string]bool{
		"policy.inherit_from_parent":                true,
		maskingRulesPath + ".condition.description": true,
		maskingRulesPath + ".condition.location":    true,
	})
}

func TestAdministerMaskingRulesChangesNothingUnlessApplied(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":        {&mcp.ElicitResult{Action: "decline"}, "Not changed: the user declined, so the masking rules stay as they are"},
		"cancelled":       {&mcp.ElicitResult{Action: "cancel"}, "Not changed: the user cancelled, so the masking rules stay as they are"},
		"changed in chat": {adminAnswer(adminChoiceChange), "Not changed: the user wants to change this request"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &maskingMock{}
			mock.setStored()
			_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: tc.answer}).handle})
			res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, maskingRule("r1", "", `resource.environment_id == "prod"`, "bb.default")))))
			require.Contains(t, resultText(t, res), tc.want)
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerMaskingRulesRefusesAClientThatCannotConfirm(t *testing.T) {
	mock := &maskingMock{}
	_, session := connectToMaskingServer(t, mock, nil)
	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(true, maskingRule("r1", "", `resource.environment_id == "prod"`, "bb.default")))))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "CONFIRMATION_UNAVAILABLE")
	require.Contains(t, resultText(t, res), "https://bb.example.com/global-masking")
	require.Empty(t, mock.calls())
}
