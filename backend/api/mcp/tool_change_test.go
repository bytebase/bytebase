package mcp

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// --- Change tool test infrastructure ---

// changeMock provides configurable mock handlers for all APIs used by propose_database_change.
type changeMock struct {
	databases []map[string]any

	// Per-step response configuration.
	sheetResponse        map[string]any
	sheetStatus          int
	planResponse         map[string]any
	planStatus           int
	runPlanChecksStatus  int
	planCheckRunResponse map[string]any
	planCheckRunStatus   int
	issueResponse        map[string]any
	issueStatus          int
	rolloutResponse      map[string]any
	rolloutStatus        int

	// Capture request bodies for assertions.
	mu              sync.Mutex
	capturedSheet   map[string]any
	capturedPlan    map[string]any
	capturedIssue   map[string]any
	capturedRollout map[string]any
}

func newChangeMock(databases []map[string]any) *changeMock {
	return &changeMock{
		databases:            databases,
		sheetResponse:        map[string]any{"name": "projects/hr-system/sheets/1001"},
		sheetStatus:          http.StatusOK,
		planResponse:         map[string]any{"name": "projects/hr-system/plans/2002"},
		planStatus:           http.StatusOK,
		runPlanChecksStatus:  http.StatusOK,
		planCheckRunResponse: defaultPlanCheckDone(),
		planCheckRunStatus:   http.StatusOK,
		issueResponse:        defaultIssueResponse("APPROVED"),
		issueStatus:          http.StatusOK,
		rolloutResponse:      map[string]any{"name": "projects/hr-system/rollouts/4004"},
		rolloutStatus:        http.StatusOK,
	}
}

func defaultPlanCheckDone() map[string]any {
	return map[string]any{
		"status":  "DONE",
		"results": []any{},
	}
}

func defaultIssueResponse(approvalStatus string) map[string]any {
	return map[string]any{
		"name":           "projects/hr-system/issues/3003",
		"approvalStatus": approvalStatus,
	}
}

func (m *changeMock) handler() http.Handler {
	listHandler := mockListDatabases(m.databases)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "SheetService/CreateSheet"):
			m.captureBody(r, &m.capturedSheet)
			w.WriteHeader(m.sheetStatus)
			_ = json.NewEncoder(w).Encode(m.sheetResponse)

		case strings.Contains(r.URL.Path, "PlanService/RunPlanChecks"):
			w.WriteHeader(m.runPlanChecksStatus)
			_ = json.NewEncoder(w).Encode(map[string]any{})

		case strings.Contains(r.URL.Path, "PlanService/GetPlanCheckRun"):
			w.WriteHeader(m.planCheckRunStatus)
			_ = json.NewEncoder(w).Encode(m.planCheckRunResponse)

		case strings.Contains(r.URL.Path, "PlanService/CreatePlan"):
			m.captureBody(r, &m.capturedPlan)
			w.WriteHeader(m.planStatus)
			_ = json.NewEncoder(w).Encode(m.planResponse)

		case strings.Contains(r.URL.Path, "IssueService/CreateIssue"):
			m.captureBody(r, &m.capturedIssue)
			w.WriteHeader(m.issueStatus)
			_ = json.NewEncoder(w).Encode(m.issueResponse)

		case strings.Contains(r.URL.Path, "RolloutService/CreateRollout"):
			m.captureBody(r, &m.capturedRollout)
			w.WriteHeader(m.rolloutStatus)
			_ = json.NewEncoder(w).Encode(m.rolloutResponse)

		case strings.Contains(r.URL.Path, "DatabaseService/ListDatabases"):
			listHandler.ServeHTTP(w, r)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "unknown path"})
		}
	})
}

func (m *changeMock) captureBody(r *http.Request, target *map[string]any) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	m.mu.Lock()
	*target = body
	m.mu.Unlock()
}

func (m *changeMock) getCapturedSheet() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedSheet
}

func (m *changeMock) getCapturedPlan() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedPlan
}

func (m *changeMock) getCapturedIssue() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedIssue
}

func (m *changeMock) getCapturedRollout() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedRollout
}

func newChangeTestServer(t *testing.T, mock *changeMock) *Server {
	t.Helper()
	s := newTestServerWithMock(t, mock.handler())
	s.profile.ExternalURL = "https://bytebase.example.com"
	// Use a short poll budget to avoid 10s waits in tests.
	s.planCheckPollBudgetOverride = 100 * time.Millisecond
	return s
}

// --- Happy path tests ---

func TestChange_HappyPath_StopAtIssue(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("PENDING")
	s := newChangeTestServer(t, mock)

	result, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE users ADD COLUMN status VARCHAR(20)",
		Title:    "Add status column",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Equal(t, "instances/prod-pg/databases/employee_db", output.Database)
	require.Equal(t, "projects/hr-system", output.Project)
	require.Equal(t, "MIGRATE", output.ResolvedChangeType)
	require.Equal(t, 1, output.TargetCount)
	require.Equal(t, "projects/hr-system/sheets/1001", output.Sheet)
	require.Equal(t, "projects/hr-system/plans/2002", output.Plan)
	require.Equal(t, "projects/hr-system/issues/3003", output.Issue)
	require.False(t, output.RolloutCreated)
	require.Equal(t, "NOT_REQUESTED", output.RolloutDeferredReason)
	require.Equal(t, "APPROVE_ISSUE", output.NextAction)
	require.Contains(t, output.Links.Issue, "projects/hr-system/issues/3003")
	require.Contains(t, output.Links.Plan, "projects/hr-system/plans/2002")
	require.Empty(t, output.Links.Rollout)
}

func TestChange_HappyPath_WithRollout(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	result, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE users ADD COLUMN status VARCHAR(20)",
		Title:         "Add status column",
		CreateRollout: true,
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.True(t, output.RolloutCreated)
	require.Equal(t, "projects/hr-system/rollouts/4004", output.Rollout)
	require.Equal(t, "MONITOR_ROLLOUT", output.NextAction)
	require.Contains(t, output.Links.Rollout, "projects/hr-system/rollouts/4004")
}

func TestChange_SDL(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:   "employee_db",
		SQL:        "CREATE TABLE users (id INT PRIMARY KEY);",
		Title:      "SDL change",
		ChangeType: "SDL",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	// changeType is reflected in the response but not sent to the API
	// (the backend auto-detects MIGRATE vs SDL from sheet content).
	require.Equal(t, "SDL", output.ResolvedChangeType)

	// Verify plan spec does NOT include a type field (removed in migration 3.14).
	plan := mock.getCapturedPlan()
	planObj, ok := plan["plan"].(map[string]any)
	require.True(t, ok)
	specs, ok := planObj["specs"].([]any)
	require.True(t, ok)
	spec, ok := specs[0].(map[string]any)
	require.True(t, ok)
	cfg, ok := spec["changeDatabaseConfig"].(map[string]any)
	require.True(t, ok)
	_, hasType := cfg["type"]
	require.False(t, hasType, "type field should not be sent to the API")
}

func TestChange_ReasonIncluded(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	_, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE users ADD COLUMN x INT",
		Title:    "Test",
		Reason:   "INC-1234",
	})
	require.NoError(t, err)

	issue := mock.getCapturedIssue()
	issueObj, ok := issue["issue"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "INC-1234", issueObj["description"])
}

func TestChange_SQLBase64Encoded(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	sql := "ALTER TABLE users ADD COLUMN status VARCHAR(20)"
	_, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      sql,
		Title:    "Test",
	})
	require.NoError(t, err)

	sheet := mock.getCapturedSheet()
	sheetObj, ok := sheet["sheet"].(map[string]any)
	require.True(t, ok)
	encoded, ok := sheetObj["content"].(string)
	require.True(t, ok)
	decoded, decodeErr := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, decodeErr)
	require.Equal(t, sql, string(decoded))
	// Sheet should not contain dead fields (title, engine removed from proto).
	_, hasTitle := sheetObj["title"]
	require.False(t, hasTitle)
	_, hasEngine := sheetObj["engine"]
	require.False(t, hasEngine)
}

// --- Defaults tests ---

func TestChange_Defaults_ChangeType(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Equal(t, "MIGRATE", output.ResolvedChangeType)
}

func TestChange_Defaults_CreateRollout(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.False(t, output.RolloutCreated)
	require.Equal(t, "NOT_REQUESTED", output.RolloutDeferredReason)
	// No rollout request should have been made.
	require.Nil(t, mock.getCapturedRollout())
}

func TestChange_Defaults_NextAction_NoApproval(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Equal(t, "CREATE_ROLLOUT", output.NextAction)
}

func TestChange_Defaults_NextAction_ApprovalNeeded(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("PENDING")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Equal(t, "APPROVE_ISSUE", output.NextAction)
}

// --- Plan check variation tests ---

func TestChange_PlanChecks_DoneWithWarnings(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planCheckRunResponse = map[string]any{
		"status": "DONE",
		"results": []any{
			map[string]any{"status": "WARNING", "title": "Column has no default", "content": "details"},
		},
	}
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.NotNil(t, output.PlanChecks)
	require.Equal(t, "DONE", output.PlanChecks.Status)
	require.Equal(t, 0, output.PlanChecks.Summary.Error)
	require.Equal(t, 1, output.PlanChecks.Summary.Warning)
	// Warnings don't block rollout.
	require.True(t, output.RolloutCreated)
}

func TestChange_PlanChecks_DoneWithErrors(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planCheckRunResponse = map[string]any{
		"status": "DONE",
		"results": []any{
			map[string]any{"status": "ERROR", "title": "Syntax error near line 1"},
			map[string]any{"status": "ERROR", "title": "Invalid column type"},
		},
	}
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y BOGUS",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.NotNil(t, output.PlanChecks)
	require.Equal(t, "DONE", output.PlanChecks.Status)
	require.Equal(t, 2, output.PlanChecks.Summary.Error)
	// Errors block rollout.
	require.False(t, output.RolloutCreated)
	require.Equal(t, "PLAN_CHECK_ERROR", output.RolloutDeferredReason)
	require.Equal(t, "FIX_SQL_AND_RETRY", output.NextAction)
}

func TestChange_PlanChecks_StillRunning(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planCheckRunResponse = map[string]any{
		"status":  "RUNNING",
		"results": []any{},
	}
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.NotNil(t, output.PlanChecks)
	require.Equal(t, "RUNNING", output.PlanChecks.Status)
	require.NotEmpty(t, output.PlanChecks.PlanCheckRun)
	require.False(t, output.RolloutCreated)
	require.Equal(t, "PLAN_CHECK_PENDING", output.RolloutDeferredReason)
	require.Equal(t, "WAIT_PLAN_CHECK", output.NextAction)
	// Issue should still be created.
	require.NotEmpty(t, output.Issue)
}

func TestChange_PlanChecks_Failed(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planCheckRunResponse = map[string]any{
		"status":  "FAILED",
		"results": []any{},
	}
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.NotNil(t, output.PlanChecks)
	require.Equal(t, "FAILED", output.PlanChecks.Status)
	require.False(t, output.RolloutCreated)
	require.Equal(t, "PLAN_CHECK_ERROR", output.RolloutDeferredReason)
	require.Equal(t, "FIX_SQL_AND_RETRY", output.NextAction)
}

func TestChange_PlanChecks_RunPlanChecksAPIFailure(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.runPlanChecksStatus = http.StatusInternalServerError
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.NotNil(t, output.PlanChecks)
	// Trigger failure is treated as RUNNING (not FAILED) to avoid misleading
	// FIX_SQL_AND_RETRY when the issue is a permission or transient error.
	require.Equal(t, "RUNNING", output.PlanChecks.Status)
	require.NotEmpty(t, output.PlanChecks.PlanCheckRun)
	// Issue still created.
	require.NotEmpty(t, output.Issue)
}

func TestChange_PlanChecks_PollFailure(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planCheckRunStatus = http.StatusInternalServerError
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.NotNil(t, output.PlanChecks)
	require.Equal(t, "RUNNING", output.PlanChecks.Status)
	// Issue still created.
	require.NotEmpty(t, output.Issue)
}

// --- Rollout gating tests ---

func TestChange_Rollout_ApprovalPending(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("PENDING")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.False(t, output.RolloutCreated)
	require.Equal(t, "APPROVAL_PENDING", output.RolloutDeferredReason)
	require.Equal(t, "APPROVE_ISSUE", output.NextAction)
}

func TestChange_Rollout_ApprovalSkipped(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("SKIPPED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.True(t, output.RolloutCreated)
	require.Equal(t, "MONITOR_ROLLOUT", output.NextAction)
}

func TestChange_Rollout_CreateFailed(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("APPROVED")
	mock.rolloutStatus = http.StatusInternalServerError
	mock.rolloutResponse = map[string]any{"message": "internal error"}
	s := newChangeTestServer(t, mock)

	result, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.False(t, output.RolloutCreated)
	require.Equal(t, "ROLLOUT_CREATE_FAILED", output.RolloutDeferredReason)
	// Should NOT be APPROVE_ISSUE — don't assume approval pending.
	require.NotEqual(t, "APPROVE_ISSUE", output.NextAction)
	// Backend error should be forwarded in the text output.
	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "rollout creation failed")
}

func TestChange_Rollout_GateOrder(t *testing.T) {
	// Plan checks have errors AND approval is pending.
	// PLAN_CHECK_ERROR should win (checked first).
	mock := newChangeMock(employeeDB())
	mock.planCheckRunResponse = map[string]any{
		"status": "DONE",
		"results": []any{
			map[string]any{"status": "ERROR", "title": "Bad SQL"},
		},
	}
	mock.issueResponse = defaultIssueResponse("PENDING")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y BOGUS",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Equal(t, "PLAN_CHECK_ERROR", output.RolloutDeferredReason)
	require.Equal(t, "FIX_SQL_AND_RETRY", output.NextAction)
}

func TestChange_Rollout_ApprovalRejected(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("REJECTED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.False(t, output.RolloutCreated)
	require.Equal(t, "APPROVAL_REJECTED", output.RolloutDeferredReason)
	require.Equal(t, "FIX_SQL_AND_RETRY", output.NextAction)
	// Rollout should NOT have been attempted.
	require.Nil(t, mock.getCapturedRollout())
}

func TestChange_Rollout_RequestShape(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	rollout := mock.getCapturedRollout()
	require.NotNil(t, rollout)
	// CreateRollout uses parent=<plan> with no rollout body payload.
	require.Equal(t, "projects/hr-system/plans/2002", rollout["parent"])
	_, hasRollout := rollout["rollout"]
	require.False(t, hasRollout, "CreateRollout should not have a rollout body field")
}

// --- Resolution and validation tests ---

func TestChange_DatabaseNotFound(t *testing.T) {
	mock := newChangeMock(nil) // no databases
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "nonexistent",
		SQL:      "SELECT 1",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "DATABASE_NOT_FOUND")
}

func TestChange_AmbiguousDatabase_ElicitationCancelled(t *testing.T) {
	// Two databases with the same short name on different instances.
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/app", "instances/prod-pg", "projects/payments", "POSTGRES", "ds-1"),
		makeDatabase("instances/staging-pg/databases/app", "instances/staging-pg", "projects/staging", "POSTGRES", "ds-2"),
	}
	mock := newChangeMock(databases)
	s := newChangeTestServer(t, mock)

	// nil request means no session -> elicitation unavailable -> fallback to AMBIGUOUS_TARGET.
	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "app",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "AMBIGUOUS_TARGET")
	require.Contains(t, text, "prod-pg")
	require.Contains(t, text, "staging-pg")
}

func TestChange_ProjectDerived(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Equal(t, "projects/hr-system", output.Project)

	// All API calls use the correct project parent.
	sheet := mock.getCapturedSheet()
	require.Equal(t, "projects/hr-system", sheet["parent"])
	plan := mock.getCapturedPlan()
	require.Equal(t, "projects/hr-system", plan["parent"])
	issue := mock.getCapturedIssue()
	require.Equal(t, "projects/hr-system", issue["parent"])
}

func TestChange_ProjectMismatch(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
		Project:  "wrong-project",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "PROJECT_MISMATCH")
}

func TestChange_InvalidChangeType(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:   "employee_db",
		SQL:        "ALTER TABLE x ADD COLUMN y INT",
		Title:      "Test",
		ChangeType: "DML_FIX",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "INVALID_ARGUMENT")
	require.Contains(t, text, "MIGRATE")
	require.Contains(t, text, "SDL")
}

func TestChange_MissingRequiredFields(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	tests := []struct {
		name  string
		input ChangeInput
		field string
	}{
		{
			name:  "missing sql",
			input: ChangeInput{Database: "employee_db", Title: "Test"},
			field: "sql",
		},
		{
			name:  "missing title",
			input: ChangeInput{Database: "employee_db", SQL: "SELECT 1"},
			field: "title",
		},
		{
			name:  "missing database",
			input: ChangeInput{SQL: "SELECT 1", Title: "Test"},
			field: "database",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := s.handleChange(testContext(), nil, tc.input)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.field)
		})
	}
}

func TestChange_ReasonTruncation(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	longReason := strings.Repeat("x", 2000)
	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
		Reason:   longReason,
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "reason truncated to 1000 chars")

	// Issue description should be truncated.
	issue := mock.getCapturedIssue()
	issueObj, ok := issue["issue"].(map[string]any)
	require.True(t, ok)
	desc, ok := issueObj["description"].(string)
	require.True(t, ok)
	require.Len(t, desc, 1000) // 997 chars + "..."
}

// --- Partial failure tests ---

func TestChange_SheetCreateFailed(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.sheetStatus = http.StatusInternalServerError
	mock.sheetResponse = map[string]any{"message": "internal error"}
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SHEET_CREATE_FAILED")
	// No partialRefs since nothing was created.
	require.NotContains(t, text, "partialRefs")
}

func TestChange_PlanCreateFailed(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planStatus = http.StatusInternalServerError
	mock.planResponse = map[string]any{"message": "internal error"}
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "PLAN_CREATE_FAILED")
	require.Contains(t, text, "sheets/1001") // partialRefs includes sheet
}

func TestChange_IssueCreateFailed(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueStatus = http.StatusInternalServerError
	mock.issueResponse = map[string]any{"message": "internal error"}
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "ISSUE_CREATE_FAILED")
	require.Contains(t, text, "sheets/1001") // partialRefs
	require.Contains(t, text, "plans/2002")  // partialRefs
}

func TestChange_IssueCreateFailed_SqlReview(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planCheckRunResponse = map[string]any{
		"status": "DONE",
		"results": []any{
			map[string]any{"status": "ERROR", "title": "SQL review failed"},
		},
	}
	mock.issueStatus = http.StatusBadRequest
	mock.issueResponse = map[string]any{"message": "enforceSqlReview: checks failed"}
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y BOGUS",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "ISSUE_CREATE_FAILED")
	require.Contains(t, text, "plan checks must pass")
	// Verify the output is valid JSON (guidance folded into suggestion, not appended as raw text).
	var parsed changeError
	require.NoError(t, json.Unmarshal([]byte(text), &parsed))
	require.Equal(t, "ISSUE_CREATE_FAILED", parsed.Code)
	require.Contains(t, parsed.Suggestion, "plan checks must pass")
}

// --- Permission tests ---

func TestChange_PermissionDenied_Sheet(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.sheetStatus = http.StatusForbidden
	mock.sheetResponse = map[string]any{"message": "permission denied"}
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "PERMISSION_DENIED")
	require.Contains(t, text, "bb.sheets.create")
}

func TestChange_PermissionDenied_Plan(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.planStatus = http.StatusForbidden
	mock.planResponse = map[string]any{"message": "permission denied"}
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "PERMISSION_DENIED")
	require.Contains(t, text, "bb.plans.create")
	require.Contains(t, text, "sheets/1001") // partialRefs
}

func TestChange_PermissionDenied_Issue(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueStatus = http.StatusForbidden
	mock.issueResponse = map[string]any{"message": "permission denied"}
	s := newChangeTestServer(t, mock)

	result, _, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "PERMISSION_DENIED")
	require.Contains(t, text, "bb.issues.create")
	require.Contains(t, text, "sheets/1001") // partialRefs
	require.Contains(t, text, "plans/2002")  // partialRefs
}

// --- Link construction tests ---

func TestChange_Links_Constructed(t *testing.T) {
	mock := newChangeMock(employeeDB())
	mock.issueResponse = defaultIssueResponse("APPROVED")
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database:      "employee_db",
		SQL:           "ALTER TABLE x ADD COLUMN y INT",
		Title:         "Test",
		CreateRollout: true,
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Equal(t, "https://bytebase.example.com/projects/hr-system/issues/3003", output.Links.Issue)
	require.Equal(t, "https://bytebase.example.com/projects/hr-system/plans/2002", output.Links.Plan)
	require.Equal(t, "https://bytebase.example.com/projects/hr-system/rollouts/4004", output.Links.Rollout)
}

func TestChange_Links_TrailingSlash(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)
	s.profile.ExternalURL = "https://bytebase.example.com/"

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	// No double slash.
	require.Equal(t, "https://bytebase.example.com/projects/hr-system/issues/3003", output.Links.Issue)
}

func TestChange_Links_RolloutOmitted(t *testing.T) {
	mock := newChangeMock(employeeDB())
	s := newChangeTestServer(t, mock)

	_, structured, err := s.handleChange(testContext(), nil, ChangeInput{
		Database: "employee_db",
		SQL:      "ALTER TABLE x ADD COLUMN y INT",
		Title:    "Test",
	})
	require.NoError(t, err)

	output, ok := structured.(*ChangeOutput)
	require.True(t, ok)
	require.Empty(t, output.Links.Rollout)
}
