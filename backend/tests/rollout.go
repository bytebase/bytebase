package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	v1 "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// waitRollout waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitRollout(ctx context.Context, rolloutName string) error {
	// Sleep for 1 second between issues so that we don't get migration version conflict because we are using second-level timestamp for the version string. We choose sleep because it mimics the user's behavior.
	time.Sleep(1 * time.Second)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1.GetRolloutRequest{
			Name: rolloutName,
		})
		if err != nil {
			return err
		}
		var anyError error
		completed := true
		var runTasks []string
		for _, stage := range rollout.Stages {
			for _, task := range stage.Tasks {
				switch task.Status {
				case v1.Task_NOT_STARTED:
					runTasks = append(runTasks, task.Name)
					completed = false
				case v1.Task_DONE:
					continue
				case v1.Task_FAILED:
					resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, &v1.ListTaskRunsRequest{Parent: task.Name, PageSize: 1})
					if err != nil {
						return err
					}
					if len(resp.TaskRuns) > 0 {
						anyError = errors.Errorf(resp.TaskRuns[0].Detail)
					}
				default:
					completed = false
				}
			}
		}

		// Rollout tasks.
		if len(runTasks) > 0 {
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, &v1.BatchRunTasksRequest{
				Parent: fmt.Sprintf("%s/stages/-", rolloutName),
				Tasks:  runTasks,
			})
			if err != nil {
				return err
			}
		}

		if completed {
			return anyError
		}
	}
	return nil
}
