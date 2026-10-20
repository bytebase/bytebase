package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/width"
	"google.golang.org/grpc/codes"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const decideIssueName = "projects/hr/issues/42"

// issueUpdateTime is the mock issue's update time, as protojson writes it.
const issueUpdateTime = "2026-10-13T08:00:00.123456Z"

// decisionMock is the internal API behind decide_issue: the reads the
// confirmation is built from, and the two decisions, recorded with the
// confirmation each request's context carried.
type decisionMock struct {
	issue map[string]any
	plan  map[string]any
	sheet map[string]any
	grant map[string]any
	// refusal, when set, is the message the decision is refused with.
	refusal string
	// failing, when set, is a read path that answers 500.
	failing string

	mu        sync.Mutex
	decisions []decisionCall
}

type decisionCall struct {
	path      string
	body      map[string]any
	confirmed *auth.MCPConfirmedDecision
}

func (m *decisionMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reply := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	m.mu.Lock()
	issue, plan, sheet, grant := m.issue, m.plan, m.sheet, m.grant
	m.mu.Unlock()
	if r.URL.Path == m.failing {
		w.WriteHeader(http.StatusInternalServerError)
		reply(map[string]any{"code": "internal", "message": "the read failed"})
		return
	}
	switch r.URL.Path {
	case "/bytebase.v1.IssueService/GetIssue":
		reply(issue)
	case "/bytebase.v1.PlanService/GetPlan":
		reply(plan)
	case "/bytebase.v1.SheetService/GetSheet":
		reply(sheet)
	case "/bytebase.v1.AccessGrantService/GetAccessGrant":
		reply(grant)
	case "/bytebase.v1.AccessGrantService/SearchMyAccessGrants":
		var search struct{ Parent, Filter string }
		_ = json.NewDecoder(r.Body).Decode(&search)
		mine := []map[string]any{}
		if grant != nil && search.Parent == "projects/hr" && search.Filter == fmt.Sprintf("name == %q", issue["accessGrant"]) {
			mine = append(mine, grant)
		}
		reply(map[string]any{"accessGrants": mine})
	case v1connect.IssueServiceApproveIssueProcedure, v1connect.IssueServiceRejectIssueProcedure:
		call := decisionCall{path: r.URL.Path}
		_ = json.NewDecoder(r.Body).Decode(&call.body)
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(r.Context()); ok {
			call.confirmed = &confirmed
		}
		m.mu.Lock()
		m.decisions = append(m.decisions, call)
		m.mu.Unlock()
		if m.refusal != "" {
			w.WriteHeader(http.StatusForbidden)
			reply(map[string]any{"code": "permission_denied", "message": m.refusal})
			return
		}
		reply(map[string]any{"name": decideIssueName, "status": "DONE", "approvalStatus": "APPROVED"})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *decisionMock) calls() []decisionCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]decisionCall(nil), m.decisions...)
}

// update changes what the reads return while a call is under way.
func (m *decisionMock) update(change func(*decisionMock)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	change(m)
}

// changeIssueMock is a database change waiting on the second of two steps.
func changeIssueMock() *decisionMock {
	return &decisionMock{
		issue: map[string]any{
			"name":           decideIssueName,
			"title":          "Add status column",
			"type":           "DATABASE_CHANGE",
			"status":         "OPEN",
			"creator":        "users/bob@example.com",
			"plan":           "projects/hr/plans/7",
			"approvalStatus": "PENDING",
			"approvalTemplate": map[string]any{"flow": map[string]any{
				"roles": []string{"roles/projectOwner", "roles/workspaceDBA"},
			}},
			"approvers":  []map[string]any{{"status": "APPROVED", "principal": "users/carol@example.com"}},
			"updateTime": issueUpdateTime,
		},
		plan: map[string]any{"updateTime": "2026-10-13T07:00:00Z", "specs": []map[string]any{{
			"id": "spec-1",
			"changeDatabaseConfig": map[string]any{
				"targets": []string{"instances/prod/databases/hr"},
				"sheet":   "projects/hr/sheets/9",
			},
		}}},
		sheet: map[string]any{
			"content": base64.StdEncoding.EncodeToString([]byte("ALTER TABLE users\n  ADD COLUMN status VARCHAR(20);")),
		},
	}
}

// newDecideServer stands up the real /mcp route over the mock and returns the
// endpoint and a bearer for alice.
func newDecideServer(t *testing.T, mock *decisionMock) (string, string) {
	t.Helper()
	return newDecideServerOn(t, mock, newTestServerStore())
}

func newDecideServerOn(t *testing.T, mock *decisionMock, stores *testServerStore) (string, string) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(stores, profile, revalidationSecret, mock)
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return ts.URL + mcpResourcePath, mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
}

// asker is a client elicitation handler that records what it was asked and
// gives a fixed answer.
type asker struct {
	answer *mcp.ElicitResult
	err    error
	// whileAsked, when set, runs while the form is open.
	whileAsked func()

	asked atomic.Int32
	mu    sync.Mutex
	last  *mcp.ElicitParams
}

func (a *asker) handle(_ context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	a.asked.Add(1)
	a.mu.Lock()
	a.last = req.Params
	a.mu.Unlock()
	if a.whileAsked != nil {
		a.whileAsked()
	}
	return a.answer, a.err
}

func (a *asker) lastParams(t *testing.T) *mcp.ElicitParams {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	require.NotNil(t, a.last, "the user was never asked")
	return a.last
}

func confirmAnswer(checked any) *mcp.ElicitResult {
	return &mcp.ElicitResult{Action: "accept", Content: map[string]any{confirmDecisionField: checked}}
}

func callDecideIssue(t *testing.T, session *mcp.ClientSession, decision string) (*mcp.CallToolResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return session.CallTool(ctx, &mcp.CallToolParams{
		Name: "decide_issue",
		Arguments: map[string]any{
			"issue":    decideIssueName,
			"decision": decision,
			"comment":  "checked the statement",
		},
	})
}

func TestDecideIssueApprovesOnConfirmation(t *testing.T) {
	mock := changeIssueMock()
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: confirmAnswer(true)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res, err := callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), user.asked.Load(), "the user must be asked exactly once")

	asked := user.lastParams(t)
	require.Equal(t, "form", asked.Mode)
	require.Equal(t, strings.Join([]string{
		`Approve issue "projects/hr/issues/42" titled "Add status column" as alice@example.com?`,
		`Change on "instances/prod/databases/hr"`,
		`SQL: "ALTER TABLE users\n  ADD COLUMN status VARCHAR(20);"`,
		`Approval: step 2 of 2, role "roles/workspaceDBA". Requested by "bob@example.com". Comment: "checked the statement"`,
	}, "\n"), asked.Message, "Claude Code shows four lines of a form whole, and only three of a longer one")
	require.NotContains(t, asked.Message, "https://", "a form must not carry a clickable URL")
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"object","required":["confirm"],"properties":{"confirm":{"type":"boolean",
		"title":"Approve this issue",
		"description":"Check to approve the issue described above. Leaving it unchecked, declining or cancelling changes nothing."}}}`,
		string(schema))

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, v1connect.IssueServiceApproveIssueProcedure, calls[0].path)
	require.Equal(t, decideIssueName, calls[0].body["name"])
	require.Equal(t, decisionNote+"\n\nchecked the statement", calls[0].body["comment"],
		"the comment must record that the decision was confirmed in the client")
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure:       v1connect.IssueServiceApproveIssueProcedure,
		Issue:           decideIssueName,
		IssueUpdateTime: time.Date(2026, 10, 13, 8, 0, 0, 123456000, time.UTC),
	}, calls[0].confirmed, "the approval must carry the confirmation the gate admits it on, and the issue version it applies to")

	require.Contains(t, resultText(t, res), `Approved projects/hr/issues/42. Approval: "APPROVED". Issue status: DONE.`)
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"issue":"projects/hr/issues/42","decision":"approve","approvalStatus":"APPROVED",
		"status":"DONE","link":"https://bb.example.com/projects/hr/issues/42"}`, string(output))
}

func TestDecideIssueRejectsOnConfirmation(t *testing.T) {
	mock := changeIssueMock()
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: confirmAnswer(true)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res, err := callDecideIssue(t, session, "REJECT")
	require.NoError(t, err)
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, user.lastParams(t).Message, `Reject issue "projects/hr/issues/42" titled "Add status column"`)

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, v1connect.IssueServiceRejectIssueProcedure, calls[0].path)
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure:       v1connect.IssueServiceRejectIssueProcedure,
		Issue:           decideIssueName,
		IssueUpdateTime: time.Date(2026, 10, 13, 8, 0, 0, 123456000, time.UTC),
	}, calls[0].confirmed)
}

func TestDecideIssueChangesNothingUnlessConfirmed(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":           {&mcp.ElicitResult{Action: "decline"}, "the user declined"},
		"cancelled":          {&mcp.ElicitResult{Action: "cancel"}, "the user cancelled"},
		"accepted, unticked": {confirmAnswer(false), "the form was submitted without the confirmation checked"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := changeIssueMock()
			endpoint, token := newDecideServer(t, mock)
			user := &asker{answer: tc.answer}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			res, err := callDecideIssue(t, session, "approve")
			require.NoError(t, err)
			require.Equal(t, int32(1), user.asked.Load())
			require.Empty(t, mock.calls(), "an unconfirmed answer must not reach the decision")
			text := resultText(t, res)
			require.Contains(t, text, "Not approved: "+tc.want+", so nothing changed.")
			require.Contains(t, text, "https://bb.example.com/projects/hr/issues/42")
		})
	}
}

// The SDK validates a legacy client's answer against the form and fails the
// whole call on a mismatch, so a malformed answer never reaches the handler;
// confirmationOutcome covers the shapes that can.
func TestDecideIssueMalformedAnswerChangesNothing(t *testing.T) {
	mock := changeIssueMock()
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: confirmAnswer("yes")}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	_, err := callDecideIssue(t, session, "approve")
	require.ErrorContains(t, err, "elicitation result content does not match requested schema")
	require.Empty(t, mock.calls())

	for name, response := range map[string]mcp.InputResponse{
		"no answer at all":     nil,
		"a nil elicitation":    (*mcp.ElicitResult)(nil),
		"no content":           &mcp.ElicitResult{Action: "accept"},
		"a string, not a bool": confirmAnswer("true"),
		"an unknown action":    &mcp.ElicitResult{Action: "approve", Content: map[string]any{confirmDecisionField: true}},
	} {
		confirmed, why := confirmationOutcome(response)
		require.False(t, confirmed, name)
		require.NotEmpty(t, why, name)
	}
	confirmed, _ := confirmationOutcome(confirmAnswer(true))
	require.True(t, confirmed)
}

// Under the multi round-trip shape the SDK, not the handler, sends the
// elicitation, and its failure fails the call before the handler runs again.
func TestDecideIssueTransportErrorChangesNothing(t *testing.T) {
	mock := changeIssueMock()
	endpoint, token := newDecideServer(t, mock)
	user := &asker{err: errors.New("the client lost its window")}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	_, err := callDecideIssue(t, session, "approve")
	require.Error(t, err)
	require.Contains(t, err.Error(), `input request "confirm"`)
	require.Empty(t, mock.calls())
}

func TestDecideIssueRefusesAClientThatCannotConfirm(t *testing.T) {
	for name, opts := range map[string]*mcp.ClientOptions{
		"no elicitation": nil,
		"URL elicitation only": {
			Capabilities: &mcp.ClientCapabilities{
				Elicitation: &mcp.ElicitationCapabilities{URL: &mcp.URLElicitationCapabilities{}},
			},
			ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
				return confirmAnswer(true), nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := changeIssueMock()
			endpoint, token := newDecideServer(t, mock)
			session := connectLegacy(t, endpoint, token, opts)

			res, err := callDecideIssue(t, session, "approve")
			require.NoError(t, err, "a client that cannot be asked must get a result, not a protocol error")
			require.True(t, res.IsError)
			text := resultText(t, res)
			require.Contains(t, text, "CONFIRMATION_UNAVAILABLE")
			require.Contains(t, text, "decide the issue signed in to the Bytebase console: https://bb.example.com/projects/hr/issues/42")
			require.Empty(t, mock.calls())
		})
	}
}

func TestDecideIssueRefusesAnUnaskedAnswerFromAClientThatCannotConfirm(t *testing.T) {
	mock := changeIssueMock()
	endpoint, token := newDecideServer(t, mock)
	session := connectLegacy(t, endpoint, token, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:           "decide_issue",
		Arguments:      map[string]any{"issue": decideIssueName, "decision": "approve"},
		InputResponses: mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(true)},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "CONFIRMATION_UNAVAILABLE")
	require.Empty(t, mock.calls())
}

func TestDecideIssueAsksDespiteAnUnaskedAnswer(t *testing.T) {
	for name, state := range map[string]string{
		"no request state":       "",
		"a forged request state": "AAAA.BBBB",
	} {
		t.Run(name, func(t *testing.T) {
			mock := changeIssueMock()
			endpoint, token := newDecideServer(t, mock)
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			res, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:           "decide_issue",
				Arguments:      map[string]any{"issue": decideIssueName, "decision": "approve"},
				InputResponses: mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(true)},
				RequestState:   state,
			})
			require.NoError(t, err)
			require.Equal(t, int32(1), user.asked.Load(), "an answer nobody asked for must not stand in for asking")
			require.Contains(t, resultText(t, res), "Not approved: the user declined, so nothing changed.")
			require.Empty(t, mock.calls())
		})
	}
}

func TestConfirmationStateIsBoundToTheCallerAndTheDecision(t *testing.T) {
	s, err := newServerWithStore(newTestServerStore(), &config.Profile{Mode: common.ReleaseModeDev}, revalidationSecret, http.NotFoundHandler())
	require.NoError(t, err)
	caller := func(workspace, email, client string) context.Context {
		return withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), workspace), email), client)
	}
	alice := caller("ws-test", "alice@example.com", "client-A")
	approve := &issueDecision{issue: decideIssueName, procedure: v1connect.IssueServiceApproveIssueProcedure}
	digest := sha256.Sum256([]byte("what the form was built from"))
	state := s.signConfirmationState(alice, approve, digest[:])

	signedDigest, signed := s.verifyConfirmationState(alice, approve, state)
	require.True(t, signed)
	require.Equal(t, digest[:], signedDigest)

	otherDigest := sha256.Sum256([]byte("something else"))
	_, mac, _ := strings.Cut(state, ".")
	for name, tc := range map[string]struct {
		ctx      context.Context
		decision *issueDecision
		state    string
	}{
		"another user":       {caller("ws-test", "bob@example.com", "client-A"), approve, state},
		"another client":     {caller("ws-test", "alice@example.com", "client-B"), approve, state},
		"another workspace":  {caller("ws-other", "alice@example.com", "client-A"), approve, state},
		"the other decision": {alice, &issueDecision{issue: decideIssueName, procedure: v1connect.IssueServiceRejectIssueProcedure}, state},
		"another issue":      {alice, &issueDecision{issue: "projects/hr/issues/43", procedure: v1connect.IssueServiceApproveIssueProcedure}, state},
		"another digest":     {alice, approve, base64.RawURLEncoding.EncodeToString(otherDigest[:]) + "." + mac},
		"no signature":       {alice, approve, base64.RawURLEncoding.EncodeToString(digest[:])},
	} {
		_, signed := s.verifyConfirmationState(tc.ctx, tc.decision, tc.state)
		require.False(t, signed, name)
	}
}

// threeStepIssueMock is a database change waiting on the second of three steps.
func threeStepIssueMock() *decisionMock {
	mock := changeIssueMock()
	mock.issue["approvalTemplate"] = map[string]any{"flow": map[string]any{
		"roles": []string{"roles/projectOwner", "roles/workspaceDBA", "roles/workspaceAdmin"},
	}}
	return mock
}

func TestDecideIssueRefusesWhenTheIssueChangesWhileAsking(t *testing.T) {
	for name, change := range map[string]func(*decisionMock){
		"the plan's statement replaced": func(m *decisionMock) {
			m.plan = map[string]any{"specs": []map[string]any{{
				"id": "spec-1",
				"changeDatabaseConfig": map[string]any{
					"targets": []string{"instances/prod/databases/hr"},
					"sheet":   "projects/hr/sheets/10",
				},
			}}}
			m.sheet = map[string]any{"content": base64.StdEncoding.EncodeToString([]byte("DROP TABLE users;"))}
		},
		"the step decided by another approver": func(m *decisionMock) {
			m.issue["approvers"] = []map[string]any{{"status": "APPROVED"}, {"status": "APPROVED"}}
		},
		"the title changed": func(m *decisionMock) {
			m.issue["title"] = "Drop users"
		},
		"the plan edited where the form does not show it, such as prior backup": func(m *decisionMock) {
			m.plan["updateTime"] = "2026-10-13T09:00:00Z"
		},
		"the issue rejected and requested again, back at the same step": func(m *decisionMock) {
			m.issue["updateTime"] = "2026-10-13T09:00:00Z"
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := threeStepIssueMock()
			endpoint, token := newDecideServer(t, mock)
			user := &asker{answer: confirmAnswer(true), whileAsked: func() { mock.update(change) }}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			res, err := callDecideIssue(t, session, "approve")
			require.NoError(t, err)
			require.Equal(t, int32(1), user.asked.Load())
			require.True(t, res.IsError)
			text := resultText(t, res)
			require.Contains(t, text, "STALE_CONFIRMATION")
			require.Contains(t, text, "call decide_issue again")
			require.Empty(t, mock.calls(), "an answer about the issue as it was must not decide the issue as it is")
		})
	}
}

func TestDecideIssueReportsADeclineEvenWhenTheIssueChanged(t *testing.T) {
	mock := threeStepIssueMock()
	endpoint, token := newDecideServer(t, mock)
	user := &asker{
		answer:     &mcp.ElicitResult{Action: "decline"},
		whileAsked: func() { mock.update(func(m *decisionMock) { m.issue["title"] = "Drop users" }) },
	}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res, err := callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	text := resultText(t, res)
	require.Contains(t, text, "Not approved: the user declined, so nothing changed. Do not retry unless the user asks.")
	require.NotContains(t, text, "STALE_CONFIRMATION", "a stale result would send the agent back to a user who said no")
	require.Empty(t, mock.calls())
}

// newDecideMRTRServer returns a decide_issue call for a client on the
// 2026-07-28 protocol, which answers the form and echoes the request state
// itself.
func newDecideMRTRServer(t *testing.T, mock *decisionMock) func(mcp.InputResponseMap, string, string) *mcp.CallToolResult {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	session := connectMRTR(t, mrtrEndpoint(t, s), token)
	return func(responses mcp.InputResponseMap, state, comment string) *mcp.CallToolResult {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:           "decide_issue",
			Arguments:      map[string]any{"issue": decideIssueName, "decision": "approve", "comment": comment},
			InputResponses: responses,
			RequestState:   state,
		})
		require.NoError(t, err)
		return res
	}
}

func TestDecideIssueUnderMRTRClient(t *testing.T) {
	mock := changeIssueMock()
	call := newDecideMRTRServer(t, mock)
	accept := mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(true)}

	first := call(nil, "", "checked the statement")
	require.True(t, first.NeedsInput())
	require.NotEmpty(t, first.RequestState)
	asked, ok := first.InputRequests[confirmDecisionRequestID].(*mcp.ElicitParams)
	require.True(t, ok)
	require.Contains(t, asked.Message, `Approve issue "projects/hr/issues/42" titled "Add status column"`)

	unsigned := call(accept, "", "checked the statement")
	require.True(t, unsigned.NeedsInput(), "an answer without the request state must be asked again")
	require.Empty(t, mock.calls())

	swapped := call(accept, first.RequestState, "a comment the user never saw")
	require.True(t, swapped.IsError)
	require.Contains(t, resultText(t, swapped), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())

	decided := call(accept, first.RequestState, "checked the statement")
	require.False(t, decided.IsError, resultText(t, decided))
	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, decisionNote+"\n\nchecked the statement", calls[0].body["comment"])
}

// callDecideIssueWith calls decide_issue on an issue of the given type,
// optionally carrying an answer the client sends without being asked.
func callDecideIssueWith(t *testing.T, issueType string, unasked mcp.InputResponseMap) (*decisionMock, *asker, *mcp.CallToolResult, *testServerStore) {
	t.Helper()
	mock := changeIssueMock()
	mock.issue["type"] = issueType
	stores := newTestServerStore()
	endpoint, token := newDecideServerOn(t, mock, stores)
	user := &asker{answer: confirmAnswer(true)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:           "decide_issue",
		Arguments:      map[string]any{"issue": decideIssueName, "decision": "approve"},
		InputResponses: unasked,
	})
	require.NoError(t, err)
	return mock, user, res, stores
}

func TestDecideIssueRefusesARoleRequest(t *testing.T) {
	for name, unasked := range map[string]mcp.InputResponseMap{
		"asked as usual":              nil,
		"with an answer sent unasked": {confirmDecisionRequestID: confirmAnswer(true)},
	} {
		t.Run(name, func(t *testing.T) {
			mock, user, res, stores := callDecideIssueWith(t, "ROLE_GRANT", unasked)
			require.True(t, res.IsError)
			text := resultText(t, res)
			require.Contains(t, text, "ROLE_GRANT_NOT_SUPPORTED")
			require.Contains(t, text, "approving a role request grants a project role, which MCP does not do yet")
			require.Contains(t, text, "decide it signed in to the Bytebase console: https://bb.example.com/projects/hr/issues/42")
			require.Zero(t, user.asked.Load(), "no form may be sent for a role request")
			require.Empty(t, mock.calls(), "no decision, and so no confirmation, may reach the API")

			require.Len(t, stores.auditRows, 1, "the refusal must be recorded, as its creation-side twin is")
			row := stores.auditRows[0]
			require.Equal(t, "projects/hr", row.Parent)
			require.Equal(t, v1connect.IssueServiceApproveIssueProcedure, row.Method)
			require.Equal(t, decideIssueName, row.Resource)
			require.Equal(t, "users/alice@example.com", row.User)
			require.Equal(t, int32(codes.PermissionDenied), row.Status.GetCode())
			require.Contains(t, row.Status.GetMessage(), "approving a role request grants a project role")
			require.Equal(t, "client-A", row.McpDelegation.GetClientId())
		})
	}
}

func TestDecideIssueRefusesAnIssueTypeItDoesNotKnow(t *testing.T) {
	mock, user, res, stores := callDecideIssueWith(t, "SOMETHING_NEW", nil)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "ISSUE_TYPE_NOT_SUPPORTED")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
	require.Len(t, stores.auditRows, 1)
}

// The tool passes a refusal on in the server's words: the workflow's and the
// ceiling's messages each name their own remedy.
func TestDecideIssueReportsTheRefusalInTheServersWords(t *testing.T) {
	for name, refusal := range map[string]string{
		"the workflow": "cannot approve because self-approval is not allowed for this project",
		"the ceiling": "/bytebase.v1.IssueService/ApproveIssue is a WRITE method and this workspace's MCP capability ceiling is READ_ONLY, " +
			"which serves READ methods. Ask a workspace admin to raise the MCP ceiling in the workspace settings, " +
			"or perform this action signed in to the Bytebase console instead",
	} {
		t.Run(name, func(t *testing.T) {
			mock := changeIssueMock()
			mock.refusal = refusal
			endpoint, token := newDecideServer(t, mock)
			user := &asker{answer: confirmAnswer(true)}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			res, err := callDecideIssue(t, session, "approve")
			require.NoError(t, err)
			require.True(t, res.IsError)
			text := resultText(t, res)
			require.Contains(t, text, "failed to approve projects/hr/issues/42: HTTP 403: "+refusal)
			require.Contains(t, text, "check the issue's current state: https://bb.example.com/projects/hr/issues/42")
			require.NotContains(t, text, "approval rules", "the tool must not put its own cause on the server's refusal")
			require.Len(t, mock.calls(), 1, "the server, not the tool, refuses")
		})
	}
}

// accessRequestMock is an access request waiting on its only step.
func accessRequestMock() *decisionMock {
	return &decisionMock{
		issue: map[string]any{
			"name":             decideIssueName,
			"title":            "Read salaries",
			"type":             "ACCESS_GRANT",
			"creator":          "users/bob@example.com",
			"accessGrant":      "projects/hr/accessGrants/5",
			"approvalStatus":   "PENDING",
			"approvalTemplate": map[string]any{"flow": map[string]any{"roles": []string{"roles/projectOwner"}}},
			"updateTime":       issueUpdateTime,
		},
		grant: map[string]any{
			"targets":    []string{"instances/prod/databases/hr"},
			"query":      "SELECT salary\nFROM employee",
			"unmask":     true,
			"ttl":        "3600s",
			"updateTime": "2026-10-13T07:00:00Z",
		},
	}
}

func TestDecideIssueDescribesAnAccessRequest(t *testing.T) {
	require.Equal(t, strings.Join([]string{
		`Approve issue "projects/hr/issues/42" titled "Read salaries" as alice@example.com?`,
		`Access on "instances/prod/databases/hr"`,
		`Query: "SELECT salary\nFROM employee"`,
		`Unmasked: yes. Export: no. Expires: "1h0m0s" after approval. Approval: step 1 of 1, role "roles/projectOwner". Requested by "bob@example.com". Comment: "checked the statement"`,
	}, "\n"), askedMessage(t, accessRequestMock()))
}

func TestDecideIssueReadsTheCallersOwnAccessRequestAsTheConsoleDoes(t *testing.T) {
	mock := accessRequestMock()
	mock.issue["creator"] = "users/alice@example.com"
	mock.failing = "/bytebase.v1.AccessGrantService/GetAccessGrant"
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: confirmAnswer(true)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res, err := callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	require.False(t, res.IsError, "a requester without bb.accessGrants.get reads their own grant: %s", resultText(t, res))
	require.Contains(t, user.lastParams(t).Message, `Query: "SELECT salary\nFROM employee"`)
	require.Len(t, mock.calls(), 1)

	mock.update(func(m *decisionMock) { m.grant = nil })
	user = &asker{answer: confirmAnswer(true)}
	session = connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res, err = callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "is not among the access grants you requested")
	require.Zero(t, user.asked.Load(), "a form missing the grant must not be sent")
	require.Len(t, mock.calls(), 1)
}

func TestDecideIssueShowsANewDatabasesSettings(t *testing.T) {
	for name, tc := range map[string]struct {
		config map[string]any
		want   string
	}{
		"a ClickHouse cluster": {
			map[string]any{"target": "instances/ch", "database": "events", "cluster": "ch-main"},
			`Cluster "ch-main"`,
		},
		"a PostgreSQL owner": {
			map[string]any{"target": "instances/ch", "database": "events", "owner": "billing_admin"},
			`Owner "billing_admin"`,
		},
		"a MySQL character set and collation": {
			map[string]any{"target": "instances/ch", "database": "events", "characterSet": "utf8mb4", "collation": "utf8mb4_bin"},
			`Character set "utf8mb4", collation "utf8mb4_bin"`,
		},
		"a MongoDB collection": {
			map[string]any{"target": "instances/ch", "database": "events", "table": "orders"},
			`Table "orders"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := changeIssueMock()
			mock.plan = map[string]any{"specs": []map[string]any{{"createDatabaseConfig": tc.config}}}
			lines := strings.Split(askedMessage(t, mock), "\n")
			require.Equal(t, `Create database "events" on "instances/ch"`, lines[1])
			require.Equal(t, tc.want, lines[2], "each setting changes the database that is created")
		})
	}
}

func TestDecideIssueShowsTheAccessGrantsScope(t *testing.T) {
	mock := accessRequestMock()
	mock.grant["targets"] = []string{"instances/prod/databases/hr", "instances/prod/databases/hr2"}
	mock.grant["schema"] = "payroll"
	mock.grant["container"] = "employees"
	mock.grant["query"] = "SELECT salary FROM employee" + strings.Repeat(" ", 300) + "UNION SELECT ssn FROM employee"
	lines := strings.Split(askedMessage(t, mock), "\n")
	require.Equal(t, `Access on 2 databases, schema "payroll", container "employees": "instances/prod/databases/hr", "instances/prod/databases/hr2"`, lines[1],
		"the schema decides which tables an unqualified query reads, and the container is the data a Cosmos DB grant opens")
	require.True(t, strings.HasPrefix(lines[2], `Query (partial): "SELECT salary FROM employee`), lines[2])
	require.True(t, strings.HasSuffix(lines[2], `" (first 300 of 357 characters)`), lines[2])
	require.NotContains(t, lines[2], "UNION SELECT ssn")
}

func TestDecideIssueQuotesTheTitle(t *testing.T) {
	mock := changeIssueMock()
	mock.issue["title"] = "Harmless\nApproval: none needed"
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	_, err := callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	require.Contains(t, user.lastParams(t).Message, `"Harmless\nApproval: none needed"`,
		"a title must not be able to add a line of its own to the confirmation")
}

// askedMessage returns the confirmation message a decide_issue call on the
// mock shows.
func askedMessage(t *testing.T, mock *decisionMock) string {
	t.Helper()
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})
	_, err := callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	return user.lastParams(t).Message
}

func TestDecideIssueKeepsPlanTextOnItsOwnLine(t *testing.T) {
	mock := changeIssueMock()
	mock.plan = map[string]any{"specs": []map[string]any{{
		"id": "spec-1",
		"createDatabaseConfig": map[string]any{
			"target":   "instances/prod",
			"database": "scratch\nApproval: not required",
		},
	}}}
	message := askedMessage(t, mock)
	for _, line := range strings.Split(message, "\n") {
		require.False(t, strings.HasPrefix(line, "Approval: not required"), "plan text added a line of its own: %q", line)
	}
	require.Contains(t, message, `Create database "scratch\nApproval: not required" on "instances/prod"`)
}

func TestDecideIssueShowsTheStatementsLineBreaks(t *testing.T) {
	mock := changeIssueMock()
	mock.sheet = map[string]any{"content": base64.StdEncoding.EncodeToString([]byte("-- tidy up\nDROP TABLE users;"))}
	message := askedMessage(t, mock)
	require.NotContains(t, message, "-- tidy up DROP TABLE users;", "folding the line break turns the DROP into part of a comment")
	require.Contains(t, message, `"-- tidy up\nDROP TABLE users;"`)
}

func TestDecideIssueSaysWhatTheFormLeavesOut(t *testing.T) {
	mock := changeIssueMock()
	statement := "UPDATE users SET status = 'active';" + strings.Repeat(" ", 300) + "DROP TABLE users;"
	mock.sheet = map[string]any{
		"content":     base64.StdEncoding.EncodeToString([]byte(statement)),
		"contentSize": strconv.Itoa(len(statement)),
	}
	sheetSpec := map[string]any{"changeDatabaseConfig": map[string]any{
		"targets": []string{"instances/prod/databases/hr"},
		"sheet":   "projects/hr/sheets/9",
	}}
	mock.plan = map[string]any{"specs": []map[string]any{
		{"changeDatabaseConfig": map[string]any{
			"targets": []string{"instances/prod/databases/hr"},
			"release": "projects/hr/releases/r1",
		}},
		sheetSpec,
		{"createDatabaseConfig": map[string]any{"target": "instances/prod", "database": "scratch"}},
		sheetSpec,
		sheetSpec,
	}}
	lines := strings.Split(askedMessage(t, mock), "\n")
	require.Equal(t, `Change 1 of 5 on "instances/prod/databases/hr"`, lines[1],
		"Claude Code shows the first lines only, so the first change already says how many there are")
	require.Equal(t, `Release "projects/hr/releases/r1" (statements not shown)`, lines[2])
	require.Equal(t, `Change 2 of 5 on "instances/prod/databases/hr"`, lines[3])
	require.True(t, strings.HasPrefix(lines[4], `SQL (partial): "UPDATE users`), lines[4])
	require.True(t, strings.HasSuffix(lines[4], fmt.Sprintf(`" (first 300 characters of %d bytes)`, len(statement))), lines[4])
	require.NotContains(t, lines[4], "DROP TABLE users;")
	require.Equal(t, `Change 3 of 5: create database "scratch" on "instances/prod"`, lines[5])
	require.Equal(t, "(3 of 5 changes shown)", lines[6])
}

func TestDecideIssueFormShowsTheChangeInClaudeCodesDialog(t *testing.T) {
	longStatement := "ALTER TABLE users ADD COLUMN status VARCHAR(20);" + strings.Repeat(" ", 300) + "DROP TABLE users;"
	manyTargets := make([]string, 12)
	for i := range manyTargets {
		manyTargets[i] = fmt.Sprintf("instances/prod-%02d/databases/hr", i)
	}
	sheetSpec := func(targets ...string) map[string]any {
		return map[string]any{"changeDatabaseConfig": map[string]any{"targets": targets, "sheet": "projects/hr/sheets/9"}}
	}
	withStatement := func(mock *decisionMock, statement string) *decisionMock {
		mock.sheet = map[string]any{
			"content":     base64.StdEncoding.EncodeToString([]byte(statement)),
			"contentSize": strconv.Itoa(len(statement)),
		}
		return mock
	}
	for name, tc := range map[string]struct {
		mock *decisionMock
		// The start of each line after the question that must stay visible.
		scope, statement, last string
	}{
		"one long target": {
			mock: func() *decisionMock {
				mock := changeIssueMock()
				mock.plan = map[string]any{"specs": []map[string]any{sheetSpec("instances/" + strings.Repeat("i", 60) + "/databases/" + strings.Repeat("d", 60))}}
				return mock
			}(),
			scope:     `Change on "instances/iiii`,
			statement: `SQL: "ALTER TABLE users\n  ADD COLUMN status`,
			last:      `Approval: step 2 of 2`,
		},
		"many targets": {
			mock: func() *decisionMock {
				mock := changeIssueMock()
				mock.plan = map[string]any{"specs": []map[string]any{sheetSpec(manyTargets...)}}
				return mock
			}(),
			scope:     `Change on 12 targets: "instances/prod-00/databases/hr"`,
			statement: `SQL: "ALTER TABLE users\n  ADD COLUMN status`,
			last:      `Approval: step 2 of 2`,
		},
		"a long statement": {
			mock:      withStatement(changeIssueMock(), longStatement),
			scope:     `Change on "instances/prod/databases/hr"`,
			statement: `SQL (partial): "ALTER TABLE users ADD COLUMN status`,
			last:      `Approval: step 2 of 2`,
		},
		"several changes": {
			mock: func() *decisionMock {
				mock := withStatement(changeIssueMock(), longStatement)
				mock.plan = map[string]any{"specs": []map[string]any{sheetSpec(manyTargets...), sheetSpec(manyTargets...), sheetSpec(manyTargets...)}}
				return mock
			}(),
			scope:     `Change 1 of 3 on 12 targets: "instances/prod-00/databases/hr"`,
			statement: `SQL (partial): "ALTER TABLE users ADD COLUMN status`,
			last:      `… (+`,
		},
		"an access request": {
			mock: func() *decisionMock {
				mock := accessRequestMock()
				mock.grant["targets"] = manyTargets
				mock.grant["schema"] = "payroll"
				mock.grant["container"] = "employees"
				mock.grant["query"] = "SELECT salary FROM employee" + strings.Repeat(" ", 300) + "UNION SELECT ssn FROM employee"
				return mock
			}(),
			scope:     `Access on 12 databases, schema "payroll", container "employees": "`,
			statement: `Query (partial): "SELECT salary FROM employee`,
			last:      `Unmasked: yes. Export: no. Expires: "1h0m0s" after approval.`,
		},
		"an access request on one long database": {
			mock: func() *decisionMock {
				mock := accessRequestMock()
				mock.grant["targets"] = []string{"instances/pg-prod-eu-west-1/databases/payroll_reporting"}
				mock.grant["schema"] = "payroll"
				return mock
			}(),
			scope:     `Access on "instances/pg-prod-eu-west-1/databases/payroll_reporting"`,
			statement: `Query: "SELECT salary`,
			last:      `Unmasked: yes. Export: no. Expires: "1h0m0s" after approval.`,
		},
		"an access request with a long schema": {
			mock: func() *decisionMock {
				mock := accessRequestMock()
				mock.grant["schema"] = "regional_payroll_reporting_archive_for_financial_year_2026"
				return mock
			}(),
			scope:     `Access on "instances/prod/databases/hr", schema "regional_payroll`,
			statement: `Query: "SELECT salary`,
			last:      `Unmasked: yes. Export: no. Expires: "1h0m0s" after approval.`,
		},
		"an access request with a long container": {
			mock: func() *decisionMock {
				mock := accessRequestMock()
				mock.grant["container"] = "employee_compensation_history_partitioned_by_region_and_year"
				return mock
			}(),
			scope:     `Access on "instances/prod/databases/hr", container "employee_compensation`,
			statement: `Query: "SELECT salary`,
			last:      `Unmasked: yes. Export: no. Expires: "1h0m0s" after approval.`,
		},
		"wide characters": {
			mock: func() *decisionMock {
				mock := withStatement(changeIssueMock(), "UPDATE 用户 SET 状态 = '有效';")
				mock.issue["title"] = strings.Repeat("为用户表添加状态列", 6)
				return mock
			}(),
			scope:     `Change on "instances/prod/databases/hr"`,
			statement: `SQL: "UPDATE 用户 SET 状态 = '有效';"`,
			last:      `Approval: step 2 of 2`,
		},
	} {
		message := askedMessage(t, tc.mock)
		for _, columns := range []int{80, 120} {
			t.Run(fmt.Sprintf("%s at %d columns", name, columns), func(t *testing.T) {
				view := claudeCodeView(message, columns)
				require.Len(t, view, 4, "%q", view)
				require.True(t, strings.HasPrefix(view[0], "Approve issue "), "the decision: %q", view)
				require.True(t, strings.HasPrefix(view[1], tc.scope), "the scope: %q", view)
				require.True(t, strings.HasPrefix(view[2], tc.statement), "the statement: %q", view)
				require.True(t, strings.HasPrefix(view[3], tc.last), "the approval, or how much is left out: %q", view)
			})
		}
	}
}

// claudeCodeView is what Claude Code 2.1.302 shows of a form's message in a
// terminal of the given width (BP in its bundle): four lines, or three and
// "… (+N more lines)", each cut six columns short of the width and ended with
// "…". A wide character takes two columns.
func claudeCodeView(message string, columns int) []string {
	lines := strings.Split(message, "\n")
	if len(lines) > 4 {
		lines = append(lines[:3:3], fmt.Sprintf("… (+%d more lines)", len(lines)-3))
	}
	limit := max(20, columns-6)
	view := make([]string, len(lines))
	for i, line := range lines {
		view[i] = cutToColumns(line, limit)
	}
	return view
}

func cutToColumns(line string, limit int) string {
	total := 0
	for _, r := range line {
		total += runeColumns(r)
	}
	if total <= limit {
		return line
	}
	used := 0
	for i, r := range line {
		if used+runeColumns(r) > limit-1 {
			return line[:i] + "…"
		}
		used += runeColumns(r)
	}
	return line
}

func runeColumns(r rune) int {
	if unicode.Is(unicode.Mn, r) {
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

func TestDecideIssueAsksNothingWhenAReadFails(t *testing.T) {
	for name, tc := range map[string]struct {
		mock *decisionMock
		path string
	}{
		"the plan":         {changeIssueMock(), "/bytebase.v1.PlanService/GetPlan"},
		"the statement":    {changeIssueMock(), "/bytebase.v1.SheetService/GetSheet"},
		"the access grant": {accessRequestMock(), "/bytebase.v1.AccessGrantService/GetAccessGrant"},
	} {
		t.Run(name, func(t *testing.T) {
			tc.mock.failing = tc.path
			endpoint, token := newDecideServer(t, tc.mock)
			user := &asker{answer: confirmAnswer(true)}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			res, err := callDecideIssue(t, session, "approve")
			require.NoError(t, err)
			require.True(t, res.IsError)
			require.Zero(t, user.asked.Load(), "a form missing part of the issue must not be sent")
			require.Empty(t, tc.mock.calls())
		})
	}
}

func TestDecideIssueRefusesInvalidArguments(t *testing.T) {
	for name, input := range map[string]DecideIssueInput{
		"an issue that is not a name": {Issue: "42", Decision: "approve"},
		"an unknown decision":         {Issue: decideIssueName, Decision: "skip"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := changeIssueMock()
			endpoint, token := newDecideServer(t, mock)
			user := &asker{answer: confirmAnswer(true)}
			session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			res, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "decide_issue",
				Arguments: map[string]any{"issue": input.Issue, "decision": input.Decision},
			})
			require.NoError(t, err)
			require.True(t, res.IsError)
			require.Contains(t, resultText(t, res), "INVALID_ARGUMENT")
			require.Zero(t, user.asked.Load())
			require.Empty(t, mock.calls())
		})
	}
}

func TestCallAPICarriesNoConfirmation(t *testing.T) {
	mock := changeIssueMock()
	endpoint, token := newDecideServer(t, mock)
	session := connectLegacy(t, endpoint, token, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "call_api",
		Arguments: map[string]any{
			"operationId": "IssueService/ApproveIssue",
			"body":        map[string]any{"name": decideIssueName},
		},
	})
	require.NoError(t, err)
	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Nil(t, calls[0].confirmed)
}

// confirmationProbe is an IssueService that answers ApproveIssue.
type confirmationProbe struct {
	v1connect.UnimplementedIssueServiceHandler
}

func (*confirmationProbe) ApproveIssue(context.Context, *connect.Request[v1pb.ApproveIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	return connect.NewResponse(&v1pb.Issue{Name: decideIssueName}), nil
}

// TestConfirmationReachesTheConnectInterceptors pins the path the gate reads
// the confirmation from: the in-memory transport hands the tool's context to
// the connect handler, and connect hands it to every interceptor, the MCP gate
// among them.
func TestConfirmationReachesTheConnectInterceptors(t *testing.T) {
	var seen atomic.Pointer[auth.MCPConfirmedDecision]
	recorder := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if confirmed, ok := auth.MCPConfirmedDecisionFromContext(ctx); ok {
				seen.Store(&confirmed)
			}
			return next(ctx, req)
		}
	})
	mux := http.NewServeMux()
	mux.Handle(v1connect.NewIssueServiceHandler(&confirmationProbe{}, connect.WithInterceptors(recorder)))
	s := newTestServerWithMock(t, mux)

	want := auth.MCPConfirmedDecision{
		Procedure:       v1connect.IssueServiceApproveIssueProcedure,
		Issue:           decideIssueName,
		IssueUpdateTime: time.Date(2026, 10, 13, 8, 0, 0, 123456000, time.UTC),
	}
	resp, err := s.apiRequest(auth.WithMCPConfirmedDecision(context.Background(), want),
		v1connect.IssueServiceApproveIssueProcedure, map[string]any{"name": decideIssueName})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))
	require.Equal(t, &want, seen.Load())
}
