package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestBatchRunTasks_Idempotent verifies that BatchRunTasks is idempotent and safe for concurrent calls.
// This test covers both sequential double-clicks and concurrent requests scenarios.
func TestBatchRunTasks_Idempotent(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Provision an instance
	instanceRootDir := t.TempDir()
	instanceName := "testInstanceIdempotent"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	// Create a database
	databaseName := "testIdempotentDb"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, databaseName, "")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	// Create a sheet with SQL
	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte("SELECT 1;"),
		},
	}))
	a.NoError(err)
	sheet := sheetResp.Msg

	// Create a plan with change database config
	spec := &v1pb.Plan_Spec{
		Id: uuid.NewString(),
		Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
				Targets: []string{database.Name},
				Sheet:   sheet.Name,
			},
		},
	}

	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{spec},
		},
	}))
	a.NoError(err)
	plan := planResp.Msg

	// Create a rollout
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: plan.Name,
	}))
	a.NoError(err)
	rollout := rolloutResp.Msg

	// Get the tasks from the rollout
	a.GreaterOrEqual(len(rollout.Stages), 1, "rollout should have at least one stage")
	stage := rollout.Stages[0]
	a.GreaterOrEqual(len(stage.Tasks), 1, "stage should have at least one task")

	var taskNames []string
	for _, task := range stage.Tasks {
		taskNames = append(taskNames, task.Name)
	}

	// Test 1: Concurrent calls (hardest case - simulates rapid multiple clicks)
	// Make 5 concurrent BatchRunTasks calls
	type result struct {
		err error
	}
	results := make(chan result, 5)

	for range 5 {
		go func() {
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
				Parent: stage.Name,
				Tasks:  taskNames,
			}))
			results <- result{err: err}
		}()
	}

	// Collect results - all should succeed
	for i := range 5 {
		res := <-results
		a.NoError(res.err, "concurrent BatchRunTasks call %d should succeed", i+1)
	}

	// Verify no duplicate task runs were created despite concurrent requests
	for _, taskName := range taskNames {
		taskRunsResp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
			Parent: taskName,
		}))
		a.NoError(err)
		a.Equal(1, len(taskRunsResp.Msg.TaskRuns), "task %s should have exactly one task run after concurrent calls", taskName)
	}

	// Test 2: Sequential double-click (simulates user accidentally clicking twice)
	_, err = ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
		Parent: stage.Name,
		Tasks:  taskNames,
	}))
	a.NoError(err, "sequential BatchRunTasks call should succeed")

	// Verify still exactly one task run per task
	for _, taskName := range taskNames {
		taskRunsResp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
			Parent: taskName,
		}))
		a.NoError(err)
		a.Equal(1, len(taskRunsResp.Msg.TaskRuns), "task %s should still have exactly one task run after sequential call", taskName)
	}
}
