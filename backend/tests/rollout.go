package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) changeDatabase(ctx context.Context, project *v1pb.Project, database *v1pb.Database, sheet *v1pb.Sheet, changeType v1pb.Plan_ChangeDatabaseConfig_Type) (*v1pb.Plan, *v1pb.Rollout, error) {
	plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Steps: []*v1pb.Plan_Step{
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
								ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
									Target: database.Name,
									Sheet:  sheet.Name,
									Type:   changeType,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create plan")
	}
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: project.Name, Plan: plan.Name})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create rollout")
	}
	_, err = ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       fmt.Sprintf("change database %s", database.Name),
			Description: fmt.Sprintf("change database %s", database.Name),
			Plan:        plan.Name,
			Rollout:     rollout.Name,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create issue")
	}
	err = ctl.waitRollout(ctx, rollout.Name)
	if err != nil {
		return nil, nil, err
	}
	return plan, rollout, nil
}

// waitRollout waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitRollout(ctx context.Context, rolloutName string) error {
	// Sleep for 1 second between issues so that we don't get migration version conflict because we are using second-level timestamp for the version string. We choose sleep because it mimics the user's behavior.
	time.Sleep(1 * time.Second)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{
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
				case v1pb.Task_NOT_STARTED:
					runTasks = append(runTasks, task.Name)
					completed = false
				case v1pb.Task_DONE:
					continue
				case v1pb.Task_FAILED:
					resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, &v1pb.ListTaskRunsRequest{Parent: task.Name, PageSize: 1})
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
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, &v1pb.BatchRunTasksRequest{
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
