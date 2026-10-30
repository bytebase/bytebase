package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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
	databases http.Handler
	// refusesConfirmed has Query refuse an INSERT even when it is confirmed.
	refusesConfirmed bool
	mu               sync.Mutex
	queries          []queryCall
}

func newConfirmingQueryMock(databases []map[string]any) *confirmingQueryMock {
	return &confirmingQueryMock{databases: mockListDatabases(databases)}
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
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	marker, marked := auth.MCPConfirmedDecisionFromContext(r.Context())
	confirmed := marked && bytes.Equal(marker.Digest, auth.MCPStatementDigest(body.Statement, body.Name, body.Schema))
	m.mu.Lock()
	m.queries = append(m.queries, queryCall{Statement: body.Statement, Confirmed: confirmed})
	m.mu.Unlock()
	if (!confirmed || m.refusesConfirmed) && strings.HasPrefix(body.Statement, "INSERT") {
		_ = connect.NewErrorWriter().Write(w, r, confirmationRequiredError(body.Schema))
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
func confirmationRequiredError(schema string) error {
	err := connect.NewError(connect.CodeFailedPrecondition, errors.New("this statement may write"))
	if detail, detailErr := connect.NewErrorDetail(&errdetails.ErrorInfo{
		Reason:   auth.MCPConfirmationReason,
		Domain:   auth.MCPConfirmationDomain,
		Metadata: map[string]string{"environment": "prod", "schema": schema},
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

func queryParams(statement, schema string) *mcp.CallToolParams {
	arguments := map[string]any{"database": "employee_db", "statement": statement}
	if schema != "" {
		arguments["schema"] = schema
	}
	return &mcp.CallToolParams{Name: "query_database", Arguments: arguments}
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
		name    string
		schema  string
		message string
	}{
		{name: "the database's default schema", message: `Run on database "employee_db" (instance "prod-pg", environment "prod"):`},
		{name: "a named schema", schema: "app", message: `Run on database "employee_db" (instance "prod-pg", environment "prod", schema "app"):`},
	} {
		t.Run(row.name, func(t *testing.T) {
			mock := newConfirmingQueryMock(employeeDatabase())
			_, endpoint, token := newQueryConfirmServer(t, mock)
			user := &asker{answer: runAnswer(runChoiceRun)}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			const insert = "INSERT INTO employee VALUES (1)"
			res := callQuery(t, session, queryParams(insert, row.schema))
			require.False(t, res.IsError, resultText(t, res))
			require.EqualValues(t, 1, user.asked.Load())
			asked := user.lastParams(t)
			require.Equal(t, row.message+"\n"+insert, asked.Message)
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

			res := callQuery(t, session, queryParams("INSERT INTO employee VALUES (1)", ""))
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

	res := callQuery(t, session, queryParams("INSERT INTO employee VALUES (1)", ""))
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, []queryCall{{Statement: "INSERT INTO employee VALUES (1)", Confirmed: true}}, mock.calls())
}

// A confirmed call that Query still refuses reports the refusal: asking again
// in the same call would send a second form.
func TestQueryDatabaseDoesNotAskAgainAfterARefusedConfirmation(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	mock.refusesConfirmed = true
	_, endpoint, token := newQueryConfirmServer(t, mock)
	user := &asker{answer: runAnswer(runChoiceRun)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callQuery(t, session, queryParams("INSERT INTO employee VALUES (1)", ""))
	require.True(t, res.IsError)
	require.Equal(t, "the user has not confirmed this statement, so nothing ran", resultText(t, res))
	require.EqualValues(t, 1, user.asked.Load())
}

func TestQueryDatabaseReadsWithoutAsking(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	_, endpoint, token := newQueryConfirmServer(t, mock)
	user := &asker{answer: runAnswer(runChoiceRun)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callQuery(t, session, queryParams("SELECT id FROM employee", ""))
	require.False(t, res.IsError, resultText(t, res))
	require.Zero(t, user.asked.Load())
	require.Equal(t, []queryCall{{Statement: "SELECT id FROM employee"}}, mock.calls())
}

func TestQueryDatabaseShowsALongRequestUnderTheChoice(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	_, endpoint, token := newQueryConfirmServer(t, mock)
	user := &asker{answer: runAnswer(runChoiceChange)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	insert := "INSERT INTO employee (id, name)\nVALUES\n  (1, 'Alice'),\n  (2, 'Bob')"
	callQuery(t, session, queryParams(insert, ""))
	asked := user.lastParams(t)
	require.Equal(t, `Run on database "employee_db" (instance "prod-pg", environment "prod"):`+
		"\nThe request runs to 4 lines, shown in full below.\nRun now runs it at once, with no issue or approval.", asked.Message)
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	description, err := json.Marshal(insert)
	require.NoError(t, err)
	require.Contains(t, string(schema), `"description":`+string(description))
}

// The call asked which database to use, so it cannot ask again: an older client
// answers one form per call.
func TestQueryDatabaseAsksOnceACall(t *testing.T) {
	mock := newConfirmingQueryMock(ambiguousDatabases())
	_, endpoint, token := newQueryConfirmServer(t, mock)
	user := &asker{answer: &mcp.ElicitResult{Action: "accept", Content: map[string]any{databaseChoiceField: prodLabel}}}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callQuery(t, session, queryParams("INSERT INTO employee VALUES (1)", ""))
	require.False(t, res.IsError)
	require.Equal(t, "Not run: this statement may write, so the user confirms it in a form, and this call already asked which database to use. "+
		`Call query_database again with database "employee_db" and instance "prod-pg" to ask the user.`, resultText(t, res))
	require.EqualValues(t, 1, user.asked.Load())
	require.Equal(t, []queryCall{{Statement: "INSERT INTO employee VALUES (1)"}}, mock.calls())
}

func TestQueryDatabaseConfirmationUnderMRTRClient(t *testing.T) {
	mock := newConfirmingQueryMock(employeeDatabase())
	s, _, token := newQueryConfirmServer(t, mock)
	session := connectMRTR(t, mrtrEndpoint(t, s), token)
	call := func(statement string, responses mcp.InputResponseMap, state string) *mcp.CallToolResult {
		params := queryParams(statement, "")
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

func TestConfirmationRequiredIn(t *testing.T) {
	body := func(err error) json.RawMessage {
		req := httptest.NewRequest(http.MethodPost, "/bytebase.v1.SQLService/Query", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		recorder := httptest.NewRecorder()
		require.NoError(t, connect.NewErrorWriter().Write(recorder, req, err))
		return recorder.Body.Bytes()
	}
	require.Equal(t, &confirmationRequired{environment: "prod", schema: "app"}, confirmationRequiredIn(body(confirmationRequiredError("app"))))
	require.Nil(t, confirmationRequiredIn(body(connect.NewError(connect.CodeFailedPrecondition, errors.New("another refusal")))))

	other := connect.NewError(connect.CodeFailedPrecondition, errors.New("another reason"))
	detail, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: "OTHER", Domain: auth.MCPConfirmationDomain})
	require.NoError(t, err)
	other.AddDetail(detail)
	require.Nil(t, confirmationRequiredIn(body(other)))
}
