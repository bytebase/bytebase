package review

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	require.Equal(t, []Event{
		ApprovalRequestedEvent{},
		IssueCommentEvent{
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
	require.Equal(t, []Event{
		IssueCommentEvent{
			ActorEmail:     "reviewer@example.com",
			Comment:        "looks good",
			ApprovalStatus: storepb.IssuePayloadApproval_Approver_APPROVED,
		},
		ApprovalRequestedEvent{},
		IssueApprovedEvent{},
		CreateRolloutEvent{},
	}, result.Events)
}

func TestLastPlanEditorApproval(t *testing.T) {
	newIssue := func(ctx context.Context, t *testing.T, stores *store.Store, clearLastPlanEditor bool) *store.IssueMessage {
		t.Helper()
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: "project-a",
			Name:      "change database",
			Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
		}, "reviewer@example.com")
		require.NoError(t, err)
		if clearLastPlanEditor {
			_, err := stores.GetDB().ExecContext(ctx, "UPDATE plan SET last_plan_editor = NULL WHERE project = $1 AND id = $2", plan.ProjectID, plan.UID)
			require.NoError(t, err)
		}
		issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
			ProjectID:    "project-a",
			CreatorEmail: "creator@example.com",
			Title:        "change database",
			Type:         storepb.Issue_DATABASE_CHANGE,
			PlanUID:      &plan.UID,
			Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
				ApprovalTemplate:     &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
			}},
		})
		require.NoError(t, err)
		return issue
	}

	for _, test := range []struct {
		name                string
		action              Action
		allowLastPlanEditor bool
		clearLastPlanEditor bool
		wantError           ErrorCode
	}{
		{name: "approve blocked", action: ActionApprove, wantError: ErrorFailedPrecondition},
		{name: "reject allowed", action: ActionReject},
		{name: "approve allowed by setting", action: ActionApprove, allowLastPlanEditor: true},
		{name: "legacy Plan falls back to creator", action: ActionApprove, clearLastPlanEditor: true, wantError: ErrorFailedPrecondition},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
				Workspace: "default",
				Member:    common.FormatUserEmail("reviewer@example.com"),
				Roles:     []string{"roles/projectOwner"},
			})
			require.NoError(t, err)
			if test.allowLastPlanEditor {
				projectID := "project-a"
				require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
					Workspace:  "default",
					ResourceID: projectID,
					Setting:    &storepb.Project{AllowLastPlanEditorApproval: true},
				}))
			}
			issue := newIssue(ctx, t, stores, test.clearLastPlanEditor)
			_, err = NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
				Workspace: "default",
				ProjectID: "project-a",
				IssueUID:  issue.UID,
				Actor:     &store.UserMessage{Email: "reviewer@example.com"},
				Action:    test.action,
			})
			if test.wantError == ErrorInternal {
				require.NoError(t, err)
				return
			}
			var workflowErr *Error
			require.ErrorAs(t, err, &workflowErr)
			require.Equal(t, test.wantError, workflowErr.Code)
		})
	}
}

func TestReviewIssueEvaluatesPolicyForEachApprovalStep(t *testing.T) {
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
	plan, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner", "roles/projectOwner"})
	workflow := NewWorkflow(stores)
	lastPlanEditor := "reviewer@example.com"
	_, err := workflow.UpdatePlan(ctx, UpdatePlanInput{
		Workspace:      "default",
		ProjectID:      "project-a",
		PlanUID:        plan.UID,
		LastPlanEditor: &lastPlanEditor,
	})
	require.NoError(t, err)

	_, err = workflow.ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: lastPlanEditor},
		Action:    ActionApprove,
	})
	var workflowErr *Error
	require.ErrorAs(t, err, &workflowErr)
	require.Equal(t, ErrorFailedPrecondition, workflowErr.Code)

	first, err := workflow.ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: "reviewer2@example.com"},
		Action:    ActionApprove,
	})
	require.NoError(t, err)
	require.False(t, first.Approved)

	_, err = workflow.ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: lastPlanEditor},
		Action:    ActionApprove,
	})
	require.ErrorAs(t, err, &workflowErr)
	require.Equal(t, ErrorFailedPrecondition, workflowErr.Code)

	second, err := workflow.ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: "reviewer2@example.com"},
		Action:    ActionApprove,
	})
	require.NoError(t, err)
	require.True(t, second.Approved)
}

func TestPlanSpecsUpdateResetsApprovalAndChangesLastPlanEditor(t *testing.T) {
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
	plan, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner", "roles/projectOwner"})
	workflow := NewWorkflow(stores)

	first, err := workflow.ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: "reviewer@example.com"},
		Action:    ActionApprove,
	})
	require.NoError(t, err)
	require.False(t, first.Approved)

	newEditor := "reviewer2@example.com"
	updated, err := workflow.UpdatePlanSpecs(ctx, UpdatePlanSpecsInput{
		Workspace:      "default",
		ProjectID:      "project-a",
		PlanUID:        plan.UID,
		Specs:          proto.CloneOf(plan.Config).GetSpecs(),
		LastPlanEditor: &newEditor,
	})
	require.NoError(t, err)
	require.True(t, updated.ApprovalReset)
	require.Empty(t, updated.Events)
	require.NotNil(t, updated.Plan.LastPlanEditor)
	require.Equal(t, newEditor, *updated.Plan.LastPlanEditor)
	require.Empty(t, updated.Issue.Payload.GetApproval().GetApprovers())
	require.False(t, updated.Issue.Payload.GetApproval().GetApprovalFindingDone())

	_, err = stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: updated.Plan.Config.GetApprovalInputVersion(),
			ApprovalTemplate:     &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner", "roles/projectOwner"}}},
		}},
	})
	require.NoError(t, err)

	priorEditor, err := workflow.ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: "reviewer@example.com"},
		Action:    ActionApprove,
	})
	require.NoError(t, err)
	require.False(t, priorEditor.Approved)

	_, err = workflow.ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: newEditor},
		Action:    ActionApprove,
	})
	var workflowErr *Error
	require.ErrorAs(t, err, &workflowErr)
	require.Equal(t, ErrorFailedPrecondition, workflowErr.Code)
}

func TestPlanlessGrantApprovalsDoNotRequireAPlan(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("reviewer@example.com"),
		Roles:     []string{"roles/projectOwner"},
	})
	require.NoError(t, err)

	for _, issueType := range []storepb.Issue_Type{storepb.Issue_ROLE_GRANT, storepb.Issue_ACCESS_GRANT} {
		issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
			ProjectID:    "project-a",
			CreatorEmail: "creator@example.com",
			Title:        issueType.String(),
			Type:         issueType,
			Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplate:    &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
			}},
		})
		require.NoError(t, err)

		result, err := NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
			Actor:     &store.UserMessage{Email: "reviewer@example.com"},
			Action:    ActionApprove,
		})
		require.NoError(t, err)
		require.True(t, result.Approved)
		require.Contains(t, result.Events, CompleteAccessRequestEvent{})
	}
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

func TestReviewIssueAppliesAConfirmedActionOnlyToTheConfirmedVersion(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("reviewer@example.com"),
		Roles:     []string{"roles/projectOwner"},
	})
	require.NoError(t, err)
	confirmedApproval := func(issue *store.IssueMessage) IssueInput {
		return IssueInput{
			Workspace:           "default",
			ProjectID:           "project-a",
			IssueUID:            issue.UID,
			Actor:               &store.UserMessage{Email: "reviewer@example.com"},
			Action:              ActionApprove,
			ConfirmedUpdateTime: &issue.UpdatedAt,
		}
	}

	t.Run("unchanged", func(t *testing.T) {
		_, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner"})
		result, err := NewWorkflow(stores).ReviewIssue(ctx, confirmedApproval(issue))
		require.NoError(t, err)
		require.True(t, result.Approved)
	})

	t.Run("changed after the workflow read it", func(t *testing.T) {
		_, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner"})
		workflow := NewWorkflow(stores)
		// The edit lands between the workflow's unlocked read and its lock and
		// leaves the approval as it was, so only the confirmed version refuses.
		workflow.beforeCommit = func() {
			title := "drop database"
			_, err := stores.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{Title: &title})
			require.NoError(t, err)
		}

		_, err := workflow.ReviewIssue(ctx, confirmedApproval(issue))
		var workflowErr *Error
		require.True(t, errors.As(err, &workflowErr))
		require.Equal(t, ErrorConflict, workflowErr.Code)
		require.Equal(t, ReasonChangedSinceConfirmed, workflowErr.Reason)
		after, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
		require.NoError(t, err)
		require.Empty(t, after.Payload.GetApproval().GetApprovers(), "a refused action records no approval")
	})
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

func TestApprovalStartedDuringPlanUpdateCannotUseStaleEditorState(t *testing.T) {
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
	planLocked := make(chan struct{})
	releasePlan := make(chan struct{})
	approvalProposed := make(chan struct{})
	workflow.beforePlanMutation = func() {
		close(planLocked)
		<-releasePlan
	}
	workflow.beforeCommit = func() {
		close(approvalProposed)
	}

	newEditor := "reviewer@example.com"
	type planOutcome struct {
		result *UpdatePlanSpecsResult
		err    error
	}
	planDone := make(chan planOutcome, 1)
	go func() {
		result, err := workflow.UpdatePlanSpecs(ctx, UpdatePlanSpecsInput{
			Workspace:      "default",
			ProjectID:      "project-a",
			PlanUID:        plan.UID,
			Specs:          []*storepb.PlanConfig_Spec{{Id: "new"}},
			LastPlanEditor: &newEditor,
		})
		planDone <- planOutcome{result: result, err: err}
	}()
	select {
	case <-planLocked:
	case <-time.After(5 * time.Second):
		t.Fatal("Plan update did not acquire its workflow locks")
	}

	approvalDone := make(chan error, 1)
	go func() {
		_, err := workflow.ReviewIssue(ctx, IssueInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
			Actor:     &store.UserMessage{Email: newEditor},
			Action:    ActionApprove,
		})
		approvalDone <- err
	}()
	select {
	case <-approvalProposed:
	case <-time.After(5 * time.Second):
		t.Fatal("approval did not reach the commit seam")
	}
	close(releasePlan)

	var updated *UpdatePlanSpecsResult
	select {
	case outcome := <-planDone:
		require.NoError(t, outcome.err)
		updated = outcome.result
		require.True(t, updated.ApprovalReset)
	case <-time.After(5 * time.Second):
		t.Fatal("Plan update deadlocked with approval")
	}
	select {
	case err := <-approvalDone:
		var workflowErr *Error
		require.ErrorAs(t, err, &workflowErr)
		require.Equal(t, ErrorConflict, workflowErr.Code)
	case <-time.After(5 * time.Second):
		t.Fatal("approval deadlocked with Plan update")
	}

	_, err = stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: updated.Plan.Config.GetApprovalInputVersion(),
			ApprovalTemplate:     &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
		}},
	})
	require.NoError(t, err)
	_, err = NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: newEditor},
		Action:    ActionApprove,
	})
	var workflowErr *Error
	require.ErrorAs(t, err, &workflowErr)
	require.Equal(t, ErrorFailedPrecondition, workflowErr.Code)
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
	require.Equal(t, []Event{PlanUpdatedEvent{
		FromSpecs: oldSpecs,
		ToSpecs:   newSpecs,
	}}, result.Events)
}

func TestUpdatePlanSpecsIncludesIssueLinkedAfterProposal(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs:                []*storepb.PlanConfig_Spec{{Id: "old"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	workflow := NewWorkflow(stores)
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	workflow.beforePlanCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}
	type updateOutcome struct {
		result *UpdatePlanSpecsResult
		err    error
	}
	updateDone := make(chan updateOutcome, 1)
	go func() {
		result, err := workflow.UpdatePlanSpecs(ctx, UpdatePlanSpecsInput{
			Workspace: "default",
			ProjectID: "project-a",
			PlanUID:   plan.UID,
			Specs:     []*storepb.PlanConfig_Spec{{Id: "new"}},
		})
		updateDone <- updateOutcome{result: result, err: err}
	}()

	select {
	case <-proposalReady:
	case <-time.After(5 * time.Second):
		t.Fatal("Plan update did not reach the commit seam")
	}
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "change database",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}},
	})
	require.NoError(t, err)
	close(releaseCommit)

	outcome := <-updateDone
	require.NoError(t, outcome.err)
	require.NotNil(t, outcome.result.Issue)
	require.Equal(t, issue.UID, outcome.result.Issue.UID)
	require.True(t, outcome.result.ApprovalReset)
	require.False(t, outcome.result.Issue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 3, outcome.result.Issue.Payload.GetApproval().GetApprovalInputVersion())
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

	evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
	evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = finding.Approval
		issue.Payload.RiskLevel = finding.RiskLevel
		return nil
	}
	result, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.True(t, result.Issue.Payload.GetApproval().Equal(finding.Approval))
	require.Equal(t, storepb.RiskLevel_HIGH, result.Issue.Payload.GetRiskLevel())
	require.Equal(t, []Event{ApprovalRequestedEvent{}}, result.Events)
}

func TestPlanCheckRerunMakesPendingApprovalFindingStale(t *testing.T) {
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
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalInputVersion: 2,
		}},
	})
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)
	planCheckRun, err := stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NoError(t, stores.UpdatePlanCheckRun(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 2,
	}, planCheckRun.UID))

	evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
	evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	evaluator.beforeCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}
	type findingOutcome struct {
		result *ApplyApprovalTemplateResult
		err    error
	}
	applyDone := make(chan findingOutcome, 1)
	go func() {
		result, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
		})
		applyDone <- findingOutcome{result: result, err: err}
	}()
	<-proposalReady

	refreshed, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, refreshed)
	close(releaseCommit)

	finding := <-applyDone
	require.Nil(t, finding.result)
	var workflowErr *Error
	require.ErrorAs(t, finding.err, &workflowErr)
	require.Equal(t, ErrorConflict, workflowErr.Code)
	got, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
}

func TestPlanCheckCreatedDuringApprovalMakesPendingFindingStale(t *testing.T) {
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
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalInputVersion: 2,
		}},
	})
	require.NoError(t, err)

	evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
	evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: "project-a",
			PlanUID:   plan.UID,
			Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
		})
		require.NoError(t, err)
		require.True(t, created)
		planCheckRun, err := stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
		require.NoError(t, err)
		require.NoError(t, stores.UpdatePlanCheckRun(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
			ApprovalInputVersion: 2,
		}, planCheckRun.UID))
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	evaluator.beforeCommit = func() {
		refreshed, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: "project-a",
			PlanUID:   plan.UID,
			Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
		})
		require.NoError(t, err)
		require.True(t, refreshed)
	}

	result, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
	})
	require.Nil(t, result)
	var workflowErr *Error
	require.ErrorAs(t, err, &workflowErr)
	require.Equal(t, ErrorConflict, workflowErr.Code)
	got, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
}

func TestUpdateIssueMetadataCommitsMixedPatchAndResetsApproval(t *testing.T) {
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
		Description:  "old description",
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

	title := "renamed database change"
	description := ""
	labels := []string{"new"}
	result, err := NewWorkflow(stores).UpdateIssueMetadata(ctx, UpdateIssueMetadataInput{
		Workspace:   "default",
		ProjectID:   "project-a",
		IssueUID:    issue.UID,
		Title:       &title,
		Description: &description,
		Labels:      &labels,
	})
	require.NoError(t, err)
	require.Equal(t, title, result.Issue.Title)
	require.Empty(t, result.Issue.Description)
	require.Equal(t, []string{"new"}, result.Issue.Payload.GetLabels())
	require.True(t, result.ApprovalReset)
	require.False(t, result.Issue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, result.Issue.Payload.GetApproval().GetApprovalInputVersion())
	require.Equal(t, []Event{
		IssueTitleUpdatedEvent{FromTitle: "change database", ToTitle: title},
		IssueDescriptionUpdatedEvent{FromDescription: "old description", ToDescription: ""},
		IssueLabelsUpdatedEvent{FromLabels: []string{"old"}, ToLabels: []string{"new"}},
		ApprovalCheckEvent{},
	}, result.Events)
}

func TestUpdateIssueMetadataDoesNotRequirePlanForDraftLabels(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft database change",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "draft database change",
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{
			Draft:  true,
			Labels: []string{"old"},
		},
	})
	require.NoError(t, err)
	planTx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer planTx.Rollback()
	var planUID int64
	require.NoError(t, planTx.QueryRowContext(ctx, `
		SELECT id
		FROM plan
		WHERE project = $1 AND id = $2
		FOR UPDATE`, issue.ProjectID, plan.UID).Scan(&planUID))

	labels := []string{"new"}
	updateCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result, err := NewWorkflow(stores).UpdateIssueMetadata(updateCtx, UpdateIssueMetadataInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Labels:    &labels,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"new"}, result.Issue.Payload.GetLabels())
	require.False(t, result.ApprovalReset)
	require.Equal(t, []Event{
		IssueLabelsUpdatedEvent{FromLabels: []string{"old"}, ToLabels: []string{"new"}},
	}, result.Events)
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
	configPayload, err := protojson.Marshal(config)
	require.NoError(t, err)
	result, err := stores.GetDB().ExecContext(ctx, `
		UPDATE plan
		SET config = $1
		WHERE project = $2 AND id = $3
	`, configPayload, plan.ProjectID, plan.UID)
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)

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

	updatedLabels := []string{"security"}
	labels, err := NewWorkflow(stores).UpdateIssueMetadata(ctx, UpdateIssueMetadataInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Labels:    &updatedLabels,
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

	evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
	evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	evaluator.beforeCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}
	applyDone := make(chan error, 1)
	go func() {
		_, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
			Workspace: "default",
			ProjectID: "project-a",
			IssueUID:  issue.UID,
		})
		applyDone <- err
	}()
	<-proposalReady

	approved := &storepb.IssuePayloadApproval{
		ApprovalFindingDone:  true,
		ApprovalInputVersion: 2,
	}
	_, err = stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: approved},
	})
	require.NoError(t, err)
	_, err = NewWorkflow(stores).CreateRollout(ctx, CreateRolloutInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		BuildTasks: func(context.Context, *store.PlanMessage, *store.ProjectMessage) ([]*store.TaskMessage, error) {
			return nil, nil
		},
	})
	require.NoError(t, err)
	close(releaseCommit)

	err = <-applyDone
	var workflowErr *Error
	require.True(t, errors.As(err, &workflowErr))
	require.Equal(t, ErrorConflict, workflowErr.Code)
}

func TestPlanMutationMakesIssueLessRolloutStale(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs:                []*storepb.PlanConfig_Spec{{Id: "old"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	buildStarted := make(chan struct{})
	releaseBuild := make(chan struct{})
	type rolloutOutcome struct {
		result *CreateRolloutResult
		err    error
	}
	rolloutDone := make(chan rolloutOutcome, 1)
	go func() {
		result, err := NewWorkflow(stores).CreateRollout(ctx, CreateRolloutInput{
			Workspace: "default",
			ProjectID: "project-a",
			PlanUID:   plan.UID,
			BuildTasks: func(_ context.Context, _ *store.PlanMessage, _ *store.ProjectMessage) ([]*store.TaskMessage, error) {
				close(buildStarted)
				<-releaseBuild
				return nil, nil
			},
		})
		rolloutDone <- rolloutOutcome{result: result, err: err}
	}()
	<-buildStarted

	updated, err := NewWorkflow(stores).UpdatePlanSpecs(ctx, UpdatePlanSpecsInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Specs:     []*storepb.PlanConfig_Spec{{Id: "new"}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, updated.Plan.Config.GetApprovalInputVersion())
	close(releaseBuild)

	outcome := <-rolloutDone
	require.Nil(t, outcome.result)
	var workflowErr *Error
	require.ErrorAs(t, outcome.err, &workflowErr)
	require.Equal(t, ErrorConflict, workflowErr.Code)
	got, err := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, got.Config.GetHasRollout())
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

	evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
	evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	proposalReady := make(chan struct{})
	releaseCommit := make(chan struct{})
	evaluator.beforeCommit = func() {
		close(proposalReady)
		<-releaseCommit
	}
	type findingOutcome struct {
		result *ApplyApprovalTemplateResult
		err    error
	}
	applyDone := make(chan findingOutcome, 1)
	go func() {
		result, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
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
	_, stores := setupWorkflowStoreWithDB(ctx, t)
	return stores
}

func setupWorkflowStoreWithDB(ctx context.Context, t *testing.T) (*sql.DB, *store.Store) {
	t.Helper()

	db, stores, _ := testcontainer.NewMetadataDB(t)

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES
			('creator', 'creator@example.com', 'unused'),
			('reviewer', 'reviewer@example.com', 'unused'),
			('reviewer2', 'reviewer2@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
	`)
	require.NoError(t, err)

	return db, stores
}

func createPendingDatabaseChangeApproval(ctx context.Context, t *testing.T, stores *store.Store, roles []string) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	return createPendingDatabaseChangeApprovalBy(ctx, t, stores, "creator@example.com", roles)
}

func createPendingDatabaseChangeApprovalBy(ctx context.Context, t *testing.T, stores *store.Store, creator string, roles []string) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, creator)
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: creator,
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

func TestApproverRoleRefusalCarriesItsReason(t *testing.T) {
	for _, test := range []struct {
		name       string
		actor      string
		actorRoles []string
		wantReason ErrorReason
	}{
		{
			name:       "no approver role",
			actor:      "reviewer@example.com",
			wantReason: ReasonApproverRoleRequired,
		},
		{
			name:       "self-approval",
			actor:      "creator@example.com",
			actorRoles: []string{"roles/projectOwner"},
			wantReason: ReasonUnspecified,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			if len(test.actorRoles) > 0 {
				_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
					Workspace: "default",
					Member:    common.FormatUserEmail(test.actor),
					Roles:     test.actorRoles,
				})
				require.NoError(t, err)
			}
			_, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner"})

			_, err := NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
				Workspace: "default",
				ProjectID: "project-a",
				IssueUID:  issue.UID,
				Actor:     &store.UserMessage{Email: test.actor},
				Action:    ActionApprove,
			})

			var workflowErr *Error
			require.ErrorAs(t, err, &workflowErr)
			require.Equal(t, ErrorPermissionDenied, workflowErr.Code)
			require.Equal(t, test.wantReason, workflowErr.Reason,
				"only the IAM verdict is marked for the audit log; the self-approval setting is not")
		})
	}
}

func TestFindUnapprovableStep(t *testing.T) {
	const owner, releaser = "roles/projectOwner", "roles/projectReleaser"
	grant := func(member, role string) func(context.Context, *testing.T, *store.Store) {
		return func(ctx context.Context, t *testing.T, stores *store.Store) {
			_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{Workspace: "default", Member: member, Roles: []string{role}})
			require.NoError(t, err)
		}
	}
	// grantInProject binds the role in the project policy only, so the member
	// gets no workspace binding from it.
	grantInProject := func(member, role string) func(context.Context, *testing.T, *store.Store) {
		return func(ctx context.Context, t *testing.T, stores *store.Store) {
			_, err := stores.PatchIamPolicy(ctx, "default", storepb.Policy_PROJECT, common.FormatProject("project-a"), func(policy *storepb.IamPolicy) error {
				policy.Bindings = append(policy.Bindings, &storepb.Binding{Role: role, Members: []string{member}})
				return nil
			})
			require.NoError(t, err)
		}
	}
	for _, test := range []struct {
		name  string
		saas  bool
		flow  []string
		setup []func(context.Context, *testing.T, *store.Store)
		// approved counts the steps approved before the check.
		approved int
		want     *UnapprovableStep
	}{
		{
			name:  "only the creator holds the role, and self-approval is off",
			flow:  []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){grant("users/creator@example.com", owner)},
			want:  &UnapprovableStep{Position: 1, Role: owner, CreatorHoldsRole: true},
		},
		{
			name: "only the creator holds the role, and the project allows self-approval",
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				grant("users/creator@example.com", owner),
				func(ctx context.Context, t *testing.T, stores *store.Store) {
					require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
						Workspace: "default", ResourceID: "project-a", Setting: &storepb.Project{AllowSelfApproval: true},
					}))
				},
			},
		},
		{
			name:  "another person holds the role",
			flow:  []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){grant("users/creator@example.com", owner), grant("users/reviewer@example.com", owner)},
		},
		{
			name: "another person holds the role through a group",
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				func(ctx context.Context, t *testing.T, stores *store.Store) {
					_, err := stores.CreateGroup(ctx, &store.GroupMessage{
						Workspace: "default",
						Email:     "approvers@example.com",
						Title:     "Approvers",
						Payload:   &storepb.GroupPayload{Members: []*storepb.GroupMember{{Member: "users/reviewer@example.com", Role: storepb.GroupMember_MEMBER}}},
					})
					require.NoError(t, err)
				},
				grant("groups/approvers@example.com", owner),
			},
		},
		{
			name: "no person holds the role",
			flow: []string{owner},
			want: &UnapprovableStep{Position: 1, Role: owner},
		},
		{
			name:  "a later step",
			flow:  []string{owner, releaser},
			setup: []func(context.Context, *testing.T, *store.Store){grant("users/reviewer@example.com", owner)},
			want:  &UnapprovableStep{Position: 2, Role: releaser},
		},
		{
			name:     "an approved step is not checked",
			flow:     []string{releaser, owner},
			setup:    []func(context.Context, *testing.T, *store.Store){grant("users/reviewer@example.com", owner)},
			approved: 1,
		},
		{
			name: "the other holder is deactivated",
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				grant("users/creator@example.com", owner),
				grant("users/reviewer@example.com", owner),
				func(ctx context.Context, t *testing.T, stores *store.Store) {
					reviewer, err := stores.GetUserByEmail(ctx, "reviewer@example.com")
					require.NoError(t, err)
					_, err = stores.UpdateUser(ctx, reviewer, &store.UpdateUserMessage{Delete: new(true)})
					require.NoError(t, err)
				},
			},
			want: &UnapprovableStep{Position: 1, Role: owner, CreatorHoldsRole: true},
		},
		{
			name: "the other holder is bound under another principal type",
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				grant("users/creator@example.com", owner),
				grant(common.FormatServiceAccountEmail("reviewer@example.com"), owner),
			},
			want: &UnapprovableStep{Position: 1, Role: owner, CreatorHoldsRole: true},
		},
		{
			name: "self-hosted, the other holder has only a project binding",
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				grant("users/creator@example.com", owner),
				grantInProject("users/reviewer@example.com", owner),
			},
		},
		{
			name: "SaaS, the other holder has only a project binding",
			saas: true,
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				grant("users/creator@example.com", owner),
				grantInProject("users/reviewer@example.com", owner),
			},
			want: &UnapprovableStep{Position: 1, Role: owner, CreatorHoldsRole: true},
		},
		{
			name: "SaaS, the other holder's workspace binding has expired",
			saas: true,
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				grant("users/creator@example.com", owner),
				grantInProject("users/reviewer@example.com", owner),
				func(ctx context.Context, t *testing.T, stores *store.Store) {
					_, err := stores.PatchIamPolicy(ctx, "default", storepb.Policy_WORKSPACE, common.FormatWorkspace("default"), func(policy *storepb.IamPolicy) error {
						policy.Bindings = append(policy.Bindings, &storepb.Binding{
							Role:      "roles/workspaceMember",
							Members:   []string{"users/reviewer@example.com"},
							Condition: &expr.Expr{Expression: `request.time < timestamp("2020-01-01T00:00:00Z")`},
						})
						return nil
					})
					require.NoError(t, err)
				},
			},
			want: &UnapprovableStep{Position: 1, Role: owner, CreatorHoldsRole: true},
		},
		{
			// Self-hosted, where no membership check runs, so only the principal
			// type can leave the account out.
			name: "only a service account holds the role",
			flow: []string{owner},
			setup: []func(context.Context, *testing.T, *store.Store){
				func(ctx context.Context, t *testing.T, stores *store.Store) {
					_, err := stores.GetDB().ExecContext(ctx,
						`INSERT INTO service_account (name, email, workspace, service_key_hash) VALUES ('bot', 'bot@service.bytebase.com', 'default', 'unused')`)
					require.NoError(t, err)
					account, err := stores.GetAccountByEmail(ctx, "bot@service.bytebase.com")
					require.NoError(t, err)
					require.Equal(t, storepb.PrincipalType_SERVICE_ACCOUNT, account.Type, "the holder must resolve to an account")
				},
				grant(common.FormatServiceAccountEmail("bot@service.bytebase.com"), owner),
			},
			want: &UnapprovableStep{Position: 1, Role: owner},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			for _, setup := range test.setup {
				setup(ctx, t, stores)
			}
			var approvers []*storepb.IssuePayloadApproval_Approver
			for range test.approved {
				approvers = append(approvers, &storepb.IssuePayloadApproval_Approver{
					Status: storepb.IssuePayloadApproval_Approver_APPROVED, Principal: "users/reviewer2@example.com",
				})
			}
			issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
				ProjectID:    "project-a",
				CreatorEmail: "creator@example.com",
				Title:        "access grant request",
				Type:         storepb.Issue_ACCESS_GRANT,
				Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone: true,
					ApprovalTemplate:    &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: test.flow}},
					Approvers:           approvers,
				}},
			})
			require.NoError(t, err)

			got, err := FindUnapprovableStep(ctx, stores, test.saas, "default", common.FormatIssue("project-a", issue.UID))
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

// The check counts a holder only where the approval check would let them
// approve: a binding names a typed principal, not an email.
func TestFindUnapprovableStepAgreesWithTheApprovalCheckOnPrincipalType(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	for member, role := range map[string]string{
		"users/creator@example.com":                              "roles/projectOwner",
		common.FormatServiceAccountEmail("reviewer@example.com"): "roles/projectOwner",
	} {
		_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{Workspace: "default", Member: member, Roles: []string{role}})
		require.NoError(t, err)
	}
	_, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner"})

	step, err := FindUnapprovableStep(ctx, stores, false, "default", common.FormatIssue("project-a", issue.UID))
	require.NoError(t, err)
	require.Equal(t, &UnapprovableStep{Position: 1, Role: "roles/projectOwner", CreatorHoldsRole: true}, step)

	reviewer, err := stores.GetUserByEmail(ctx, "reviewer@example.com")
	require.NoError(t, err)
	_, err = NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     reviewer,
		Action:    ActionApprove,
	})
	var workflowErr *Error
	require.ErrorAs(t, err, &workflowErr)
	require.Equal(t, ReasonApproverRoleRequired, workflowErr.Reason, "the approval check refuses the same person")
}

// A holder the lookup cannot resolve leaves the answer unknown, so the check
// fails instead of reporting that no one else can approve.
func TestFindUnapprovableStepFailsWhenAHolderCannotBeResolved(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	for _, member := range []string{"users/creator@example.com", "users/"} {
		_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{Workspace: "default", Member: member, Roles: []string{"roles/projectOwner"}})
		require.NoError(t, err)
	}
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "access grant request",
		Type:         storepb.Issue_ACCESS_GRANT,
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: true,
			ApprovalTemplate:    &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
		}},
	})
	require.NoError(t, err)

	step, err := FindUnapprovableStep(ctx, stores, false, "default", common.FormatIssue("project-a", issue.UID))
	require.ErrorContains(t, err, `failed to parse email from identifier users/`)
	require.Nil(t, step)
}

func TestApproverRoleLookupFailureIsNotARefusal(t *testing.T) {
	ctx := context.Background()
	db, stores := setupWorkflowStoreWithDB(ctx, t)
	_, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{"roles/projectOwner"})

	// A policy the store cannot parse is the cheapest stand-in for any read
	// that fails: an outage, a permission error on the metadata database, a
	// payload written by an older release.
	_, err := db.ExecContext(ctx, `
		INSERT INTO policy (workspace, resource_type, resource, type, payload)
		VALUES ('default', 'PROJECT', 'projects/project-a', 'IAM', '{"bindings": "not a list"}');
	`)
	require.NoError(t, err)

	_, err = NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
		Workspace: "default",
		ProjectID: "project-a",
		IssueUID:  issue.UID,
		Actor:     &store.UserMessage{Email: "reviewer@example.com"},
		Action:    ActionApprove,
	})

	var workflowErr *Error
	require.ErrorAs(t, err, &workflowErr)
	require.Equal(t, ErrorInternal, workflowErr.Code,
		"a policy the store cannot read is an outage, not a verdict about the caller")
	require.Equal(t, ReasonUnspecified, workflowErr.Reason,
		"an outage must not be marked as a permission denial in the audit log")
}

const serviceNowApproverRole = "roles/servicenow-approver"

// bindApproverRole grants the role to one policy member, in the project policy
// or the workspace policy.
func bindApproverRole(ctx context.Context, t *testing.T, stores *store.Store, inProject bool, member string) {
	t.Helper()
	resourceType, resource := storepb.Policy_WORKSPACE, common.FormatWorkspace("default")
	if inProject {
		resourceType, resource = storepb.Policy_PROJECT, common.FormatProject("project-a")
	}
	_, err := stores.PatchIamPolicy(ctx, "default", resourceType, resource, func(policy *storepb.IamPolicy) error {
		policy.Bindings = append(policy.Bindings, &storepb.Binding{
			Role:    serviceNowApproverRole,
			Members: []string{member},
		})
		return nil
	})
	require.NoError(t, err)
}

func TestNonHumanPrincipalApprovesTheStepItsRoleHolds(t *testing.T) {
	for _, test := range []struct {
		name      string
		actor     *store.UserMessage
		inProject bool
	}{
		{
			name:  "workspace service account bound in the workspace policy",
			actor: &store.UserMessage{Email: "integration@service.bytebase.com", Type: storepb.PrincipalType_SERVICE_ACCOUNT},
		},
		{
			name:      "workspace service account bound in the project policy",
			actor:     &store.UserMessage{Email: "integration@service.bytebase.com", Type: storepb.PrincipalType_SERVICE_ACCOUNT},
			inProject: true,
		},
		{
			name:      "project service account bound in the project policy",
			actor:     &store.UserMessage{Email: "testdt@project-a.service.bytebase.com", Type: storepb.PrincipalType_SERVICE_ACCOUNT},
			inProject: true,
		},
		{
			name:      "workload identity bound in the project policy",
			actor:     &store.UserMessage{Email: "deploy@project-a.workload.bytebase.com", Type: storepb.PrincipalType_WORKLOAD_IDENTITY},
			inProject: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			bindApproverRole(ctx, t, stores, test.inProject, common.FormatPrincipalMember(test.actor.Email, test.actor.Type))
			_, issue := createPendingDatabaseChangeApproval(ctx, t, stores, []string{serviceNowApproverRole})

			result, err := NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
				Workspace: "default",
				ProjectID: "project-a",
				IssueUID:  issue.UID,
				Actor:     test.actor,
				Action:    ActionApprove,
			})
			require.NoError(t, err)
			require.True(t, result.Approved)
			approvers := result.Issue.Payload.GetApproval().GetApprovers()
			require.Len(t, approvers, 1)
			require.True(t, proto.Equal(&storepb.IssuePayloadApproval_Approver{
				Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
				Principal: common.FormatUserEmail(test.actor.Email),
			}, approvers[0]), "the approver is recorded as users/{email} whatever its type, like the issue creator")
		})
	}
}

func TestNonHumanApproverIsRefusedLikeAPerson(t *testing.T) {
	serviceAccount := &store.UserMessage{Email: "integration@service.bytebase.com", Type: storepb.PrincipalType_SERVICE_ACCOUNT}

	for _, test := range []struct {
		name       string
		member     string
		creator    string
		wantReason ErrorReason
	}{
		{
			name:       "bound under the users/ prefix, which holds nothing for a service account",
			member:     common.FormatUserEmail(serviceAccount.Email),
			creator:    "creator@example.com",
			wantReason: ReasonApproverRoleRequired,
		},
		{
			name:       "created the issue it would approve",
			member:     common.FormatPrincipalMember(serviceAccount.Email, serviceAccount.Type),
			creator:    serviceAccount.Email,
			wantReason: ReasonUnspecified,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			bindApproverRole(ctx, t, stores, true, test.member)
			_, issue := createPendingDatabaseChangeApprovalBy(ctx, t, stores, test.creator, []string{serviceNowApproverRole})

			_, err := NewWorkflow(stores).ReviewIssue(ctx, IssueInput{
				Workspace: "default",
				ProjectID: "project-a",
				IssueUID:  issue.UID,
				Actor:     serviceAccount,
				Action:    ActionApprove,
			})

			var workflowErr *Error
			require.ErrorAs(t, err, &workflowErr)
			require.Equal(t, ErrorPermissionDenied, workflowErr.Code)
			require.Equal(t, test.wantReason, workflowErr.Reason)
		})
	}
}
