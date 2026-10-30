package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// queryCall is one SQLService/Query request the confirming mock received.
type queryCall struct {
	Statement string
	Confirmed bool
}

// confirmingQueryMock serves the database list and a Query that refuses an
// INSERT, as SQLService/Query refuses a possible write, until the request
// carries the confirmation of that exact request.
type confirmingQueryMock struct {
	databases   http.Handler
	environment string
	// refusesConfirmed has Query refuse an INSERT even when it is confirmed.
	refusesConfirmed bool
	mu               sync.Mutex
	queries          []queryCall
}

func newConfirmingQueryMock(databases []map[string]any) *confirmingQueryMock {
	return &confirmingQueryMock{databases: mockListDatabases(databases), environment: "prod"}
}

func (m *confirmingQueryMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "SQLService/Query") {
		m.databases.ServeHTTP(w, r)
		return
	}
	var body struct {
		Name      string `json:"name"`
		Statement string `json:"statement"`
		Schema    string `json:"schema"`
		Container string `json:"container"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	marker, marked := auth.MCPConfirmedDecisionFromContext(r.Context())
	confirmed := marked && bytes.Equal(marker.Digest, auth.MCPStatementDigest(body.Statement, body.Name, body.Schema, body.Container))
	m.mu.Lock()
	m.queries = append(m.queries, queryCall{Statement: body.Statement, Confirmed: confirmed})
	m.mu.Unlock()
	if (!confirmed || m.refusesConfirmed) && strings.HasPrefix(body.Statement, "INSERT") {
		_ = connect.NewErrorWriter().Write(w, r, confirmationRequiredError(m.environment, body.Schema))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(makeQueryResponse([]string{"id"}, []string{"int4"}, [][]any{{"1"}}, "0.001s"))
}

func (m *confirmingQueryMock) calls() []queryCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]queryCall(nil), m.queries...)
}

// confirmationRequiredError is SQLService/Query's refusal of an unconfirmed
// possible write.
func confirmationRequiredError(environment, schema string) error {
	err := connect.NewError(connect.CodeFailedPrecondition, errors.New("this statement may write"))
	if detail, detailErr := connect.NewErrorDetail(&errdetails.ErrorInfo{
		Reason:   auth.MCPConfirmationReason,
		Domain:   auth.MCPConfirmationDomain,
		Metadata: map[string]string{"environment": environment, "schema": schema},
	}); detailErr == nil {
		err.AddDetail(detail)
	}
	return err
}

func employeeDatabase() []map[string]any {
	return []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
	}
}

func newQueryConfirmServer(t *testing.T, mock *confirmingQueryMock) (*Server, string, string) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return s, ts.URL + mcpResourcePath, mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
}

func runAnswer(choice any) *mcp.ElicitResult {
	return &mcp.ElicitResult{Action: "accept", Content: map[string]any{runChoiceField: choice}}
}

func queryParams(statement string) *mcp.CallToolParams {
	return &mcp.CallToolParams{Name: "query_database", Arguments: map[string]any{"database": "employee_db", "statement": statement}}
}

func callQuery(t *testing.T, session *mcp.ClientSession, params *mcp.CallToolParams) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, params)
	require.NoError(t, err)
	return res
}

func TestQueryDatabaseAsksBeforeAWrite(t *testing.T) {
	for _, row := range []struct {
		name        string
		databases   []map[string]any
		environment string
		schema      string
		container   string
		header      string
	}{
		{name: "the database's default schema", databases: employeeDatabase(), environment: "prod",
			header: `Run in environment "prod" on database "employee_db",` + "\n" + `instance "prod-pg":`},
		{name: "a named schema", databases: employeeDatabase(), environment: "prod", schema: "app",
			header: `Run in environment "prod" on database "employee_db",` + "\n" + `instance "prod-pg", schema "app":`},
		{name: "a project instance", databases: []map[string]any{
			makeDatabase("projects/hr/instances/prod-pg/databases/employee_db", "projects/hr/instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
		}, environment: "prod", header: `Run in environment "prod" on database "employee_db",` + "\n" + `instance "prod-pg":`},
		{name: "no environment", databases: employeeDatabase(),
			header: `Run on database "employee_db",` + "\n" + `instance "prod-pg":`},
		{name: "a CosmosDB container", databases: employeeDatabase(), environment: "prod", container: "orders",
			header: `Run in environment "prod" on database "employee_db",` + "\n" + `instance "prod-pg", container "orders":`},
	} {
		t.Run(row.name, func(t *testing.T) {
			mock := newConfirmingQueryMock(row.databases)
			mock.environment = row.environment
			_, endpoint, token := newQueryConfirmServer(t, mock)
			user := &asker{answer: runAnswer(runChoiceRun)}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			const insert = "INSERT INTO employee VALUES (1)"
			arguments := map[string]any{"database": "employee_db", "statement": insert}
			if row.schema != "" {
				arguments["schema"] = row.schema
			}
			if row.container != "" {
				arguments["container"] = row.container
			}
			res := callQuery(t, session, &mcp.CallToolParams{Name: "query_database", Arguments: arguments})
			require.False(t, res.IsError, resultText(t, res))
			require.EqualValues(t, 1, user.asked.Load())
			asked := user.lastParams(t)
			require.Equal(t, row.header+"\n"+insert, asked.Message)
			schema, err := json.Marshal(asked.RequestedSchema)
			require.NoError(t, err)
			require.Contains(t, string(schema), `"default":"run"`)
			require.Contains(t, string(schema), `"oneOf":[{"const":"run","title":"Run now"},{"const":"change","title":"Change it in chat"}]`)
			require.Contains(t, string(schema), `"description":"Run now runs it at once, with no issue or approval."`)
			require.Equal(t, []queryCall{{Statement: insert}, {Statement: insert, Confirmed: true}}, mock.calls())
		})
	}
}

func TestQueryDatabaseRunsNothingUnlessTheUserRuns(t *testing.T) {
	for _, row := range []struct {
		name   string
		answer *mcp.ElicitResult
		text   string
	}{
		{name: "change it in chat", answer: runAnswer(runChoiceChange),
			text: "Not run: the user wants to change this request. Ask them what to change, then call query_database with the revised request."},
		{name: "decline", answer: &mcp.ElicitResult{Action: "decline"},
			text: "Not run: the user declined, so nothing changed. Do not retry unless the user asks."},
		{name: "cancel", answer: &mcp.ElicitResult{Action: "cancel"},
			text: "Not run: the user cancelled, so nothing changed. Do not retry unless the user asks."},
	} {
		t.Run(row.name, func(t *testing.T) {
			mock := newConfirmingQueryMock(employeeDatabase())
			_, endpoint, token := newQueryConfirmServer(t, mock)
			user := &asker{answer: row.answer}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			res := callQuery(t, session, queryParams("INSERT INTO employee VALUES (1)"))
			require.False(t, res.IsError)
			require.Equal(t, row.text, resultText(t, res))
			require.Nil(t, res.StructuredContent, "a client shows text alone to the model whole")
			require.Equal(t, []queryCall{{Statement: "INSERT INTO employee VALUES (1)"}}, mock.calls())
		})
	}
}

func TestQueryDatabaseRunsAtOnceWhenTheClientCannotAsk(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	_, endpoint, token := newQueryConfirmServer(t, mock)
	session := connectLegacy(t, endpoint, token, nil)

	res := callQuery(t, session, queryParams("INSERT INTO employee VALUES (1)"))
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, []queryCall{{Statement: "INSERT INTO employee VALUES (1)", Confirmed: true}}, mock.calls())
}

func TestQueryDatabaseDoesNotAskAgainAfterARefusedConfirmation(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	mock.refusesConfirmed = true
	_, endpoint, token := newQueryConfirmServer(t, mock)
	user := &asker{answer: runAnswer(runChoiceRun)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callQuery(t, session, queryParams("INSERT INTO employee VALUES (1)"))
	require.True(t, res.IsError)
	require.Equal(t, "Bytebase refused this statement's confirmation, so nothing ran; asking the user again will not help", resultText(t, res))
	require.EqualValues(t, 1, user.asked.Load())
}

func TestQueryDatabaseReadsWithoutAsking(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	_, endpoint, token := newQueryConfirmServer(t, mock)
	user := &asker{answer: runAnswer(runChoiceRun)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callQuery(t, session, queryParams("SELECT id FROM employee"))
	require.False(t, res.IsError, resultText(t, res))
	require.Zero(t, user.asked.Load())
	require.Equal(t, []queryCall{{Statement: "SELECT id FROM employee"}}, mock.calls())
}

func TestQueryDatabaseFormFitsClaudeCode(t *testing.T) {
	const header = `Run in environment "prod" on database "employee_db",` + "\n" + `instance "prod-pg":`
	const insert = "INSERT INTO employee VALUES (1, '"
	for _, row := range []struct {
		name      string
		statement string
		inMessage bool
	}{
		{name: "a line at the width", statement: insert + strings.Repeat("a", 35) + "')", inMessage: true},
		{name: "a line past the width", statement: insert + strings.Repeat("a", 36) + "')"},
		{name: "two lines at the width", statement: insert + strings.Repeat("a", 35) + "')\n" + insert + "b')", inMessage: true},
		{name: "three lines", statement: "INSERT INTO employee\nVALUES\n(1)"},
		{name: "wide characters past the width", statement: insert + strings.Repeat("名", 18) + "')"},
		{name: "four lines", statement: "INSERT INTO employee (id, name)\nVALUES\n  (1, 'Alice'),\n  (2, 'Bob')"},
	} {
		t.Run(row.name, func(t *testing.T) {
			mock := newConfirmingQueryMock(employeeDatabase())
			_, endpoint, token := newQueryConfirmServer(t, mock)
			user := &asker{answer: runAnswer(runChoiceChange)}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			callQuery(t, session, queryParams(row.statement))
			asked := user.lastParams(t)
			schema, err := json.Marshal(asked.RequestedSchema)
			require.NoError(t, err)
			if row.inMessage {
				require.Equal(t, header+"\n"+row.statement, asked.Message)
			} else {
				require.Equal(t, header+fmt.Sprintf("\nThe request is under the choice (%d characters).", utf8.RuneCountInString(row.statement))+
					"\nRun now runs it at once, with no issue or approval.", asked.Message)
				description, err := json.Marshal(row.statement)
				require.NoError(t, err)
				require.Contains(t, string(schema), `"description":`+string(description))
			}
			for _, line := range claudeCodeView(asked.Message, 80) {
				require.NotContains(t, line, "…", "an 80-column terminal cuts a line of the message")
			}
		})
	}
}

func TestQueryDatabaseShowsLongNamesWhole(t *testing.T) {
	// The longest PostgreSQL database name, and the longest resource ID.
	longName := "payroll_" + strings.Repeat("x", 55)
	longEnvironment := "prod-" + strings.Repeat("x", 58)
	const runsAtOnce = "Run now runs it at once, with no issue or approval."
	for _, row := range []struct {
		name        string
		database    string
		environment string
		schema      string
		container   string
		statement   string
		message     []string
		description string
	}{
		{name: "one line left for the request", database: longName, environment: "prod", statement: "INSERT INTO employee VALUES (1)",
			message:     []string{`Run in "prod":`, fmt.Sprintf("Database %q", longName), `Instance "prod-pg"`, "INSERT INTO employee VALUES (1)"},
			description: runsAtOnce},
		{name: "a request longer than the line left", database: longName, environment: "prod", statement: "INSERT INTO employee\nVALUES (1)",
			message:     []string{`Run in "prod":`, fmt.Sprintf("Database %q", longName), `Instance "prod-pg"`, "The request is under the choice (31 characters)."},
			description: "INSERT INTO employee\nVALUES (1)\n\n" + runsAtOnce},
		{name: "no line left", database: longName, environment: "prod", schema: "app", statement: "INSERT INTO employee VALUES (1)",
			message:     []string{`Run in "prod":`, fmt.Sprintf("Database %q", longName), `Instance "prod-pg"`, `Schema "app"`},
			description: "INSERT INTO employee VALUES (1)\n\n" + runsAtOnce},
		{name: "the longest environment ID", database: "employee_db", environment: longEnvironment, statement: "INSERT INTO employee VALUES (1)",
			message:     []string{fmt.Sprintf("Run in %q:", longEnvironment), `Database "employee_db"`, `Instance "prod-pg"`, "INSERT INTO employee VALUES (1)"},
			description: runsAtOnce},
		{name: "no environment", database: longName, statement: "INSERT INTO employee VALUES (1)",
			message:     []string{fmt.Sprintf("Database %q", longName), `Instance "prod-pg"`, "INSERT INTO employee VALUES (1)"},
			description: runsAtOnce},
		{name: "a schema and a container", database: longName, environment: "prod", schema: "app", container: "orders", statement: "INSERT INTO employee VALUES (1)",
			message:     []string{`Run in "prod":`, fmt.Sprintf("Database %q", longName), `Instance "prod-pg"`, `Schema "app", container "orders"`},
			description: "INSERT INTO employee VALUES (1)\n\n" + runsAtOnce},
	} {
		t.Run(row.name, func(t *testing.T) {
			mock := newConfirmingQueryMock([]map[string]any{
				makeDatabase("instances/prod-pg/databases/"+row.database, "instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
			})
			mock.environment = row.environment
			_, endpoint, token := newQueryConfirmServer(t, mock)
			user := &asker{answer: runAnswer(runChoiceChange)}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			arguments := map[string]any{"database": row.database, "statement": row.statement}
			if row.schema != "" {
				arguments["schema"] = row.schema
			}
			if row.container != "" {
				arguments["container"] = row.container
			}
			callQuery(t, session, &mcp.CallToolParams{Name: "query_database", Arguments: arguments})
			asked := user.lastParams(t)
			require.Equal(t, strings.Join(row.message, "\n"), asked.Message)
			schema, err := json.Marshal(asked.RequestedSchema)
			require.NoError(t, err)
			description, err := json.Marshal(row.description)
			require.NoError(t, err)
			require.Contains(t, string(schema), `"description":`+string(description))
			for _, line := range claudeCodeView(asked.Message, 80) {
				require.NotContains(t, line, "…", "an 80-column terminal cuts a line of the message")
			}
		})
	}
}

func TestQueryDatabaseAsksOnceACall(t *testing.T) {
	const projectDatabase = "projects/hr/instances/prod-pg/databases/employee_db"
	for _, row := range []struct {
		name      string
		databases []map[string]any
		label     string
		instance  string
	}{
		{name: "a workspace instance", databases: ambiguousDatabases(), label: prodLabel, instance: "prod-pg"},
		{name: "a project instance", databases: []map[string]any{
			makeDatabase(projectDatabase, "projects/hr/instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
			makeDatabase("instances/staging-pg/databases/employee_db", "instances/staging-pg", "projects/hr", "POSTGRES", "ds-2"),
		}, label: projectDatabase + " (prod-pg, POSTGRES)", instance: "projects/hr/instances/prod-pg"},
	} {
		t.Run(row.name, func(t *testing.T) {
			mock := newConfirmingQueryMock(row.databases)
			_, endpoint, token := newQueryConfirmServer(t, mock)
			user := &asker{answer: &mcp.ElicitResult{Action: "accept", Content: map[string]any{databaseChoiceField: row.label}}}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			const insert = "INSERT INTO employee VALUES (1)"
			res := callQuery(t, session, queryParams(insert))
			require.False(t, res.IsError)
			require.Equal(t, "Not run: this statement may write, so the user confirms it in a form, and this call already asked which database to use. "+
				fmt.Sprintf("Call query_database again with database \"employee_db\" and instance %q to ask the user.", row.instance), resultText(t, res))
			require.EqualValues(t, 1, user.asked.Load())
			require.Equal(t, []queryCall{{Statement: insert}}, mock.calls())

			user.answer = runAnswer(runChoiceChange)
			again := callQuery(t, session, &mcp.CallToolParams{Name: "query_database", Arguments: map[string]any{
				"database": "employee_db", "instance": row.instance, "statement": insert,
			}})
			require.Contains(t, resultText(t, again), "the user wants to change this request")
			require.EqualValues(t, 2, user.asked.Load())
			require.True(t, strings.HasPrefix(user.lastParams(t).Message, `Run in environment "prod" on database "employee_db",`),
				"the call the text names asks to run, not which database: %q", user.lastParams(t).Message)
		})
	}
}

func TestQueryDatabaseConfirmationUnderMRTRClient(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	s, _, token := newQueryConfirmServer(t, mock)
	session := connectMRTR(t, mrtrEndpoint(t, s), token)
	call := func(statement string, responses mcp.InputResponseMap, state string) *mcp.CallToolResult {
		params := queryParams(statement)
		params.InputResponses = responses
		params.RequestState = state
		return callQuery(t, session, params)
	}
	const insert = "INSERT INTO employee VALUES (1)"
	run := mcp.InputResponseMap{confirmDecisionRequestID: runAnswer(runChoiceRun)}

	first := call(insert, nil, "")
	require.True(t, first.NeedsInput())
	require.NotEmpty(t, first.RequestState)

	unsigned := call(insert, run, "")
	require.True(t, unsigned.NeedsInput(), "an answer without the request state must be asked again")

	changed := call("INSERT INTO employee VALUES (2)", run, first.RequestState)
	require.True(t, changed.IsError)
	require.Contains(t, resultText(t, changed), "STALE_CONFIRMATION")

	moved := &mcp.CallToolParams{Name: "query_database", Arguments: map[string]any{"database": "employee_db", "statement": insert, "container": "orders"}}
	moved.InputResponses = run
	moved.RequestState = first.RequestState
	require.Contains(t, resultText(t, callQuery(t, session, moved)), "STALE_CONFIRMATION", "a confirmation is for one container")

	unknown := call(insert, mcp.InputResponseMap{confirmDecisionRequestID: runAnswer("skip")}, first.RequestState)
	require.Equal(t, "Not run: the answer chose neither option, so nothing changed.", resultText(t, unknown))

	ran := call(insert, run, first.RequestState)
	require.False(t, ran.IsError, resultText(t, ran))
	require.Equal(t, []queryCall{
		{Statement: insert},
		{Statement: insert},
		{Statement: insert, Confirmed: true},
	}, mock.calls(), "only the answer signed for this request runs it")
}

func TestQueryDatabaseReportsAnAnswerThatRunsNothingBeforeAStaleOne(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	s, _, token := newQueryConfirmServer(t, mock)
	session := connectMRTR(t, mrtrEndpoint(t, s), token)
	const insert = "INSERT INTO employee VALUES (1)"
	const database = "instances/prod-pg/databases/employee_db"
	alice := withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), "ws-test"), "alice@example.com"), "client-A")
	old := s.signConfirmationState(alice, runConfirmation{database: database}, auth.MCPStatementDigest(insert, database, "", ""), time.Now().Add(-11*time.Minute))
	call := func(answer *mcp.ElicitResult) *mcp.CallToolResult {
		params := queryParams(insert)
		params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: answer}
		params.RequestState = old
		return callQuery(t, session, params)
	}

	declined := call(&mcp.ElicitResult{Action: "decline"})
	require.False(t, declined.IsError, resultText(t, declined))
	require.Contains(t, resultText(t, declined), "Not run: the user declined")
	changed := call(runAnswer(runChoiceChange))
	require.False(t, changed.IsError, resultText(t, changed))
	require.Contains(t, resultText(t, changed), "Not run: the user wants to change this request")
	ran := call(runAnswer(runChoiceRun))
	require.True(t, ran.IsError)
	require.Contains(t, resultText(t, ran), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, ran), "the user was asked more than 10 minutes ago")
	require.Empty(t, mock.calls())
}

func TestConfirmationRequiredIn(t *testing.T) {
	body := func(err error) json.RawMessage {
		req := httptest.NewRequest(http.MethodPost, "/bytebase.v1.SQLService/Query", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		recorder := httptest.NewRecorder()
		require.NoError(t, connect.NewErrorWriter().Write(recorder, req, err))
		return recorder.Body.Bytes()
	}
	require.Equal(t, &confirmationRequired{environment: "prod", schema: "app"}, confirmationRequiredIn(body(confirmationRequiredError("prod", "app"))))
	require.Nil(t, confirmationRequiredIn(body(connect.NewError(connect.CodeFailedPrecondition, errors.New("another refusal")))))

	for _, info := range []*errdetails.ErrorInfo{
		{Reason: "OTHER", Domain: auth.MCPConfirmationDomain},
		{Reason: auth.MCPConfirmationReason, Domain: "example.com"},
	} {
		other := connect.NewError(connect.CodeFailedPrecondition, errors.New("another refusal"))
		detail, err := connect.NewErrorDetail(info)
		require.NoError(t, err)
		other.AddDetail(detail)
		require.Nil(t, confirmationRequiredIn(body(other)), "%s from %s", info.GetReason(), info.GetDomain())
	}
}
