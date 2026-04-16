package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

const (
	reasonMaxLen          = 1000
	planCheckPollBudget   = 10 * time.Second
	planCheckPollInterval = 1 * time.Second
)

// Change types.
const (
	changeTypeMigrate = "MIGRATE"
	changeTypeSDL     = "SDL"
)

// nextAction enum constants.
const (
	nextActionApproveIssue   = "APPROVE_ISSUE"
	nextActionCreateRollout  = "CREATE_ROLLOUT"
	nextActionMonitorRollout = "MONITOR_ROLLOUT"
	nextActionWaitPlanCheck  = "WAIT_PLAN_CHECK"
	nextActionFixSQLRetry    = "FIX_SQL_AND_RETRY"
)

// rolloutDeferredReason enum constants.
const (
	rolloutNotRequested     = "NOT_REQUESTED"
	rolloutApprovalPending  = "APPROVAL_PENDING"
	rolloutApprovalRejected = "APPROVAL_REJECTED"
	rolloutPlanCheckPending = "PLAN_CHECK_PENDING"
	rolloutPlanCheckError   = "PLAN_CHECK_ERROR"
	rolloutCreateFailed     = "ROLLOUT_CREATE_FAILED"
)

// planChecks.status enum constants.
const (
	planCheckDone    = "DONE"
	planCheckRunning = "RUNNING"
	planCheckFailed  = "FAILED"
)

// ChangeInput is the input for the propose_database_change tool.
type ChangeInput struct {
	Database      string `json:"database"`
	SQL           string `json:"sql"`
	Title         string `json:"title"`
	Instance      string `json:"instance,omitempty"`
	Project       string `json:"project,omitempty"`
	ChangeType    string `json:"changeType,omitempty"`
	CreateRollout bool   `json:"createRollout,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// ChangeOutput is the output for the propose_database_change tool.
type ChangeOutput struct {
	Database              string         `json:"database"`
	Project               string         `json:"project"`
	ResolvedChangeType    string         `json:"resolvedChangeType"`
	TargetCount           int            `json:"targetCount"`
	Sheet                 string         `json:"sheet"`
	Plan                  string         `json:"plan"`
	PlanChecks            *PlanCheckInfo `json:"planChecks,omitempty"`
	Issue                 string         `json:"issue"`
	RolloutCreated        bool           `json:"rolloutCreated"`
	Rollout               string         `json:"rollout,omitempty"`
	RolloutDeferredReason string         `json:"rolloutDeferredReason,omitempty"`
	NextAction            string         `json:"nextAction"`
	Links                 ChangeLinks    `json:"links"`
}

// PlanCheckInfo holds plan check status and results.
type PlanCheckInfo struct {
	Status       string            `json:"status"`
	Summary      *PlanCheckSummary `json:"summary,omitempty"`
	Results      []PlanCheckResult `json:"results,omitempty"`
	PlanCheckRun string            `json:"planCheckRun,omitempty"`
}

// PlanCheckSummary holds error/warning counts.
type PlanCheckSummary struct {
	Error   int `json:"error"`
	Warning int `json:"warning"`
}

// PlanCheckResult holds a single plan check result.
type PlanCheckResult struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ChangeLinks holds URLs for created resources.
type ChangeLinks struct {
	Issue   string `json:"issue"`
	Plan    string `json:"plan"`
	Rollout string `json:"rollout,omitempty"`
}

// changeError is a structured error with partial refs for the change tool.
type changeError struct {
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	Suggestion  string            `json:"suggestion,omitempty"`
	PartialRefs map[string]string `json:"partialRefs,omitempty"`
}

func (e *changeError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// proposeChangeDescription is the description for the propose_database_change tool.
const proposeChangeDescription = `Run DDL/DML changes (ALTER TABLE, CREATE TABLE, INSERT, UPDATE, migrations) against a Bytebase database.

This is the primary tool for making database changes. Use this instead of manually calling CreateSheet/CreatePlan/CreateIssue via call_api.

Provide plain SQL — the tool handles database resolution, sheet creation, plan creation with automatic plan checks, and issue creation in one call. Optionally creates a rollout.

| Parameter     | Required | Description |
|---------------|----------|-------------|
| database      | Yes      | Database name or substring |
| sql           | Yes      | SQL statement(s) in plain text (NOT base64) |
| title         | Yes      | Title for the issue/plan |
| instance      | No       | Narrow database resolution |
| project       | No       | Narrow to a specific project |
| changeType    | No       | "MIGRATE" (default) or "SDL" |
| createRollout | No       | If true, attempts rollout creation after issue |
| reason        | No       | Context or ticket reference (max 1000 chars) |

**Examples:**
propose_database_change(database="app", sql="ALTER TABLE users ADD COLUMN status VARCHAR(20)", title="Add status column")
propose_database_change(database="app", sql="UPDATE orders SET status='shipped' WHERE id=42", title="Ship order 42", createRollout=true)

**Notes:**
- v1 supports single database targets only. For batch changes across multiple databases, use get_skill("database-change").
- Plan checks run automatically; results included in response when available.
- Requires bb.sheets.create, bb.plans.create, bb.issues.create permissions.
- If createRollout=true but policy gates aren't satisfied, returns success with rolloutCreated=false and a reason.`

func (s *Server) registerChangeTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "propose_database_change",
		Description: proposeChangeDescription,
	}, s.handleChange)
}

func (s *Server) handleChange(ctx context.Context, req *mcp.CallToolRequest, input ChangeInput) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate input.
	if input.Database == "" {
		return nil, nil, errors.New("database is required")
	}
	if input.SQL == "" {
		return nil, nil, errors.New("sql is required")
	}
	if input.Title == "" {
		return nil, nil, errors.New("title is required")
	}

	changeType := input.ChangeType
	if changeType == "" {
		changeType = changeTypeMigrate
	}
	if changeType != changeTypeMigrate && changeType != changeTypeSDL {
		return formatToolError(&toolError{
			Code:       "INVALID_ARGUMENT",
			Message:    fmt.Sprintf("invalid changeType %q", changeType),
			Suggestion: "allowed values: MIGRATE, SDL",
		}), nil, nil
	}

	var warnings []string
	reason := input.Reason
	runes := []rune(reason)
	if len(runes) > reasonMaxLen {
		reason = string(runes[:reasonMaxLen-3]) + "..."
		warnings = append(warnings, "reason truncated to 1000 chars")
	}

	// Step 2: Resolve database.
	resolved, resolveResult := s.resolveChangeTarget(ctx, req, input)
	if resolveResult != nil {
		return resolveResult, nil, nil
	}

	// Step 3: Use project from resolved database.
	// When input.Project is set, the resolver already filtered by it.
	project := resolved.project

	// Step 4: Create sheet.
	sheetName, err := s.createSheet(ctx, project, resolved.engine, input.Title, input.SQL)
	if err != nil {
		return formatChangeStepError(err, "SHEET_CREATE_FAILED", "bb.sheets.create", nil), nil, nil
	}

	// Step 5: Create plan.
	planName, err := s.createPlan(ctx, project, input.Title, resolved.resourceName, sheetName, changeType)
	if err != nil {
		return formatChangeStepError(err, "PLAN_CREATE_FAILED", "bb.plans.create", map[string]string{"sheet": sheetName}), nil, nil
	}

	// Step 6: Run plan checks (hybrid async, 10s budget).
	planChecks := s.runPlanChecks(ctx, planName)

	// Step 7: Create issue.
	issueName, approvalStatus, err := s.createIssue(ctx, project, input.Title, planName, reason)
	if err != nil {
		partialRefs := map[string]string{"sheet": sheetName, "plan": planName}
		// If plan checks had errors, enforceSqlReview may have blocked issue creation.
		suggestion := ""
		if planChecks != nil && planChecks.Status == planCheckDone && planChecks.Summary != nil && planChecks.Summary.Error > 0 {
			suggestion = "plan checks must pass before issue creation; fix SQL and retry"
		}
		return formatChangeStepErrorWithHint(err, "ISSUE_CREATE_FAILED", "bb.issues.create", partialRefs, suggestion), nil, nil
	}

	// Step 8: Determine nextAction from approvalStatus.
	nextAction := deriveNextAction(approvalStatus)

	// Step 9: Conditionally create rollout.
	rolloutCreated := false
	rolloutName := ""
	rolloutDeferredReason := ""

	switch {
	case !input.CreateRollout:
		rolloutDeferredReason = rolloutNotRequested
	case planChecks != nil && planChecks.Status == planCheckRunning:
		rolloutDeferredReason = rolloutPlanCheckPending
		nextAction = nextActionWaitPlanCheck
	case planChecksHaveErrors(planChecks):
		rolloutDeferredReason = rolloutPlanCheckError
		nextAction = nextActionFixSQLRetry
	case approvalStatus == "PENDING" || approvalStatus == "CHECKING":
		rolloutDeferredReason = rolloutApprovalPending
		nextAction = nextActionApproveIssue
	case approvalStatus == "REJECTED":
		rolloutDeferredReason = rolloutApprovalRejected
		nextAction = nextActionFixSQLRetry
	default:
		rolloutName, err = s.createRollout(ctx, planName)
		if err != nil {
			rolloutDeferredReason = rolloutCreateFailed
			warnings = append(warnings, fmt.Sprintf("rollout creation failed: %s", err.Error()))
			// Keep nextAction from step 8 — do NOT assume APPROVAL_PENDING.
		} else {
			rolloutCreated = true
			nextAction = nextActionMonitorRollout
		}
	}

	// Step 10: Build response.
	output := &ChangeOutput{
		Database:              resolved.resourceName,
		Project:               project,
		ResolvedChangeType:    changeType,
		TargetCount:           1,
		Sheet:                 sheetName,
		Plan:                  planName,
		PlanChecks:            planChecks,
		Issue:                 issueName,
		RolloutCreated:        rolloutCreated,
		Rollout:               rolloutName,
		RolloutDeferredReason: rolloutDeferredReason,
		NextAction:            nextAction,
		Links: ChangeLinks{
			Issue: s.buildResourceURL(issueName),
			Plan:  s.buildResourceURL(planName),
		},
	}
	if rolloutCreated {
		output.Links.Rollout = s.buildResourceURL(rolloutName)
	}

	text := formatChangeOutput(output, warnings)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}

// resolveChangeTarget runs the shared database resolver with elicitation fallback.
// The project parameter is passed to the server-side filter so it can disambiguate
// when the same database name exists in multiple projects.
func (s *Server) resolveChangeTarget(ctx context.Context, req *mcp.CallToolRequest, input ChangeInput) (*resolvedDatabase, *mcp.CallToolResult) {
	resolveCtx, resolveCancel := context.WithTimeout(ctx, resolveTimeout)
	defer resolveCancel()

	resolved, err := s.resolveDatabase(resolveCtx, input.Database, input.Instance, input.Project)
	if err != nil {
		return nil, formatToolError(err)
	}
	if !resolved.ambiguous {
		return resolved, nil
	}
	picked, elicitErr := s.elicitDatabaseChoice(ctx, req, resolved)
	if elicitErr != nil {
		return nil, formatAmbiguousResult(input.Database, resolved.candidates)
	}
	return picked, nil
}

// createSheet creates a sheet with base64-encoded SQL content.
// Note: the Sheet proto only has name, content, and content_size.
// title and engine were removed; the backend discards unknown fields.
func (s *Server) createSheet(ctx context.Context, project, _, _, sql string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(sql))
	body := map[string]any{
		"parent": project,
		"sheet": map[string]any{
			"content": encoded,
		},
	}

	resp, err := s.apiRequest(ctx, "/bytebase.v1.SheetService/CreateSheet", body)
	if err != nil {
		return "", errors.Wrap(err, "sheet creation request failed")
	}
	if err := checkAPIResponse(resp, "create sheets", "bb.sheets.create"); err != nil {
		return "", err
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse CreateSheet response")
	}
	return result.Name, nil
}

// createPlan creates a plan with a single changeDatabaseConfig spec.
// Note: the proto ChangeDatabaseConfig has no `type` field (MIGRATE vs SDL was
// merged in migration 3.14). The backend auto-detects from sheet content.
// changeType is accepted as input for forward-compat but not sent on the wire.
func (s *Server) createPlan(ctx context.Context, project, title, target, sheet, _ string) (string, error) {
	body := map[string]any{
		"parent": project,
		"plan": map[string]any{
			"title": title,
			"specs": []map[string]any{
				{
					"id": "spec-1",
					"changeDatabaseConfig": map[string]any{
						"targets": []string{target},
						"sheet":   sheet,
					},
				},
			},
		},
	}

	resp, err := s.apiRequest(ctx, "/bytebase.v1.PlanService/CreatePlan", body)
	if err != nil {
		return "", errors.Wrap(err, "plan creation request failed")
	}
	if err := checkAPIResponse(resp, "create plans", "bb.plans.create"); err != nil {
		return "", err
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse CreatePlan response")
	}
	return result.Name, nil
}

// planCheckBudget returns the effective poll budget, allowing tests to override.
func (s *Server) planCheckBudget() time.Duration {
	if s.planCheckPollBudgetOverride > 0 {
		return s.planCheckPollBudgetOverride
	}
	return planCheckPollBudget
}

// runPlanChecks triggers plan checks and polls for results within the poll budget.
func (s *Server) runPlanChecks(ctx context.Context, planName string) *PlanCheckInfo {
	checkRunName := planName + "/planCheckRun"

	// Trigger plan checks. If this fails, still poll — CreatePlan already
	// initializes plan checks server-side, so they may be running regardless.
	_, _ = s.apiRequest(ctx, "/bytebase.v1.PlanService/RunPlanChecks", map[string]any{
		"name": planName,
	})

	pendingResult := &PlanCheckInfo{
		Status:       planCheckRunning,
		PlanCheckRun: checkRunName,
	}

	// Poll for results. Transient errors (404 before row visible, brief 500)
	// are retried within the budget rather than bailing immediately.
	deadline := time.Now().Add(s.planCheckBudget())
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return pendingResult
		}

		pollResp, err := s.apiRequest(ctx, "/bytebase.v1.PlanService/GetPlanCheckRun", map[string]any{
			"name": checkRunName,
		})
		if err != nil || pollResp.Status >= 400 {
			time.Sleep(planCheckPollInterval)
			continue
		}

		var checkRun planCheckRunResponse
		if err := json.Unmarshal(pollResp.Body, &checkRun); err != nil {
			time.Sleep(planCheckPollInterval)
			continue
		}

		switch checkRun.Status {
		case "DONE":
			return buildPlanCheckInfo(checkRun)
		case "FAILED", "CANCELED":
			return &PlanCheckInfo{Status: planCheckFailed}
		default:
			// RUNNING or unknown — keep polling.
		}

		time.Sleep(planCheckPollInterval)
	}

	// Budget exhausted.
	return pendingResult
}

// planCheckRunResponse mirrors the PlanCheckRun proto response.
type planCheckRunResponse struct {
	Status  string                   `json:"status"`
	Results []planCheckResultMessage `json:"results"`
}

// planCheckResultMessage mirrors a single result in PlanCheckRun.
type planCheckResultMessage struct {
	Status  string `json:"status"` // SUCCESS, WARNING, ERROR
	Title   string `json:"title"`
	Content string `json:"content"`
}

// buildPlanCheckInfo converts a completed plan check run into PlanCheckInfo.
func buildPlanCheckInfo(run planCheckRunResponse) *PlanCheckInfo {
	info := &PlanCheckInfo{
		Status:  planCheckDone,
		Summary: &PlanCheckSummary{},
	}
	for _, r := range run.Results {
		msg := r.Title
		if msg == "" {
			msg = r.Content
		}
		switch r.Status {
		case "ERROR":
			info.Summary.Error++
			info.Results = append(info.Results, PlanCheckResult{Type: "ERROR", Message: msg})
		case "WARNING":
			info.Summary.Warning++
			info.Results = append(info.Results, PlanCheckResult{Type: "WARNING", Message: msg})
		default:
			// SUCCESS and other statuses are not included in the output.
		}
	}
	return info
}

// createIssue creates an issue linked to a plan.
func (s *Server) createIssue(ctx context.Context, project, title, planName, description string) (string, string, error) {
	issue := map[string]any{
		"title": title,
		"type":  "DATABASE_CHANGE",
		"plan":  planName,
	}
	if description != "" {
		issue["description"] = description
	}
	body := map[string]any{
		"parent": project,
		"issue":  issue,
	}

	resp, err := s.apiRequest(ctx, "/bytebase.v1.IssueService/CreateIssue", body)
	if err != nil {
		return "", "", errors.Wrap(err, "issue creation request failed")
	}
	if err := checkAPIResponse(resp, "create issues", "bb.issues.create"); err != nil {
		return "", "", err
	}

	var result struct {
		Name           string `json:"name"`
		ApprovalStatus string `json:"approvalStatus"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", "", errors.Wrap(err, "failed to parse CreateIssue response")
	}
	return result.Name, result.ApprovalStatus, nil
}

// createRollout creates a rollout for a plan.
func (s *Server) createRollout(ctx context.Context, planName string) (string, error) {
	body := map[string]any{
		"parent": planName,
	}

	resp, err := s.apiRequest(ctx, "/bytebase.v1.RolloutService/CreateRollout", body)
	if err != nil {
		return "", errors.Wrap(err, "rollout creation request failed")
	}
	if err := checkAPIResponse(resp, "create rollouts", "bb.rollouts.create"); err != nil {
		return "", err
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse CreateRollout response")
	}
	return result.Name, nil
}

// deriveNextAction maps approvalStatus to the next agent action.
func deriveNextAction(approvalStatus string) string {
	switch approvalStatus {
	case "APPROVED", "SKIPPED":
		return nextActionCreateRollout
	case "REJECTED":
		return nextActionFixSQLRetry
	default: // PENDING, CHECKING, or unknown
		return nextActionApproveIssue
	}
}

// planChecksHaveErrors returns true if plan checks completed with errors or failed.
func planChecksHaveErrors(info *PlanCheckInfo) bool {
	if info == nil {
		return false
	}
	if info.Status == planCheckFailed {
		return true
	}
	if info.Status == planCheckDone && info.Summary != nil && info.Summary.Error > 0 {
		return true
	}
	return false
}

// buildResourceURL constructs a URL from externalURL and resource name.
func (s *Server) buildResourceURL(resourceName string) string {
	base := strings.TrimRight(s.profile.ExternalURL, "/")
	if base == "" {
		return resourceName
	}
	return base + "/" + resourceName
}

// formatChangeOutput produces a text header + JSON body.
func formatChangeOutput(output *ChangeOutput, warnings []string) string {
	var sb strings.Builder

	for _, w := range warnings {
		fmt.Fprintf(&sb, "Note: %s\n", w)
	}

	fmt.Fprintf(&sb, "Database: %s\n", output.Database)
	fmt.Fprintf(&sb, "Project: %s\n", output.Project)
	fmt.Fprintf(&sb, "Change: %s | Target count: %d\n", output.ResolvedChangeType, output.TargetCount)
	fmt.Fprintf(&sb, "Issue: %s | Next action: %s\n", output.Issue, output.NextAction)

	if output.RolloutCreated {
		fmt.Fprintf(&sb, "Rollout: %s (created)\n", output.Rollout)
	} else if output.RolloutDeferredReason != "" {
		fmt.Fprintf(&sb, "Rollout: deferred (%s)\n", output.RolloutDeferredReason)
	}

	sb.WriteString("\n")
	jsonBytes, _ := json.Marshal(output)
	sb.Write(jsonBytes)

	return sb.String()
}

// checkAPIResponse checks an API response for permission or general errors.
// Returns nil if the response is OK, or a structured error otherwise.
func checkAPIResponse(resp *apiResponse, operation, permission string) error {
	if resp.Status == http.StatusForbidden || resp.Status == http.StatusUnauthorized {
		return &toolError{
			Code:       "PERMISSION_DENIED",
			Message:    fmt.Sprintf("you don't have permission to %s", operation),
			Suggestion: fmt.Sprintf("ask your workspace admin to grant you the %s permission", permission),
		}
	}
	if resp.Status >= 400 {
		return errors.Errorf("%s failed: HTTP %d: %s", operation, resp.Status, parseError(resp.Body))
	}
	return nil
}

// formatChangeError formats a changeError into an MCP error result.
func formatChangeError(err *changeError) *mcp.CallToolResult {
	jsonBytes, _ := json.MarshalIndent(err, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
		IsError: true,
	}
}

// formatChangeStepError formats a step failure into an MCP error result.
// It detects permission errors and wraps them appropriately.
func formatChangeStepError(err error, defaultCode, permission string, partialRefs map[string]string) *mcp.CallToolResult {
	return formatChangeStepErrorWithHint(err, defaultCode, permission, partialRefs, "")
}

// formatChangeStepErrorWithHint is like formatChangeStepError but replaces the
// default suggestion with hint when non-empty, keeping the JSON shape intact.
func formatChangeStepErrorWithHint(err error, defaultCode, permission string, partialRefs map[string]string, hint string) *mcp.CallToolResult {
	var te *toolError
	if errors.As(err, &te) && te.Code == "PERMISSION_DENIED" {
		return formatChangeError(&changeError{
			Code:        "PERMISSION_DENIED",
			Message:     te.Message,
			Suggestion:  fmt.Sprintf("ask your workspace admin to grant you the %s permission", permission),
			PartialRefs: partialRefs,
		})
	}
	suggestion := "check the error and retry"
	if hint != "" {
		suggestion = hint
	}
	return formatChangeError(&changeError{
		Code:        defaultCode,
		Message:     err.Error(),
		Suggestion:  suggestion,
		PartialRefs: partialRefs,
	})
}
