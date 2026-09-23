package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
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
	ctl, ctx := startWorkspace(ctx, t)

	fixture := setupCollidingProjects(ctx, t, ctl)

	// The fixture issues carry rollouts; a rollout freezes the plan's SQL, so
	// review is refused.
	_, err := runReview(ctx, ctl, fixture.IssueA.Name, "rule")
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	// Fresh colliding plan+issue pairs without rollouts.
	_, issueA := createPlanAndIssue(ctx, t, ctl, fixture.ProjectA, fixture.DatabaseA, "Review lifecycle A")
	_, issueB := createPlanAndIssue(ctx, t, ctl, fixture.ProjectB, fixture.DatabaseB, "Review lifecycle B")
	projectAID, issueAUID, err := common.GetProjectIDIssueUID(issueA.Name)
	a.NoError(err)
	projectBID, issueBUID, err := common.GetProjectIDIssueUID(issueB.Name)
	a.NoError(err)
	a.Equal(issueAUID, issueBUID, "fixture issue ids should collide")

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
	// the real test databases through omni's SQL Review V2 contract; until
	// omni ships this engine's review package, the run fails honestly rather
	// than reporting a vacuous DONE. Flip these to DONE when it registers.
	rowA := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal("FAILED", rowA.Status, "rule review on project A, got payload %s", rowA.Payload)
	a.Contains(rowA.Payload, "has no standard rule reviewer")
	a.Equal(int64(0), rowA.Attempt)
	rowB := waitReviewRunTerminal(ctx, t, ctl, projectBID, issueBUID, "RULE")
	a.Equal("FAILED", rowB.Status, "rule review on project B, got payload %s", rowB.Payload)
	a.Equal(int64(0), rowB.Attempt)

	// Re-running A bumps only A's attempt; B's colliding slot is untouched.
	beforeB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Greater(len(beforeB), 0, "project B should have review_run rows")
	runA2, err := runReview(ctx, ctl, issueA.Name, "rule")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AVAILABLE, runA2.Status)
	rowA2 := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal(int64(1), rowA2.Attempt)
	a.Equal("FAILED", rowA2.Status)
	afterB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Equal(beforeB, afterB, "project B review_run rows must be untouched by A's re-run")

	// With AI enabled but the model unreachable, the AI slot is created,
	// claimed, and fails honestly with the model's error.
	setAISetting(ctx, t, ctl, "https://ai.invalid")
	aiRun, err := runReview(ctx, ctl, issueA.Name, "ai")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AI, aiRun.Type)
	aiRow := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "AI")
	a.Equal("FAILED", aiRow.Status, "payload %s", aiRow.Payload)
	a.Contains(aiRow.Payload, "model call 1 failed")
	// The failed AI run left A's rule slot alone.
	rowA3 := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal(int64(1), rowA3.Attempt)
	a.Equal("FAILED", rowA3.Status)

	// With a model that answers, the run completes and posts the finding as
	// an open thread on the sheet, named as the AI reviewer's result and
	// listing the database it applies to.
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates": [{"content": {"parts": [{"text": "{\"findings\": [{\"title\": \"Remove the placeholder statement\", \"severity\": \"P2\", \"line\": 1, \"rule\": \"Anything the policy below asks for.\", \"evidence\": \"SELECT 1 changes nothing.\", \"fix\": \"Delete the statement.\"}], \"notes\": []}"}]}}], "usageMetadata": {"totalTokenCount": 10}}`))
	}))
	defer model.Close()
	setAISetting(ctx, t, ctl, model.URL)
	aiRun, err = runReview(ctx, ctl, issueA.Name, "ai")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AVAILABLE, aiRun.Status)
	aiRow = waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "AI")
	a.Equal("DONE", aiRow.Status, "payload %s", aiRow.Payload)
	a.Equal(int64(1), aiRow.Attempt)
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
	a.Equal("Remove the placeholder statement\n\nSELECT 1 changes nothing.\n\nRule: Anything the policy below asks for.\n\nSuggested fix: Delete the statement.", result.Comment)
	a.Equal(v1pb.IssueComment_ReviewMetadata_P2, result.GetReviewMetadata().GetPriority())
	a.Len(result.GetReviewMetadata().GetTargets(), 1)
	a.True(strings.HasSuffix(result.GetReviewMetadata().GetTargets()[0], fixture.DatabaseA.Name), "target %q", result.GetReviewMetadata().GetTargets()[0])
	a.Equal(int32(1), result.GetStatementAnchor().GetStartPosition().GetLine())
	a.Equal(v1pb.IssueComment_OPEN, result.GetThreadState())
}

// setAISetting enables AI in the test workspace with a Gemini model served at
// endpoint.
func setAISetting(ctx context.Context, t *testing.T, ctl *controller, endpoint string) {
	t.Helper()
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_AI.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Ai{
					Ai: &v1pb.AISetting{
						Enabled:  true,
						Provider: v1pb.AISetting_GEMINI,
						Endpoint: endpoint,
						ApiKey:   "unused",
						Model:    "unused",
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"value.ai.enabled", "value.ai.provider", "value.ai.endpoint", "value.ai.api_key", "value.ai.model"},
		},
	}))
	require.NoError(t, err)
}
