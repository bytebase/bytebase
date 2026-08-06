package taskrun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCheckTaskDrift(t *testing.T) {
	prodEnv := "prod"
	stagingEnv := "staging"
	dbName := "mydb"

	tests := []struct {
		name        string
		task        *store.TaskMessage
		database    *store.DatabaseMessage
		plan        *store.PlanMessage
		wantErr     bool
		errContains string
	}{
		{
			name: "no drift",
			task: &store.TaskMessage{
				InstanceID:   "inst-1",
				DatabaseName: &dbName,
				Environment:  "prod",
				Type:         storepb.Task_DATABASE_MIGRATE,
			},
			database: &store.DatabaseMessage{
				ProjectID:              "proj-1",
				EffectiveEnvironmentID: &prodEnv,
			},
			plan: &store.PlanMessage{
				ProjectID: "proj-1",
			},
			wantErr: false,
		},
		{
			name: "project drift",
			task: &store.TaskMessage{
				InstanceID:   "inst-1",
				DatabaseName: &dbName,
				Environment:  "prod",
				Type:         storepb.Task_DATABASE_MIGRATE,
			},
			database: &store.DatabaseMessage{
				ProjectID:              "proj-other",
				EffectiveEnvironmentID: &prodEnv,
			},
			plan: &store.PlanMessage{
				ProjectID: "proj-original",
			},
			wantErr:     true,
			errContains: "project",
		},
		{
			name: "environment drift",
			task: &store.TaskMessage{
				InstanceID:   "inst-1",
				DatabaseName: &dbName,
				Environment:  "prod",
				Type:         storepb.Task_DATABASE_MIGRATE,
			},
			database: &store.DatabaseMessage{
				ProjectID:              "proj-1",
				EffectiveEnvironmentID: &stagingEnv,
			},
			plan: &store.PlanMessage{
				ProjectID: "proj-1",
			},
			wantErr:     true,
			errContains: "environment",
		},
		{
			name: "project and environment drift returns project error first",
			task: &store.TaskMessage{
				InstanceID:   "inst-1",
				DatabaseName: &dbName,
				Environment:  "prod",
				Type:         storepb.Task_DATABASE_MIGRATE,
			},
			database: &store.DatabaseMessage{
				ProjectID:              "proj-other",
				EffectiveEnvironmentID: &stagingEnv,
			},
			plan: &store.PlanMessage{
				ProjectID: "proj-original",
			},
			wantErr:     true,
			errContains: "project",
		},
		{
			name: "empty task environment skips env check",
			task: &store.TaskMessage{
				InstanceID:   "inst-1",
				DatabaseName: &dbName,
				Environment:  "",
				Type:         storepb.Task_DATABASE_MIGRATE,
			},
			database: &store.DatabaseMessage{
				ProjectID:              "proj-1",
				EffectiveEnvironmentID: &stagingEnv,
			},
			plan: &store.PlanMessage{
				ProjectID: "proj-1",
			},
			wantErr: false,
		},
		{
			name: "nil effective environment skips env check",
			task: &store.TaskMessage{
				InstanceID:   "inst-1",
				DatabaseName: &dbName,
				Environment:  "prod",
				Type:         storepb.Task_DATABASE_MIGRATE,
			},
			database: &store.DatabaseMessage{
				ProjectID:              "proj-1",
				EffectiveEnvironmentID: nil,
			},
			plan: &store.PlanMessage{
				ProjectID: "proj-1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTaskDrift(tt.task, tt.database, tt.plan)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateTaskFreshnessChecksInstanceArchival(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutCreatorStore(ctx, t)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "freshness validation plan",
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
		Type:       storepb.Task_DATABASE_CREATE,
		Payload:    &storepb.Task{},
	}})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.NoError(t, tx.Commit())

	scheduler := &Scheduler{store: s}
	// DATABASE_CREATE has no database target yet, but its target instance must
	// still be live for the task run to be allowed to start.
	require.NoError(t, scheduler.validateTaskFreshness(ctx, tasks[0]), "a live instance should pass freshness validation")

	// Seed a legacy state to keep freshness validation defensive for archived
	// DATABASE_CREATE tasks that have no database target yet.
	_, err = s.GetDB().ExecContext(ctx, `
		UPDATE instance SET deleted = TRUE WHERE resource_id = $1
	`, instanceID)
	require.NoError(t, err)
	err = scheduler.validateTaskFreshness(ctx, tasks[0])
	require.Error(t, err)
	require.Contains(t, err.Error(), "archived", "an archived instance must block a DATABASE_CREATE task from starting")
}

func TestScheduleRunningTaskRunsSkipsArchivedInstance(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutCreatorStore(ctx, t)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "archived instance claim plan",
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
		Type:       storepb.Task_DATABASE_CREATE,
		Payload:    &storepb.Task{},
	}})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.NoError(t, tx.Commit())
	require.NoError(t, s.CreatePendingTaskRuns(ctx, "", &store.TaskRunMessage{
		ProjectID: plan.ProjectID,
		TaskUID:   tasks[0].ID,
	}))
	taskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: plan.ProjectID})
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	_, err = s.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:              taskRuns[0].ID,
		ProjectID:       plan.ProjectID,
		Status:          storepb.TaskRun_AVAILABLE,
		AllowedStatuses: []storepb.TaskRun_Status{storepb.TaskRun_PENDING},
	})
	require.NoError(t, err)

	// Seed a legacy state that predates the archive precondition. New archive
	// requests reject AVAILABLE task runs, but claiming must remain defensive
	// for existing inconsistent rows.
	_, err = s.GetDB().ExecContext(ctx, `
		UPDATE instance SET deleted = TRUE WHERE resource_id = $1
	`, instanceID)
	require.NoError(t, err)

	b, err := bus.New()
	require.NoError(t, err)
	scheduler := &Scheduler{store: s, bus: b, profile: &config.Profile{ReplicaID: "replica-a"}}
	require.NoError(t, scheduler.scheduleRunningTaskRuns(ctx))

	taskRuns, err = s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: plan.ProjectID})
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	require.Equal(t, storepb.TaskRun_AVAILABLE, taskRuns[0].Status, "an archived instance's available task run must not be claimed")
}
