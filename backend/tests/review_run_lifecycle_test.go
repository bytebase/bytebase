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
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	component "github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// waitReviewRunTerminal polls the review_run slot raw until it reaches a
// terminal status and returns the row.
func waitReviewRunTerminal(ctx context.Context, t *testing.T, ctl *controller, projectID string, issueUID int64, reviewType string) *reviewRun {
	t.Helper()
	a := require.New(t)
	var row *reviewRun
	a.Eventually(func() bool {
		for _, r := range listReviewRuns(ctx, t, ctl, projectID) {
			if r.IssueID == issueUID && r.Type == reviewType && (r.Status == "DONE" || r.Status == "FAILED") {
				row = r
				return true
			}
		}
		return false
	}, 90*time.Second, 500*time.Millisecond,
		"review run (%s, %d, %s) should reach a terminal status", projectID, issueUID, reviewType)
	return row
}

func runReview(ctx context.Context, ctl *controller, issueName, reviewRunID string) (*v1pb.ReviewRun, error) {
	resp, err := ctl.issueServiceClient.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
		Name: issueName + "/reviewRuns/" + reviewRunID,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// TestCollision_RunReviewLifecycle drives RunReview end to end on two
// projects whose issue ids collide: the scheduler claims and executes the
// rule review for real, completions land on the right project, and a re-run
// on one project leaves the other project's slot untouched.
func TestCollision_RunReviewLifecycle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	ctl, ctx := startWorkspace(ctx, t, func(p *component.Profile) { p.SQLReviewV2 = true })

	fixture := setupCollidingProjects(ctx, t, ctl)

	// The fixture issues carry rollouts; a rollout freezes the plan's SQL, so
	// review is refused.
	_, err := runReview(ctx, ctl, fixture.IssueA.Name, "rule")
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	// Fresh colliding plan+issue pairs without rollouts.
	planA, issueA := createPlanAndIssue(ctx, t, ctl, fixture.ProjectA, fixture.DatabaseA, "Review lifecycle A")
	_, issueB := createPlanAndIssue(ctx, t, ctl, fixture.ProjectB, fixture.DatabaseB, "Review lifecycle B")
	projectAID, issueAUID, err := common.GetProjectIDIssueUID(issueA.Name)
	a.NoError(err)
	projectBID, issueBUID, err := common.GetProjectIDIssueUID(issueB.Name)
	a.NoError(err)
	a.Equal(issueAUID, issueBUID, "fixture issue ids should collide")

	// Creating an issue starts its rule review.
	rowA := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal("DONE", rowA.Status, "rule review on project A, got payload %s", rowA.Payload)
	a.Equal(int64(0), rowA.Attempt)
	rowB := waitReviewRunTerminal(ctx, t, ctl, projectBID, issueBUID, "RULE")
	a.Equal("DONE", rowB.Status, "rule review on project B, got payload %s", rowB.Payload)
	a.Equal(int64(0), rowB.Attempt)

	// Unknown reviewer type.
	_, err = runReview(ctx, ctl, issueA.Name, "bogus")
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// AI review requires AI to be enabled, which is off in the test workspace.
	_, err = runReview(ctx, ctl, issueA.Name, "ai")
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	// Rule review on both colliding issues.
	runA, err := runReview(ctx, ctl, issueA.Name, "rule")
	a.NoError(err)
	a.Equal(issueA.Name+"/reviewRuns/rule", runA.Name)
	a.Equal(v1pb.ReviewRun_RULE, runA.Type)
	a.Equal(v1pb.ReviewRun_AVAILABLE, runA.Status)
	a.NotNil(runA.CreateTime)
	a.Nil(runA.EndTime)
	a.Empty(runA.Error)
	runB, err := runReview(ctx, ctl, issueB.Name, "rule")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AVAILABLE, runB.Status)

	// The scheduler claims and executes both. The rule executor evaluates
	// the real test databases through omni's SQL Review V2 contract.
	rowA = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal("DONE", rowA.Status, "rule review on project A, got payload %s", rowA.Payload)
	a.Equal(int64(1), rowA.Attempt)
	rowB = waitReviewRunTerminal(ctx, t, ctl, projectBID, issueBUID, "RULE")
	a.Equal("DONE", rowB.Status, "rule review on project B, got payload %s", rowB.Payload)
	a.Equal(int64(1), rowB.Attempt)

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
	runA2, err := runReview(ctx, ctl, issueA.Name, "rule")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AVAILABLE, runA2.Status)
	rowA2 := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal(int64(2), rowA2.Attempt)
	a.Equal("DONE", rowA2.Status)
	afterB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Equal(beforeB, afterB, "project B review_run rows must be untouched by A's re-run")

	// Running the plan checks re-runs the rule review, and the checks leave
	// the SQL review to it.
	_, err = ctl.planServiceClient.RunPlanChecks(ctx, connect.NewRequest(&v1pb.RunPlanChecksRequest{Name: planA.Name}))
	a.NoError(err)
	rowA2 = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal(int64(3), rowA2.Attempt)
	a.Equal("DONE", rowA2.Status)
	waitPlanCheckRunSettled(ctx, t, ctl, planA.Name)
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

	setting := setAISetting(ctx, t, ctl, brokenModel.URL, false)
	a.False(setting.GetValue().GetAi().GetSqlReviewEnabled())
	_, err = runReview(ctx, ctl, issueA.Name, "ai")
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	a.ErrorContains(err, "AI SQL review is not enabled")

	// With the review switched on but the model failing, the AI slot is
	// created, claimed, and fails honestly with the model's error.
	setting = setAISetting(ctx, t, ctl, brokenModel.URL, true)
	a.True(setting.GetValue().GetAi().GetSqlReviewEnabled())

	// The server info follows a switch written outside the server too.
	a.True(actuatorAISQLReviewEnabled(ctx, t, ctl))
	setStoredAISQLReviewEnabled(ctx, t, ctl, false)
	a.False(actuatorAISQLReviewEnabled(ctx, t, ctl))
	setStoredAISQLReviewEnabled(ctx, t, ctl, true)
	a.True(actuatorAISQLReviewEnabled(ctx, t, ctl))

	aiRun, err := runReview(ctx, ctl, issueA.Name, "ai")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AI, aiRun.Type)
	aiRow := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "AI")
	a.Equal("FAILED", aiRow.Status, "payload %s", aiRow.Payload)
	a.Contains(aiRow.Payload, "model call 1 failed")
	// The failed AI run left A's rule slot alone.
	rowA3 := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal(int64(3), rowA3.Attempt)
	a.Equal("DONE", rowA3.Status)

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
	// and posts the finding as an open thread on the sheet, named as the AI
	// reviewer's result and listing the database it applies to.
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
					reply = `{"candidates": [{"content": {"parts": [{"text": "{\"findings\": [{\"title\": \"Remove the placeholder statement\", \"severity\": \"P2\", \"line\": 1, \"rule\": \"fails\", \"evidence\": \"SELECT 1 changes nothing.\", \"fix\": \"Delete the statement.\"}], \"notes\": []}"}]}}], "usageMetadata": {"totalTokenCount": 10}}`
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	defer model.Close()
	setAISetting(ctx, t, ctl, model.URL, true)
	aiRun, err = runReview(ctx, ctl, issueA.Name, "ai")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AVAILABLE, aiRun.Status)
	aiRow = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "AI")
	a.Equal("DONE", aiRow.Status, "payload %s", aiRow.Payload)
	a.Equal(int64(1), aiRow.Attempt)
	readResult, ok := toolResults.Load("read")
	a.True(ok, "the model read the table")
	a.Contains(readResult, `CREATE TABLE \"public\".\"review_target\"`, "the tools read the synced schema of the database under review")
	a.Contains(readResult, `\"note\" text`)
	comments, err := ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: issueA.Name}))
	a.NoError(err)
	var results []*v1pb.IssueComment
	for _, comment := range comments.Msg.IssueComments {
		if comment.GetReviewMetadata().GetRunType() == v1pb.ReviewRun_AI {
			results = append(results, comment)
		}
	}
	a.Len(results, 1, "one AI review result")
	result := results[0]
	a.Equal("Remove the placeholder statement\n\nSELECT 1 changes nothing.\n\nRule: The statement fails on this database, for example because an object it creates already exists, an object it uses does not exist, or existing rows violate a constraint or a foreign key.\n\nSuggested fix: Delete the statement.", result.Comment)
	a.Equal(v1pb.IssueComment_ReviewMetadata_P2, result.GetReviewMetadata().GetPriority())
	a.Len(result.GetReviewMetadata().GetTargets(), 1)
	a.True(strings.HasSuffix(result.GetReviewMetadata().GetTargets()[0], fixture.DatabaseA.Name), "target %q", result.GetReviewMetadata().GetTargets()[0])
	a.Equal(int32(1), result.GetStatementAnchor().GetStartPosition().GetLine())
	a.Equal(v1pb.IssueComment_OPEN, result.GetThreadState())

	// Switching the review off stops a run in flight at its next model call.
	// This model answers its first call with a tool call, which asks for a
	// second call, and the review is switched off in between.
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
	_, err = runReview(ctx, ctl, issueA.Name, "ai")
	a.NoError(err)
	stoppedRow := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "AI")
	select {
	case err := <-switchedOff:
		a.NoError(err)
	default:
		a.Fail("the model was never called")
	}
	a.Equal("FAILED", stoppedRow.Status, "payload %s", stoppedRow.Payload)
	a.Contains(stoppedRow.Payload, "AI SQL review is not enabled")
	a.Equal(int32(1), modelCalls.Load(), "no call reaches the model after the review is switched off")
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
