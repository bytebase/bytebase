package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCreateRolloutAddsMissingTasksAfterRolloutExists(t *testing.T) {
	ctx := context.Background()
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "plan-a",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)
	workflow := NewWorkflow(stores)

	result, err := workflow.CreateRollout(ctx, CreateRolloutInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		BuildTasks: func(context.Context, *store.PlanMessage, *store.ProjectMessage) ([]*store.TaskMessage, error) {
			return []*store.TaskMessage{newRolloutReviewTask("database-a", "sheet-a")}, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)

	result, err = workflow.CreateRollout(ctx, CreateRolloutInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		BuildTasks: func(context.Context, *store.PlanMessage, *store.ProjectMessage) ([]*store.TaskMessage, error) {
			return []*store.TaskMessage{
				newRolloutReviewTask("database-a", "sheet-a"),
				newRolloutReviewTask("database-b", "sheet-b"),
			}, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)
	require.Equal(t, "database-b", result.Tasks[0].GetDatabaseName())
}

func newRolloutReviewTask(databaseName, sheetSHA256 string) *store.TaskMessage {
	return &store.TaskMessage{
		InstanceID:   "instance-a",
		DatabaseName: &databaseName,
		Type:         storepb.Task_DATABASE_MIGRATE,
		Payload: &storepb.Task{
			Source: &storepb.Task_SheetSha256{SheetSha256: sheetSHA256},
		},
	}
}
