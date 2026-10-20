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

// projectMock is the internal API behind set_just_in_time_access: the project
// read, and the update, recorded with the confirmation its context carried.
type projectMock struct {
	mu      sync.Mutex
	enabled bool
	// refusal, when set, is the message the update is refused with.
	refusal string
	updates []decisionCall
}

func (m *projectMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reply := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	switch r.URL.Path {
	case "/bytebase.v1.ProjectService/GetProject":
		m.mu.Lock()
		enabled := m.enabled
		m.mu.Unlock()
		reply(map[string]any{"name": "projects/hr", "allowJustInTimeAccess": enabled})
	case v1connect.ProjectServiceUpdateProjectProcedure:
		call := decisionCall{path: r.URL.Path}
		_ = json.NewDecoder(r.Body).Decode(&call.body)
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(r.Context()); ok {
			call.confirmed = &confirmed
		}
		m.mu.Lock()
		m.updates = append(m.updates, call)
		refusal := m.refusal
		m.mu.Unlock()
		if refusal != "" {
			w.WriteHeader(http.StatusForbidden)
			reply(map[string]any{"code": "permission_denied", "message": refusal})
			return
		}
		reply(map[string]any{"name": "projects/hr"})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *projectMock) calls() []decisionCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]decisionCall(nil), m.updates...)
}

func (m *projectMock) set(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

func newProjectServer(t *testing.T, mock *projectMock) *Server {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	return s
}

func connectToProjectServer(t *testing.T, mock *projectMock, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	s := newProjectServer(t, mock)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	return connectLegacy(t, ts.URL+mcpResourcePath, token, opts)
}

func callSetJustInTimeAccess(t *testing.T, session *mcp.ClientSession, enabled bool) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "set_just_in_time_access",
		Arguments: map[string]any{"project": "projects/hr", "enabled": enabled},
	})
	require.NoError(t, err)
	return res
}

func TestSetJustInTimeAccessTurnsItOnOnConfirmation(t *testing.T) {
	mock := &projectMock{}
	user := &asker{answer: confirmAnswer(true)}
	session := connectToProjectServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callSetJustInTimeAccess(t, session, true)
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), user.asked.Load())
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Turn on just-in-time access in project "projects/hr" as alice@example.com?`,
		`Active grants that have not expired apply again at once.`,
	}, "\n"), asked.Message)
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"object","required":["confirm"],"properties":{"confirm":{"type":"boolean",
		"title":"Turn on just-in-time access",
		"description":"Check to turn on just-in-time access in the project above. Leaving it unchecked, declining or cancelling changes nothing."}}}`,
		string(schema))

	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, map[string]any{
		"project":    map[string]any{"name": "projects/hr", "allowJustInTimeAccess": true},
		"updateMask": "allowJustInTimeAccess",
	}, calls[0].body, "the update must change that one setting")
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure: v1connect.ProjectServiceUpdateProjectProcedure,
		Resource:  "projects/hr",
	}, calls[0].confirmed, "the update must carry the confirmation the gate admits it on")
	require.Contains(t, resultText(t, res), "Turned on just-in-time access in projects/hr.\nProject settings: https://bb.example.com/projects/hr/settings")
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"project":"projects/hr","enabled":true,"changed":true,"link":"https://bb.example.com/projects/hr/settings"}`, string(output))
}

func TestSetJustInTimeAccessSaysWhatTurningItOffDoes(t *testing.T) {
	mock := &projectMock{enabled: true}
	user := &asker{answer: confirmAnswer(true)}
	session := connectToProjectServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callSetJustInTimeAccess(t, session, false)
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, strings.Join([]string{
		`Turn off just-in-time access in project "projects/hr" as alice@example.com?`,
		`Active grants here stop applying (not revoked); new requests are refused.`,
	}, "\n"), user.lastParams(t).Message)
	calls := mock.calls()
	require.Len(t, calls, 1)
	require.Equal(t, false, calls[0].body["project"].(map[string]any)["allowJustInTimeAccess"])
}

func TestSetJustInTimeAccessAsksNothingWhenAlreadySet(t *testing.T) {
	mock := &projectMock{enabled: true}
	user := &asker{answer: confirmAnswer(true)}
	session := connectToProjectServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callSetJustInTimeAccess(t, session, true)
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "Just-in-time access is already on in projects/hr, so nothing changed.")
	require.Zero(t, user.asked.Load())
	require.Empty(t, mock.calls())
}

func TestSetJustInTimeAccessReportsASettingChangedWhileAsking(t *testing.T) {
	mock := &projectMock{}
	user := &asker{answer: confirmAnswer(true), whileAsked: func() { mock.set(true) }}
	session := connectToProjectServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callSetJustInTimeAccess(t, session, true)
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "already on in projects/hr, so nothing changed")
	require.Empty(t, mock.calls())
}

func TestSetJustInTimeAccessChangesNothingUnlessConfirmed(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":           {&mcp.ElicitResult{Action: "decline"}, "the user declined"},
		"cancelled":          {&mcp.ElicitResult{Action: "cancel"}, "the user cancelled"},
		"accepted, unticked": {confirmAnswer(false), "the form was submitted without the confirmation checked"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &projectMock{}
			user := &asker{answer: tc.answer}
			session := connectToProjectServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

			res := callSetJustInTimeAccess(t, session, true)
			require.False(t, res.IsError, resultText(t, res))
			require.Contains(t, resultText(t, res), "was not turned on: "+tc.want+", so nothing changed")
			require.Empty(t, mock.calls())
		})
	}
}

func TestSetJustInTimeAccessRefusesAClientThatCannotConfirm(t *testing.T) {
	mock := &projectMock{}
	session := connectToProjectServer(t, mock, nil)

	res := callSetJustInTimeAccess(t, session, true)
	require.True(t, res.IsError)
	text := resultText(t, res)
	require.Contains(t, text, "CONFIRMATION_UNAVAILABLE")
	require.Contains(t, text, "change it signed in to the Bytebase console: https://bb.example.com/projects/hr/settings")
	require.Empty(t, mock.calls())
}

func TestSetJustInTimeAccessReportsTheRefusalInTheServersWords(t *testing.T) {
	mock := &projectMock{refusal: "permission denied: bb.projects.update"}
	session := connectToProjectServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(true)}).handle})

	res := callSetJustInTimeAccess(t, session, true)
	require.True(t, res.IsError)
	text := resultText(t, res)
	require.Contains(t, text, "failed to turn on just-in-time access in projects/hr: HTTP 403: permission denied: bb.projects.update")
	require.NotContains(t, text, "Turned on")
	require.Nil(t, res.StructuredContent)
	require.Len(t, mock.calls(), 1)
}

func TestSetJustInTimeAccessTellsAClientThatCannotConfirmWhenNothingIsNeeded(t *testing.T) {
	mock := &projectMock{enabled: true}
	session := connectToProjectServer(t, mock, nil)

	res := callSetJustInTimeAccess(t, session, true)
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "already on in projects/hr, so nothing changed")
	require.Empty(t, mock.calls())
}

func TestSetJustInTimeAccessRefusesAnUnaskedAnswerFromAClientThatCannotConfirm(t *testing.T) {
	mock := &projectMock{}
	session := connectToProjectServer(t, mock, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:           "set_just_in_time_access",
		Arguments:      map[string]any{"project": "projects/hr", "enabled": true},
		InputResponses: mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(true)},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "CONFIRMATION_UNAVAILABLE")
	require.Empty(t, mock.calls())
}

func TestSetJustInTimeAccessBindsTheAnswerToTheArguments(t *testing.T) {
	mock := &projectMock{}
	s := newProjectServer(t, mock)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	session := connectMRTR(t, mrtrEndpoint(t, s), token)
	call := func(project string, enabled bool, responses mcp.InputResponseMap, state string) *mcp.CallToolResult {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:           "set_just_in_time_access",
			Arguments:      map[string]any{"project": project, "enabled": enabled},
			InputResponses: responses,
			RequestState:   state,
		})
		require.NoError(t, err)
		return res
	}
	accept := mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(true)}

	first := call("projects/hr", true, nil, "")
	require.True(t, first.NeedsInput())

	mock.set(true)
	flipped := call("projects/hr", false, accept, first.RequestState)
	require.True(t, flipped.IsError, resultText(t, flipped))
	require.Contains(t, resultText(t, flipped), "STALE_CONFIRMATION", "an answer resent with the other value is not what the user confirmed")

	mock.set(false)
	elsewhere := call("projects/finance", true, accept, first.RequestState)
	require.True(t, elsewhere.NeedsInput(), "an answer for another project is asked again")
	require.Empty(t, mock.calls())
}

func TestSetJustInTimeAccessRefusesAnInvalidProject(t *testing.T) {
	mock := &projectMock{}
	session := connectToProjectServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(true)}).handle})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for _, project := range []string{"hr", "projects/hr/issues/1", ""} {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "set_just_in_time_access",
			Arguments: map[string]any{"project": project, "enabled": true},
		})
		require.NoError(t, err)
		require.True(t, res.IsError, project)
		require.Contains(t, resultText(t, res), "INVALID_ARGUMENT", project)
	}
	require.Empty(t, mock.calls())
}

func TestSetJustInTimeAccessFormShowsTheChangeInClaudeCodesDialog(t *testing.T) {
	longProject := "projects/" + strings.Repeat("p", 63)
	for _, enabled := range []bool{true, false} {
		change := &justInTimeChange{project: longProject, enabled: enabled}
		message := describeJustInTimeChange(withUserEmail(context.Background(), "alexander.smith@example.com"), change)
		for _, columns := range []int{80, 120} {
			t.Run(fmt.Sprintf("enabled %v at %d columns", enabled, columns), func(t *testing.T) {
				view := claudeCodeView(message, columns)
				require.Len(t, view, 2, "%q", view)
				require.True(t, strings.HasPrefix(view[0], fmt.Sprintf("Turn %s just-in-time access in project ", change.state())), "%q", view)
				if columns == 120 {
					require.Contains(t, view[0], `"`+longProject, "at 120 columns a 63-character project ID shows whole; the email is cut: %q", view)
				}
				require.True(t, strings.HasPrefix(view[1], "Active grants"), "the effect: %q", view)
				require.NotContains(t, view[1], "…", "the effect line must fit whole: %q", view)
			})
		}
	}
}
