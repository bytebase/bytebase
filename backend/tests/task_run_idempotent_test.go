package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

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
			Environment: new("environments/prod"),
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
	runTime := timestamppb.New(time.Now().Add(time.Hour))

	for range 5 {
		go func() {
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
				Parent:  stage.Name,
				Tasks:   taskNames,
				RunTime: runTime,
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
		Parent:  stage.Name,
		Tasks:   taskNames,
		RunTime: runTime,
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

	_, err = ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
		Parent: stage.Name,
		Tasks:  taskNames,
		Reason: "must not skip pending tasks",
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestBatchRunTasks_RejectsSkippedTasks(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "testInstanceSkippedTask")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "testInstanceSkippedTask",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)

	const databaseName = "testSkippedTaskDb"
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil /* environment */, databaseName, ""))
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, databaseName),
	}))
	a.NoError(err)

	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
	}))
	a.NoError(err)

	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{Specs: []*v1pb.Plan_Spec{{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
					Targets: []string{databaseResp.Msg.Name},
					Sheet:   sheetResp.Msg.Name,
				},
			},
		}}},
	}))
	a.NoError(err)

	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: planResp.Msg.Name,
	}))
	a.NoError(err)
	stage := rolloutResp.Msg.Stages[0]
	task := stage.Tasks[0]

	_, err = ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
		Parent: stage.Name,
		Tasks:  []string{task.Name},
		Reason: "not needed",
	}))
	a.NoError(err)

	_, err = ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
		Parent: stage.Name,
		Tasks:  []string{task.Name},
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	_, err = ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
		Parent: stage.Name,
		Tasks:  []string{task.Name},
		Reason: "replacement reason",
	}))
	a.NoError(err)
	rolloutResp, err = ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
		Name: rolloutResp.Msg.Name,
	}))
	a.NoError(err)
	a.Equal(v1pb.Task_SKIPPED, rolloutResp.Msg.Stages[0].Tasks[0].Status)
	a.Equal("not needed", rolloutResp.Msg.Stages[0].Tasks[0].SkippedReason)

	taskRunsResp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
		Parent: task.Name,
	}))
	a.NoError(err)
	a.Empty(taskRunsResp.Msg.TaskRuns)
}
