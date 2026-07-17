package review

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCreateDraftIssueRevalidatesPlanAfterProposal(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(context.Context, *testing.T, *store.Store, *store.PlanMessage)
		errorCode ErrorCode
	}{
		{
			name: "Plan closed",
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage) {
				deleted := true
				_, err := NewWorkflow(stores).UpdatePlan(ctx, UpdatePlanInput{
					Workspace: "default", ProjectID: plan.ProjectID, PlanUID: plan.UID, Deleted: &deleted,
				})
				require.NoError(t, err)
			},
			errorCode: ErrorFailedPrecondition,
		},
		{
			name: "Plan kind changed",
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage) {
				specs := []*storepb.PlanConfig_Spec{{
					Config: &storepb.PlanConfig_Spec_ExportDataConfig{
						ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{},
					},
				}}
				_, err := NewWorkflow(stores).UpdatePlan(ctx, UpdatePlanInput{
					Workspace: "default", ProjectID: plan.ProjectID, PlanUID: plan.UID, Specs: &specs,
				})
				require.NoError(t, err)
			},
			errorCode: ErrorInvalidAction,
		},
		{
			name: "rollout started",
			mutate: func(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage) {
				_, err := NewWorkflow(stores).CreateRollout(ctx, CreateRolloutInput{
					Workspace: "default", ProjectID: plan.ProjectID, PlanUID: plan.UID,
					BuildTasks: func(context.Context, *store.PlanMessage, *store.ProjectMessage) ([]*store.TaskMessage, error) {
						return nil, nil
					},
				})
				require.NoError(t, err)
			},
			errorCode: ErrorFailedPrecondition,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			stores := setupWorkflowStore(ctx, t)
			plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
				ProjectID: "project-a", Name: "draft Plan", Description: "from Plan",
				Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
					},
				}}},
			}, "creator@example.com")
			require.NoError(t, err)
			workflow := NewWorkflow(stores)
			workflow.beforeCreateDraft = func() { test.mutate(ctx, t, stores, plan) }

			result, err := workflow.CreateDraftIssue(ctx, CreateDraftIssueInput{
				Workspace: "default",
				Issue: &store.IssueMessage{
					ProjectID: plan.ProjectID, CreatorEmail: "creator@example.com", PlanUID: &plan.UID,
					Title: "stale request title", Type: storepb.Issue_DATABASE_CHANGE,
					Payload: &storepb.Issue{Draft: true, Approval: &storepb.IssuePayloadApproval{}},
				},
			})
			require.Nil(t, result)
			var workflowErr *Error
			require.True(t, errors.As(err, &workflowErr))
			require.Equal(t, test.errorCode, workflowErr.Code)
			issue, getErr := stores.GetIssue(ctx, &store.FindIssueMessage{
				ProjectIDs: []string{plan.ProjectID}, PlanUID: &plan.UID,
			})
			require.NoError(t, getErr)
			require.Nil(t, issue)
		})
	}
}

func TestCreateDraftIssueUsesLockedPlanMetadata(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a", Name: "Plan title", Description: "Plan description",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)
	workflow := NewWorkflow(stores)
	workflow.beforeCreateDraft = func() {
		title := "Updated Plan title"
		description := "Updated Plan description"
		_, updateErr := NewWorkflow(stores).UpdatePlan(ctx, UpdatePlanInput{
			Workspace: "default", ProjectID: plan.ProjectID, PlanUID: plan.UID,
			Title: &title, Description: &description,
		})
		require.NoError(t, updateErr)
	}
	result, err := workflow.CreateDraftIssue(ctx, CreateDraftIssueInput{
		Workspace: "default",
		Issue: &store.IssueMessage{
			ProjectID: plan.ProjectID, CreatorEmail: "creator@example.com", PlanUID: &plan.UID,
			Title: "request title", Description: "request description", Type: storepb.Issue_DATABASE_CHANGE,
			Payload: &storepb.Issue{Draft: true, Approval: &storepb.IssuePayloadApproval{}},
		},
	})
	require.NoError(t, err)
	require.True(t, result.Created)
	require.Equal(t, "Updated Plan title", result.Issue.Title)
	require.Equal(t, "Updated Plan description", result.Issue.Description)
}
