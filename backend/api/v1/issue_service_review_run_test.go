package v1

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// reviewRunName is the name of an issue's review run.
func reviewRunName(projectID string, issueUID int64) string {
	return fmt.Sprintf("projects/%s/issues/%d/reviewRun", projectID, issueUID)
}

// TestRunReviewStartsWithEveryRuleOff pins that switching every rule off does
// not refuse a review: the run completes with no findings, which resolves the
// findings of earlier runs.
func TestRunReviewStartsWithEveryRuleOff(t *testing.T) {
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
		Name: reviewRunName("project-a", issue.UID),
	}))
	require.NoError(t, err)
	require.Equal(t, v1pb.ReviewRun_RUNNING, resp.Msg.State)
	run, err := stores.GetReviewRun(ctx, "project-a", issue.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status)
}

func TestGetReviewRun(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	issue := createReviewRunIssue(ctx, t, stores, "project-a")
	service := newIssueServiceForTest(t, stores)
	name := reviewRunName("project-a", issue.UID)
	getRun := func() *v1pb.ReviewRun {
		t.Helper()
		run, err := getReviewRunForTest(ctx, service, name)
		require.NoError(t, err)
		return run
	}
	// start starts a run and claims it, as the scheduler would.
	start := func() (*v1pb.ReviewRun, *store.ClaimedReviewRun) {
		t.Helper()
		run, err := service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{Name: name}))
		require.NoError(t, err)
		require.Equal(t, v1pb.ReviewRun_RUNNING, run.Msg.State, "a queued run reads as RUNNING")
		require.True(t, proto.Equal(run.Msg, getRun()), "got %v, want %v", getRun(), run.Msg)
		claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica-1")
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		return run.Msg, claimed[0]
	}
	complete := func(claimed *store.ClaimedReviewRun, status storepb.ReviewRun_Status, payload *storepb.ReviewRunPayload) *v1pb.ReviewRun {
		t.Helper()
		updated, err := stores.CompleteReviewRun(ctx, claimed, "replica-1", status, payload, nil)
		require.NoError(t, err)
		require.True(t, updated)
		return getRun()
	}

	_, err := getReviewRunForTest(ctx, service, name)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a run that never started")

	started, claimed := start()
	got := getRun()
	require.Equal(t, name, got.Name)
	require.Equal(t, v1pb.ReviewRun_RUNNING, got.State, "a claimed run reads as RUNNING")
	require.Nil(t, got.Error)
	require.Nil(t, got.EndTime)

	got = complete(claimed, storepb.ReviewRun_FAILED, &storepb.ReviewRunPayload{Error: "boom"})
	require.Equal(t, v1pb.ReviewRun_FAILED, got.State)
	require.Equal(t, int32(code.Code_UNKNOWN), got.GetError().GetCode(), "a failure stored with no code")
	require.Equal(t, "boom", got.GetError().GetMessage())
	require.True(t, proto.Equal(started.CreateTime, got.CreateTime))
	require.NotNil(t, got.EndTime)

	_, claimed = start()
	got = complete(claimed, storepb.ReviewRun_FAILED, &storepb.ReviewRunPayload{Error: "rate limited", Code: code.Code_RESOURCE_EXHAUSTED})
	require.Equal(t, v1pb.ReviewRun_FAILED, got.State)
	require.Equal(t, int32(code.Code_RESOURCE_EXHAUSTED), got.GetError().GetCode())
	require.Equal(t, "rate limited", got.GetError().GetMessage())

	_, claimed = start()
	got = complete(claimed, storepb.ReviewRun_DONE, &storepb.ReviewRunPayload{})
	require.Equal(t, v1pb.ReviewRun_SUCCEEDED, got.State)
	require.Nil(t, got.Error)
	require.NotNil(t, got.EndTime)

	for _, tc := range []struct {
		description string
		name        string
		wantCode    connect.Code
	}{
		{"issue does not exist", reviewRunName("project-a", 999), connect.CodeNotFound},
		{"a name that names a reviewer", fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/rule", issue.UID), connect.CodeInvalidArgument},
		{"issue id not a number", "projects/project-a/issues/abc/reviewRun", connect.CodeInvalidArgument},
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
	nameA := reviewRunName("project-a", issueA.UID)
	nameB := reviewRunName("project-b", issueB.UID)

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
	require.Equal(t, v1pb.ReviewRun_FAILED, gotA.State)
	require.Equal(t, "boom", gotA.GetError().GetMessage())
	gotB, err := getReviewRunForTest(ctx, service, nameB)
	require.NoError(t, err)
	require.Equal(t, nameB, gotB.Name)
	require.Equal(t, v1pb.ReviewRun_RUNNING, gotB.State)
	require.Nil(t, gotB.Error)
	require.Nil(t, gotB.EndTime)
	require.True(t, proto.Equal(runB.Msg.CreateTime, gotB.CreateTime))
}

// TestRunReviewStartsOneRunWhateverTheAISetting pins that under SQL Review V2
// the AI setting decides nothing at the start: the review is one run, and
// whether the AI review takes part is the executor's to read.
func TestRunReviewStartsOneRunWhateverTheAISetting(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	service := newIssueServiceForTest(t, stores)

	for attempt, setting := range []*storepb.AISetting{
		nil,
		{Enabled: true},
		{Enabled: true, SqlReviewEnabled: true},
		{SqlReviewEnabled: true},
	} {
		if setting != nil {
			_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
				Name:      storepb.SettingName_AI,
				Workspace: "default",
				Value:     setting,
			})
			require.NoError(t, err)
		}
		resp, err := service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
			Name: reviewRunName("project-a", issue.UID),
		}))
		require.NoError(t, err, "setting %v", setting)
		require.Equal(t, v1pb.ReviewRun_RUNNING, resp.Msg.State)
		claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
		require.NoError(t, err)
		require.Len(t, claimed, 1, "setting %v", setting)
		require.Equal(t, int64(attempt), claimed[0].Attempt, "every start resets the same slot")
	}
	var slots int
	require.NoError(t, stores.GetDB().QueryRowContext(ctx, `
		SELECT count(*) FROM review_run WHERE project = 'project-a' AND issue_id = $1
	`, issue.UID).Scan(&slots))
	require.Equal(t, 1, slots)
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

// TestRunReviewWithoutSQLReviewV2 pins a server without SQL Review V2: no
// standard rule review runs there, so a review is refused unless the AI review
// is on.
func TestRunReviewWithoutSQLReviewV2(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	service := newIssueServiceForTest(t, stores)
	service.profile = &config.Profile{}
	runReview := func() (*connect.Response[v1pb.ReviewRun], error) {
		return service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
			Name: reviewRunName("project-a", issue.UID),
		}))
	}
	setAISetting := func(setting *storepb.AISetting) {
		t.Helper()
		_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
			Name:      storepb.SettingName_AI,
			Workspace: "default",
			Value:     setting,
		})
		require.NoError(t, err)
	}

	_, err := runReview()
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorContains(t, err, "AI is not enabled")
	setAISetting(&storepb.AISetting{Enabled: true})
	_, err = runReview()
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorContains(t, err, "AI SQL review is not enabled")
	run, err := stores.GetReviewRun(ctx, "project-a", issue.UID)
	require.NoError(t, err)
	require.Nil(t, run, "a refused review creates no run")

	setAISetting(&storepb.AISetting{Enabled: true, SqlReviewEnabled: true})
	resp, err := runReview()
	require.NoError(t, err, "the AI review does not depend on SQL Review V2")
	require.Equal(t, v1pb.ReviewRun_RUNNING, resp.Msg.State)
	claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
}

func TestStartReview(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	claimedAttempts := func() []int64 {
		t.Helper()
		claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
		require.NoError(t, err)
		var attempts []int64
		for _, c := range claimed {
			attempts = append(attempts, c.Attempt)
		}
		return attempts
	}
	sqlReviewV2 := &config.Profile{SQLReviewV2: true}

	issue := createReviewRunIssue(ctx, t, stores, "project-a")
	startReview(ctx, &config.Profile{}, stores, b, issue)
	require.Empty(t, claimedAttempts(), "SQL Review V2 is off")
	require.Empty(t, b.ReviewRunTickleChan)

	startReview(ctx, sqlReviewV2, stores, b, issue)
	require.Equal(t, []int64{0}, claimedAttempts())
	require.Len(t, b.ReviewRunTickleChan, 1)

	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_AI,
		Workspace: "default",
		Value:     &storepb.AISetting{Enabled: true, SqlReviewEnabled: true},
	})
	require.NoError(t, err)
	startReview(ctx, sqlReviewV2, stores, b, issue)
	require.Equal(t, []int64{1}, claimedAttempts(), "the review is one run with the AI review on too")

	issue.Status = storepb.Issue_DONE
	startReview(ctx, sqlReviewV2, stores, b, issue)
	require.Empty(t, claimedAttempts(), "a closed issue is skipped")
	startReview(ctx, sqlReviewV2, stores, b, nil)
	require.Empty(t, claimedAttempts(), "a plan with no issue is skipped")
}

// TestReviewStartsWhenTheSQLIsSubmittedOrChanged pins the points the review
// starts on its own: creating the issue and changing the plan's SQL. Rerunning
// the plan checks leaves it alone.
func TestReviewStartsWhenTheSQLIsSubmittedOrChanged(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	projectID := "project-a"
	_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			Activation:  true,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "prod",
		DatabaseName: "app",
		ProjectID:    projectID,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	sheets, err := stores.CreateSheets(ctx, projectID,
		&store.SheetMessage{Statement: "ALTER TABLE t ADD COLUMN a INT;"},
		&store.SheetMessage{Statement: "ALTER TABLE t ADD COLUMN b INT;"},
	)
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_AI,
		Workspace: "default",
		Value:     &storepb.AISetting{Enabled: true, SqlReviewEnabled: true},
	})
	require.NoError(t, err)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: projectID,
		Name:      "plan",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: "spec-1",
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
					Targets:     []string{"instances/prod/databases/app"},
					SheetSha256: sheets[0].Sha256,
				},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)

	issueService := newIssueServiceForTest(t, stores)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	planService := NewPlanService(stores, issueService.bus, iamManager, webhook.NewManager(stores, nil), nil, &config.Profile{SQLReviewV2: true})
	getRun := func(issueUID int64) *store.ReviewRunMessage {
		t.Helper()
		run, err := stores.GetReviewRun(ctx, projectID, issueUID)
		require.NoError(t, err)
		require.NotNil(t, run)
		return run
	}
	claimAll := func() {
		t.Helper()
		_, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
		require.NoError(t, err)
	}

	created, err := issueService.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/" + projectID,
		Issue: &v1pb.Issue{
			Type: v1pb.Issue_DATABASE_CHANGE,
			Plan: common.FormatPlan(projectID, plan.UID),
		},
	}))
	require.NoError(t, err)
	_, issueUID, err := common.GetProjectIDIssueUID(created.Msg.Name)
	require.NoError(t, err)
	run := getRun(issueUID)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status)
	require.Equal(t, int64(0), run.Attempt)
	claimAll()

	_, err = planService.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: common.FormatPlan(projectID, plan.UID),
			Specs: []*v1pb.Plan_Spec{{
				Id: "spec-1",
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{"instances/prod/databases/app"},
						Sheet:   common.FormatSheet(projectID, sheets[1].Sha256),
					},
				},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	require.NoError(t, err)
	updatedPlan, err := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: &plan.UID})
	require.NoError(t, err)
	run = getRun(issueUID)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status, "changing the SQL restarts the review")
	require.Equal(t, int64(1), run.Attempt)
	require.Equal(t, updatedPlan.Config.GetApprovalInputVersion(), run.Payload.GetApprovalInputVersion(), "the run reviews the new SQL")
	claimAll()

	_, err = planService.RunPlanChecks(ctx, connect.NewRequest(&v1pb.RunPlanChecksRequest{
		Name: common.FormatPlan(projectID, plan.UID),
	}))
	require.NoError(t, err)
	run = getRun(issueUID)
	require.Equal(t, storepb.ReviewRun_RUNNING, run.Status, "rerunning the checks leaves the review alone")
	require.Equal(t, int64(1), run.Attempt)
}

func TestGetReviewRunReportsAnOutdatedRun(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	issue := createReviewRunIssue(ctx, t, stores, "project-a")
	_, err := stores.CreateReviewRun(ctx, issue.ProjectID, issue.UID)
	require.NoError(t, err)
	getRun := func() *v1pb.ReviewRun {
		t.Helper()
		resp, err := service.GetReviewRun(ctx, connect.NewRequest(&v1pb.GetReviewRunRequest{
			Name: reviewRunName("project-a", issue.UID),
		}))
		require.NoError(t, err)
		return resp.Msg
	}
	require.False(t, getRun().Outdated)

	_, err = stores.GetDB().ExecContext(ctx, `
		UPDATE plan SET config = jsonb_set(config, '{approvalInputVersion}', '"1"')
		WHERE project = $1 AND id = $2
	`, issue.ProjectID, *issue.PlanUID)
	require.NoError(t, err)
	require.True(t, getRun().Outdated, "the plan's SQL changed after the run started")

	_, err = stores.CreateReviewRun(ctx, issue.ProjectID, issue.UID)
	require.NoError(t, err)
	require.False(t, getRun().Outdated)
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

// TestCreateIssueStartsTheReviewBeforeTheWorkflow pins the order that keeps a
// new issue's rollout held: the workflow's approval check can create the
// rollout, so the review slot must exist by the time it is sent.
func TestCreateIssueStartsTheReviewBeforeTheWorkflow(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "plan",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: "spec-1",
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
					Targets: []string{"instances/prod/databases/app"},
				},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)

	// Unbuffered and not yet received, so the workflow's send blocks: a slot
	// that is not in place before the send cannot appear until the listener
	// takes it.
	approvalCheck := make(chan bus.IssueRef)
	service.bus.ApprovalCheckChan = approvalCheck
	slotBeforeApprovalCheck := make(chan bool, 1)
	go func() {
		defer func() { <-approvalCheck }()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			issue, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, PlanUID: &plan.UID})
			if err == nil && issue != nil {
				run, err := stores.GetReviewRun(ctx, issue.ProjectID, issue.UID)
				if err == nil && run != nil {
					slotBeforeApprovalCheck <- true
					return
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		slotBeforeApprovalCheck <- false
	}()

	_, err = service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type: v1pb.Issue_DATABASE_CHANGE,
			Plan: common.FormatPlan("project-a", plan.UID),
		},
	}))
	require.NoError(t, err)
	require.True(t, <-slotBeforeApprovalCheck, "the review slot exists before the approval check is sent")
}

// TestUpdatePlanStartsTheReviewBeforeTheChecks pins the order that keeps an
// edited plan's rollout held: the plan checks and the approval check can each
// lead to the rollout, so the review slot must be reset first.
func TestUpdatePlanStartsTheReviewBeforeTheChecks(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	projectID := "project-a"
	_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			Activation:  true,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "prod",
		DatabaseName: "app",
		ProjectID:    projectID,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	sheets, err := stores.CreateSheets(ctx, projectID, &store.SheetMessage{Statement: "ALTER TABLE t ADD COLUMN c INT;"})
	require.NoError(t, err)
	issue := createReviewRunIssue(ctx, t, stores, projectID)
	_, err = stores.CreateReviewRun(ctx, issue.ProjectID, issue.UID)
	require.NoError(t, err)
	claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err := stores.CompleteReviewRun(ctx, claimed[0], "replica", storepb.ReviewRun_DONE, &storepb.ReviewRunPayload{}, nil)
	require.NoError(t, err)
	require.True(t, updated)

	b, err := bus.New()
	require.NoError(t, err)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := NewPlanService(stores, b, iamManager, webhook.NewManager(stores, nil), nil, &config.Profile{SQLReviewV2: true})
	// Unbuffered and not yet received, so either signal blocks the update: a
	// slot not reset before the first signal stays DONE until the listener
	// takes it.
	b.PlanCheckTickleChan = make(chan int)
	b.ApprovalCheckChan = make(chan bus.IssueRef)
	slotResetBeforeSignal := make(chan bool, 1)
	signals := make(chan int, 1)
	updateDone := make(chan struct{})
	go func() {
		reset := false
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			run, err := stores.GetReviewRun(ctx, issue.ProjectID, issue.UID)
			if err == nil && run != nil && run.Status == storepb.ReviewRun_AVAILABLE {
				reset = true
				break
			}
		}
		slotResetBeforeSignal <- reset
		received := 0
		for {
			select {
			case <-b.PlanCheckTickleChan:
				received++
			case <-b.ApprovalCheckChan:
				received++
			case <-updateDone:
				signals <- received
				return
			}
		}
	}()

	_, err = service.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: common.FormatPlan(projectID, *issue.PlanUID),
			Specs: []*v1pb.Plan_Spec{{
				Id: "spec-1",
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{"instances/prod/databases/app"},
						Sheet:   common.FormatSheet(projectID, sheets[0].Sha256),
					},
				},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	close(updateDone)
	require.NoError(t, err)
	require.Positive(t, <-signals, "the update signals the plan checks or the approval check")
	require.True(t, <-slotResetBeforeSignal, "the review slot is reset before the first signal")
}
