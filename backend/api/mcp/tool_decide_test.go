package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const decideIssueName = "projects/hr/issues/42"

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
	switch r.URL.Path {
	case "/bytebase.v1.IssueService/GetIssue":
		reply(issue)
	case "/bytebase.v1.PlanService/GetPlan":
		reply(plan)
	case "/bytebase.v1.SheetService/GetSheet":
		reply(sheet)
	case "/bytebase.v1.AccessGrantService/GetAccessGrant":
		reply(grant)
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
			"approvers": []map[string]any{{"status": "APPROVED", "principal": "users/carol@example.com"}},
		},
		plan: map[string]any{"specs": []map[string]any{{
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
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
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
	for _, line := range []string{
		`Approve issue "Add status column" as alice@example.com?`,
		"Issue: projects/hr/issues/42 (DATABASE_CHANGE, requested by bob@example.com)",
		"Change on instances/prod/databases/hr: ALTER TABLE users ADD COLUMN status VARCHAR(20);",
		"Approval: step 2 of 2, role roles/workspaceDBA",
		"Details: https://bb.example.com/projects/hr/issues/42",
	} {
		require.Contains(t, asked.Message, line)
	}
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
		Procedure: v1connect.IssueServiceApproveIssueProcedure,
		Issue:     decideIssueName,
	}, calls[0].confirmed, "the approval must carry the confirmation the gate admits it on")

	require.Contains(t, resultText(t, res), "Approved projects/hr/issues/42. Approval: APPROVED. Issue status: DONE.")
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
	require.Contains(t, user.lastParams(t).Message, `Reject issue "Add status column"`)

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, v1connect.IssueServiceRejectIssueProcedure, calls[0].path)
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure: v1connect.IssueServiceRejectIssueProcedure,
		Issue:     decideIssueName,
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
	approve := &issueDecision{issue: decideIssueName, approve: true, procedure: v1connect.IssueServiceApproveIssueProcedure}
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
		"another issue":      {alice, &issueDecision{issue: "projects/hr/issues/43", approve: true, procedure: v1connect.IssueServiceApproveIssueProcedure}, state},
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
	require.Contains(t, asked.Message, `Approve issue "Add status column"`)

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
func callDecideIssueWith(t *testing.T, issueType string, unasked mcp.InputResponseMap) (*decisionMock, *asker, *mcp.CallToolResult) {
	t.Helper()
	mock := changeIssueMock()
	mock.issue["type"] = issueType
	endpoint, token := newDecideServer(t, mock)
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
	return mock, user, res
}

func TestDecideIssueRefusesARoleRequest(t *testing.T) {
	for name, unasked := range map[string]mcp.InputResponseMap{
		"asked as usual":              nil,
		"with an answer sent unasked": {confirmDecisionRequestID: confirmAnswer(true)},
	} {
		t.Run(name, func(t *testing.T) {
			mock, user, res := callDecideIssueWith(t, "ROLE_GRANT", unasked)
			require.True(t, res.IsError)
			text := resultText(t, res)
			require.Contains(t, text, "ROLE_GRANT_NOT_SUPPORTED")
			require.Contains(t, text, "approving a role request grants a project role, which MCP does not do yet")
			require.Contains(t, text, "decide it signed in to the Bytebase console: https://bb.example.com/projects/hr/issues/42")
			require.Zero(t, user.asked.Load(), "no form may be sent for a role request")
			require.Empty(t, mock.calls(), "no decision, and so no confirmation, may reach the API")
		})
	}
}

func TestDecideIssueRefusesAnIssueTypeItDoesNotKnow(t *testing.T) {
	mock, user, res := callDecideIssueWith(t, "SOMETHING_NEW", nil)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "ISSUE_TYPE_NOT_SUPPORTED")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
}

func TestDecideIssueReportsTheWorkflowRefusal(t *testing.T) {
	mock := changeIssueMock()
	mock.refusal = "cannot approve because self-approval is not allowed for this project"
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: confirmAnswer(true)}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res, err := callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	require.True(t, res.IsError)
	text := resultText(t, res)
	require.Contains(t, text, "DECISION_REFUSED")
	require.Contains(t, text, "nothing changed: cannot approve because self-approval is not allowed for this project")
	require.Len(t, mock.calls(), 1, "the workflow, not the tool, refuses")
}

func TestDecideIssueDescribesAnAccessRequest(t *testing.T) {
	mock := &decisionMock{
		issue: map[string]any{
			"name":             decideIssueName,
			"title":            "Read salaries",
			"type":             "ACCESS_GRANT",
			"creator":          "users/bob@example.com",
			"accessGrant":      "projects/hr/accessGrants/5",
			"approvalStatus":   "PENDING",
			"approvalTemplate": map[string]any{"flow": map[string]any{"roles": []string{"roles/projectOwner"}}},
		},
		grant: map[string]any{
			"creator": "users/bob@example.com",
			"targets": []string{"instances/prod/databases/hr"},
			"query":   "SELECT salary\nFROM employee",
			"unmask":  true,
			"ttl":     "3600s",
		},
	}
	endpoint, token := newDecideServer(t, mock)
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{ElicitationHandler: user.handle})

	_, err := callDecideIssue(t, session, "approve")
	require.NoError(t, err)
	message := user.lastParams(t).Message
	for _, line := range []string{
		"Issue: projects/hr/issues/42 (ACCESS_GRANT, requested by bob@example.com)",
		"Access: run SELECT salary FROM employee on instances/prod/databases/hr",
		"Unmasked: yes. Export: no. Requested by bob@example.com",
		"Expires: 1h0m0s after approval",
		"Approval: step 1 of 1, role roles/projectOwner",
	} {
		require.Contains(t, message, line)
	}
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

// TestCallAPICarriesNoConfirmation is the other half of the provenance rule:
// call_api reaches the decision procedures with no confirmation on the
// context, which is what the gate refuses.
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

// confirmationProbe is an IssueService whose only method records the
// confirmation its interceptor saw.
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

	want := auth.MCPConfirmedDecision{Procedure: v1connect.IssueServiceApproveIssueProcedure, Issue: decideIssueName}
	resp, err := s.apiRequest(auth.WithMCPConfirmedDecision(context.Background(), want),
		v1connect.IssueServiceApproveIssueProcedure, map[string]any{"name": decideIssueName})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))
	require.Equal(t, &want, seen.Load())
}
