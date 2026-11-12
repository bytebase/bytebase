package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	component "github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// waitReviewRunTerminal polls the issue's review_run slot raw until it
// reaches a terminal status and returns the row.
func waitReviewRunTerminal(ctx context.Context, t *testing.T, ctl *controller, projectID string, issueUID int64) *reviewRun {
	t.Helper()
	a := require.New(t)
	var row *reviewRun
	a.Eventually(func() bool {
		for _, r := range listReviewRuns(ctx, t, ctl, projectID) {
			if r.IssueID == issueUID && (r.Status == "DONE" || r.Status == "FAILED") {
				row = r
				return true
			}
		}
		return false
	}, 90*time.Second, 500*time.Millisecond,
		"review run (%s, %d) should reach a terminal status", projectID, issueUID)
	return row
}

func runReview(ctx context.Context, ctl *controller, issueName string) (*v1pb.ReviewRun, error) {
	resp, err := ctl.issueServiceClient.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
		Name: issueName + "/reviewRun",
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func getReviewRun(ctx context.Context, t *testing.T, ctl *controller, issueName string) *v1pb.ReviewRun {
	t.Helper()
	resp, err := ctl.issueServiceClient.GetReviewRun(ctx, connect.NewRequest(&v1pb.GetReviewRunRequest{
		Name: issueName + "/reviewRun",
	}))
	require.NoError(t, err)
	return resp.Msg
}

// TestCollision_RunReviewLifecycle drives RunReview end to end on two
// projects whose issue ids collide: the scheduler claims and executes the
// review for real, completions land on the right project, and a re-run on one
// project leaves the other project's slot untouched.
func TestCollision_RunReviewLifecycle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	ctl, ctx := startWorkspace(ctx, t, func(p *component.Profile) { p.SQLReviewV2 = true })

	fixture := setupCollidingProjects(ctx, t, ctl)

	// The fixture issues carry rollouts; a rollout freezes the plan's SQL, so
	// review is refused.
	_, err := runReview(ctx, ctl, fixture.IssueA.Name)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	// Fresh colliding plan+issue pairs without rollouts.
	planA, issueA := createPlanAndIssue(ctx, t, ctl, fixture.ProjectA, fixture.DatabaseA, "Review lifecycle A")
	_, issueB := createPlanAndIssue(ctx, t, ctl, fixture.ProjectB, fixture.DatabaseB, "Review lifecycle B")
	projectAID, issueAUID, err := common.GetProjectIDIssueUID(issueA.Name)
	a.NoError(err)
	projectBID, issueBUID, err := common.GetProjectIDIssueUID(issueB.Name)
	a.NoError(err)
	a.Equal(issueAUID, issueBUID, "fixture issue ids should collide")

	// Creating an issue starts its review.
	rowA := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	a.Equal("DONE", rowA.Status, "review on project A, got payload %s", rowA.Payload)
	a.Equal(int64(0), rowA.Attempt)
	rowB := waitReviewRunTerminal(ctx, t, ctl, projectBID, issueBUID)
	a.Equal("DONE", rowB.Status, "review on project B, got payload %s", rowB.Payload)
	a.Equal(int64(0), rowB.Attempt)

	// A name that names a reviewer is not a review run.
	_, err = ctl.issueServiceClient.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{Name: issueA.Name + "/reviewRuns/rule"}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Review on both colliding issues. AI is off in the test workspace, so
	// the review is the standard rules alone.
	runA, err := runReview(ctx, ctl, issueA.Name)
	a.NoError(err)
	a.Equal(issueA.Name+"/reviewRun", runA.Name)
	a.Equal(v1pb.ReviewRun_RUNNING, runA.State)
	a.NotNil(runA.CreateTime)
	a.Nil(runA.EndTime)
	a.Nil(runA.Error)
	runB, err := runReview(ctx, ctl, issueB.Name)
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_RUNNING, runB.State)

	// The scheduler claims and executes both. The rule executor evaluates
	// the real test databases through omni's SQL Review V2 contract.
	rowA = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	a.Equal("DONE", rowA.Status, "review on project A, got payload %s", rowA.Payload)
	a.Equal(int64(1), rowA.Attempt)
	rowB = waitReviewRunTerminal(ctx, t, ctl, projectBID, issueBUID)
	a.Equal("DONE", rowB.Status, "review on project B, got payload %s", rowB.Payload)
	a.Equal(int64(1), rowB.Attempt)
	doneA := getReviewRun(ctx, t, ctl, issueA.Name)
	a.Equal(v1pb.ReviewRun_SUCCEEDED, doneA.State)
	a.NotNil(doneA.EndTime)

	// Re-running A bumps only A's attempt; B's colliding slot is untouched.
	// The fixture's own issues started reviews too; they settle first.
	a.Eventually(func() bool {
		for _, r := range listReviewRuns(ctx, t, ctl, projectBID) {
			if r.Status != "DONE" && r.Status != "FAILED" {
				return false
			}
		}
		return true
	}, 90*time.Second, 500*time.Millisecond, "project B review runs should settle")
	beforeB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Greater(len(beforeB), 0, "project B should have review_run rows")
	runA2, err := runReview(ctx, ctl, issueA.Name)
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_RUNNING, runA2.State)
	rowA2 := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	a.Equal(int64(2), rowA2.Attempt)
	a.Equal("DONE", rowA2.Status)
	afterB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Equal(beforeB, afterB, "project B review_run rows must be untouched by A's re-run")

	// Running the plan checks leaves the review alone, and the checks leave
	// the SQL review to it.
	_, err = ctl.planServiceClient.RunPlanChecks(ctx, connect.NewRequest(&v1pb.RunPlanChecksRequest{Name: planA.Name}))
	a.NoError(err)
	waitPlanCheckRunSettled(ctx, t, ctl, planA.Name)
	rowA2 = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	a.Equal(int64(2), rowA2.Attempt, "rerunning the plan checks starts no review")
	checkRun, err := ctl.planServiceClient.GetPlanCheckRun(ctx, connect.NewRequest(&v1pb.GetPlanCheckRunRequest{Name: planA.Name + "/planCheckRun"}))
	a.NoError(err)
	var checkTypes []v1pb.PlanCheckRun_Result_Type
	for _, result := range checkRun.Msg.Results {
		checkTypes = append(checkTypes, result.Type)
	}
	a.Contains(checkTypes, v1pb.PlanCheckRun_Result_STATEMENT_SUMMARY_REPORT)
	a.NotContains(checkTypes, v1pb.PlanCheckRun_Result_STATEMENT_ADVISE)

	brokenModel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error": {"message": "quota exceeded"}}`, http.StatusServiceUnavailable)
	}))
	defer brokenModel.Close()

	// With the AI review switched off, the review skips it: the model is
	// never called, and the run succeeds on the standard rules.
	setting := setAISetting(ctx, t, ctl, brokenModel.URL, false)
	a.False(setting.GetValue().GetAi().GetSqlReviewEnabled())
	_, err = runReview(ctx, ctl, issueA.Name)
	a.NoError(err)
	rowA2 = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	a.Equal(int64(3), rowA2.Attempt)
	a.Equal("DONE", rowA2.Status, "payload %s", rowA2.Payload)

	// With the AI review switched on but the model failing, the whole review
	// fails honestly with the model's error.
	setting = setAISetting(ctx, t, ctl, brokenModel.URL, true)
	a.True(setting.GetValue().GetAi().GetSqlReviewEnabled())

	// The server info follows a switch written outside the server too.
	a.True(actuatorAISQLReviewEnabled(ctx, t, ctl))
	setStoredAISQLReviewEnabled(ctx, t, ctl, false)
	a.False(actuatorAISQLReviewEnabled(ctx, t, ctl))
	setStoredAISQLReviewEnabled(ctx, t, ctl, true)
	a.True(actuatorAISQLReviewEnabled(ctx, t, ctl))

	aiRun, err := runReview(ctx, ctl, issueA.Name)
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_RUNNING, aiRun.State)
	aiRow := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	a.Equal(int64(4), aiRow.Attempt)
	a.Equal("FAILED", aiRow.Status, "payload %s", aiRow.Payload)
	a.Contains(aiRow.Payload, "model call 1 failed")
	failedRun := getReviewRun(ctx, t, ctl, issueA.Name)
	a.Equal(v1pb.ReviewRun_FAILED, failedRun.State)
	a.Equal(int32(code.Code_UNAVAILABLE), failedRun.GetError().GetCode(), "the model answered 503")
	a.Contains(failedRun.GetError().GetMessage(), "model call 1 failed")

	// A table in A's database for the model to read through the tools.
	pgContainer, err := pgContainerOf(fixture.Instance)
	a.NoError(err)
	_, databaseAName, err := common.GetInstanceDatabaseID(fixture.DatabaseA.Name)
	a.NoError(err)
	databaseA, err := sql.Open("pgx", fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s?sslmode=disable", pgContainer.GetHost(), pgContainer.GetPort(), databaseAName))
	a.NoError(err)
	defer databaseA.Close()
	_, err = databaseA.ExecContext(ctx, "CREATE TABLE review_target (id bigint PRIMARY KEY, note text)")
	a.NoError(err)
	_, err = ctl.databaseServiceClient.SyncDatabase(ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{Name: fixture.DatabaseA.Name}))
	a.NoError(err)

	// With a model that reads the table and then answers, the run completes
	// and posts the finding as an open thread on the sheet, listing the
	// database it applies to.
	var toolResults sync.Map
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Contents []struct {
				Parts []struct {
					FunctionResponse *struct {
						Name     string          `json:"name"`
						Response json.RawMessage `json:"response"`
					} `json:"functionResponse"`
				} `json:"parts"`
			} `json:"contents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		reply := `{"candidates": [{"content": {"parts": [{"functionCall": {"name": "read", "args": {"objects": [{"schema": "public", "name": "review_target"}]}}}]}}], "usageMetadata": {"totalTokenCount": 10}}`
		for _, content := range request.Contents {
			for _, part := range content.Parts {
				if part.FunctionResponse != nil {
					toolResults.Store(part.FunctionResponse.Name, string(part.FunctionResponse.Response))
					reply = `{"candidates": [{"content": {"parts": [{"text": "{\"findings\": [{\"title\": \"Remove the placeholder statement\", \"severity\": \"P2\", \"line\": 1, \"rule\": \"Anything the policy below asks for.\", \"evidence\": \"SELECT 1 changes nothing.\", \"fix\": \"Delete the statement.\"}], \"notes\": []}"}]}}], "usageMetadata": {"totalTokenCount": 10}}`
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	defer model.Close()
	setAISetting(ctx, t, ctl, model.URL, true)
	aiRun, err = runReview(ctx, ctl, issueA.Name)
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_RUNNING, aiRun.State)
	aiRow = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	a.Equal("DONE", aiRow.Status, "payload %s", aiRow.Payload)
	a.Equal(int64(5), aiRow.Attempt)
	readResult, ok := toolResults.Load("read")
	a.True(ok, "the model read the table")
	a.Contains(readResult, `CREATE TABLE \"public\".\"review_target\"`, "the tools read the synced schema of the database under review")
	a.Contains(readResult, `\"note\" text`)
	comments, err := ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: issueA.Name}))
	a.NoError(err)
	var results []*v1pb.IssueComment
	for _, comment := range comments.Msg.IssueComments {
		// A result that no standard rule names is the AI review's.
		if metadata := comment.GetReviewMetadata(); metadata != nil && metadata.GetRuleType() == v1pb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED && !metadata.GetSuperseded() {
			results = append(results, comment)
		}
	}
	a.Len(results, 1, "one AI review result")
	result := results[0]
	a.Equal("Remove the placeholder statement", result.GetReviewMetadata().GetTitle())
	a.Equal("SELECT 1 changes nothing.\n\nRule: Anything the policy below asks for.\n\nSuggested fix: Delete the statement.", result.Comment)
	a.Equal(v1pb.IssueComment_ReviewMetadata_P2, result.GetReviewMetadata().GetPriority())
	a.Len(result.GetReviewMetadata().GetTargets(), 1)
	a.True(strings.HasSuffix(result.GetReviewMetadata().GetTargets()[0], fixture.DatabaseA.Name), "target %q", result.GetReviewMetadata().GetTargets()[0])
	a.Equal(int32(1), result.GetStatementAnchor().GetStartPosition().GetLine())
	a.Equal(v1pb.IssueComment_OPEN, result.GetThreadState())

	// Switching the AI review off stops it in flight at its next model call,
	// and the run goes on without it. This model answers its first call with
	// a tool call, which asks for a second call, and the AI review is
	// switched off in between.
	var modelCalls atomic.Int32
	switchedOff := make(chan error, 1)
	switchingModel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if modelCalls.Add(1) == 1 {
			_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
				Setting: &v1pb.Setting{
					Name:  "settings/" + v1pb.Setting_AI.String(),
					Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Ai{Ai: &v1pb.AISetting{}}},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value.ai.sql_review_enabled"}},
			}))
			switchedOff <- err
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates": [{"content": {"parts": [{"functionCall": {"name": "read", "args": {}}}]}}], "usageMetadata": {"totalTokenCount": 10}}`))
	}))
	defer switchingModel.Close()
	setAISetting(ctx, t, ctl, switchingModel.URL, true)
	_, err = runReview(ctx, ctl, issueA.Name)
	a.NoError(err)
	stoppedRow := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID)
	select {
	case err := <-switchedOff:
		a.NoError(err)
	default:
		a.Fail("the model was never called")
	}
	a.Equal("DONE", stoppedRow.Status, "payload %s", stoppedRow.Payload)
	a.Equal(int32(1), modelCalls.Load(), "no call reaches the model after the AI review is switched off")
	// A run without the AI review no longer reports its result.
	comments, err = ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: issueA.Name}))
	a.NoError(err)
	for _, comment := range comments.Msg.IssueComments {
		if comment.Name == result.Name {
			a.True(comment.GetReviewMetadata().GetSuperseded())
			a.Equal(v1pb.IssueComment_RESOLVED, comment.GetThreadState())
		}
	}
}

// setAISetting enables AI in the test workspace with a Gemini model served at
// endpoint, and switches the AI SQL review on or off.
func setAISetting(ctx context.Context, t *testing.T, ctl *controller, endpoint string, sqlReviewEnabled bool) *v1pb.Setting {
	t.Helper()
	resp, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_AI.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Ai{
					Ai: &v1pb.AISetting{
						Enabled:          true,
						Provider:         v1pb.AISetting_GEMINI,
						Endpoint:         endpoint,
						ApiKey:           "unused",
						Model:            "unused",
						SqlReviewEnabled: sqlReviewEnabled,
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"value.ai.enabled", "value.ai.provider", "value.ai.endpoint", "value.ai.api_key", "value.ai.model", "value.ai.sql_review_enabled"},
		},
	}))
	require.NoError(t, err)
	return resp.Msg
}

func actuatorAISQLReviewEnabled(ctx context.Context, t *testing.T, ctl *controller) bool {
	t.Helper()
	info, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	return info.Msg.AiSqlReviewEnabled
}

// setStoredAISQLReviewEnabled writes the switch straight into the metadata
// database, past the server and its setting cache.
func setStoredAISQLReviewEnabled(ctx context.Context, t *testing.T, ctl *controller, enabled bool) {
	t.Helper()
	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{Name: "workspaces/-"}))
	require.NoError(t, err)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	require.NoError(t, err)
	defer db.Close()
	result, err := db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{sqlReviewEnabled}', to_jsonb($2::boolean))
		WHERE workspace = $1 AND name = 'AI'
	`, strings.TrimPrefix(workspace.Msg.Name, "workspaces/"), enabled)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
}

// TestRuleReviewKeepsThreadsAcrossRuns drives the review's standard rules through a re-run
// and two SQL edits: a finding reported again keeps its thread, a person's
// resolution included, and is not posted twice; a finding the SQL no longer
// has is superseded; an edited statement is judged anew.
func TestRuleReviewKeepsThreadsAcrossRuns(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	ctl, ctx := startWorkspace(ctx, t, func(p *component.Profile) { p.SQLReviewV2 = true })
	database := ctl.createTestPostgreSQLDatabase(ctx, t)

	createSheet := func(statement string) string {
		sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte(statement)},
		}))
		a.NoError(err)
		return sheet.Msg.Name
	}
	specID := uuid.NewString()
	spec := func(sheet string) []*v1pb.Plan_Spec {
		return []*v1pb.Plan_Spec{{
			Id: specID,
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{Targets: []string{database.Name}, Sheet: sheet},
			},
		}}
	}
	plan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan:   &v1pb.Plan{Specs: spec(createSheet("DROP TABLE IF EXISTS keep_a;\nDROP TABLE IF EXISTS keep_b;"))},
	}))
	a.NoError(err)
	issue, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue:  &v1pb.Issue{Title: "Review across runs", Type: v1pb.Issue_DATABASE_CHANGE, Plan: plan.Msg.Name},
	}))
	a.NoError(err)
	projectID, issueUID, err := common.GetProjectIDIssueUID(issue.Msg.Name)
	a.NoError(err)

	waitAttempt := func(attempt int64) {
		t.Helper()
		a.Eventually(func() bool {
			for _, r := range listReviewRuns(ctx, t, ctl, projectID) {
				if r.IssueID == issueUID && r.Attempt == attempt && (r.Status == "DONE" || r.Status == "FAILED") {
					a.Equal("DONE", r.Status, "payload %s", r.Payload)
					return true
				}
			}
			return false
		}, 90*time.Second, 500*time.Millisecond, "review attempt %d should finish", attempt)
	}
	// The DROP findings, by thread name.
	drops := func() map[string]*v1pb.IssueComment {
		t.Helper()
		resp, err := ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: issue.Msg.Name}))
		a.NoError(err)
		out := make(map[string]*v1pb.IssueComment)
		for _, comment := range resp.Msg.IssueComments {
			if comment.GetReviewMetadata().GetRuleType() == v1pb.ReviewRuleType_DISALLOW_DROP_OBJECT {
				out[comment.Name] = comment
			}
		}
		return out
	}
	// The review's latest completion event.
	lastCompletion := func() *v1pb.IssueComment_ReviewRunCompletion {
		t.Helper()
		resp, err := ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: issue.Msg.Name}))
		a.NoError(err)
		var last *v1pb.IssueComment_ReviewRunCompletion
		for _, comment := range resp.Msg.IssueComments {
			if completion := comment.GetReviewRunCompletion(); completion != nil {
				a.Empty(comment.Creator)
				last = completion
			}
		}
		a.NotNil(last)
		return last
	}
	counts := func(c *v1pb.IssueComment_ReviewRunCompletion) []int64 {
		return []int64{c.GetAttempt(), int64(c.GetPostedCount()), int64(c.GetKeptCount()), int64(c.GetSupersededCount())}
	}
	about := func(threads map[string]*v1pb.IssueComment, table string) []*v1pb.IssueComment {
		var out []*v1pb.IssueComment
		for _, thread := range threads {
			if strings.Contains(thread.Comment, table) {
				out = append(out, thread)
			}
		}
		return out
	}
	updateSQL := func(statement string) string {
		t.Helper()
		sheet := createSheet(statement)
		_, err := ctl.planServiceClient.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
			Plan:       &v1pb.Plan{Name: plan.Msg.Name, Specs: spec(sheet)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
		}))
		a.NoError(err)
		return sheet
	}

	waitAttempt(0)
	a.Equal([]int64{0, 2, 0, 0}, counts(lastCompletion()), "attempt, posted, kept, superseded")
	first := drops()
	a.Len(first, 2, "%v", first)
	a.Len(about(first, "keep_a"), 1)
	a.Len(about(first, "keep_b"), 1)
	threadA, threadB := about(first, "keep_a")[0], about(first, "keep_b")[0]
	_, err = ctl.issueServiceClient.UpdateIssueComment(ctx, connect.NewRequest(&v1pb.UpdateIssueCommentRequest{
		Parent:       issue.Msg.Name,
		IssueComment: &v1pb.IssueComment{Name: threadB.Name, ThreadState: new(v1pb.IssueComment_RESOLVED)},
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"thread_state"}},
	}))
	a.NoError(err)

	// A re-run on the same SQL posts nothing and keeps the person's resolution.
	_, err = runReview(ctx, ctl, issue.Msg.Name)
	a.NoError(err)
	waitAttempt(1)
	a.Equal([]int64{1, 0, 2, 0}, counts(lastCompletion()))
	second := drops()
	a.Len(second, 2, "%v", second)
	a.Equal(v1pb.IssueComment_OPEN, second[threadA.Name].GetThreadState())
	a.Equal(v1pb.IssueComment_RESOLVED, second[threadB.Name].GetThreadState())
	a.False(second[threadA.Name].GetReviewMetadata().GetSuperseded())
	a.False(second[threadB.Name].GetReviewMetadata().GetSuperseded())

	// Dropping keep_a and moving keep_b down a line supersedes A and keeps B,
	// which moves with its statement.
	moved := updateSQL("SELECT 1;\nDROP TABLE IF EXISTS keep_b;")
	waitAttempt(2)
	a.Equal([]int64{2, 0, 1, 1}, counts(lastCompletion()))
	third := drops()
	a.Len(third, 2, "%v", third)
	a.Equal(v1pb.IssueComment_RESOLVED, third[threadA.Name].GetThreadState())
	a.True(third[threadA.Name].GetReviewMetadata().GetSuperseded())
	a.Equal(v1pb.IssueComment_RESOLVED, third[threadB.Name].GetThreadState())
	a.False(third[threadB.Name].GetReviewMetadata().GetSuperseded())
	a.True(strings.HasSuffix(moved, "/"+third[threadB.Name].GetStatementAnchor().GetSheetSha256()))
	a.Equal(int32(2), third[threadB.Name].GetStatementAnchor().GetStartPosition().GetLine())

	// Editing B's statement judges it anew.
	_ = updateSQL("SELECT 1;\nDROP TABLE IF EXISTS keep_b CASCADE;")
	waitAttempt(3)
	a.Equal([]int64{3, 1, 0, 1}, counts(lastCompletion()))
	fourth := drops()
	a.Len(fourth, 3, "%v", fourth)
	a.True(fourth[threadB.Name].GetReviewMetadata().GetSuperseded())
	var current []*v1pb.IssueComment
	for _, thread := range about(fourth, "keep_b") {
		if !thread.GetReviewMetadata().GetSuperseded() {
			current = append(current, thread)
		}
	}
	a.Len(current, 1)
	a.NotEqual(threadB.Name, current[0].Name)
	a.Equal(v1pb.IssueComment_OPEN, current[0].GetThreadState())
	a.Equal(int64(3), current[0].GetReviewMetadata().GetAttempt())
	a.Equal(int64(0), fourth[threadB.Name].GetReviewMetadata().GetAttempt())
}
