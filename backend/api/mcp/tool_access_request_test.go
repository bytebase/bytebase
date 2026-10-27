package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/review"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// accessGrantMock is the internal API behind request_access_grant: the
// database read, and the request, recorded as sent.
type accessGrantMock struct {
	mu sync.Mutex
	// project is the project GetDatabase reports.
	project string
	// readStatus and readRefusal, when set, are how GetDatabase is refused.
	readStatus  int
	readRefusal map[string]any
	// refusalStatus and refusal, when set, are how the request is refused.
	refusalStatus int
	refusal       map[string]any
	// unapprovable and unapprovableErr are what the approval-flow check
	// reports for the filed request.
	unapprovable    *review.UnapprovableStep
	unapprovableErr error
	reads           []string
	requests        []map[string]any
	checks          []string
}

func (m *accessGrantMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reply := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.URL.Path {
	case v1connect.DatabaseServiceGetDatabaseProcedure:
		var body struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.reads = append(m.reads, body.Name)
		if m.readRefusal != nil {
			w.WriteHeader(m.readStatus)
			reply(m.readRefusal)
			return
		}
		reply(map[string]any{"name": body.Name, "project": m.project})
	case v1connect.AccessGrantServiceCreateAccessGrantProcedure:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.requests = append(m.requests, body)
		if m.refusal != nil {
			w.WriteHeader(m.refusalStatus)
			reply(m.refusal)
			return
		}
		reply(map[string]any{"name": "projects/hr/accessGrants/g1", "status": "PENDING", "issue": "projects/hr/issues/7"})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *accessGrantMock) findUnapprovableStep(_ context.Context, workspace, issue string) (*review.UnapprovableStep, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks = append(m.checks, workspace+" "+issue)
	return m.unapprovable, m.unapprovableErr
}

func (m *accessGrantMock) checked() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.checks...)
}

func (m *accessGrantMock) calls() (reads []string, requests []map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.reads...), append([]map[string]any(nil), m.requests...)
}

func connectToAccessGrantServer(t *testing.T, mock *accessGrantMock) *mcp.ClientSession {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	s.findUnapprovableStep = mock.findUnapprovableStep
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	return connectLegacy(t, ts.URL+mcpResourcePath, token, nil)
}

func callRequestAccessGrant(t *testing.T, session *mcp.ClientSession, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "request_access_grant", Arguments: arguments})
	require.NoError(t, err)
	return res
}

func TestRequestAccessGrantFilesTheRequest(t *testing.T) {
	mock := &accessGrantMock{project: "projects/hr"}
	session := connectToAccessGrantServer(t, mock)

	res := callRequestAccessGrant(t, session, map[string]any{
		"database":      "projects/hr/instances/prod/databases/hr",
		"statement":     "SELECT first_name, last_name FROM employee",
		"schema":        "public",
		"container":     "employees",
		"unmask":        true,
		"export":        true,
		"reason":        "Compensation review",
		"durationHours": 4,
	})
	require.False(t, res.IsError, resultText(t, res))

	reads, requests := mock.calls()
	require.Equal(t, []string{"projects/hr/instances/prod/databases/hr"}, reads)
	require.Len(t, requests, 1)
	require.Equal(t, map[string]any{
		"parent": "projects/hr",
		"accessGrant": map[string]any{
			"targets":   []any{"projects/hr/instances/prod/databases/hr"},
			"query":     "SELECT first_name, last_name FROM employee",
			"schema":    "public",
			"container": "employees",
			"unmask":    true,
			"export":    true,
			"reason":    "Compensation review",
			"ttl":       "14400s",
		},
	}, requests[0], "the request names the database as given, sends no creator, and asks for a duration that starts at approval")

	require.Equal(t, "Requested projects/hr/accessGrants/g1: run the statement on projects/hr/instances/prod/databases/hr, unmasked, with export for 4 hours from approval. "+
		"A person approves it in Bytebase before it takes effect.\n"+
		"Issue: https://bb.example.com/projects/hr/issues/7\n"+
		`Once it is approved, pass the statement unchanged, with the same schema and container, to call_api(operationId="SQLService/Export") to export the results, or to query_database to read up to 1000 rows.`, resultText(t, res))
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"accessGrant":"projects/hr/accessGrants/g1","issue":"projects/hr/issues/7",
		"durationHours":4,"link":"https://bb.example.com/projects/hr/issues/7"}`, string(output))
	require.Equal(t, []string{"ws-test projects/hr/issues/7"}, mock.checked(), "the filed issue's approval flow is checked in the session's workspace")
}

func TestRequestAccessGrantSaysWhenNoOneCanApprove(t *testing.T) {
	for name, tt := range map[string]struct {
		step   *review.UnapprovableStep
		unmask bool
		want   string
	}{
		"only the requester holds the role": {
			step:   &review.UnapprovableStep{Position: 1, Role: "roles/projectOwner", CreatorHoldsRole: true},
			unmask: true,
			want: "Requested projects/hr/accessGrants/g1: run the statement on projects/hr/instances/prod/databases/hr, unmasked for 4 hours from approval. " +
				"No other person can approve it: step 1 of its approval flow needs roles/projectOwner, which no other active workspace member holds, and this project does not allow self-approval.\n" +
				"Issue: https://bb.example.com/projects/hr/issues/7\n" +
				"An approver can be added only in the Bytebase console, by granting someone that role. " +
				"If the statement already runs for you with the values masked, a project owner can grant a masking exemption with grant_masking_exemption to read them without this request.",
		},
		"no person holds the role": {
			step: &review.UnapprovableStep{Position: 2, Role: "roles/workspaceDBA"},
			want: "Requested projects/hr/accessGrants/g1: run the statement on projects/hr/instances/prod/databases/hr for 4 hours from approval. " +
				"No person can approve it: step 2 of its approval flow needs roles/workspaceDBA, which no active workspace member holds.\n" +
				"Issue: https://bb.example.com/projects/hr/issues/7\n" +
				"An approver can be added only in the Bytebase console, by granting someone that role.",
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &accessGrantMock{project: "projects/hr", unapprovable: tt.step}
			session := connectToAccessGrantServer(t, mock)

			res := callRequestAccessGrant(t, session, map[string]any{
				"database":      "projects/hr/instances/prod/databases/hr",
				"statement":     "SELECT 1",
				"unmask":        tt.unmask,
				"reason":        "a look",
				"durationHours": 4,
			})
			require.False(t, res.IsError, resultText(t, res))
			require.Equal(t, tt.want, resultText(t, res))
		})
	}
}

func TestRequestAccessGrantKeepsItsResultWhenTheApprovalCheckFails(t *testing.T) {
	mock := &accessGrantMock{project: "projects/hr", unapprovableErr: errors.New("connection refused")}
	session := connectToAccessGrantServer(t, mock)

	res := callRequestAccessGrant(t, session, map[string]any{
		"database":      "projects/hr/instances/prod/databases/hr",
		"statement":     "SELECT 1",
		"reason":        "a look",
		"durationHours": 4,
	})
	require.False(t, res.IsError, "the request is filed, so a failed check must not read as a failed request: %s", resultText(t, res))
	require.Contains(t, resultText(t, res), "A person approves it in Bytebase before it takes effect.")
	_, requests := mock.calls()
	require.Len(t, requests, 1)
}

func TestRequestAccessGrantFilesInTheDatabaseProject(t *testing.T) {
	mock := &accessGrantMock{project: "projects/finance"}
	session := connectToAccessGrantServer(t, mock)

	res := callRequestAccessGrant(t, session, map[string]any{
		"database":      "instances/prod/databases/ledger",
		"statement":     "SELECT 1",
		"unmask":        true,
		"reason":        "Quarter close",
		"durationHours": 1,
	})
	require.False(t, res.IsError, resultText(t, res))

	_, requests := mock.calls()
	require.Len(t, requests, 1)
	require.Equal(t, "projects/finance", requests[0]["parent"], "a workspace instance's database name carries no project")
	require.Equal(t, "3600s", requests[0]["accessGrant"].(map[string]any)["ttl"])
	require.Contains(t, resultText(t, res), "run the statement on instances/prod/databases/ledger, unmasked for 1 hour from approval")
	require.Contains(t, resultText(t, res), "Once it is approved, rerun the statement unchanged with query_database, with the same schema and container.")
}

func TestRequestAccessGrantRefusesADurationBeforeAnyCall(t *testing.T) {
	for name, tt := range map[string]struct {
		durationHours int64
		want          string
	}{
		"no duration":         {0, "invalid durationHours 0"},
		"a negative duration": {-2, "invalid durationHours -2"},
		"a duration past the input schema's maximum": {math.MaxInt32 + 1, "validating /properties/durationHours: maximum"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := &accessGrantMock{project: "projects/hr"}
			session := connectToAccessGrantServer(t, mock)

			res := callRequestAccessGrant(t, session, map[string]any{
				"database":      "projects/hr/instances/prod/databases/hr",
				"statement":     "SELECT 1",
				"reason":        "a look",
				"durationHours": tt.durationHours,
			})
			require.True(t, res.IsError)
			require.Contains(t, resultText(t, res), tt.want)
			reads, requests := mock.calls()
			require.Empty(t, reads)
			require.Empty(t, requests)
		})
	}
}

func TestRequestAccessGrantFilesNothingWhenTheDatabaseReadFails(t *testing.T) {
	// GetDatabase's own answer to a name that is not a full database name.
	malformed := `failed to parse "hr_test": invalid request "hr_test"`
	mock := &accessGrantMock{
		readStatus:  http.StatusBadRequest,
		readRefusal: map[string]any{"code": "invalid_argument", "message": malformed},
	}
	session := connectToAccessGrantServer(t, mock)

	res := callRequestAccessGrant(t, session, map[string]any{
		"database":      "hr_test",
		"statement":     "SELECT 1",
		"reason":        "a look",
		"durationHours": 4,
	})
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), malformed)
	reads, requests := mock.calls()
	require.Equal(t, []string{"hr_test"}, reads)
	require.Empty(t, requests)
}

func TestRequestAccessGrantPassesPermissionDenialsThrough(t *testing.T) {
	for name, message := range map[string]string{
		"a request no person would approve": "/bytebase.v1.AccessGrantService/CreateAccessGrant is not available to MCP sessions for this request because no approval rule requires a person to approve it, " +
			"so the access would be granted at once with no human step. Ask a workspace admin to give access requests " +
			"an approval rule that names an approver, or request the access in the Bytebase console.",
		"a missing permission": `permission denied for method "/bytebase.v1.AccessGrantService/CreateAccessGrant", user does not have permission "bb.accessGrants.create", extra map[]`,
		"a plan feature":       "feature FEATURE_JIT is a ENTERPRISE feature, please upgrade to access it",
	} {
		t.Run(name, func(t *testing.T) {
			mock := &accessGrantMock{
				project:       "projects/hr",
				refusalStatus: http.StatusForbidden,
				refusal:       map[string]any{"code": "permission_denied", "message": message},
			}
			session := connectToAccessGrantServer(t, mock)

			res := callRequestAccessGrant(t, session, map[string]any{
				"database":      "projects/hr/instances/prod/databases/hr",
				"statement":     "SELECT 1",
				"reason":        "a look",
				"durationHours": 4,
			})
			require.True(t, res.IsError)
			var denial toolError
			require.NoError(t, json.Unmarshal([]byte(resultText(t, res)), &denial))
			require.Equal(t, toolError{Code: "PERMISSION_DENIED", Message: message}, denial, "the server's message is the whole answer")
		})
	}
}

func TestRequestAccessGrantReportsTheAPIRefusal(t *testing.T) {
	jitOff := `just-in-time access is off for project "hr", so an access grant there would not apply; ` +
		`a project owner can turn on "Allow access grants" in the project settings`
	mock := &accessGrantMock{
		project:       "projects/hr",
		refusalStatus: http.StatusBadRequest,
		refusal:       map[string]any{"code": "failed_precondition", "message": jitOff},
	}
	session := connectToAccessGrantServer(t, mock)

	res := callRequestAccessGrant(t, session, map[string]any{
		"database":      "projects/hr/instances/prod/databases/hr",
		"statement":     "SELECT 1",
		"reason":        "a look",
		"durationHours": 4,
	})
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), jitOff)
}

func TestOnlyTheGrantingToolsRequireUserInteraction(t *testing.T) {
	session := connectToAccessGrantServer(t, &accessGrantMock{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)

	var names, flagged []string
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
		if required, ok := tool.Meta["anthropic/requiresUserInteraction"].(bool); ok && required {
			flagged = append(flagged, tool.Name)
		}
	}
	slices.Sort(names)
	require.Equal(t, []string{
		"call_api", "decide_issue", "get_schema", "get_skill", "grant_masking_exemption", "propose_database_change",
		"query_database", "reauthorize", "request_access_grant", "revoke_masking_exemption", "search_api", "set_just_in_time_access",
	}, names)
	require.ElementsMatch(t, []string{"request_access_grant", "grant_masking_exemption"}, flagged)
}
