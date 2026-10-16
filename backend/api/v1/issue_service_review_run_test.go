package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestRunReviewCreatesRuleRunWithReviewOff pins that switching every rule off
// does not refuse a rule run: the run completes with no findings, which
// resolves the findings of earlier runs.
func TestRunReviewCreatesRuleRunWithReviewOff(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	service := newIssueServiceForTest(t, stores)

	// REQUIRE_WHERE without SYNTAX is review switched off.
	payload, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: []storepb.ReviewRuleType{storepb.ReviewRuleType_REQUIRE_WHERE}})
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Type:         storepb.Policy_REVIEW_RULE,
		Payload:      string(payload),
		Enforce:      true,
	})
	require.NoError(t, err)

	resp, err := service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
		Name: fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/rule", issue.UID),
	}))
	require.NoError(t, err)
	require.Equal(t, v1pb.ReviewRun_RULE, resp.Msg.Type)
	require.Equal(t, v1pb.ReviewRun_AVAILABLE, resp.Msg.Status)
}

func TestGetReviewRun(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	issue := createReviewRunIssue(ctx, t, stores, "project-a")
	service := newIssueServiceForTest(t, stores)
	ruleName := fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/rule", issue.UID)

	_, err := getReviewRunForTest(ctx, service, ruleName)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a run that never started")

	run, err := service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{Name: ruleName}))
	require.NoError(t, err)
	got, err := getReviewRunForTest(ctx, service, ruleName)
	require.NoError(t, err)
	require.True(t, proto.Equal(run.Msg, got), "got %v, want %v", got, run.Msg)

	claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err := stores.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_FAILED, &storepb.ReviewRunPayload{Error: "boom"}, nil)
	require.NoError(t, err)
	require.True(t, updated)
	got, err = getReviewRunForTest(ctx, service, ruleName)
	require.NoError(t, err)
	require.Equal(t, ruleName, got.Name)
	require.Equal(t, v1pb.ReviewRun_RULE, got.Type)
	require.Equal(t, v1pb.ReviewRun_FAILED, got.Status)
	require.Equal(t, "boom", got.Error)
	require.True(t, proto.Equal(run.Msg.CreateTime, got.CreateTime))
	require.NotNil(t, got.EndTime)

	for _, tc := range []struct {
		description string
		name        string
		wantCode    connect.Code
	}{
		{"slot never created", fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/ai", issue.UID), connect.CodeNotFound},
		{"issue does not exist", "projects/project-a/issues/999/reviewRuns/rule", connect.CodeNotFound},
		{"unknown reviewer", fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/guideline", issue.UID), connect.CodeInvalidArgument},
		{"issue id not a number", "projects/project-a/issues/abc/reviewRuns/rule", connect.CodeInvalidArgument},
	} {
		_, err := getReviewRunForTest(ctx, service, tc.name)
		require.Equal(t, tc.wantCode, connect.CodeOf(err), tc.description)
	}
}

func TestGetReviewRunCollidingIssues(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, err := stores.GetDB().ExecContext(ctx, `INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', 'default', 'Project B')`)
	require.NoError(t, err)
	issueA := createReviewRunIssue(ctx, t, stores, "project-a")
	issueB := createReviewRunIssue(ctx, t, stores, "project-b")
	require.Equal(t, issueA.UID, issueB.UID, "the issue ids must collide")
	service := newIssueServiceForTest(t, stores)
	nameA := fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/rule", issueA.UID)
	nameB := fmt.Sprintf("projects/project-b/issues/%d/reviewRuns/rule", issueB.UID)

	runB, err := service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{Name: nameB}))
	require.NoError(t, err)
	_, err = getReviewRunForTest(ctx, service, nameA)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "project B's run is not project A's")

	// A fails while B stays RUNNING.
	_, err = service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{Name: nameA}))
	require.NoError(t, err)
	claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 2)
	for _, c := range claimed {
		if c.ProjectID != "project-a" {
			continue
		}
		updated, err := stores.CompleteReviewRun(ctx, c, "replica-1", storepb.ReviewRun_FAILED, &storepb.ReviewRunPayload{Error: "boom"}, nil)
		require.NoError(t, err)
		require.True(t, updated)
	}

	gotA, err := getReviewRunForTest(ctx, service, nameA)
	require.NoError(t, err)
	require.Equal(t, nameA, gotA.Name)
	require.Equal(t, v1pb.ReviewRun_FAILED, gotA.Status)
	require.Equal(t, "boom", gotA.Error)
	gotB, err := getReviewRunForTest(ctx, service, nameB)
	require.NoError(t, err)
	require.Equal(t, nameB, gotB.Name)
	require.Equal(t, v1pb.ReviewRun_RUNNING, gotB.Status)
	require.Empty(t, gotB.Error)
	require.Nil(t, gotB.EndTime)
	require.True(t, proto.Equal(runB.Msg.CreateTime, gotB.CreateTime))
}

func getReviewRunForTest(ctx context.Context, service *IssueService, name string) (*v1pb.ReviewRun, error) {
	resp, err := service.GetReviewRun(ctx, connect.NewRequest(&v1pb.GetReviewRunRequest{Name: name}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// createReviewRunIssue creates an open issue whose plan has one reviewable
// spec.
func createReviewRunIssue(ctx context.Context, t *testing.T, stores *store.Store, projectID string) *store.IssueMessage {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: projectID,
		Name:      "plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Id: "spec-1",
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							Targets: []string{"instances/prod/databases/app"},
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    projectID,
		CreatorEmail: "creator@example.com",
		Title:        "issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)
	return issue
}
