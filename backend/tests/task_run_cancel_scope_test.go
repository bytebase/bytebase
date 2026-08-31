package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestBatchCancelTaskRuns_RejectsTaskRunFromAnotherStage pins T18a.
//
// BatchCancelTaskRuns authorizes the caller against the stage named in the
// request, and the stage segment of a task run name is caller-supplied. It must
// therefore confine the task runs it acts on to that stage; otherwise cancel
// rights in one environment reach a task run in another environment of the same
// project — test-environment rights killing a production migration.
func TestBatchCancelTaskRuns_RejectsTaskRunFromAnotherStage(t *testing.T) {
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	testEnvironment, err := ctl.getEnvironment(ctx, "test")
	a.NoError(err)

	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "testInstanceCancelScope",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	// One database per environment, so the rollout gets a test stage and a prod stage.
	const prodDatabase, testDatabase = "cancelScopeProdDb", "cancelScopeTestDb"
	a.NoError(ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, prodDatabase, ""))
	a.NoError(ctl.createDatabase(ctx, ctl.project, instance, testEnvironment, testDatabase, ""))

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
					Targets: []string{
						fmt.Sprintf("%s/databases/%s", instance.Name, testDatabase),
						fmt.Sprintf("%s/databases/%s", instance.Name, prodDatabase),
					},
					Sheet: sheetResp.Msg.Name,
				},
			},
		}}},
	}))
	a.NoError(err)

	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: planResp.Msg.Name,
	}))
	a.NoError(err)

	stageByID := map[string]*v1pb.Stage{}
	for _, stage := range rolloutResp.Msg.Stages {
		stageByID[stage.Id] = stage
	}
	testStage, ok := stageByID["test"]
	a.True(ok, "rollout should have a test stage")
	prodStage, ok := stageByID["prod"]
	a.True(ok, "rollout should have a prod stage")

	// Park a task run in each stage. A run time in the future leaves them PENDING,
	// which is cancelable, without executing anything.
	runTime := timestamppb.New(time.Now().Add(time.Hour))
	for _, stage := range []*v1pb.Stage{testStage, prodStage} {
		var taskNames []string
		for _, task := range stage.Tasks {
			taskNames = append(taskNames, task.Name)
		}
		a.NotEmpty(taskNames, "stage %s should have tasks", stage.Id)
		_, err = ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
			Parent:  stage.Name,
			Tasks:   taskNames,
			RunTime: runTime,
		}))
		a.NoError(err)
	}

	prodTaskRun := onlyTaskRunInStage(ctx, a, ctl, prodStage)
	a.Equal(v1pb.TaskRun_PENDING, prodTaskRun.Status)

	// The attack: address the prod task run as though it lived in the test stage.
	// Only the stage segment changes; the task run ID still resolves in the project.
	spoofed := testStage.Name + strings.TrimPrefix(prodTaskRun.Name, prodStage.Name)
	a.NotEqual(prodTaskRun.Name, spoofed, "spoofed name should differ from the real one")

	_, err = ctl.rolloutServiceClient.BatchCancelTaskRuns(ctx, connect.NewRequest(&v1pb.BatchCancelTaskRunsRequest{
		Parent:   testStage.Name + "/tasks/-",
		TaskRuns: []string{spoofed},
	}))
	a.Error(err, "canceling a prod task run through the test stage must be rejected")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	a.Equal(v1pb.TaskRun_PENDING, onlyTaskRunInStage(ctx, a, ctl, prodStage).Status,
		"the prod task run must survive the rejected cancel")

	// The legitimate path still works: the test stage can cancel its own task run.
	testTaskRun := onlyTaskRunInStage(ctx, a, ctl, testStage)
	_, err = ctl.rolloutServiceClient.BatchCancelTaskRuns(ctx, connect.NewRequest(&v1pb.BatchCancelTaskRunsRequest{
		Parent:   testStage.Name + "/tasks/-",
		TaskRuns: []string{testTaskRun.Name},
	}))
	a.NoError(err)
	a.Equal(v1pb.TaskRun_CANCELED, onlyTaskRunInStage(ctx, a, ctl, testStage).Status)
}

func onlyTaskRunInStage(ctx context.Context, a *require.Assertions, ctl *controller, stage *v1pb.Stage) *v1pb.TaskRun {
	resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
		Parent: stage.Name + "/tasks/-",
	}))
	a.NoError(err)
	a.Len(resp.Msg.TaskRuns, 1, "stage %s should have exactly one task run", stage.Id)
	return resp.Msg.TaskRuns[0]
}
