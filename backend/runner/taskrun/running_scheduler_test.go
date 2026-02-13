package taskrun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

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

func TestValidateTaskFreshness_DatabaseCreateSkipsValidation(t *testing.T) {
	// Construct a Scheduler with nil store. If validateTaskFreshness tries to
	// call the store for DATABASE_CREATE tasks, it will panic — proving the
	// early return works.
	s := &Scheduler{}
	task := &store.TaskMessage{
		Type: storepb.Task_DATABASE_CREATE,
	}
	err := s.validateTaskFreshness(context.Background(), task)
	require.NoError(t, err)
}
