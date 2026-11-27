package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
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
	return s.signConfirmationState(alice, change, maskingFormDigest(form, change.request, stored, nil), askedAt)
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
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	body := maskingBody(t, maskingRequest(true, maskingRule("7d3c1a52-1f6e-4a5b-9d0e-2b8f4c6a1e09", "Personal data in prod", journeyCondition, "bb.default")))

	res := callAdminister(t, session, maskingParams(body))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	title, labels := choiceLabels(t, asked)
	require.Equal(t, "Action", title)
	require.Equal(t, []string{"Save rules", "Change it in chat"}, labels)
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		"Add rule 1, Full mask: env \"prod\", any db, any table, columns",
		"> (\"birth_date\", \"gender\", \"amount\")",
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n") {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	require.Equal(t, "The masking rules, in the order masking tries them:\n"+
		"1. \"Personal data in prod\" (new), Full mask: `"+journeyCondition+"`", unwrapMessage(choiceDescription(t, asked)))
	for _, row := range strings.Split(choiceDescription(t, asked), "\n") {
		require.LessOrEqual(t, lineColumns(row), descriptionLineWidth, row)
	}

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
	// With no margins, strings stay as they are, and NULL and booleans are
	// replaced.
	noMargin := &v1pb.Algorithm{Mask: &v1pb.Algorithm_InnerOuterMask_{InnerOuterMask: &v1pb.Algorithm_InnerOuterMask{
		Type: v1pb.Algorithm_InnerOuterMask_OUTER, Substitution: "*"}}}
	types, err := semanticTypesFromSetting([]*v1pb.SemanticTypeSetting_SemanticType{
		semanticType("email", "Email", md5),
		semanticType("plain", "Plain", nil),
		semanticType("empty-range", "Empty range", noSlice),
		semanticType("outer", "Outer", noMargin),
		semanticType("redacted", "Redacted", fullMask),
		// A stored copy of a built-in type is ignored, as masking ignores it.
		semanticType("bb.default", "Stored default", nil),
	})
	require.NoError(t, err)
	for id, want := range map[string]struct {
		phrase  string
		unmasks bool
	}{
		"bb.default":         {"Full mask", false},
		"redacted":           {`"Redacted" (Full mask)`, false},
		"bb.default-partial": {"Partial mask", true},
		"email":              {`"Email" (Hash (MD5))`, true},
		"plain":              {`"Plain" (masks nothing)`, true},
		"empty-range":        {`"Empty range" (masks nothing)`, true},
		"outer":              {`"Outer" (Inner/Outer mask)`, true},
		"bb.typo":            {`"bb.typo" (no such type, masks nothing)`, true},
		"":                   {"no type (each column's own type)", true},
	} {
		diff, err := diffMaskingRules(nil, []*v1pb.MaskingRulePolicy_MaskingRule{maskingRule("r1", "", `resource.environment_id == "prod"`, id)}, types, "")
		require.NoError(t, err, id)
		require.Equal(t, []formChange{{"Add rule 1, " + want.phrase + ": env \"prod\", every column", want.unmasks}}, diff.changes, id)
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
			want: []formChange{{"Remove, Full mask: env \"prod\", every column", true}}},
		"a full mask added before the kept rules": {stored: rules(test), requested: rules(prod, test),
			want: []formChange{{"Add rule 1, Full mask: env \"prod\", every column", false}}},
		"a partial mask added after the kept rules": {stored: rules(prod), requested: rules(prod, test),
			want: []formChange{{"Add rule 2, Partial mask: env \"test\", every column", true}}},
		"moved": {stored: rules(prod, test), requested: rules(test, prod),
			want: []formChange{{"Move to rule 2: env \"prod\", every column", true}}},
		"given a partial mask": {stored: rules(prod),
			requested: rules(with(prod, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.SemanticType = "bb.default-partial" })),
			want:      []formChange{{"Change type, Full mask to Partial mask: env \"prod\", every column", true}}},
		"given a full mask": {stored: rules(test),
			requested: rules(with(test, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.SemanticType = "bb.default" })),
			want:      []formChange{{"Change type, Partial mask to Full mask: env \"test\", every column", false}}},
		"renamed": {stored: rules(prod),
			requested: rules(with(prod, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.Condition.Title = "Production" })),
			want:      []formChange{{"Rename to \"Production\": env \"prod\", every column", false}}},
		"renamed to no title": {stored: rules(prod),
			requested: rules(with(prod, func(r *v1pb.MaskingRulePolicy_MaskingRule) { r.Condition.Title = "" })),
			want:      []formChange{{"Rename to untitled: env \"prod\", every column", false}}},
		"given a full mask, moved and renamed": {stored: rules(test, prod),
			requested: rules(prod, with(test, func(r *v1pb.MaskingRulePolicy_MaskingRule) {
				r.SemanticType = "bb.default"
				r.Condition.Title = "Testing"
			})),
			want: []formChange{{"Change type, Partial mask to Full mask; move to rule 2; rename to \"Testing\": env \"test\", every column", true}}},
		"condition changed": {stored: rules(prod),
			requested: rules(maskingRule("r3", "Prod", `resource.environment_id == "production"`, "bb.default")),
			want: []formChange{
				{"Remove, Full mask: env \"prod\", every column", true},
				{"Add rule 1, Full mask: env \"production\", every column", false},
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
		"Remove, Full mask: env \"prod\", every column",
		maskingRulesUnmask,
	}, "\n"), user.lastParams(t).Message)
	require.Equal(t, "No masking rule is left, so each column keeps its own semantic type.", choiceDescription(t, user.lastParams(t)))
	require.Empty(t, mock.calls())
}

// With the question and three change lines, the message is full, so the fixed
// sentence ends the description. Kept rules say what they decide.
func TestAdministerMaskingRulesLeavesTheFixedSentenceToAFullDescription(t *testing.T) {
	mock := &maskingMock{}
	kept := maskingRule("r1", "A", `resource.environment_id == "a"`, "bb.default")
	mock.setStored(kept)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, proto.CloneOf(kept),
		maskingRule("r2", "B", `resource.environment_id == "b"`, "bb.default"),
		maskingRule("r3", "C", `resource.environment_id == "c"`, "bb.default"),
		maskingRule("r4", "D", `resource.environment_id == "d"`, "bb.default")))))
	require.Contains(t, resultText(t, res), "Not changed: the user declined")
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		"Add rule 2, Full mask: env \"b\", every column",
		"Add rule 3, Full mask: env \"c\", every column",
		"Add rule 4, Full mask: env \"d\", every column",
	}, "\n"), user.lastParams(t).Message)
	require.Equal(t, "The masking rules, in the order masking tries them:\n"+
		"1. \"A\" (kept), Full mask: env \"a\", every column\n"+
		"2. \"B\" (new), Full mask: `resource.environment_id == \"b\"`\n"+
		"3. \"C\" (new), Full mask: `resource.environment_id == \"c\"`\n"+
		"4. \"D\" (new), Full mask: `resource.environment_id == \"d\"`\n"+maskingRulesApply,
		choiceDescription(t, user.lastParams(t)))
}

// A move's line in the message shows no type, and the order decides which rule
// masks a column, so the description names the moved rule's type.
func TestAdministerMaskingRulesNamesAMovedRulesType(t *testing.T) {
	types, err := semanticTypesFromSetting(nil)
	require.NoError(t, err)
	full := maskingRule("r1", "SSN", `resource.column_name == "ssn"`, "bb.default")
	partial := maskingRule("r2", "PII", `resource.column_name in ["email", "ssn"]`, "bb.default-partial")
	diff, err := diffMaskingRules([]*v1pb.MaskingRulePolicy_MaskingRule{full, partial}, []*v1pb.MaskingRulePolicy_MaskingRule{partial, full}, types, "")
	require.NoError(t, err)
	form, err := maskingRulesForm(withUserEmail(context.Background(), "alice@example.com"), diff, "")
	require.NoError(t, err)
	require.Contains(t, form.message, "Move to rule 2: any db, any table, column \"ssn\"")
	require.Equal(t, "The masking rules, in the order masking tries them:\n"+
		"1. \"PII\" (kept), Partial mask: any db, any table, columns (\"email\", \"ssn\")\n"+
		"2. \"SSN\" (changed), Full mask: `resource.column_name == \"ssn\"`", unwrapMessage(form.description))
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
			user := &asker{answer: confirmAnswer(confirmChoiceApply)}
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
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
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
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
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

// A type no rule names shows nowhere on the form, so the digest binds every
// type read.
func TestAdministerMaskingRulesRefusesAnAnswerOverATypeNoRuleNames(t *testing.T) {
	md5 := &v1pb.Algorithm{Mask: &v1pb.Algorithm_Md5Mask{Md5Mask: &v1pb.Algorithm_MD5Mask{}}}
	mock := &maskingMock{types: []*v1pb.SemanticTypeSetting_SemanticType{semanticType("email", "Email", md5)}}
	mock.setStored()
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	user.whileAsked = func() {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		mock.types = []*v1pb.SemanticTypeSetting_SemanticType{semanticType("email", "Email", nil)}
	}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false,
		maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default")))))
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

// Two types can read the same on the form, so a removed rule that moves to the
// other one changes nothing the form shows; the digest binds the stored rules.
func TestAdministerMaskingRulesRefusesAnAnswerOverARemovedRuleMovedToALikeType(t *testing.T) {
	starred := &v1pb.Algorithm{Mask: &v1pb.Algorithm_FullMask_{FullMask: &v1pb.Algorithm_FullMask{Substitution: "*"}}}
	hashed := &v1pb.Algorithm{Mask: &v1pb.Algorithm_FullMask_{FullMask: &v1pb.Algorithm_FullMask{Substitution: "#"}}}
	mock := &maskingMock{types: []*v1pb.SemanticTypeSetting_SemanticType{semanticType("secret-a", "Secret", starred), semanticType("secret-b", "Secret", hashed)}}
	mock.setStored(maskingRule("r1", "SSN", `resource.column_name == "ssn"`, "secret-a"))
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	user.whileAsked = func() { mock.setStored(maskingRule("r1", "SSN", `resource.column_name == "ssn"`, "secret-b")) }
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false))))
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
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

// Four rules need five message lines. The message then counts them, and the
// description lists them first, since Claude Code's fullscreen renderer shows
// only its first row.
func TestAdministerMaskingRulesListsAChangeTooLongForTheMessageUnderTheChoice(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored()
	var rules []*v1pb.MaskingRulePolicy_MaskingRule
	for _, environment := range []string{"a", "b", "c", "d"} {
		rules = append(rules, maskingRule("r-"+environment, "", `resource.environment_id == "`+environment+`"`, "bb.default"))
	}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, rules...))))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		`Add 4 rules, listed under Action.`,
		maskingRulesApply,
	}, "\n"), asked.Message)
	require.Equal(t, strings.Join([]string{
		`Add rule 1, Full mask: env "a", every column`,
		`Add rule 2, Full mask: env "b", every column`,
		`Add rule 3, Full mask: env "c", every column`,
		`Add rule 4, Full mask: env "d", every column`,
		`The masking rules, in the order masking tries them:`,
		`1. untitled (new), Full mask`,
		`2. untitled (new), Full mask`,
		`3. untitled (new), Full mask`,
		`4. untitled (new), Full mask`,
	}, "\n"), choiceDescription(t, asked), "the changes above state each new rule's condition, so the list does not repeat it")
	require.Len(t, mock.calls(), 1, "one form makes the whole change in one write")
}

// A form taller than the terminal loses its top in Claude Code's default
// renderer, the question first, so a change that needs more rows than the
// description's budget is still made in steps.
func TestAdministerMaskingRulesRefusesAChangeTooLongForTheDescription(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored()
	var rules []*v1pb.MaskingRulePolicy_MaskingRule
	for i := range descriptionLines + 1 {
		rules = append(rules, maskingRule(fmt.Sprintf("r-%d", i), "", fmt.Sprintf(`resource.environment_id == "e%d"`, i), "bb.default"))
	}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, rules...))))
	refusal := resultToolError(t, res)
	require.Equal(t, "CHANGE_TOO_LONG", refusal.Code)
	require.Equal(t, "this change needs 11 lines, more than the 10 one confirmation form has room for under its choice, so the masking rules were not changed", refusal.Message)
	require.Contains(t, refusal.Suggestion, "make the change in smaller steps, a few rules at a time")
	require.Contains(t, refusal.Suggestion, "a smaller step changes fewer rules, never a condition")
	require.Contains(t, refusal.Suggestion, "add a rule under a new UUID before removing the one it replaces")
	require.Contains(t, refusal.Suggestion, "read the rules again with call_api OrgPolicyService/GetPolicy")
	require.Contains(t, refusal.Suggestion, "https://bb.example.com/global-masking")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())

	res = callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, rules[:descriptionLines]...))))
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, "Add 10 rules, listed under Action.", strings.Split(user.lastParams(t).Message, "\n")[1])
}

// An edit to a rule's condition removes one rule and adds another. Both go in
// one form and one write, so the edited rule keeps its id and the columns are
// never left to neither rule.
func TestAdministerMaskingRulesReplacesARuleInOneForm(t *testing.T) {
	mock := &maskingMock{}
	prod := maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default")
	mock.setStored(prod)
	condition := `resource.environment_id == "prod" && (resource.column_name == "birth_date" || resource.column_name == "gender" || resource.column_name == "amount")`
	require.Len(t, wrapMessageLine("Add rule 1, Full mask: "+maskingConditionText(condition)), 3, "the added rule takes three lines")
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, maskingRule("r1", "Personal data in prod", condition, "bb.default")))))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		`Remove 1 rule and add 1, listed under Action.`,
		maskingRulesUnmask,
	}, "\n"), asked.Message)
	description := strings.Split(unwrapMessage(choiceDescription(t, asked)), "\n")
	require.Equal(t, []string{
		`Remove, Full mask: env "prod", every column`,
		"Add rule 1, Full mask: `" + condition + "`",
		`The masking rules, in the order masking tries them:`,
		`1. "Personal data in prod" (new), Full mask`,
	}, description, "the removal leads, and each change reads as a message line would")
	require.LessOrEqual(t, len(strings.Split(choiceDescription(t, asked), "\n")), descriptionLines)
	require.EqualValues(t, 1, user.asked.Load())
	require.Len(t, mock.calls(), 1)
}

// The condition takes three of the four lines, which leaves the removal no line
// for the unmask effect.
func TestAdministerMaskingRulesRemovesALongRuleItAdded(t *testing.T) {
	mock := &maskingMock{}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	rule := maskingRule("r1", "Salary amounts", `resource.database_name == "hr_test" && resource.table_name == "salary" && (resource.column_name == "amount" || resource.column_name == "bonus")`, "bb.default")

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(true, rule))))
	require.False(t, res.IsError, resultText(t, res))
	mock.setStored(rule)
	res = callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false))))
	require.False(t, res.IsError, resultText(t, res))
	require.EqualValues(t, 2, user.asked.Load())
	require.Len(t, mock.calls(), 2)
	require.Contains(t, user.lastParams(t).Message, "can unmask data, for you too")
}

func TestAdministerMaskingRulesPutsTheUnmaskEffectInTheQuestionWhenItsLineDoesNotFit(t *testing.T) {
	fullMask := &v1pb.Algorithm{Mask: &v1pb.Algorithm_FullMask_{FullMask: &v1pb.Algorithm_FullMask{Substitution: "*"}}}
	mock := &maskingMock{types: []*v1pb.SemanticTypeSetting_SemanticType{semanticType("secret", "Secret", fullMask)}}
	ssn := maskingRule("r2", "SSN", `resource.column_name == "ssn"`, "bb.default")
	email := maskingRule("r3", "Email", `resource.column_name == "email"`, "bb.default-partial")
	mock.setStored(maskingRule("r1", "Pay", `resource.database_name == "hr_test" && (resource.column_name == "amount" || resource.column_name == "bonus")`, "secret"), ssn, email)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, proto.CloneOf(ssn), proto.CloneOf(email)))))
	require.Contains(t, resultText(t, res), "Not changed: the user declined")
	require.Equal(t, strings.Join([]string{
		"Change the masking rules (can unmask data, for you too), as alice@example.com?",
		"Remove, \"Secret\" (Full mask): `resource.database_name == \"hr_test\" &&",
		"> (resource.column_name == \"amount\" || resource.column_name ==",
		"> \"bonus\")`",
	}, "\n"), user.lastParams(t).Message)
	require.Equal(t, "The masking rules, in the order masking tries them:\n"+
		"1. \"SSN\" (kept), Full mask: any db, any table, column \"ssn\"\n"+
		"2. \"Email\" (kept), Partial mask: any db, any table, column \"email\"\n"+maskingRulesApply,
		choiceDescription(t, user.lastParams(t)))
	require.LessOrEqual(t, lineColumns(maskingRulesQuestionWithEffect), messageLineWidth, "the effect comes before the account, which a narrow terminal cuts")
}

// A long removal fits the message with the unmask effect in the question. With
// an added rule beside it, the change goes under the choice in one form.
func TestAdministerMaskingRulesReplacesALongRuleInOneForm(t *testing.T) {
	mock := &maskingMock{}
	long := maskingRule("r1", "Salary amounts", `resource.database_name == "hr_test" && resource.table_name == "salary" && (resource.column_name == "amount" || resource.column_name == "bonus")`, "bb.default")
	short := maskingRule("r2", "Amounts", `resource.column_name == "amount"`, "bb.default")
	mock.setStored(long)
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, short))))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		`Remove 1 rule and add 1, listed under Action.`,
		maskingRulesUnmask,
	}, "\n"), asked.Message)
	require.True(t, strings.HasPrefix(choiceDescription(t, asked), "Remove, Full mask: `resource.database_name == \"hr_test\" &&"), choiceDescription(t, asked))
	require.EqualValues(t, 1, user.asked.Load())
	require.Len(t, mock.calls(), 1)
}

// Each rule a change removes, adds or changes gets one change line, which is
// what lets the count line read the removals off the marks.
func TestAdministerMaskingRulesGivesEachChangedRuleOneLine(t *testing.T) {
	types, err := semanticTypesFromSetting(nil)
	require.NoError(t, err)
	rule := func(id, title, column, semanticType string) *v1pb.MaskingRulePolicy_MaskingRule {
		return maskingRule(id, title, `resource.column_name == "`+column+`"`, semanticType)
	}
	stored := []*v1pb.MaskingRulePolicy_MaskingRule{rule("a", "A", "a", "bb.default"), rule("b", "B", "b", "bb.default"), rule("c", "C", "c", "bb.default"), rule("d", "D", "d", "bb.default")}
	// b is retyped, renamed and moved at once; a and d are removed; c is kept.
	requested := []*v1pb.MaskingRulePolicy_MaskingRule{rule("c", "C", "c", "bb.default"), rule("b", "B2", "b", "bb.default-partial"), rule("e", "E", "e", "bb.default")}
	diff, err := diffMaskingRules(stored, requested, types, "")
	require.NoError(t, err)
	require.Equal(t, []string{"kept", "changed", "new"}, diff.marks)
	require.Len(t, diff.changes, 4, "two removals, and one line for each new or changed rule: %v", diff.changes)
	require.Equal(t, "Remove 2 rules, add 1 and change 1, listed under Action.", countLine(len(diff.changes), diff.marks, 0))
}

// BYT-10598: a removal states the unmask effect and adding a full-mask rule
// does not, so a rule one form could add fit no form that removed it. Whatever
// a form adds, a form removes or moves, and what the message states whole when
// it is added, the message states whole when it goes.
func TestAdministerMaskingRulesRemovesOrMovesEveryRuleOneFormAdds(t *testing.T) {
	types, err := semanticTypesFromSetting(nil)
	require.NoError(t, err)
	alice := withUserEmail(context.Background(), "alice@example.com")
	// form reports whether a form states the change, and whether its message
	// states it whole.
	form := func(stored, requested []*v1pb.MaskingRulePolicy_MaskingRule) (asked, whole bool) {
		diff, err := diffMaskingRules(stored, requested, types, "")
		require.NoError(t, err)
		form, err := maskingRulesForm(alice, diff, "")
		return err == nil, err == nil && !strings.Contains(form.message, "listed under Action")
	}
	for _, condition := range []func(name string) string{
		func(name string) string {
			return `resource.database_name == "hr_test" && resource.table_name == "salary" && resource.column_name == "` + name + `"`
		},
		func(name string) string {
			return `resource.environment_id == "prod" && (resource.column_name == "amount" || resource.column_name == "` + name + `")`
		},
	} {
		added, addedWhole := 0, 0
		for n := 1; n <= 800; n++ {
			for _, semanticType := range []string{"bb.default", "bb.default-partial"} {
				rule := maskingRule("r1", "Long", condition(strings.Repeat("c", n)), semanticType)
				asked, whole := form(nil, []*v1pb.MaskingRulePolicy_MaskingRule{rule})
				if !asked {
					continue
				}
				added++
				removed, removedWhole := form([]*v1pb.MaskingRulePolicy_MaskingRule{rule}, nil)
				// The other rule's move takes more rows, so this rule is the
				// one that moved.
				longer := maskingRule("r0", "Longer", condition(strings.Repeat("c", n+2*descriptionLineWidth)), semanticType)
				swap, err := diffMaskingRules([]*v1pb.MaskingRulePolicy_MaskingRule{rule, longer}, []*v1pb.MaskingRulePolicy_MaskingRule{longer, rule}, types, "")
				require.NoError(t, err)
				require.Len(t, swap.changes, 1)
				require.True(t, strings.HasPrefix(swap.changes[0].text, "Move to rule 2: "), swap.changes[0].text)
				moved, movedWhole := form([]*v1pb.MaskingRulePolicy_MaskingRule{rule, longer}, []*v1pb.MaskingRulePolicy_MaskingRule{longer, rule})
				require.True(t, removed && moved, "%q", rule.GetCondition().GetExpression())
				if whole {
					addedWhole++
					require.True(t, removedWhole && movedWhole, "%q", rule.GetCondition().GetExpression())
				}
			}
		}
		require.Greater(t, addedWhole, 0, "the sweep must reach a rule the message states whole")
		require.Greater(t, added, addedWhole, "and one it lists under the choice")
		require.Less(t, added, 1600, "and one no form states")
	}
}

// One rule too long for the message goes under the choice like a list of
// rules: the form says what it does, and the description starts with it.
func TestAdministerMaskingRulesListsARemovalTooLongForTheMessage(t *testing.T) {
	mock := &maskingMock{}
	condition := `resource.environment_id == "prod" && resource.database_name == "hr_prod" && (resource.column_name == "amount" || resource.column_name == "bonus" || resource.column_name == "commission" || resource.column_name == "allowance")`
	mock.setStored(maskingRule("r1", "Pay", condition, "bb.default"))
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false))))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		`Remove 1 rule, listed under Action.`,
		maskingRulesUnmask,
	}, "\n"), asked.Message)
	require.Equal(t, "Remove, Full mask: `"+condition+"`\nNo masking rule is left, so each column keeps its own semantic type.",
		unwrapMessage(choiceDescription(t, asked)))
	require.Len(t, mock.calls(), 1)
}

func TestAdministerMaskingRulesSendsARemovalNoFormFitsToTheConsole(t *testing.T) {
	mock := &maskingMock{}
	var columns []string
	for i := range 30 {
		columns = append(columns, fmt.Sprintf(`resource.column_name == "column_%02d"`, i))
	}
	mock.setStored(maskingRule("r1", "Pay", `resource.environment_id == "prod" && (`+strings.Join(columns, " || ")+`)`, "bb.default"))
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false))))
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "needs more lines than the 10 one confirmation form has room for under its choice")
	require.Contains(t, resultText(t, res), "make that change signed in to the Bytebase console")
	require.Contains(t, resultText(t, res), "do not shorten the rule's condition to make it fit")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
}

// The rule list grows with the workspace's rules. It is cut to the rows the
// description has, from the top, since the first matching rule decides.
func TestAdministerMaskingRulesCutsALongRuleListToTheDescription(t *testing.T) {
	mock := &maskingMock{}
	var stored []*v1pb.MaskingRulePolicy_MaskingRule
	for i := range 30 {
		stored = append(stored, maskingRule(fmt.Sprintf("r-%d", i), fmt.Sprintf("Rule %d", i+1), fmt.Sprintf(`resource.column_name == "column_%02d"`, i), "bb.default"))
	}
	mock.setStored(stored...)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	requested := append(slices.Clone(stored), maskingRule("r-new", "SSN", `resource.column_name == "ssn"`, "bb.default"))
	callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, requested...))))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Change the masking rules, as alice@example.com?`,
		`Add rule 31, Full mask: any db, any table, column "ssn"`,
		maskingRulesApply,
	}, "\n"), asked.Message)
	rows := strings.Split(choiceDescription(t, asked), "\n")
	require.Len(t, rows, descriptionLines)
	require.Equal(t, `The masking rules, in the order masking tries them:`, rows[0])
	require.Equal(t, `1. "Rule 1" (kept), Full mask: any db, any table, column "column_00"`, rows[1])
	require.Equal(t, `8. "Rule 8" (kept), Full mask: any db, any table, column "column_07"`, rows[8])
	require.Equal(t, `... and 23 more rules, tried after these.`, rows[9])
}

// askedMaskingChange sends requested to a workspace that stores stored, and
// returns the form the user was asked or the result when none was.
func askedMaskingChange(t *testing.T, stored, requested []*v1pb.MaskingRulePolicy_MaskingRule) (message, description, refusal string) {
	t.Helper()
	mock := &maskingMock{}
	mock.setStored(stored...)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, requested...))))
	if user.asked.Load() == 0 {
		return "", "", resultText(t, res)
	}
	asked := user.lastParams(t)
	return asked.Message, choiceDescription(t, asked), ""
}

// columnsCondition matches n columns of one table, in about 33 characters
// each.
func columnsCondition(table, prefix string, n int) string {
	var terms []string
	for i := range n {
		terms = append(terms, fmt.Sprintf(`resource.column_name == "%s%02d"`, prefix, i))
	}
	return `resource.table_name == "` + table + `" && (` + strings.Join(terms, " || ") + ")"
}

// The masking form states the move that takes fewer rows too: a swap that a
// form makes, a form swaps back, also when one of the two rules is too long for
// any form.
func TestAdministerMaskingRulesSwapsBackWhatItSwaps(t *testing.T) {
	swap := func(stored, requested []*v1pb.MaskingRulePolicy_MaskingRule) (message, refusal string) {
		message, _, refusal = askedMaskingChange(t, stored, requested)
		return message, refusal
	}
	columns := func(n int) string { return columnsCondition("orders", "c", n) }
	short := maskingRule("r-short", "Short", `resource.column_name == "ssn"`, "bb.default")
	for n := 2; n <= 30; n += 2 {
		many := maskingRule("r-many", "Many columns", columns(n), "bb.default")
		message, refusal := swap([]*v1pb.MaskingRulePolicy_MaskingRule{short, many}, []*v1pb.MaskingRulePolicy_MaskingRule{many, short})
		require.Empty(t, refusal, "%d columns", n)
		require.Contains(t, message, `Move to rule 2: any db, any table, column "ssn"`)
		_, refusal = swap([]*v1pb.MaskingRulePolicy_MaskingRule{many, short}, []*v1pb.MaskingRulePolicy_MaskingRule{short, many})
		require.Empty(t, refusal, "%d columns, swapped back", n)
	}

	// The move that takes fewer rows is the one stated, whichever way the
	// two rules trade places, also where both moves fit a form.
	for _, n := range []int{2, 30} {
		many := maskingRule("r-many", "Many columns", columns(n), "bb.default")
		message, _ := swap([]*v1pb.MaskingRulePolicy_MaskingRule{many, short}, []*v1pb.MaskingRulePolicy_MaskingRule{short, many})
		require.Contains(t, message, `Move to rule 1: any db, any table, column "ssn"`, "%d columns", n)
	}
	// Two moves that take the same rows: the rule that ends second is the
	// moved one, as before a move had a cost.
	email := maskingRule("r-email", "Email", `resource.column_name == "email"`, "bb.default")
	message, _ := swap([]*v1pb.MaskingRulePolicy_MaskingRule{email, short}, []*v1pb.MaskingRulePolicy_MaskingRule{short, email})
	require.Contains(t, message, `Move to rule 2: any db, any table, column "email"`)
	long := maskingRule("r-many", "Many columns", columns(30), "bb.default")
	// Two rules that are each too long for any form cannot trade places in
	// one, by either description or in steps.
	other := maskingRule("r-other", "Other columns", columnsCondition("orders", "d", 30), "bb.default")
	_, refusal := swap([]*v1pb.MaskingRulePolicy_MaskingRule{long, other}, []*v1pb.MaskingRulePolicy_MaskingRule{other, long})
	require.Contains(t, refusal, "CHANGE_TOO_LONG")
	require.Contains(t, refusal, "Move to rule 2: ")
	require.NotContains(t, refusal, "Move to rule 1: ")
	require.Contains(t, refusal, "make that change signed in to the Bytebase console")
	require.NotContains(t, refusal, "smaller steps")
}

// Two swaps in one request, each of a long rule and a short one: the two short
// rules are the moved ones, and their lines fit the message. A reorder with no
// room in one form is sent to smaller steps, and each step gets a form.
func TestAdministerMaskingRulesDescribesAReorderByTheMovesThatFit(t *testing.T) {
	wideA := maskingRule("r-a", "Many columns A", columnsCondition("orders", "a", 24), "bb.default")
	ssn := maskingRule("r-b", "SSN", `resource.column_name == "ssn"`, "bb.default")
	email := maskingRule("r-c", "Email", `resource.column_name == "email"`, "bb.default-partial")
	wideD := maskingRule("r-d", "Many columns D", columnsCondition("orders", "d", 24), "bb.default-partial")
	stored := []*v1pb.MaskingRulePolicy_MaskingRule{wideA, ssn, email, wideD}
	swapped := []*v1pb.MaskingRulePolicy_MaskingRule{ssn, wideA, wideD, email}
	for _, request := range [][2][]*v1pb.MaskingRulePolicy_MaskingRule{{stored, swapped}, {swapped, stored}} {
		message, _, refusal := askedMaskingChange(t, request[0], request[1])
		require.Empty(t, refusal)
		rows := strings.Split(message, "\n")
		require.Len(t, rows, 4)
		require.Regexp(t, `^Move to rule [12]: any db, any table, column "ssn"$`, rows[1])
		require.Regexp(t, `^Move to rule [34]: any db, any table, column "email"$`, rows[2])
	}

	// Ten swaps of a rule of two rows and a rule of one: the ten moves of two
	// rows have no room, and the ten of one row take the ten rows a form has.
	var pairs, traded []*v1pb.MaskingRulePolicy_MaskingRule
	for i := range 10 {
		twoRows := maskingRule(fmt.Sprintf("r-two-%d", i), "", columnsCondition(fmt.Sprintf("t%d", i), "c", 3), "bb.default")
		narrow := maskingRule(fmt.Sprintf("r-narrow-%d", i), "", fmt.Sprintf(`resource.column_name == "s%d"`, i), "bb.default")
		pairs, traded = append(pairs, twoRows, narrow), append(traded, narrow, twoRows)
	}
	message, description, refusal := askedMaskingChange(t, pairs, traded)
	require.Empty(t, refusal)
	require.Contains(t, message, "Change 10 rules, listed under Action.")
	require.Equal(t, descriptionLines, strings.Count(description, "\n")+1)
	require.True(t, strings.HasSuffix(description, `Move to rule 19: any db, any table, column "s9"`), description)

	// Eleven swaps of a wide rule and a narrow one have no room in one form,
	// and each has room alone.
	pairs, traded = nil, nil
	for i := range 11 {
		wide := maskingRule(fmt.Sprintf("r-wide-%d", i), "", columnsCondition(fmt.Sprintf("t%d", i), "c", 24), "bb.default")
		narrow := maskingRule(fmt.Sprintf("r-narrow-%d", i), "", fmt.Sprintf(`resource.column_name == "s%d"`, i), "bb.default")
		pairs, traded = append(pairs, wide, narrow), append(traded, narrow, wide)
	}
	_, _, refusal = askedMaskingChange(t, pairs, traded)
	require.Contains(t, refusal, "this change needs 11 lines, more than the 10 one confirmation form has room for")
	require.Contains(t, refusal, "make the change in smaller steps, a few rules at a time")
	from := pairs
	for pair := range 11 {
		step := slices.Clone(from)
		step[2*pair], step[2*pair+1] = traded[2*pair], traded[2*pair+1]
		message, _, refusal := askedMaskingChange(t, from, step)
		require.Empty(t, refusal, "pair %d alone", pair)
		require.Contains(t, message, fmt.Sprintf(`Move to rule %d: any db, any table, column "s%d"`, 2*pair+1, pair))
		from = step
	}
}

// A rule whose move fills the rows a form has for its changes is the moved
// one. With one row more its move fits no form, so it stays in place and the
// two rules it traded order with move instead.
func TestMaskingReorderLeavesInPlaceARuleWhoseMoveFitsNoForm(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	types, err := semanticTypesFromSetting(nil)
	require.NoError(t, err)
	ssn := maskingRule("r-ssn", "SSN", `resource.column_name == "ssn"`, "bb.default")
	email := maskingRule("r-email", "Email", `resource.column_name == "email"`, "bb.default-partial")
	diffOf := func(pad int) *maskingRulesDiff {
		wide := maskingRule("r-wide", "", columnsCondition("orders", "m", 16)+` || resource.column_name == "`+strings.Repeat("m", pad)+`"`, "bb.default")
		diff, err := diffMaskingRules([]*v1pb.MaskingRulePolicy_MaskingRule{wide, ssn, email}, []*v1pb.MaskingRulePolicy_MaskingRule{ssn, email, wide}, types, "")
		require.NoError(t, err)
		return diff
	}
	pad := 0
	for diffOf(pad+1).moves == 1 {
		pad++
		require.Less(t, pad, 400)
	}
	fits := diffOf(pad)
	require.Len(t, fits.changes, 1)
	require.True(t, strings.HasPrefix(fits.changes[0].text, "Move to rule 3: `resource.table_name == \"orders\" && ("), fits.changes[0].text)
	require.Len(t, wrapLine(fits.changes[0].text, descriptionLineWidth), descriptionLines, "the last move that fits takes every row a form has")
	form, err := maskingRulesForm(alice, fits, "")
	require.NoError(t, err)
	require.Contains(t, form.message, "Change 1 rule, listed under Action.")
	require.Equal(t, descriptionLines, strings.Count(form.description, "\n")+1)

	stays := diffOf(pad + 1)
	require.Equal(t, []formChange{
		{text: `Move to rule 1: any db, any table, column "ssn"`, effect: true},
		{text: `Move to rule 2: any db, any table, column "email"`, effect: true},
	}, stays.changes)
	form, err = maskingRulesForm(alice, stays, "")
	require.NoError(t, err)
	require.Contains(t, form.message, `Move to rule 1: any db, any table, column "ssn"`)
}

// Of every description of a masking reorder, a form states one whose moves
// cost the least: the fewest lines too long for any form, then the fewest
// moves, then the fewest rows. Each description's lines are derived here from
// the rules' places. The request gets a form when the stated lines have room,
// is sent to smaller steps when each of them has room alone, and to the
// console otherwise.
func TestMaskingReorderIsStatedByItsCheapestMoves(t *testing.T) {
	alice := withUserEmail(context.Background(), "alice@example.com")
	types, err := semanticTypesFromSetting(nil)
	require.NoError(t, err)
	random := rand.New(rand.NewPCG(22234, 13))
	semanticTypes := []string{"bb.default", "bb.default-partial", ""}
	var forms, toSteps, toConsole, moreMoves, fewerRows int
	for range 500 {
		n := 2 + random.IntN(5)
		var stored []*v1pb.MaskingRulePolicy_MaskingRule
		for i := range n {
			// Conditions of every length, so that a column more or less
			// decides a row.
			columns := []int{1, 1, 2, 4, 8, 16, 20, 24}[random.IntN(8)]
			condition := columnsCondition(fmt.Sprintf("t%d", i), "c", columns) + strings.Repeat(" ", random.IntN(40))
			stored = append(stored, maskingRule(fmt.Sprintf("r%d", i), []string{"", "Pay"}[random.IntN(2)], condition, semanticTypes[random.IntN(len(semanticTypes))]))
		}
		was := reorderedPlaces(random, n)
		var requested []*v1pb.MaskingRulePolicy_MaskingRule
		for _, place := range was {
			requested = append(requested, proto.CloneOf(stored[place]))
		}
		diff, err := diffMaskingRules(stored, requested, types, "LINK")
		require.NoError(t, err)
		if diff.moves == 0 {
			continue
		}
		lines, rows := make([]string, n), make([]int, n)
		for i, rule := range requested {
			lines[i] = fmt.Sprintf("Move to rule %d: %s", i+1, maskingConditionText(rule.GetCondition().GetExpression()))
			rows[i] = len(wrapLine(lines[i], descriptionLineWidth))
		}
		var least, fewest *moveTotal
		everyDescription(was, func(inOrder []bool) {
			total := &moveTotal{}
			for i, in := range inOrder {
				if !in {
					total.add(rows[i], descriptionLines)
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
				stated.add(rows[i], descriptionLines)
				want = append(want, lines[i])
			}
		}
		var got []string
		for _, change := range diff.changes {
			got = append(got, change.text)
		}
		require.Equal(t, want, got)
		require.Equal(t, *least, stated, "stored %v\nrequested %v", stored, requested)
		require.Equal(t, stated.moves, diff.moves)

		_, err = maskingRulesForm(alice, diff, "LINK")
		switch {
		case stated.tooLong > 0:
			require.False(t, advisesSteps(t, err), "stored %v\nrequested %v", stored, requested)
			toConsole++
		case stated.rows > descriptionLines:
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
	require.Greater(t, forms, 100)
	require.Greater(t, toSteps, 20)
	require.Greater(t, toConsole, 50)
	require.Greater(t, moreMoves, 15, "reorders stated by more moves than the fewest, since a rule too long for a form stays in place")
	require.Greater(t, fewerRows, 50, "reorders whose fewest moves can take more rows than the ones stated")
}

// On a form that lists its changes under the choice, a user who cannot read
// them there can ask for the change in steps, which the message states whole.
func TestAdministerMaskingRulesOffersStepsWhenAListedChangeIsChangedInChat(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored()
	var rules []*v1pb.MaskingRulePolicy_MaskingRule
	for _, environment := range []string{"a", "b", "c", "d"} {
		rules = append(rules, maskingRule("r-"+environment, "", `resource.environment_id == "`+environment+`"`, "bb.default"))
	}
	user := &asker{answer: confirmAnswer(confirmChoiceChange)}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	const changed = "Not changed: the user wants to change this request. Ask them what to change, then call administer_workspace with the revised rules."
	listed := resultText(t, callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, rules...)))))
	require.True(t, strings.HasPrefix(listed, changed+" The form listed the rule changes under its choice"), listed)
	require.Contains(t, listed, "make the change in smaller steps, a rule or two at a time, and keep each rule's condition unchanged")
	require.Contains(t, listed, "read the rules again with call_api OrgPolicyService/GetPolicy")

	whole := resultText(t, callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, rules[:2]...)))))
	require.Equal(t, changed, whole)

	// A rule the message states alone only with the unmask effect in its
	// question counts as one a step can state.
	folded := maskingRule("r-folded", "Pay", `resource.database_name == "hr_test" && resource.table_name == "salary" && (resource.column_name == "amount" || resource.column_name == "bonus")`, "bb.default")
	mock.setStored(folded)
	byFold := resultText(t, callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, rules[:2]...)))))
	require.True(t, strings.HasPrefix(byFold, changed+" The form listed the rule changes under its choice"), byFold)
	mock.setStored()

	// Steps cannot put a rule in the message that fits no message alone.
	long := maskingRule("r-long", "", `resource.environment_id == "prod" && resource.database_name == "hr_prod" && (resource.column_name == "amount" || resource.column_name == "bonus" || resource.column_name == "commission" || resource.column_name == "allowance")`, "bb.default")
	alone := resultText(t, callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, long)))))
	require.Equal(t, changed, alone)
	both := resultText(t, callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, long, rules[0])))))
	require.Equal(t, changed, both)

	// A form that lists a move offers no steps, though each of its lines fits
	// a message alone: a step is a request of its own, and its rules decide
	// which of two rules that trade places it calls moved.
	var pairs, swapped []*v1pb.MaskingRulePolicy_MaskingRule
	for i := range 4 {
		var columns []string
		for c := range 10 {
			columns = append(columns, fmt.Sprintf(`resource.column_name == "c%02d"`, c))
		}
		wide := maskingRule(fmt.Sprintf("r-wide-%d", i), "", fmt.Sprintf(`resource.table_name == "t%d" && (`, i)+strings.Join(columns, " || ")+")", "bb.default")
		narrow := maskingRule(fmt.Sprintf("r-narrow-%d", i), "", fmt.Sprintf(`resource.column_name == "s%d"`, i), "bb.default")
		pairs, swapped = append(pairs, wide, narrow), append(swapped, narrow, wide)
	}
	mock.setStored(pairs...)
	withMoves := resultText(t, callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false, swapped...)))))
	require.Equal(t, changed, withMoves)
	asked := user.lastParams(t)
	require.Contains(t, asked.Message, "Change 4 rules, listed under Action.")
	require.Contains(t, choiceDescription(t, asked), `Move to rule 1: any db, any table, column "s0"`)
	require.Empty(t, mock.calls())
}

func TestAdministerMaskingRulesAsksNothingWhenNothingChanges(t *testing.T) {
	mock := &maskingMock{}
	prod := maskingRule("r1", "Prod", `resource.environment_id == "prod"`, "bb.default")
	mock.setStored(prod)
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
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
		"declined":  {&mcp.ElicitResult{Action: "decline"}, "Not changed: the user declined, so the masking rules stay as they are"},
		"cancelled": {&mcp.ElicitResult{Action: "cancel"}, "Not changed: the user cancelled, so the masking rules stay as they are"},
		"changed in chat": {confirmAnswer(confirmChoiceChange),
			"Not changed: the user wants to change this request. Ask them what to change, then call administer_workspace with the revised rules."},
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
