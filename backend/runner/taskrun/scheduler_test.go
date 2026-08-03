package taskrun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCompletionWebhookEnvironmentUsesLastEnvironmentOrder(t *testing.T) {
	tasks := []*store.TaskMessage{
		{ID: 1, Environment: "dev"},
		{ID: 2, Environment: "prod"},
		{ID: 3, Environment: "test"},
	}
	environmentOrderMap := common.EnvironmentOrderMap([]*storepb.EnvironmentSetting_Environment{
		{Id: "dev"},
		{Id: "test"},
		{Id: "prod"},
	})

	require.Equal(t, "prod", completionWebhookEnvironment(tasks, environmentOrderMap))
}

func TestSchedulePendingTaskRunsSkipsArchivedProject(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutCreatorStore(ctx, t)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "archived project plan",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	_, err = s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "unused",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	tasks, err := s.CreateMissingTasksTx(ctx, tx, plan.ProjectID, plan.UID, []*store.TaskMessage{{
		InstanceID: "unused",
		Type:       storepb.Task_TASK_TYPE_UNSPECIFIED,
		Payload:    &storepb.Task{},
	}})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.NoError(t, tx.Commit())
	require.NoError(t, s.CreatePendingTaskRuns(ctx, "", &store.TaskRunMessage{
		ProjectID: plan.ProjectID,
		TaskUID:   tasks[0].ID,
	}))

	archived := true
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: plan.ProjectID,
		Workspace:  "default",
		Delete:     &archived,
	}))

	b, err := bus.New()
	require.NoError(t, err)
	scheduler := &Scheduler{store: s, bus: b}
	require.NoError(t, scheduler.schedulePendingTaskRuns(ctx))

	taskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: plan.ProjectID})
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	require.Equal(t, storepb.TaskRun_PENDING, taskRuns[0].Status)

	archived = false
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: plan.ProjectID,
		Workspace:  "default",
		Delete:     &archived,
	}))
	require.NoError(t, scheduler.schedulePendingTaskRuns(ctx))

	taskRuns, err = s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: plan.ProjectID})
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	require.Equal(t, storepb.TaskRun_AVAILABLE, taskRuns[0].Status)
}
