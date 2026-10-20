package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
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

func TestRunReviewRefusesAIReviewWhileItsSwitchIsOff(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	service := newIssueServiceForTest(t, stores)

	setAISetting := func(setting *storepb.AISetting) {
		t.Helper()
		_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
			Name:      storepb.SettingName_AI,
			Workspace: "default",
			Value:     setting,
		})
		require.NoError(t, err)
	}
	runAIReview := func() (*connect.Response[v1pb.ReviewRun], error) {
		return service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
			Name: fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/ai", issue.UID),
		}))
	}

	setAISetting(&storepb.AISetting{Enabled: true})
	_, err := runAIReview()
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorContains(t, err, "AI SQL review is not enabled")
	claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
	require.NoError(t, err)
	require.Empty(t, claimed, "a refused review creates no run")

	setAISetting(&storepb.AISetting{Enabled: true, SqlReviewEnabled: true})
	resp, err := runAIReview()
	require.NoError(t, err)
	require.Equal(t, v1pb.ReviewRun_AI, resp.Msg.Type)
	require.Equal(t, v1pb.ReviewRun_AVAILABLE, resp.Msg.Status)

	setAISetting(&storepb.AISetting{SqlReviewEnabled: true})
	_, err = runAIReview()
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorContains(t, err, "AI is not enabled")
}

func TestRunReviewRefusesRuleReviewWithoutSQLReviewV2(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	service := newIssueServiceForTest(t, stores)
	service.profile = &config.Profile{}
	runReview := func(reviewRunID string) (*connect.Response[v1pb.ReviewRun], error) {
		return service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
			Name: fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/%s", issue.UID, reviewRunID),
		}))
	}

	_, err := runReview("rule")
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
	claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
	require.NoError(t, err)
	require.Empty(t, claimed, "a refused review creates no run")

	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_AI,
		Workspace: "default",
		Value:     &storepb.AISetting{Enabled: true, SqlReviewEnabled: true},
	})
	require.NoError(t, err)
	resp, err := runReview("ai")
	require.NoError(t, err, "the AI review does not depend on SQL Review V2")
	require.Equal(t, v1pb.ReviewRun_AVAILABLE, resp.Msg.Status)
}

func TestStartRuleReview(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	claimRuleRuns := func() []*store.ClaimedReviewRun {
		t.Helper()
		claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
		require.NoError(t, err)
		return claimed
	}

	issue := createReviewRunIssue(ctx, t, stores, "project-a")
	startRuleReview(ctx, &config.Profile{}, stores, b, issue)
	require.Empty(t, claimRuleRuns(), "SQL Review V2 is off")

	startRuleReview(ctx, &config.Profile{SQLReviewV2: true}, stores, b, issue)
	claimed := claimRuleRuns()
	require.Len(t, claimed, 1)
	require.Equal(t, issue.UID, claimed[0].IssueUID)
	require.Equal(t, store.ReviewRunTypeRule, claimed[0].Type)
	require.Len(t, b.ReviewRunTickleChan, 1)

	issue.Status = storepb.Issue_DONE
	startRuleReview(ctx, &config.Profile{SQLReviewV2: true}, stores, b, issue)
	require.Empty(t, claimRuleRuns(), "a closed issue is skipped")
}

func TestResolvingAThreadRequestsARolloutCheck(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	issue := createReviewRunIssue(ctx, t, stores, "project-a")
	open := store.ThreadStateOpen
	root, err := stores.CreateIssueComments(ctx, "creator@example.com", &store.IssueCommentMessage{
		ProjectID:   "project-a",
		IssueUID:    issue.UID,
		Payload:     &storepb.IssueCommentPayload{Comment: "Is this safe?"},
		ThreadState: &open,
	})
	require.NoError(t, err)
	parent := fmt.Sprintf("projects/project-a/issues/%d", issue.UID)
	setThreadState := func(service *IssueService, state v1pb.IssueComment_ThreadState) {
		t.Helper()
		_, err := service.UpdateIssueComment(ctx, connect.NewRequest(&v1pb.UpdateIssueCommentRequest{
			Parent: parent,
			IssueComment: &v1pb.IssueComment{
				Name:        fmt.Sprintf("%s/issueComments/%s", parent, root.ResourceID),
				ThreadState: &state,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"thread_state"}},
		}))
		require.NoError(t, err)
	}

	withoutV2 := newIssueServiceForTest(t, stores)
	withoutV2.profile = &config.Profile{}
	setThreadState(withoutV2, v1pb.IssueComment_RESOLVED)
	require.Empty(t, withoutV2.bus.RolloutCreationChan)

	service := newIssueServiceForTest(t, stores)
	setThreadState(service, v1pb.IssueComment_OPEN)
	require.Empty(t, service.bus.RolloutCreationChan, "reopening holds the rollout; it releases nothing")
	setThreadState(service, v1pb.IssueComment_RESOLVED)
	require.Len(t, service.bus.RolloutCreationChan, 1)
	require.Equal(t, bus.PlanRef{ProjectID: "project-a", PlanID: *issue.PlanUID}, <-service.bus.RolloutCreationChan)
}
