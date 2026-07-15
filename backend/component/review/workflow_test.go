package review

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestReviewIssueRequestAgain(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)

	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "request access",
		Type:         storepb.Issue_ROLE_GRANT,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
				},
				Approvers: []*storepb.IssuePayloadApproval_Approver{{
					Status:    storepb.IssuePayloadApproval_Approver_REJECTED,
					Principal: common.FormatUserEmail("reviewer@example.com"),
				}},
			},
		},
	})
	require.NoError(t, err)

	result, err := NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: "creator@example.com"},
		Action:    ActionRequest,
		Comment:   "addressed",
	})
	require.NoError(t, err)
	require.Empty(t, result.Issue.Payload.GetApproval().GetApprovers())
	require.Equal(t, []*EventIntent{
		{Type: EventApprovalRequested},
		{
			Type:           EventIssueComment,
			ActorEmail:     "creator@example.com",
			Comment:        "addressed",
			ApprovalStatus: storepb.IssuePayloadApproval_Approver_PENDING,
		},
	}, result.Events)
}

func TestReviewIssueApproveCurrentPlan(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("reviewer@example.com"),
		Roles:     []string{"roles/projectOwner"},
	})
	require.NoError(t, err)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
				},
			},
		},
	})
	require.NoError(t, err)

	result, err := NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: "reviewer@example.com"},
		Action:    ActionApprove,
		Comment:   "looks good",
	})
	require.NoError(t, err)
	require.True(t, result.Approved)
	require.Len(t, result.Issue.Payload.GetApproval().GetApprovers(), 1)
	require.True(t, proto.Equal(&storepb.IssuePayloadApproval_Approver{
		Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
		Principal: common.FormatUserEmail("reviewer@example.com"),
	}, result.Issue.Payload.GetApproval().GetApprovers()[0]))
	require.Equal(t, []*EventIntent{
		{
			Type:           EventIssueComment,
			ActorEmail:     "reviewer@example.com",
			Comment:        "looks good",
			ApprovalStatus: storepb.IssuePayloadApproval_Approver_APPROVED,
		},
		{Type: EventApprovalRequested},
		{Type: EventIssueApproved},
		{Type: EventCreateRollout},
	}, result.Events)
}

func TestReviewIssueConcurrentApprovalsHaveOneWinner(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	for _, email := range []string{"reviewer@example.com", "reviewer2@example.com"} {
		_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
			Workspace: "default",
			Member:    common.FormatUserEmail(email),
			Roles:     []string{"roles/projectOwner"},
		})
		require.NoError(t, err)
	}
	_, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner", "roles/projectOwner"})
	workflow := NewWorkflow(stores)
	var proposals sync.WaitGroup
	proposals.Add(2)
	releaseCommit := make(chan struct{})
	workflow.beforeCommit = func() {
		proposals.Done()
		<-releaseCommit
	}

	type outcome struct {
		result *IssueResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, email := range []string{"reviewer@example.com", "reviewer2@example.com"} {
		wg.Go(func() {
			<-start
			result, err := workflow.ReviewIssue(ctx, IssueInput{
				Workspace: "default",
				ProjectID: "project-a",
				IssueUID:  issue.UID,
				Actor:     &store.UserMessage{Email: email},
				Action:    ActionApprove,
			})
			outcomes <- outcome{result: result, err: err}
		})
	}
	close(start)
	proposals.Wait()
	close(releaseCommit)
	wg.Wait()
	close(outcomes)

	var succeeded, rejected int
	for outcome := range outcomes {
		if outcome.err == nil {
			succeeded++
			require.False(t, outcome.result.Approved)
			require.Len(t, outcome.result.Events, 2)
			continue
		}
		var workflowErr *Error
		require.True(t, errors.As(outcome.err, &workflowErr))
		require.Equal(t, ErrorConflict, workflowErr.Code)
		require.Empty(t, outcome.result)
		rejected++
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, rejected)
}

func TestPlanUpdateMakesPendingApprovalActionStale(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("reviewer@example.com"),
		Roles:     []string{"roles/projectOwner"},
	})
	require.NoError(t, err)
	plan, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner"})
	workflow := NewWorkflow(stores)
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	workflow.beforeCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}

	actionDone := make(chan error, 1)
	go func() {
		_, err := workflow.ReviewIssue(ctx, IssueInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
			Actor:     &store.UserMessage{Email: "reviewer@example.com"},
			Action:    ActionApprove,
		})
		actionDone <- err
	}()
	<-proposalReady

	planResult, err := workflow.UpdatePlanSpecs(ctx, UpdatePlanSpecsInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Specs:     []*storepb.PlanConfig_Spec{{Id: "new"}},
	})
	require.NoError(t, err)
	require.True(t, planResult.ApprovalReset)
	close(releaseCommit)

	err = <-actionDone
	var workflowErr *Error
	require.True(t, errors.As(err, &workflowErr))
	require.Equal(t, ErrorConflict, workflowErr.Code)
	require.False(t, planResult.Issue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 3, planResult.Issue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdatePlanResetsLinkedIssueApprovalAtomically(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	oldSpecs := []*storepb.PlanConfig_Spec{{Id: "old"}}
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs:                oldSpecs,
		},
	}, "creator@example.com")
	require.NoError(t, err)
	_, err = stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
				},
			},
		},
	})
	require.NoError(t, err)
	newSpecs := []*storepb.PlanConfig_Spec{{Id: "new"}}

	result, err := NewWorkflow(stores).UpdatePlanSpecs(ctx, UpdatePlanSpecsInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Specs:     newSpecs,
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, result.Plan.Config.GetApprovalInputVersion())
	require.Equal(t, newSpecs, result.Plan.Config.GetSpecs())
	require.True(t, result.ApprovalReset)
	require.False(t, result.Issue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 3, result.Issue.Payload.GetApproval().GetApprovalInputVersion())
	require.Equal(t, []*EventIntent{{
		Type:      EventPlanUpdated,
		FromSpecs: oldSpecs,
		ToSpecs:   newSpecs,
	}}, result.Events)
}

func TestApplyApprovalTemplateCommitsFindingAndIntent(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Labels: []string{"environment:prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalInputVersion: 2,
			},
		},
	})
	require.NoError(t, err)
	finding := &storepb.Issue{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
			ApprovalTemplate: &storepb.ApprovalTemplate{
				Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
			},
		},
		RiskLevel: storepb.RiskLevel_HIGH,
	}

	workflow := NewWorkflow(stores)
	workflow.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = finding.Approval
		issue.Payload.RiskLevel = finding.RiskLevel
		return nil
	}
	result, err := workflow.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.True(t, result.Issue.Payload.GetApproval().Equal(finding.Approval))
	require.Equal(t, storepb.RiskLevel_HIGH, result.Issue.Payload.GetRiskLevel())
	require.Equal(t, []*EventIntent{{Type: EventApprovalRequested}}, result.Events)
}

func TestUpdateIssueLabelsResetsApprovalBeforeRollout(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Labels: []string{"old"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
	})
	require.NoError(t, err)

	result, err := NewWorkflow(stores).UpdateIssueLabels(ctx, UpdateIssueLabelsInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Labels:    []string{"new"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"new"}, result.Issue.Payload.GetLabels())
	require.True(t, result.ApprovalReset)
	require.False(t, result.Issue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, result.Issue.Payload.GetApproval().GetApprovalInputVersion())
	require.Equal(t, []*EventIntent{{Type: EventApprovalCheck}}, result.Events)
}

func TestLabelResetMakesPendingApprovalActionStale(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("reviewer@example.com"),
		Roles:     []string{"roles/projectOwner"},
	})
	require.NoError(t, err)
	plan, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner"})
	config := proto.CloneOf(plan.Config)
	config.Specs = []*storepb.PlanConfig_Spec{{
		Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
		},
	}}
	_, err = stores.UpdatePlan(ctx, &store.UpdatePlanMessage{UID: plan.UID, ProjectID: plan.ProjectID, Config: config})
	require.NoError(t, err)

	workflow := NewWorkflow(stores)
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	workflow.beforeCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}
	type actionOutcome struct {
		result *IssueResult
		err    error
	}
	actionDone := make(chan actionOutcome, 1)
	go func() {
		result, err := workflow.ReviewIssue(ctx, IssueInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
			Actor:     &store.UserMessage{Email: "reviewer@example.com"},
			Action:    ActionApprove,
		})
		actionDone <- actionOutcome{result: result, err: err}
	}()
	<-proposalReady

	labels, err := NewWorkflow(stores).UpdateIssueLabels(ctx, UpdateIssueLabelsInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Labels:    []string{"security"},
	})
	require.NoError(t, err)
	require.True(t, labels.ApprovalReset)
	close(releaseCommit)

	action := <-actionDone
	require.Nil(t, action.result)
	err = action.err
	var workflowErr *Error
	require.True(t, errors.As(err, &workflowErr))
	require.Equal(t, ErrorConflict, workflowErr.Code)
	got, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Empty(t, got.Payload.GetApproval().GetApprovers())
}

func TestRolloutMakesPendingApprovalFindingStale(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		Workspace:  "default",
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: true},
	}))
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{ApprovalInputVersion: 2},
		},
	})
	require.NoError(t, err)

	workflow := NewWorkflow(stores)
	workflow.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	workflow.beforeCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}
	applyDone := make(chan error, 1)
	go func() {
		_, err := workflow.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
		})
		applyDone <- err
	}()
	<-proposalReady

	marked, _, err := stores.CreateRolloutTasks(ctx, "project-a", plan.UID, nil, nil)
	require.NoError(t, err)
	require.True(t, marked)
	close(releaseCommit)

	err = <-applyDone
	var workflowErr *Error
	require.True(t, errors.As(err, &workflowErr))
	require.Equal(t, ErrorConflict, workflowErr.Code)
}

func TestPlanMutationMakesPendingApprovalFindingStale(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{ApprovalInputVersion: 2},
		},
	})
	require.NoError(t, err)

	workflow := NewWorkflow(stores)
	workflow.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	workflow.beforeCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}
	type findingOutcome struct {
		result *ApplyApprovalTemplateResult
		err    error
	}
	applyDone := make(chan findingOutcome, 1)
	go func() {
		result, err := workflow.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
		})
		applyDone <- findingOutcome{result: result, err: err}
	}()
	<-proposalReady

	updated, err := NewWorkflow(stores).UpdatePlanSpecs(ctx, UpdatePlanSpecsInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Specs:     []*storepb.PlanConfig_Spec{{Id: "new"}},
	})
	require.NoError(t, err)
	require.True(t, updated.ApprovalReset)
	close(releaseCommit)

	finding := <-applyDone
	require.Nil(t, finding.result)
	err = finding.err
	var workflowErr *Error
	require.True(t, errors.As(err, &workflowErr))
	require.Equal(t, ErrorConflict, workflowErr.Code)
	got, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 3, got.Payload.GetApproval().GetApprovalInputVersion())
}

func setupWorkflowStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES
			('creator', 'creator@example.com', 'unused'),
			('reviewer', 'reviewer@example.com', 'unused'),
			('reviewer2', 'reviewer2@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })
	return stores
}

func createPendingDatabaseChangeApproval(ctx context.Context, t *testing.T, stores *store.Store, roles []string) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: roles},
				},
			},
		},
	})
	require.NoError(t, err)
	return plan, issue
}
