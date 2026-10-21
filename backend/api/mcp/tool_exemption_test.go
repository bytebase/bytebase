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
		reply(map[string]any{"name": request["name"], "project": m.databaseProject})
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
		"schema": "public", "table": "employee", "column": "salary",
		"expireTime": "2026-10-27T00:00:00Z", "reason": "payroll audit",
	}
}

func TestGrantMaskingExemptionCreatesTheFirstExemption(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr"}
	user := &asker{answer: confirmAnswer(true)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), user.asked.Load())
	require.Equal(t, strings.Join([]string{
		`Exempt "user:bob@example.com" from masking, as alice@example.com?`,
		`Instance "prod"`,
		`Database "hr", column "salary" in "public"."employee"`,
		`Unmasked in query and export results until "2026-10-27T00:00:00.000Z". Reason: "payroll audit"`,
	}, "\n"), user.lastParams(t).Message)

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
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(true)}).handle})

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
	user := &asker{answer: confirmAnswer(true)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "database": "projects/hr/instances/team/databases/payroll",
	})
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, user.lastParams(t).Message, "Instance \"team\"\nDatabase \"payroll\", all tables")
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
		"a table":  {map[string]any{"table": "employee"}, `Database "hr", any schema, table "employee"`},
		"a column": {map[string]any{"table": "employee", "column": "salary"}, `Database "hr", any schema, column "salary" in "employee"`},
		"a schema": {map[string]any{"schema": "public"}, `Database "hr", schema "public"`},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{databaseProject: "projects/hr"}
			user := &asker{answer: confirmAnswer(true)}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			arguments := map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "database": "instances/prod/databases/hr"}
			for k, v := range tc.arguments {
				arguments[k] = v
			}
			res := callExemptionTool(t, session, "grant_masking_exemption", arguments)
			require.False(t, res.IsError, resultText(t, res))
			require.Contains(t, user.lastParams(t).Message, tc.want)
		})
	}
}

func TestGrantMaskingExemptionForEveryDatabaseIsExplicit(t *testing.T) {
	mock := &policyMock{}
	user := &asker{answer: confirmAnswer(true)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", map[string]any{
		"project": "projects/hr", "member": "user:bob@example.com", "allDatabases": true,
	})
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, strings.Join([]string{
		`Exempt "user:bob@example.com" from masking, as alice@example.com?`,
		`Project "hr"`,
		`Scope: all databases in the project, current and future`,
		`Unmasked in query and export results, with no expiry.`,
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
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(true)}).handle})
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
		"column without table":  {with(map[string]any{"database": "instances/prod/databases/hr", "column": "salary"}), "column needs table"},
		"a malformed time":      {with(map[string]any{"allDatabases": true, "expireTime": "next week"}), "invalid expireTime"},
		"a malformed database":  {with(map[string]any{"database": "hr"}), "invalid database"},
		"no member":             {map[string]any{"project": "projects/hr", "member": " ", "allDatabases": true}, "member is required"},
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
	user := &asker{answer: confirmAnswer(true)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "instances/prod/databases/hr belongs to projects/finance, not projects/hr")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
}

func TestGrantMaskingExemptionIsStaleWhenTheListChangesWhileAsking(t *testing.T) {
	mock := &policyMock{databaseProject: "projects/hr"}
	user := &asker{answer: confirmAnswer(true), whileAsked: func() {
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
	accept := mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(true)}

	first := call("grant_masking_exemption", salaryGrant(), nil, "")
	require.True(t, first.NeedsInput())
	wider := salaryGrant()
	delete(wider, "column")
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

func TestGrantMaskingExemptionChangesNothingUnlessConfirmed(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":           {&mcp.ElicitResult{Action: "decline"}, "the user declined"},
		"cancelled":          {&mcp.ElicitResult{Action: "cancel"}, "the user cancelled"},
		"accepted, unticked": {confirmAnswer(false), "the form was submitted without the confirmation checked"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &policyMock{databaseProject: "projects/hr"}
			session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: tc.answer}).handle})
			res := callExemptionTool(t, session, "grant_masking_exemption", salaryGrant())
			require.False(t, res.IsError, resultText(t, res))
			require.Contains(t, resultText(t, res), "The exemption for user:bob@example.com was not granted: "+tc.want+", so nothing changed")
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
	user := &asker{answer: confirmAnswer(true)}
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
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(true)}).handle})

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
	user := &asker{answer: confirmAnswer(true)}
	session := connectToPolicyServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callExemptionTool(t, session, "revoke_masking_exemption", map[string]any{"project": "projects/hr", "member": "user:bob@example.com", "condition": condition})
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, strings.Join([]string{
		`Revoke masking exemption: "user:bob@example.com", as alice@example.com?`,
		`Project "hr"`,
		`Condition (2 entries) "(resource.instance_id == \"prod\" && resource.database_name == \"hr\")"`,
		`Masked again in their results unless another exemption covers the data.`,
	}, "\n"), user.lastParams(t).Message)
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
	require.Contains(t, user.lastParams(t).Message, `Condition "" (all databases, no expiry)`)
	require.Empty(t, mock.calls())
}

func TestRevokeMaskingExemptionShowsTheScopeBeforeTheExpiry(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	for name, tc := range map[string]struct {
		condition, want string
	}{
		"the console's form": {
			`request.time < timestamp("2026-10-27T00:00:00.000Z") && (resource.instance_id == "prod" && resource.database_name == "hr")`,
			`Condition "(resource.instance_id == \"prod\" && resource.database_name == \"hr\")", until "2026-10-27T00:00:00.000Z"`,
		},
		"an expiry only": {`request.time < timestamp("2026-10-27T00:00:00.000Z")`, `Condition "", until "2026-10-27T00:00:00.000Z"`},
		"another form":   {`resource.database_name == "hr" && request.time < timestamp("2026-10-27T00:00:00.000Z")`, `Condition "resource.database_name == \"hr\" && request.time < timestamp(\"2026-10-27T00:00:00.000Z\")"`},
	} {
		t.Run(name, func(t *testing.T) {
			lines := strings.Split(describeExemptionRevoke(ctx, RevokeMaskingExemptionInput{Project: "projects/hr", Member: "user:bob@example.com", Condition: tc.condition}, 1), "\n")
			require.Equal(t, tc.want, lines[2])
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
			user := &asker{answer: confirmAnswer(true)}
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

func TestMaskingExemptionFormsKeepAMemberOnItsLine(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	forged := "user:bob@example.com\nInstance \"dev\"\nDatabase \"sandbox\", table \"t\""
	for name, message := range map[string]string{
		"grant":  describeExemptionGrant(ctx, &exemptionGrant{project: "projects/hr", member: forged}),
		"revoke": describeExemptionRevoke(ctx, RevokeMaskingExemptionInput{Project: "projects/hr", Member: forged}, 1),
	} {
		view := claudeCodeView(message, 120)
		require.Len(t, view, 4, "%s: %q", name, view)
		require.Equal(t, `Project "hr"`, view[1], "%s: a line break in the member stays on line 1, escaped: %q", name, view)
	}
}

func TestMaskingExemptionFormsShowTheScopeInClaudeCodesDialog(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alexander.smith@example.com")
	longest := strings.Repeat("x", 63)
	grantView := func(g *exemptionGrant, columns int) []string {
		return claudeCodeView(describeExemptionGrant(ctx, g), columns)
	}
	for _, columns := range []int{80, 120} {
		t.Run(fmt.Sprintf("%d columns", columns), func(t *testing.T) {
			member := "serviceAccount:deploy@hr-analytics-prod.service.bytebase.com"
			view := grantView(&exemptionGrant{project: "projects/hr", member: member, database: "d", instanceID: longest, databaseName: longest, table: "t", column: "c"}, columns)
			require.Len(t, view, 4, "%q", view)
			require.True(t, strings.HasPrefix(view[0], fmt.Sprintf("Exempt %q", member)), "a %d-character member shows whole, its project included: %q", len(member), view)
			require.Equal(t, fmt.Sprintf("Instance %q", longest), view[1], "the longest instance ID shows whole")
			require.True(t, strings.HasPrefix(view[2], `Database "`+longest), "the longest PostgreSQL database name shows whole, and the narrowing after it is cut: %q", view)
			require.True(t, strings.HasPrefix(view[3], "Unmasked in query and export results"), "%q", view)

			west := grantView(&exemptionGrant{member: "user:bob@example.com", database: "d", instanceID: "production-analytics-platform-west", databaseName: "hr"}, columns)
			east := grantView(&exemptionGrant{member: "user:bob@example.com", database: "d", instanceID: "production-analytics-platform-east", databaseName: "hr"}, columns)
			require.NotEqual(t, west, east, "two instances that differ only at the end must not look the same")
			prod := grantView(&exemptionGrant{member: "user:bob@example.com", database: "d", instanceID: longest, databaseName: "payroll_reporting_prod"}, columns)
			test := grantView(&exemptionGrant{member: "user:bob@example.com", database: "d", instanceID: longest, databaseName: "payroll_reporting_test"}, columns)
			require.NotEqual(t, prod, test, "two databases that differ only at the end must not look the same")

			all := grantView(&exemptionGrant{project: "projects/" + longest, member: "user:bob@example.com", expireTime: "2026-10-27T00:00:00.000Z", note: strings.Repeat("r", 80)}, columns)
			require.Len(t, all, 4, "%q", all)
			require.Equal(t, fmt.Sprintf("Project %q", longest), all[1])
			require.Equal(t, "Scope: all databases in the project, current and future", all[2])
			require.True(t, strings.HasPrefix(all[3], `Unmasked in query and export results until "2026-10-27T00:00:00.000Z".`), "the expiry shows before the reason is cut: %q", all)

			revoke := claudeCodeView(describeExemptionRevoke(ctx, RevokeMaskingExemptionInput{Project: "projects/" + longest, Member: "user:bob@example.com", Condition: strings.Repeat("c", 200)}, 3), columns)
			require.Len(t, revoke, 4, "%q", revoke)
			require.Equal(t, fmt.Sprintf("Project %q", longest), revoke[1])
			require.True(t, strings.HasPrefix(revoke[2], "Condition (3 entries) \""), "%q", revoke)
			require.NotContains(t, revoke[3], "…", "the effect line fits whole: %q", revoke)
		})
	}
}
