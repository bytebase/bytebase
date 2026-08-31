package tests

import (
	"context"
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
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	// The fixture issues carry rollouts; a rollout freezes the plan's SQL, so
	// review is refused.
	_, err = runReview(ctx, ctl, fixture.IssueA.Name, "rule")
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

	// Guideline review requires AI, which is off in the test workspace.
	_, err = runReview(ctx, ctl, issueA.Name, "guideline")
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

	// The scheduler claims and executes both; the rule engine evaluates the
	// real test databases and the runs reach DONE.
	rowA := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal("DONE", rowA.Status, "rule review on project A should succeed, got payload %s", rowA.Payload)
	a.Equal(int64(0), rowA.Attempt)
	rowB := waitReviewRunTerminal(ctx, t, ctl, projectBID, issueBUID, "RULE")
	a.Equal("DONE", rowB.Status, "rule review on project B should succeed, got payload %s", rowB.Payload)
	a.Equal(int64(0), rowB.Attempt)

	// Re-running A bumps only A's attempt; B's colliding slot is untouched.
	beforeB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Greater(len(beforeB), 0, "project B should have review_run rows")
	runA2, err := runReview(ctx, ctl, issueA.Name, "rule")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_AVAILABLE, runA2.Status)
	rowA2 := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal(int64(1), rowA2.Attempt)
	a.Equal("DONE", rowA2.Status)
	afterB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Equal(beforeB, afterB, "project B review_run rows must be untouched by A's re-run")

	// With AI enabled, the guideline slot is created, claimed, and fails
	// honestly: the executor is not implemented yet.
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_AI.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Ai{
					Ai: &v1pb.AISetting{
						Enabled:  true,
						Provider: v1pb.AISetting_GEMINI,
						Endpoint: "https://ai.invalid",
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
	a.NoError(err)
	runG, err := runReview(ctx, ctl, issueA.Name, "guideline")
	a.NoError(err)
	a.Equal(v1pb.ReviewRun_GUIDELINE, runG.Type)
	rowG := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "GUIDELINE")
	a.Equal("FAILED", rowG.Status)
	a.Contains(rowG.Payload, "not implemented")
	// The failed guideline run left A's rule slot alone.
	rowA3 := waitReviewRunTerminal(ctx, t, ctl, projectAID, issueAUID, "RULE")
	a.Equal(int64(1), rowA3.Attempt)
	a.Equal("DONE", rowA3.Status)
}
