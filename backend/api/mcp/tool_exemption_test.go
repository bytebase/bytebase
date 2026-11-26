package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const exemptionPolicyName = "projects/hr/policies/masking_exemption"

// policyMock is the internal API behind the masking exemption tools: the
// project, policy and database reads, and the update, recorded with the
// confirmation its context carried. A read for another name is not found.
type policyMock struct {
	mu sync.Mutex
	// exemptions is the policy's list; nil means the project has no policy.
	exemptions []map[string]any
	// databaseProject is the project GetDatabase reports.
	databaseProject string
	// policyReadStatus, when set, is the status GetPolicy fails with.
	policyReadStatus int
	refusal          string
	updates          []decisionCall
}

func (m *policyMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reply := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	notFound := func() {
		w.WriteHeader(http.StatusNotFound)
		reply(map[string]any{"code": "not_found", "message": "not found"})
	}
	var request map[string]any
	_ = json.NewDecoder(r.Body).Decode(&request)
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.URL.Path {
	case "/bytebase.v1.ProjectService/GetProject":
		if request["name"] != "projects/hr" {
			notFound()
			return
		}
		reply(map[string]any{"name": "projects/hr"})
	case "/bytebase.v1.OrgPolicyService/GetPolicy":
		if m.policyReadStatus != 0 {
			w.WriteHeader(m.policyReadStatus)
			reply(map[string]any{"code": "internal", "message": "the read failed"})
			return
		}
		if request["name"] != exemptionPolicyName || m.exemptions == nil {
			notFound()
			return
		}
		reply(map[string]any{"name": exemptionPolicyName, "type": "MASKING_EXEMPTION",
			"maskingExemptionPolicy": map[string]any{"exemptions": m.exemptions}})
	case "/bytebase.v1.DatabaseService/GetDatabase":
		// A database on a project-owned instance always reports its owner.
		project := m.databaseProject
		if name, ok := request["name"].(string); ok {
			if owner, _, _, err := common.GetDatabaseResourceName(name); err == nil && owner != nil {
				project = common.FormatProject(*owner)
			}
		}
		reply(map[string]any{"name": request["name"], "project": project})
	case v1connect.OrgPolicyServiceUpdatePolicyProcedure:
		call := decisionCall{path: r.URL.Path, body: request}
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(r.Context()); ok {
			call.confirmed = &confirmed
		}
		m.updates = append(m.updates, call)
		if m.refusal != "" {
			w.WriteHeader(http.StatusForbidden)
			reply(map[string]any{"code": "permission_denied", "message": m.refusal})
			return
		}
		reply(map[string]any{"name": exemptionPolicyName})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *policyMock) calls() []decisionCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]decisionCall(nil), m.updates...)
}

func newPolicyServer(t *testing.T, mock *policyMock) *Server {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	return s
}

func connectToPolicyServer(t *testing.T, mock *policyMock, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	e := echo.New()
	newPolicyServer(t, mock).RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	return connectLegacy(t, ts.URL+mcpResourcePath, token, opts)
}

func callExemptionTool(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err)
	return res
}

func writtenExemptions(t *testing.T, call decisionCall) []any {
	t.Helper()
	policy, ok := call.body["policy"].(map[string]any)
	require.True(t, ok, "%v", call.body)
	payload, ok := policy["maskingExemptionPolicy"].(map[string]any)
	require.True(t, ok, "%v", policy)
	exemptions, ok := payload["exemptions"].([]any)
	require.True(t, ok, "%v", payload)
	return exemptions
}

func salaryGrant() map[string]any {
	return map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/prod/databases/hr",
		"schema": "public", "table": "employee", "columns": []string{"salary"},
		"expireTime": "2026-10-27T00:00:00Z", "reason": "payroll audit",
	}
}

func TestGrantMaskingExemptionCreatesTheFirstExemption(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr"}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), user.asked.Load())
	require.Equal(t, strings.Join([]string{
		`Exempt "user:bob@example.com" from masking, as alice@example.com?`,
		`Database "hr" on instance "prod", column "salary" in`,
		`> "public"."employee"`,
		`Bypasses masking until "2026-10-27T00:00:00.000Z"; adds no masking.`,
	}, "\n"), user.lastParams(t).Message)
	require.Equal(t, `Reason: "payroll audit"`, choiceDescription(t, user.lastParams(t)), "the reason goes under the choice")

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, map[string]any{
		"policy": map[string]any{
			"name": exemptionPolicyName,
			"type": "MASKING_EXEMPTION",
			"maskingExemptionPolicy": map[string]any{"exemptions": []any{map[string]any{
				"members": []any{"user:bob@example.com"},
				"condition": map[string]any{
					"expression":  `request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.instance_id == "prod" && resource.database_name == "hr" && resource.schema_name == "public" && resource.table_name == "employee" && resource.column_name == "salary")`,
					"description": "payroll audit",
				},
			}}},
		},
		"updateMask":   "maskingExemptionPolicy",
		"allowMissing": true,
	}, calls[0].body, "the first exemption creates the policy, as the console does, in the console's condition form")
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure: v1connect.OrgPolicyServiceUpdatePolicyProcedure,
		Resource:  exemptionPolicyName,
	}, calls[0].confirmed)
	require.Contains(t, resultText(t, res), "Exempted user:bob@example.com from masking in projects/hr.\nMasking exemptions: https://bb.example.com/projects/hr/masking-exemption")
}

func TestGrantMaskingExemptionKeepsEveryOtherEntry(t *testing.T) {
	existing := map[string]any{
		"members": []any{"group:auditors@example.com", "user:carol@example.com"},
		"condition": map[string]any{
			"title": "Q3", "description": "made in the console", "location": "console",
			"expression": `(resource.instance_id == "prod" && resource.database_name == "hr") || (resource.instance_id == "prod" && resource.database_name == "finance")`,
		},
	}
	mock := &policyMock{databaseProject: "projects/hr", exemptions: []map[string]any{existing}}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/prod/databases/hr",
	})
	require.False(t, res.IsError, resultText(t, res))
	calls := mock.calls()
	require.Len(t, calls, 1)
	written := writtenExemptions(t, calls[0])
	require.Len(t, written, 2)
	require.Equal(t, map[string]any{
		"members": []any{"group:auditors@example.com", "user:carol@example.com"},
		"condition": map[string]any{
			"title": "Q3", "description": "made in the console", "location": "console",
			"expression": `(resource.instance_id == "prod" && resource.database_name == "hr") || (resource.instance_id == "prod" && resource.database_name == "finance")`,
		},
	}, written[0], "an entry the tool does not change is written back with every field it read")
	require.Equal(t, map[string]any{"expression": `(resource.instance_id == "prod" && resource.database_name == "hr")`}, written[1].(map[string]any)["condition"])
}

func TestGrantMaskingExemptionNamesADatabaseOnAProjectOwnedInstance(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr"}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "database": "projects/hr/instances/team/databases/payroll",
	})
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, user.lastParams(t).Message, "\n"+`Database "payroll" on instance "team", all tables`+"\n")
	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, map[string]any{"expression": `(resource.instance_id == "team" && resource.database_name == "payroll")`}, writtenExemptions(t, calls[0])[0].(map[string]any)["condition"])

	res = callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "database": "projects/finance/instances/team/databases/payroll",
	})
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "belongs to projects/finance, not projects/hr")
	require.Len(t, mock.calls(), 1)
}

func TestGrantMaskingExemptionSaysWhenATableIsInAnySchema(t *testing.T) {
	for name, tc := range map[string]struct {
		arguments map[string]any
		want      string
	}{
		"a table":     {map[string]any{"table": "employee"}, `Database "hr" on instance "prod", any schema, table "employee"`},
		"a column":    {map[string]any{"table": "employee", "columns": []string{"salary"}}, `Database "hr" on instance "prod", any schema, column "salary" in "employee"`},
		"two columns": {map[string]any{"table": "employee", "columns": []string{"first_name", "last_name"}}, `Database "hr" on instance "prod", any schema, columns "first_name", "last_name" in "employee"`},
		"a schema":    {map[string]any{"schema": "public"}, `Database "hr" on instance "prod", schema "public"`},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{databaseProject: "projects/hr"}
			user := &asker{answer: confirmAnswer(confirmChoiceApply)}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			arguments := map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/prod/databases/hr"}
			for k, v := range tc.arguments {
				arguments[k] = v
			}
			res := callExemptionTool(t, session, "grant_masking_exemption", arguments)
			require.False(t, res.IsError, resultText(t, res))
			require.Contains(t, unwrapMessage(user.lastParams(t).Message), tc.want)
		})
	}
}

func TestGrantMaskingExemptionCoversSeveralColumnsInOneEntry(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr"}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	grant := salaryGrant()
	grant["columns"] = []string{"first_name", "last_name"}
	res := callExemptionTool(t, session, "grant_masking_exemption", grant)
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), user.asked.Load())
	require.Contains(t, unwrapMessage(user.lastParams(t).Message), "\n"+`Database "hr" on instance "prod", columns "first_name", "last_name" in "public"."employee"`+"\n")
	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, map[string]any{
		"expression":  `request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.instance_id == "prod" && resource.database_name == "hr" && resource.schema_name == "public" && resource.table_name == "employee" && resource.column_name in ["first_name", "last_name"])`,
		"description": "payroll audit",
	}, writtenExemptions(t, calls[0])[0].(map[string]any)["condition"], "the columns are one entry, in the list form the console writes")
}

func TestGrantMaskingExemptionForEveryDatabaseIsExplicit(t *testing.T) {
	mock := &policyMock{}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "allDatabases": true,
	})
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, strings.Join([]string{
		`Exempt "user:bob@example.com" from masking, as alice@example.com?`,
		`Project "hr": all databases, current and future`,
		`Bypasses masking here, with no expiry; adds no masking.`,
	}, "\n"), user.lastParams(t).Message)
	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, map[string]any{"expression": ""}, writtenExemptions(t, calls[0])[0].(map[string]any)["condition"],
		"an empty condition is every database without expiry, as the proto documents")

	res = callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/ghost", "member": "user:bob@example.com", "allDatabases": true,
	})
	require.True(t, res.IsError, "an exemption under a project that does not exist would apply to one created later with that ID")
	require.Len(t, mock.calls(), 1)
}

func TestGrantMaskingExemptionRefusesBadArguments(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr"}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
	base := map[string]any{"project": "projects/hr", "member": "user:bob@example.com"}
	with := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	for name, tc := range map[string]struct {
		arguments map[string]any
		want      string
	}{
		"no scope":              {base, "needs a scope: database, or allDatabases set to true"},
		"both scopes":           {with(map[string]any{"database": "instances/prod/databases/hr", "allDatabases": true}), "not both"},
		"narrowing without one": {with(map[string]any{"allDatabases": true, "table": "employee"}), "they need database"},
		"columns without table": {with(map[string]any{"database": "instances/prod/databases/hr", "columns": []string{"salary"}}), "columns need table"},
		"a malformed time":      {with(map[string]any{"allDatabases": true, "expireTime": "next week"}), "invalid expireTime"},
		"a malformed database":  {with(map[string]any{"database": "hr"}), "invalid database"},
		"no member":             {map[string]any{"project": "projects/hr", "member": " ", "allDatabases": true}, "invalid member"},
		"every user":            {map[string]any{"project": "projects/hr", "member": "allUsers", "allDatabases": true}, "would unmask the scope for everyone"},
		"no prefix":             {map[string]any{"project": "projects/hr", "member": "bob@example.com", "allDatabases": true}, "invalid member"},
		"a prefix alone":        {map[string]any{"project": "projects/hr", "member": "user:", "allDatabases": true}, "invalid member"},
		"an invalid project":    {map[string]any{"project": "hr", "member": "user:bob@example.com", "allDatabases": true}, "invalid project"},
	} {
		t.Run(name, func(t *testing.T) {
			res := callExemptionTool(t, session, "grant_masking_exemption", tc.arguments)
			require.True(t, res.IsError)
			require.Contains(t, resultText(t, res), tc.want)
		})
	}
	require.Empty(t, mock.calls())
}

func TestGrantMaskingExemptionRefusesADatabaseInAnotherProject(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/finance"}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "instances/prod/databases/hr belongs to projects/finance, not projects/hr")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
}

func TestGrantMaskingExemptionIsStaleWhenTheListChangesWhileAsking(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr"}
	user := &asker{answer: confirmAnswer(confirmChoiceApply), whileAsked: func() {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		mock.exemptions = []map[string]any{{"members": []any{"user:carol@example.com"}, "condition": map[string]any{"expression": ""}}}
	}}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls(), "an exemption added while the form was open must not be overwritten by the old list")
}

func TestMaskingExemptionToolsBindTheAnswerToTheArguments(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr", exemptions: []map[string]any{
		{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}},
	}}
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	session := connectMRTR(t, mrtrEndpoint(t, newPolicyServer(t, mock)), token)
	call := func(name string, arguments map[string]any, responses mcp.InputResponseMap, state string) *mcp.CallToolResult {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments, InputResponses: responses, RequestState: state})
		require.NoError(t, err)
		return res
	}
	accept := mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}

	first := call("grant_masking_exemption", salaryGrant(), nil, "")
	require.True(t, first.NeedsInput())
	wider := salaryGrant()
	delete(wider, "columns")
	stale := call("grant_masking_exemption", wider, accept, first.RequestState)
	require.True(t, stale.IsError, resultText(t, stale))
	require.Contains(t, resultText(t, stale), "STALE_CONFIRMATION", "an answer resent with a wider scope is not what the user confirmed")

	revoke := map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": ""}
	first = call("revoke_masking_exemption", revoke, nil, "")
	require.True(t, first.NeedsInput())
	mock.mu.Lock()
	mock.exemptions = append(mock.exemptions, map[string]any{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}})
	mock.mu.Unlock()
	stale = call("revoke_masking_exemption", revoke, accept, first.RequestState)
	require.True(t, stale.IsError, resultText(t, stale))
	require.Contains(t, resultText(t, stale), "STALE_CONFIRMATION", "a revoke would now drop an entry the form did not count")
	require.Empty(t, mock.calls())
}

// Journey 3's exemption, as the agent sent it.
func TestMaskingExemptionFormsForTheJourneysExemption(t *testing.T) {
	condition := `(resource.instance_id == "prod-pg" && resource.database_name == "hr_prod")`
	mock := &policyMock{databaseProject: "projects/hr", exemptions: []map[string]any{
		{"members": []any{"group:hr-team@example.com"}, "condition": map[string]any{"expression": condition}},
	}}
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "group:hr-team@example.com", "database": "instances/prod-pg/databases/hr_prod",
		"reason": "HR team needs to see employee personal details (birth dates, salaries) in production",
	})
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Exempt "group:hr-team@example.com" from masking, as alice@example.com?`,
		`Database "hr_prod" on instance "prod-pg", all tables`,
		`Bypasses masking here, with no expiry; adds no masking.`,
	}, "\n"), asked.Message)
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"object","required":["choice"],"properties":{"choice":{"type":"string",
		"title":"Action",
		"description":"Reason: \"HR team needs to see employee personal details (birth dates, salaries) in production\"",
		"oneOf":[{"const":"apply","title":"Exempt"},{"const":"change","title":"Change it in chat"}],
		"default":"apply"}}}`, string(schema))

	callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "group:hr-team@example.com", "condition": condition,
	})
	asked = user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Revoke "group:hr-team@example.com"'s masking exemption, as alice@example.com?`,
		`Database "hr_prod" on instance "prod-pg", all tables`,
		`Ends their bypass of masking here, unless another exemption covers it.`,
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n")[1:] {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	schema, err = json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"object","required":["choice"],"properties":{"choice":{"type":"string",
		"title":"Action",
		"oneOf":[{"const":"apply","title":"Revoke"},{"const":"change","title":"Change it in chat"}],
		"default":"apply"}}}`, string(schema), "one matching entry needs no description")
	require.Empty(t, mock.calls())
}

// grantMessage and revokeMessage return a form's message, and why no form
// states the exemption.
func grantMessage(ctx context.Context, g *exemptionGrant) (message, tooLong string) {
	form, tooLong, _ := describeExemptionGrant(ctx, g)
	return form.message, tooLong
}

func revokeMessage(ctx context.Context, input RevokeMaskingExemptionInput) (message, tooLong string) {
	form, tooLong := describeExemptionRevoke(ctx, input)
	return form.message, tooLong
}

func TestMaskingExemptionToolsRefuseAFormTooLongBeforeAsking(t *testing.T) {
	longest := strings.Repeat("x", 63)
	const tooManyLines = "more than the 4 one confirmation form shows, so nothing changed"
	var manyColumns []string
	for i := range 60 {
		manyColumns = append(manyColumns, fmt.Sprintf("column_%02d", i))
	}
	for name, tc := range map[string]struct {
		tool      string
		arguments map[string]any
		reason    string
		want      string
	}{
		"a grant naming more columns than the description has rows for": {"grant_masking_exemption", map[string]any{
			"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/prod-analytics-replica/databases/hr_reporting",
			"table": "employee_compensation", "columns": manyColumns,
		}, "the names of this exemption's columns need 12 lines, more than the 8 one confirmation form has room for under its choice, so nothing changed",
			"exempt fewer columns at a time, or grant it signed in to the Bytebase console: https://bb.example.com/projects/hr/masking-exemption"},
		// Fewer columns can leave the reason alone under the choice, where no
		// row is counted, so the advice names the reason only.
		"a grant whose reason needs more rows than its columns leave": {"grant_masking_exemption", map[string]any{
			"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/prod-analytics-replica/databases/hr_reporting",
			"table": "employee_compensation", "columns": manyColumns[:39], "reason": strings.Repeat("HR needs birth dates and salaries for the payroll audit. ", 3)[:126],
		}, "this exemption's reason needs 3 lines under the form's choice, and the names of its columns leave it 2, so nothing changed",
			"give a shorter reason, or grant it signed in to the Bytebase console: https://bb.example.com/projects/hr/masking-exemption"},
		"a grant naming columns that do not fit even one at a time": {"grant_masking_exemption", map[string]any{
			"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/" + longest + "/databases/" + longest,
			"table": "employee", "columns": []string{"birth_date", "gender"},
		}, tooManyLines, "grant it signed in to the Bytebase console: https://bb.example.com/projects/hr/masking-exemption"},
		// The counted scope is longer than a scope that names one column.
		"a grant whose columns each fit a form, where their count does not": {"grant_masking_exemption", map[string]any{
			"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/inst-payments-eu-west/databases/payroll_archive_2026",
			"schema": "public", "table": "employee_compensation_hist", "columns": []string{"base_salary", "bonus", "ssn"},
		}, tooManyLines, "exempt fewer columns at a time, or grant it signed in to the Bytebase console: https://bb.example.com/projects/hr/masking-exemption"},
		"a grant of one long column": {"grant_masking_exemption", map[string]any{
			"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/" + longest + "/databases/" + longest,
			"table": "employee", "columns": []string{"birth_date"},
		}, tooManyLines, "grant it signed in to the Bytebase console: https://bb.example.com/projects/hr/masking-exemption"},
		"a revoke of a long condition": {"revoke_masking_exemption", map[string]any{
			"project": "projects/hr", "member": "user:bob@example.com", "condition": strings.Repeat("c", 200),
		}, tooManyLines, "remove it signed in to the Bytebase console: https://bb.example.com/projects/hr/masking-exemption"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{databaseProject: "projects/hr", exemptions: []map[string]any{
				{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": strings.Repeat("c", 200)}},
				{"members": []any{"user:" + strings.Repeat("m", 50) + "@example.com"}, "condition": map[string]any{"expression": ""}},
			}}
			user := &asker{answer: confirmAnswer(confirmChoiceApply)}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callExemptionTool(t, session, tc.tool, tc.arguments)
			require.True(t, res.IsError, resultText(t, res))
			var refusal toolError
			require.NoError(t, json.Unmarshal([]byte(resultText(t, res)), &refusal))
			require.Equal(t, "CHANGE_TOO_LONG", refusal.Code)
			require.Contains(t, refusal.Message, tc.reason)
			require.Equal(t, tc.want, refusal.Suggestion)
			require.Zero(t, user.asked.Load(), "the user is not asked about a form that cannot show the change whole")
			require.Empty(t, mock.calls())
		})
	}
}

func TestRevokeMaskingExemptionChangedInChatChangesNothing(t *testing.T) {
	mock := &policyMock{exemptions: []map[string]any{{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}}}}
	user := &asker{answer: confirmAnswer(confirmChoiceChange)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": ""})
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "The exemption for user:bob@example.com was not removed: the user wants to change this request. "+
		"Ask them what to change, such as the member or the exemption's condition, then call revoke_masking_exemption again.")
	require.Empty(t, mock.calls())
}

func TestGrantMaskingExemptionChangesNothingUnlessConfirmed(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":  {&mcp.ElicitResult{Action: "decline"}, "was not granted: the user declined, so nothing changed. Do not retry unless the user asks."},
		"cancelled": {&mcp.ElicitResult{Action: "cancel"}, "was not granted: the user cancelled, so nothing changed"},
		"changed in chat": {confirmAnswer(confirmChoiceChange), "was not granted: the user wants to change this request. " +
			"Ask them what to change, such as the member, the scope, the expiry or the reason, then call grant_masking_exemption again."},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{databaseProject: "projects/hr"}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: tc.answer}).handle})
			res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
			require.False(t, res.IsError, resultText(t, res))
			require.Contains(t, resultText(t, res), "The exemption for user:bob@example.com "+tc.want)
			require.Empty(t, mock.calls())
		})
	}
}

func TestMaskingExemptionToolsRefuseAClientThatCannotConfirm(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr", exemptions: []map[string]any{
		{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}},
	}}
	session := connectToPolicyServer(t, mock, nil)

	res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "CONFIRMATION_UNAVAILABLE")
	require.Contains(t, resultText(t, res), "grant it signed in to the Bytebase console: https://bb.example.com/projects/hr/masking-exemption")

	res = callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": ""})
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "CONFIRMATION_UNAVAILABLE")
	require.Empty(t, mock.calls())
}

func TestMaskingExemptionToolsChangeNothingWhenThePolicyReadFails(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr", policyReadStatus: http.StatusInternalServerError}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	for _, call := range []struct {
		name      string
		arguments map[string]any
	}{
		{"grant_masking_exemption", salaryGrant()},
		{"revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": ""}},
	} {
		res := callExemptionTool(t, session, call.name, call.arguments)
		require.True(t, res.IsError, call.name)
		require.Contains(t, resultText(t, res), "HTTP 500", call.name)
	}
	require.Zero(t, user.asked.Load(), "a failed read is not an empty list: the user is not asked about a write that would erase every exemption")
	require.Empty(t, mock.calls())
}

func TestMaskingExemptionToolsReportTheRefusalInTheServersWords(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr", refusal: "permission denied: bb.policies.updateMaskingExemptionPolicy",
		exemptions: []map[string]any{{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}}}}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})

	for _, call := range []struct {
		name      string
		arguments map[string]any
		done      string
	}{
		{"grant_masking_exemption", salaryGrant(), "Exempted"},
		{"revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": ""}, "Removed"},
	} {
		res := callExemptionTool(t, session, call.name, call.arguments)
		require.True(t, res.IsError, call.name)
		require.Contains(t, resultText(t, res), "HTTP 403: permission denied: bb.policies.updateMaskingExemptionPolicy", call.name)
		require.NotContains(t, resultText(t, res), call.done, call.name)
		require.Nil(t, res.StructuredContent, call.name)
	}
}

func TestRevokeMaskingExemptionRemovesTheMemberFromEveryMatch(t *testing.T) {
	condition := `(resource.instance_id == "prod" && resource.database_name == "hr")`
	untouched := map[string]any{"members": []any{"user:dave@example.com"}, "condition": map[string]any{"expression": condition, "title": "dave's"}}
	mock := &policyMock{exemptions: []map[string]any{
		{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": condition}},
		untouched,
		{"members": []any{"user:carol@example.com", "user:bob@example.com"}, "condition": map[string]any{"expression": condition, "description": "kept"}},
		{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}},
	}}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": condition})
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, strings.Join([]string{
		`Revoke "user:bob@example.com"'s masking exemption, as alice@example.com?`,
		`Database "hr" on instance "prod", all tables`,
		`Ends their bypass of masking here, unless another exemption covers it.`,
	}, "\n"), user.lastParams(t).Message)
	require.Equal(t, "Removes the member from all 2 exemptions with this condition.", choiceDescription(t, user.lastParams(t)))
	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, []any{
		map[string]any{"members": []any{"user:dave@example.com"}, "condition": map[string]any{"expression": condition, "title": "dave's"}},
		map[string]any{"members": []any{"user:carol@example.com"}, "condition": map[string]any{"expression": condition, "description": "kept"}},
		map[string]any{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}},
	}, writtenExemptions(t, calls[0]), "the member leaves every exact match, an entry left empty is dropped, and every other entry stays")
	require.Contains(t, resultText(t, res), "Removed user:bob@example.com's masking exemption in projects/hr.")
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, fmt.Sprintf(`{"project":"projects/hr","member":"user:bob@example.com","condition":%q,"link":"https://bb.example.com/projects/hr/masking-exemption"}`, condition), string(output))
}

func TestRevokeMaskingExemptionChangesNothingUnlessConfirmed(t *testing.T) {
	mock := &policyMock{exemptions: []map[string]any{{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": ""}}}}
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": ""})
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "The exemption for user:bob@example.com was not removed: the user declined, so nothing changed")
	require.Contains(t, user.lastParams(t).Message, `Project "hr": all databases, current and future`)
	require.Empty(t, mock.calls())
}

func TestRevokeMaskingExemptionNamesWhatItEnds(t *testing.T) {
	for name, tc := range map[string]struct {
		condition, scope, expiry string
	}{
		"the console's form": {
			`request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.instance_id == "prod" && resource.database_name == "hr")`,
			`Database "hr" on instance "prod", all tables`, "2026-10-27T00:00:00.000Z",
		},
		"an expiry only": {`request.time < timestamp("2026-10-27T00:00:00.000Z")`, `Project "hr": all databases, current and future`, "2026-10-27T00:00:00.000Z"},
		"no condition":   {"", `Project "hr": all databases, current and future`, ""},
		"a grant's columns": {
			`request.time < timestamp("2099-01-01T00:00:00.000Z") && (resource.instance_id == "prod" && resource.database_name == "hr" && resource.table_name == "employee" && resource.column_name in ["salary", "bonus"])`,
			`Database "hr" on instance "prod", any schema, columns "salary", "bonus" in "employee"`, "2099-01-01T00:00:00.000Z",
		},
		"a grant's one column": {
			`(resource.instance_id == "prod" && resource.database_name == "hr" && resource.schema_name == "public" && resource.table_name == "employee" && resource.column_name == "salary")`,
			`Database "hr" on instance "prod", column "salary" in "public"."employee"`, "",
		},
		"a grant's terms in another order": {
			`(resource.database_name == "hr" && resource.instance_id == "prod")`,
			"Project \"hr\": `(resource.database_name == \"hr\" && resource.instance_id == \"prod\")`", "",
		},
		"a column list": {
			`request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.table_name == "employee" && resource.column_name in ["first_name", "last_name"])`,
			"Project \"hr\": `resource.table_name == \"employee\" && resource.column_name in [\"first_name\", \"last_name\"]`", "2026-10-27T00:00:00.000Z",
		},
		"an expiry last": {
			`resource.database_name == "hr" && request.time < timestamp("2026-10-27T00:00:00.000Z")`,
			"Project \"hr\": `resource.database_name == \"hr\" && request.time < timestamp(\"2026-10-27T00:00:00.000Z\")`", "",
		},
		"a top-level or": {
			`request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.database_name == "hr") || (resource.database_name == "finance")`,
			"Project \"hr\": `request.time < timestamp(\"2026-10-27T00:00:00.000Z\") && (resource.database_name == \"hr\") || (resource.database_name == \"finance\")`", "",
		},
		"single-quoted parentheses around a top-level or": {
			`request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.table_name == '(') || (resource.database_name == ')')`,
			"Project \"hr\": `request.time < timestamp(\"2026-10-27T00:00:00.000Z\") && (resource.table_name == '(') || (resource.database_name == ')')`", "",
		},
		"a parenthesis in a name": {
			`request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.table_name == "a)b")`,
			"Project \"hr\": `resource.table_name == \"a)b\"`", "2026-10-27T00:00:00.000Z",
		},
		"not CEL": {`((`, "Project \"hr\": `((`", ""},
		"an empty list of tables": {
			`(resource.instance_id == "prod" && resource.database_name == "hr" && resource.table_name in [])`,
			"Project \"hr\": `(resource.instance_id == \"prod\" && resource.database_name == \"hr\" && resource.table_name in [])`", "",
		},
		"an empty list of columns": {
			`(resource.instance_id == "prod" && resource.database_name == "hr" && resource.table_name == "t" && resource.column_name in [])`,
			"Project \"hr\": `(resource.instance_id == \"prod\" && resource.database_name == \"hr\" && resource.table_name == \"t\" && resource.column_name in [])`", "",
		},
		"one column in a list": {
			`(resource.instance_id == "prod" && resource.database_name == "hr" && resource.table_name == "t" && resource.column_name in ["salary"])`,
			"Project \"hr\": `(resource.instance_id == \"prod\" && resource.database_name == \"hr\" && resource.table_name == \"t\" && resource.column_name in [\"salary\"])`", "",
		},
		"a list on an attribute with one value": {
			`(resource.instance_id in ["prod", "test"] && resource.database_name == "hr")`,
			"Project \"hr\": `(resource.instance_id in [\"prod\", \"test\"] && resource.database_name == \"hr\")`", "",
		},
		"a column with no table": {
			`(resource.instance_id == "prod" && resource.database_name == "hr" && resource.column_name == "salary")`,
			"Project \"hr\": `(resource.instance_id == \"prod\" && resource.database_name == \"hr\" && resource.column_name == \"salary\")`", "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope, expiry := revokeScope(RevokeMaskingExemptionInput{Project: "projects/hr", Member: "user:bob@example.com", Condition: tc.condition})
			require.Equal(t, tc.scope, scope)
			require.Equal(t, tc.expiry, expiry)
		})
	}
	scope, _ := revokeScope(RevokeMaskingExemptionInput{Project: "projects/hr", Member: "allUsers",
		Condition: `(resource.instance_id == "prod" && resource.database_name == "hr" && resource.table_name == "employee" && resource.column_name in ["salary", "bonus"])`})
	require.Equal(t, `Database "hr" on instance "prod", any schema, columns "salary", "bonus" in "employee"`, scope,
		"a member a grant refuses, such as allUsers, still reads in the grant's words")
}

func TestRevokeMaskingExemptionPutsTheExpiryWhereItFits(t *testing.T) {
	for name, tc := range map[string]struct {
		condition, inMessage, description string
	}{
		"a database": {
			`request.time < timestamp("2099-01-01T00:00:00.000Z") && (resource.instance_id == "prod-pg" && resource.database_name == "hr_prod")`,
			`until "2099-01-01T00:00:00.000Z"`, "",
		},
		"two columns of a table": {
			`request.time < timestamp("2099-01-01T00:00:00.000Z") && (resource.instance_id == "inst-0123456789" && resource.database_name == "db-0123456789" && resource.table_name == "employee" && resource.column_name in ["salary", "bonus"])`,
			`columns "salary", "bonus" in "employee"`, `It would expire "2099-01-01T00:00:00.000Z".`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{exemptions: []map[string]any{{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": tc.condition}}}}
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": tc.condition})
			require.False(t, res.IsError, resultText(t, res))
			asked := user.lastParams(t)
			require.LessOrEqual(t, len(strings.Split(asked.Message, "\n")), messageLines, asked.Message)
			require.Contains(t, unwrapMessage(asked.Message), tc.inMessage)
			require.Equal(t, tc.description, choiceDescription(t, asked))
		})
	}
}

func TestRevokeMaskingExemptionRefusesWithoutAMatch(t *testing.T) {
	for name, tc := range map[string]struct {
		exemptions []map[string]any
		want       string
	}{
		"no policy": {nil, "user:bob@example.com holds no exemption in projects/hr"},
		"another condition": {
			[]map[string]any{{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": "resource.database_name == \"finance\""}}},
			"holds exemptions with these conditions",
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{exemptions: tc.exemptions}
			user := &asker{answer: confirmAnswer(confirmChoiceApply)}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": `resource.database_name == "hr"`})
			require.True(t, res.IsError)
			var refusal toolError
			require.NoError(t, json.Unmarshal([]byte(resultText(t, res)), &refusal))
			require.Equal(t, "NOT_FOUND", refusal.Code)
			require.Contains(t, refusal.Suggestion, tc.want)
			if len(tc.exemptions) > 0 {
				require.Contains(t, refusal.Suggestion, `resource.database_name == \"finance\"`, "the refusal names the conditions the member holds")
			}
			require.Zero(t, user.asked.Load())
			require.Empty(t, mock.calls())
		})
	}
}

// unwrapMessage joins each continued line back to the line it continues.
func unwrapMessage(message string) string {
	return strings.ReplaceAll(message, "\n"+continuedLine, "")
}

func TestMaskingExemptionFormsKeepAMemberOnItsLine(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	forged := "user:bob@example.com\nInstance \"dev\"\nDatabase \"sandbox\", table \"t\""
	grant, _ := grantMessage(ctx, &exemptionGrant{project: "projects/hr", member: forged})
	revoke, _ := revokeMessage(ctx, RevokeMaskingExemptionInput{Project: "projects/hr", Member: forged})
	for name, message := range map[string]string{"grant": grant, "revoke": revoke} {
		lines := strings.Split(message, "\n")
		require.Len(t, lines, messageLines, "%s: %q", name, lines)
		require.True(t, strings.HasPrefix(lines[1], continuedLine), "%s: a line break in the member is escaped, and its second row is marked: %q", name, lines)
		require.True(t, strings.HasPrefix(lines[2], `Project "hr": all databases`), "%s: %q", name, lines)
	}
}

func TestMaskingExemptionFormsShowEveryValueWhole(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alexander.smith@example.com")
	long := "production-analytics-platform-west-2"
	longest := strings.Repeat("x", 63)
	member := "serviceAccount:deploy@hr-analytics-prod.service.bytebase.com"
	requireWhole := func(t *testing.T, message string, values ...string) {
		t.Helper()
		lines := strings.Split(message, "\n")
		require.LessOrEqual(t, len(lines), messageLines, "%q", lines)
		for _, line := range lines[1:] {
			require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
		}
		for _, value := range values {
			require.Contains(t, unwrapMessage(message), value)
		}
	}
	for name, g := range map[string]*exemptionGrant{
		"a long project, an expiry and a reason": {project: "projects/" + long, member: member, expireTime: "2026-10-27T00:00:00.000Z", note: strings.Repeat("r", 80)},
		"a long database and a column":           {member: member, database: "d", instanceID: "prod-pg", databaseName: long, table: "employee", columns: []string{"birth_date"}},
		"three columns": {member: member, database: "d", instanceID: "prod-pg", databaseName: "hr_prod", table: "employee",
			columns: []string{"birth_date", "gender", "salary"}},
	} {
		t.Run(name, func(t *testing.T) {
			message, tooLong := grantMessage(ctx, g)
			require.Empty(t, tooLong, "%q", message)
			requireWhole(t, message, exemptionScope(g, false))
			require.NotContains(t, message, "rrr", "the reason goes to the description")
			lines := strings.Split(message, "\n")
			require.True(t, strings.HasPrefix(lines[len(lines)-1], "Bypasses masking") && strings.HasSuffix(lines[len(lines)-1], "adds no masking."),
				"the effect takes one line: %q", lines)
		})
	}
	revoke, tooLong := revokeMessage(ctx, RevokeMaskingExemptionInput{Project: "projects/hr", Member: member,
		Condition: `request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.instance_id == "prod-pg" && resource.database_name == "hr_prod")`})
	require.Empty(t, tooLong, "%q", revoke)
	requireWhole(t, revoke, `Database "hr_prod" on instance "prod-pg", all tables, until "2026-10-27T00:00:00.000Z"`)

	_, tooLong = grantMessage(ctx, &exemptionGrant{member: member, database: "d", instanceID: longest, databaseName: longest, table: "t", columns: []string{"c"}})
	require.NotEmpty(t, tooLong, "two 63-character names and a column need more lines than a form shows")
	grant, tooLong := grantMessage(ctx, &exemptionGrant{project: "projects/" + longest, member: member})
	require.Empty(t, tooLong, "a 63-character project ID starts on the scope's first line, so the scope takes two")
	requireWhole(t, grant, longest)
	_, tooLong = revokeMessage(ctx, RevokeMaskingExemptionInput{Project: "projects/hr", Member: member, Condition: strings.Repeat("c", 200)})
	require.NotEmpty(t, tooLong, "a 200-character condition needs more lines than a form shows")

	// 61 characters, so `Exempt "…"` and `Revoke "…"` end at column 70.
	edge := "user:" + strings.Repeat("m", 44) + "@example.com"
	_, tooLong = grantMessage(ctx, &exemptionGrant{project: "projects/hr", member: edge})
	require.Empty(t, tooLong)
	_, tooLong = revokeMessage(ctx, RevokeMaskingExemptionInput{Project: "projects/hr", Member: edge})
	require.Empty(t, tooLong)
	// One character more, and the member takes a second row, whole.
	wide := edge + "m"
	grant, tooLong = grantMessage(ctx, &exemptionGrant{project: "projects/hr", member: wide})
	require.Empty(t, tooLong)
	revoke, tooLong = revokeMessage(ctx, RevokeMaskingExemptionInput{Project: "projects/hr", Member: wide})
	require.Empty(t, tooLong)
	for question, message := range map[string]string{
		`Exempt "` + wide + `" from masking, as alexander.smith@example.com?`:        grant,
		`Revoke "` + wide + `"'s masking exemption, as alexander.smith@example.com?`: revoke,
	} {
		lines := strings.Split(message, "\n")
		require.Len(t, lines, messageLines, "%q", lines)
		require.Equal(t, question, unwrapMessage(strings.Join(lines[:2], "\n")))
		require.LessOrEqual(t, lineColumns(lines[0]), messageLineWidth)
	}
}

// A list of columns the message has no room for goes under the choice, and the
// message counts it. The database, the instance and the table stay.
func TestGrantMaskingExemptionListsColumnsTheMessageHasNoRoomFor(t *testing.T) {
	arguments := map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com",
		"database": "instances/prod-payments-us-east-1/databases/payments_ledger_prod", "table": "salary",
		"columns":    []string{"amount", "bonus", "base_pay", "equity", "ssn", "iban"},
		"expireTime": "2026-12-01T00:00:00Z", "reason": "Quarter-end audit",
	}
	const scope = `Database "payments_ledger_prod" on instance "prod-payments-us-east-1",` + "\n" +
		`> any schema, 6 columns in "salary", listed under Action`
	const columns = `Columns: "amount", "bonus", "base_pay", "equity", "ssn", "iban"`

	mock := &policyMock{databaseProject: "projects/hr"}
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callExemptionTool(t, session, "grant_masking_exemption", arguments)
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Exempt "user:bob@example.com" from masking, as alice@example.com?`,
		scope,
		`Bypasses masking until "2026-12-01T00:00:00.000Z"; adds no masking.`,
	}, "\n"), asked.Message)
	require.Equal(t, columns+"\nReason: \"Quarter-end audit\"", choiceDescription(t, asked))
	require.Len(t, mock.calls(), 1)

	// What the grant wrote, a revoke states the same way: BYT-10598 was a rule
	// one form could add and no form could remove.
	grant, err := parseExemptionGrant(GrantMaskingExemptionInput{
		Project: "projects/hr", Member: "user:bob@example.com", Database: "instances/prod-payments-us-east-1/databases/payments_ledger_prod",
		Table: "salary", Columns: []string{"amount", "bonus", "base_pay", "equity", "ssn", "iban"}, ExpireTime: "2026-12-01T00:00:00Z",
	})
	require.NoError(t, err)
	mock = &policyMock{exemptions: []map[string]any{{"members": []any{"user:bob@example.com"}, "condition": map[string]any{"expression": grant.condition()}}}}
	user = &asker{answer: confirmAnswer(confirmChoiceApply)}
	session = connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res = callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "condition": grant.condition(),
	})
	require.False(t, res.IsError, resultText(t, res))
	asked = user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Revoke "user:bob@example.com"'s masking exemption, as alice@example.com?`,
		scope,
		`Ends their bypass of masking here, unless another exemption covers it.`,
	}, "\n"), asked.Message)
	require.Equal(t, columns+"\nIt would expire \"2026-12-01T00:00:00.000Z\".", choiceDescription(t, asked))
	require.Len(t, mock.calls(), 1)
}

// Whatever exemption a form grants, a form revokes: the two share the scope's
// line and the rows that name the columns.
func TestRevokeMaskingExemptionStatesEveryExemptionAFormGrants(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	granted, listed, refused := 0, 0, 0
	for _, member := range []string{"user:bob@example.com", "user:" + strings.Repeat("m", 50) + "@example.com"} {
		for _, name := range []int{4, 20, 63} {
			for n := 0; n <= 70; n++ {
				for _, expireTime := range []string{"", "2026-12-01T00:00:00Z"} {
					for _, schema := range []string{"", "public"} {
						input := GrantMaskingExemptionInput{
							Project: "projects/payments", Member: member, ExpireTime: expireTime,
							Database: "instances/" + strings.Repeat("i", name) + "/databases/" + strings.Repeat("d", name),
						}
						if n > 0 {
							input.Schema, input.Table = schema, "salary"
						}
						for i := range n {
							input.Columns = append(input.Columns, fmt.Sprintf("column_%02d", i))
						}
						g, err := parseExemptionGrant(input)
						require.NoError(t, err)
						grant, tooLong, _ := describeExemptionGrant(ctx, g)
						if tooLong != "" {
							refused++
							continue
						}
						granted++
						if len(grant.columns) > 0 {
							listed++
						}
						revoke, tooLong := describeExemptionRevoke(ctx, RevokeMaskingExemptionInput{Project: input.Project, Member: member, Condition: g.condition()})
						require.Empty(t, tooLong, "%d columns, names of %d: %q", n, name, revoke.message)
						require.Equal(t, grant.columns, revoke.columns)
						require.LessOrEqual(t, len(strings.Split(revoke.message, "\n")), messageLines)
					}
				}
			}
		}
	}
	require.Positive(t, listed, "the sweep must reach a grant that lists its columns under the choice")
	require.Greater(t, granted, listed)
	require.Positive(t, refused, "and one no form states")
}

// The rows that name the columns leave two, for a revoke's note on matching
// entries and its expiry.
func TestMaskingExemptionColumnsLeaveTheNotesTheirRows(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	input := GrantMaskingExemptionInput{
		Project: "projects/hr", Member: "user:bob@example.com", Database: "instances/prod-payments-us-east-1/databases/payments_ledger_prod",
		Table: "salary", ExpireTime: "2026-12-01T00:00:00Z",
	}
	most := 0
	for n := 4; n <= 70; n++ {
		input.Columns = append(input.Columns[:0], make([]string, n)...)
		for i := range input.Columns {
			input.Columns[i] = fmt.Sprintf("column_%02d", i)
		}
		g, err := parseExemptionGrant(input)
		require.NoError(t, err)
		grant, tooLong, instead := describeExemptionGrant(ctx, g)
		if tooLong != "" {
			require.Equal(t, fewerColumns, instead, "%d columns", n)
			continue
		}
		revoke, tooLong := describeExemptionRevoke(ctx, RevokeMaskingExemptionInput{Project: input.Project, Member: input.Member, Condition: g.condition()})
		require.Empty(t, tooLong)
		require.NotEmpty(t, revoke.expiry, "the counted scope's line has no room for the expiry")
		require.LessOrEqual(t, len(revoke.columns)+2, descriptionLines)
		most = max(most, len(grant.columns))
	}
	require.Equal(t, exemptionColumnRows, most, "the sweep must reach the most rows the columns take")
}

// Beside listed columns the reason is wrapped and counted, so a description
// that lists columns keeps to the rows an 80-by-24 terminal shows: the grant
// gets a form exactly when its reason fits the rows its columns leave. A
// reason alone stays one line of any length.
func TestGrantMaskingExemptionCountsTheReasonBesideItsColumns(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	input := GrantMaskingExemptionInput{
		Project: "projects/hr", Member: "user:bob@example.com", Database: "instances/prod-payments-us-east-1/databases/payments_ledger_prod",
		Table: "salary", ExpireTime: "2026-12-01T00:00:00Z",
	}
	text := strings.Repeat("HR needs birth dates and salaries for the payroll audit. ", 13)
	alone, formed, refused := 0, 0, 0
	for n := 1; n <= 39; n++ {
		input.Columns, input.Reason = make([]string, n), ""
		for i := range input.Columns {
			input.Columns[i] = fmt.Sprintf("column_%02d", i)
		}
		g, err := parseExemptionGrant(input)
		require.NoError(t, err)
		plain, tooLong, _ := describeExemptionGrant(ctx, g)
		require.Empty(t, tooLong, "%d columns", n)
		room := descriptionLines - len(plain.columns)
		for length := 1; length <= len(text); length++ {
			input.Reason = text[:length]
			g, err := parseExemptionGrant(input)
			require.NoError(t, err)
			line := fmt.Sprintf("Reason: %q", input.Reason)
			form, tooLong, instead := describeExemptionGrant(ctx, g)
			if len(plain.columns) == 0 {
				require.Empty(t, tooLong)
				require.Equal(t, []string{line}, form.reason, "%d columns", n)
				alone++
				continue
			}
			// The fewest rows the line takes: 68 columns, then 67 after each mark.
			rows := 1 + (max(0, len(line)-descriptionLineWidth)+descriptionLineWidth-2)/(descriptionLineWidth-1)
			if rows > room {
				require.Equal(t, fmt.Sprintf("this exemption's reason needs %d lines under the form's choice, and the names of its columns leave it %d, so nothing changed", rows, room), tooLong)
				require.Equal(t, shorterReason, instead)
				refused++
				continue
			}
			require.Empty(t, tooLong, "%d columns, a reason of %d", n, length)
			require.Equal(t, plain.columns, form.columns)
			require.Len(t, form.reason, rows, "%d columns, a reason of %d", n, length)
			require.Equal(t, line, unwrapMessage(strings.Join(form.reason, "\n")))
			for _, row := range form.reason {
				require.LessOrEqual(t, lineColumns(row), descriptionLineWidth)
			}
			formed++
		}
	}
	require.Positive(t, alone, "the sweep must reach a grant whose message names its columns")
	require.Positive(t, formed)
	require.Positive(t, refused)
}

func TestGrantMaskingExemptionShowsTheReasonUnderTheChoice(t *testing.T) {
	const reason = "HR team needs to see employee personal details (birth dates, salaries) in production"
	long := strings.Repeat("HR needs birth dates and salaries for the payroll audit. ", 13)[:700]
	for name, tc := range map[string]struct {
		columns []string
		reason  string
		want    string
	}{
		"beside listed columns, in rows": {[]string{"amount", "bonus", "base_pay", "equity", "ssn", "iban"}, reason, strings.Join([]string{
			`Columns: "amount", "bonus", "base_pay", "equity", "ssn", "iban"`,
			`Reason: "HR team needs to see employee personal details (birth`,
			`> dates, salaries) in production"`,
		}, "\n")},
		// As on main, where no description lists columns.
		"alone, as one line of any length": {[]string{"amount"}, long, fmt.Sprintf("Reason: %q", long)},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{databaseProject: "projects/hr"}
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
				"project": "projects/hr", "member": "user:bob@example.com",
				"database": "instances/prod-payments-us-east-1/databases/payments_ledger_prod", "table": "salary",
				"columns": tc.columns, "reason": tc.reason,
			})
			require.False(t, res.IsError, resultText(t, res))
			require.Equal(t, tc.want, choiceDescription(t, user.lastParams(t)))
		})
	}
}

// The advice to exempt fewer columns at a time is given exactly when each
// column gets a form alone, by the path a one-column grant takes. A long
// column name fits a message only as a count, and the count of one column is
// a character shorter than the count of two.
func TestGrantMaskingExemptionAdvisesFewerColumnsWhenEachGetsAForm(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	refused := func(input GrantMaskingExemptionInput) (bool, bool) {
		g, err := parseExemptionGrant(input)
		require.NoError(t, err)
		_, tooLong, instead := describeExemptionGrant(ctx, g)
		return tooLong != "", instead == fewerColumns
	}
	advised, sent := 0, 0
	for n := 1; n <= 63; n++ {
		input := GrantMaskingExemptionInput{
			Project: "projects/hr", Member: "user:bob@example.com",
			Database: "instances/inst-payments-eu-west/databases/" + strings.Repeat("d", n),
			Schema:   "public", Table: "employee_compensation_hist",
			Columns: []string{"base_salary_before_tax_local", "bonus_paid_in_local_currency"},
		}
		isRefused, fewer := refused(input)
		if !isRefused {
			continue
		}
		alone := true
		for _, column := range input.Columns {
			one := input
			one.Columns = []string{column}
			if oneRefused, _ := refused(one); oneRefused {
				alone = false
			}
		}
		require.Equal(t, alone, fewer, "database name of %d", n)
		if fewer {
			advised++
		} else {
			sent++
		}
	}
	require.Positive(t, advised, "the sweep must reach a grant refused only for its count of columns")
	require.Positive(t, sent, "and one whose columns fit no form")
}
