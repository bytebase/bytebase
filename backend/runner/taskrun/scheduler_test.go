package taskrun

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/productmetrics"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestTaskCyclesRecordEmptySuccess(t *testing.T) {
	ctx := context.Background()
	stores := setupRolloutCreatorStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	metrics := productmetrics.New(nil, nil)
	scheduler := &Scheduler{
		store:          stores,
		bus:            b,
		profile:        &config.Profile{ReplicaID: "replica-a"},
		productMetrics: metrics,
	}

	require.NoError(t, scheduler.schedulePendingTaskRuns(ctx))
	require.NoError(t, scheduler.scheduleRunningTaskRuns(ctx))
	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	acquired, err := store.TryAdvisoryXactLock(ctx, tx, store.AdvisoryLockKeyPendingScheduler)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, scheduler.schedulePendingTaskRuns(ctx))
	require.NoError(t, tx.Rollback())
	panicScheduler := &Scheduler{productMetrics: metrics}
	require.Error(t, panicScheduler.schedulePendingTaskRuns(ctx))
	require.Error(t, panicScheduler.scheduleRunningTaskRuns(ctx))
	require.Equal(t, uint64(1), runnerRunCount(t, metrics, productmetrics.RunnerTaskPending, productmetrics.ResultSuccess))
	require.Equal(t, uint64(1), runnerRunCount(t, metrics, productmetrics.RunnerTaskDispatch, productmetrics.ResultSuccess))
	require.Equal(t, uint64(1), runnerRunCount(t, metrics, productmetrics.RunnerTaskPending, productmetrics.ResultFailure))
	require.Equal(t, uint64(1), runnerRunCount(t, metrics, productmetrics.RunnerTaskDispatch, productmetrics.ResultFailure))
	require.Equal(t, uint64(1), runnerRunCount(t, metrics, productmetrics.RunnerTaskPending, productmetrics.ResultSkipped))

	canceledMetrics := productmetrics.New(nil, nil)
	scheduler.productMetrics = canceledMetrics
	canceledContext, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, scheduler.schedulePendingTaskRuns(canceledContext))
	require.Error(t, scheduler.scheduleRunningTaskRuns(canceledContext))
	require.Zero(t, runnerRunCount(t, canceledMetrics, productmetrics.RunnerTaskPending, productmetrics.ResultFailure))
	require.Zero(t, runnerRunCount(t, canceledMetrics, productmetrics.RunnerTaskDispatch, productmetrics.ResultFailure))
}

func runnerRunCount(t *testing.T, metrics *productmetrics.ProductMetrics, runner productmetrics.Runner, result productmetrics.RunnerResult) uint64 {
	t.Helper()
	ch := make(chan prometheus.Metric, 16)
	go func() {
		metrics.Collect(ch)
		close(ch)
	}()

	var count uint64
	for metric := range ch {
		var dtoMetric dto.Metric
		if metric.Write(&dtoMetric) != nil {
			continue
		}
		labels := map[string]string{}
		for _, label := range dtoMetric.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}
		if labels["runner"] == string(runner) && labels["result"] == string(result) {
			count += dtoMetric.GetHistogram().GetSampleCount()
		}
	}
	return count
}

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

func TestSchedulePendingTaskRunsSkipsArchivedInstance(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutCreatorStore(ctx, t)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "archived instance plan",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	instanceID := "instance-a"
	_, err = s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: instanceID,
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
		InstanceID: instanceID,
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

	// Seed a legacy state that predates the archive precondition. New archive
	// requests reject PENDING task runs, but the scheduler must still avoid
	// promoting existing inconsistent rows.
	_, err = s.GetDB().ExecContext(ctx, `
		UPDATE instance SET deleted = TRUE WHERE resource_id = $1
	`, instanceID)
	require.NoError(t, err)

	b, err := bus.New()
	require.NoError(t, err)
	scheduler := &Scheduler{store: s, bus: b}
	require.NoError(t, scheduler.schedulePendingTaskRuns(ctx))

	taskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: plan.ProjectID})
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	require.Equal(t, storepb.TaskRun_PENDING, taskRuns[0].Status, "an archived instance's pending task run must not be promoted")
}
