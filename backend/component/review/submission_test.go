package review

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSubmitIssueRejectsStateChangedAfterProposal(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(context.Context, *testing.T, *store.Store)
		mutate     func(context.Context, *testing.T, *store.Store, *store.PlanMessage, *store.IssueMessage)
		errorCode  ErrorCode
		withLabels bool
	}{
		{
			name: "required labels removed",
			configure: func(ctx context.Context, t *testing.T, stores *store.Store) {
				require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
					Workspace: "default", ResourceID: "project-a",
					Setting: &storepb.Project{ForceIssueLabels: true},
				}))
			},
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, _ *store.PlanMessage, issue *store.IssueMessage) {
				labels := []string(nil)
				_, err := NewWorkflow(stores).UpdateIssueMetadata(ctx, UpdateIssueMetadataInput{
					Workspace: "default", ProjectID: "project-a", IssueUID: issue.UID, Labels: &labels,
				})
				require.NoError(t, err)
			},
			errorCode:  ErrorInvalidAction,
			withLabels: true,
		},
		{
			name: "Plan closed",
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage, _ *store.IssueMessage) {
				deleted := true
				_, err := NewWorkflow(stores).UpdatePlan(ctx, UpdatePlanInput{
					Workspace: "default", ProjectID: "project-a", PlanUID: plan.UID, Deleted: &deleted,
				})
				require.NoError(t, err)
			},
			errorCode: ErrorConflict,
		},
		{
			name: "Plan title cleared",
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage, _ *store.IssueMessage) {
				title := ""
				_, err := NewWorkflow(stores).UpdatePlan(ctx, UpdatePlanInput{
					Workspace: "default", ProjectID: "project-a", PlanUID: plan.UID, Title: &title,
				})
				require.NoError(t, err)
			},
			errorCode: ErrorInvalidAction,
		},
		{
			name: "issue status changed",
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, _ *store.PlanMessage, issue *store.IssueMessage) {
				status := storepb.Issue_CANCELED
				_, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{Status: &status})
				require.NoError(t, err)
			},
			errorCode: ErrorConflict,
		},
		{
			name: "Plan checks rerun",
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage, _ *store.IssueMessage) {
				created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
					ProjectID: "project-a", PlanUID: plan.UID,
					Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: plan.Config.GetApprovalInputVersion()},
				})
				require.NoError(t, err)
				require.True(t, created)
			},
			errorCode: ErrorFailedPrecondition,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			if test.configure != nil {
				test.configure(ctx, t, stores)
			}
			plan, issue := createReadyDraft(ctx, t, stores, test.withLabels)
			workflow := NewWorkflow(stores)
			workflow.beforeSubmit = func() { test.mutate(ctx, t, stores, plan, issue) }

			_, err := workflow.SubmitIssue(ctx, SubmitIssueInput{
				Workspace: "default", ProjectID: "project-a", IssueUID: issue.UID,
			})
			var workflowErr *Error
			require.True(t, errors.As(err, &workflowErr))
			require.Equal(t, test.errorCode, workflowErr.Code)
			got, getErr := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
			require.NoError(t, getErr)
			require.True(t, got.Payload.GetDraft())
		})
	}
}

func TestConcurrentSubmitIssueEmitsEffectsOnce(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	_, issue := createReadyDraft(ctx, t, stores, false)
	workflow := NewWorkflow(stores)
	var ready sync.WaitGroup
	ready.Add(2)
	release := make(chan struct{})
	workflow.beforeSubmit = func() {
		ready.Done()
		<-release
	}

	type outcome struct {
		result *SubmitIssueResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	var calls sync.WaitGroup
	for range 2 {
		calls.Go(func() {
			result, err := workflow.SubmitIssue(ctx, SubmitIssueInput{
				Workspace: "default", ProjectID: "project-a", IssueUID: issue.UID,
			})
			outcomes <- outcome{result: result, err: err}
		})
	}
	ready.Wait()
	close(release)
	calls.Wait()
	close(outcomes)

	var effects int
	for outcome := range outcomes {
		require.NoError(t, outcome.err)
		effects += len(outcome.result.Events)
	}
	require.Equal(t, 3, effects)
}

func TestUpdatePlanSynchronizesDraftMetadata(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, issue := createReadyDraft(ctx, t, stores, false)
	title := "renamed draft"
	description := "updated description"
	deleted := true
	result, err := NewWorkflow(stores).UpdatePlan(ctx, UpdatePlanInput{
		Workspace: "default", ProjectID: "project-a", PlanUID: plan.UID,
		Title: &title, Description: &description, Deleted: &deleted,
	})
	require.NoError(t, err)
	require.Equal(t, issue.UID, result.Issue.UID)
	require.Equal(t, title, result.Issue.Title)
	require.Equal(t, description, result.Issue.Description)
	require.Equal(t, storepb.Issue_CANCELED, result.Issue.Status)
	require.True(t, result.Issue.Payload.GetDraft())
}

func TestApprovalFindingLabelFreshnessMatrix(t *testing.T) {
	tests := []struct {
		name           string
		spec           *storepb.PlanConfig_Spec
		expectConflict bool
	}{
		{
			name: "change database depends on labels",
			spec: &storepb.PlanConfig_Spec{Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
			}},
			expectConflict: true,
		},
		{
			name: "create database ignores labels",
			spec: &storepb.PlanConfig_Spec{Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
				ProjectID: "project-a", Name: "database Plan",
				Config: &storepb.PlanConfig{ApprovalInputVersion: 2, Specs: []*storepb.PlanConfig_Spec{test.spec}},
			}, "creator@example.com")
			require.NoError(t, err)
			issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
				ProjectID: "project-a", CreatorEmail: "creator@example.com", Title: plan.Name,
				Type: storepb.Issue_DATABASE_CHANGE, PlanUID: &plan.UID,
				Payload: &storepb.Issue{
					Labels:   []string{"before"},
					Approval: &storepb.IssuePayloadApproval{ApprovalInputVersion: 2},
				},
			})
			require.NoError(t, err)

			evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
			evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
				issue.Payload.Approval = &storepb.IssuePayloadApproval{ApprovalFindingDone: true, ApprovalInputVersion: 2}
				return nil
			}
			evaluator.beforeCommit = func() {
				labels := []string{"after"}
				_, updateErr := NewWorkflow(stores).UpdateIssueMetadata(ctx, UpdateIssueMetadataInput{
					Workspace: "default", ProjectID: "project-a", IssueUID: issue.UID, Labels: &labels,
				})
				require.NoError(t, updateErr)
			}
			result, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
				Workspace: "default", ProjectID: "project-a", IssueUID: issue.UID,
			})
			if test.expectConflict {
				var workflowErr *Error
				require.True(t, errors.As(err, &workflowErr))
				require.Equal(t, ErrorConflict, workflowErr.Code)
				require.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.True(t, result.Applied)
			require.Equal(t, []string{"after"}, result.Issue.Payload.GetLabels())
		})
	}
}

func createReadyDraft(ctx context.Context, t *testing.T, stores *store.Store, withLabels bool) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change database",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "change",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets:     []string{"instances/instance-a/databases/database-a"},
						SheetSha256: "sha256",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	labels := []string(nil)
	if withLabels {
		labels = []string{"environment:prod"}
	}
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID: "project-a", CreatorEmail: "creator@example.com", Title: plan.Name,
		Type: storepb.Issue_DATABASE_CHANGE, PlanUID: &plan.UID,
		Payload: &storepb.Issue{Draft: true, Labels: labels, Approval: &storepb.IssuePayloadApproval{}},
	})
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a", PlanUID: plan.UID,
		Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)
	run, err := stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NoError(t, stores.UpdatePlanCheckRun(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 2,
	}, run.UID))
	return plan, issue
}
