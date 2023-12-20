package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) changeDatabase(ctx context.Context, project *v1pb.Project, database *v1pb.Database, sheet *v1pb.Sheet, changeType v1pb.Plan_ChangeDatabaseConfig_Type) error {
	_, _, _, err := ctl.changeDatabaseWithConfig(ctx, project, []*v1pb.Plan_Step{
		{
			Specs: []*v1pb.Plan_Spec{
				{
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Target:        database.Name,
							Sheet:         sheet.Name,
							Type:          changeType,
							SchemaVersion: uuid.NewString(),
						},
					},
				},
			},
		},
	},
	)
	return err
}

func (ctl *controller) changeDatabaseWithConfig(ctx context.Context, project *v1pb.Project, steps []*v1pb.Plan_Step) (*v1pb.Plan, *v1pb.Rollout, *v1pb.Issue, error) {
	plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Steps: steps,
		},
	})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to create plan")
	}
	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       "change database",
			Description: "change database",
			Plan:        plan.Name,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to create issue")
	}
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: project.Name, Plan: plan.Name})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to create rollout")
	}
	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	if err != nil {
		return nil, nil, nil, err
	}
	return plan, rollout, issue, nil
}

// waitRollout waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitRollout(ctx context.Context, issueName, rolloutName string) error {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		issue, err := ctl.issueServiceClient.GetIssue(ctx, &v1pb.GetIssueRequest{Name: issueName})
		if err != nil {
			return err
		}
		if issue.ApprovalFindingError != "" {
			return errors.Errorf("approval finding error: %v", issue.ApprovalFindingError)
		}
		if issue.ApprovalFindingDone {
			break
		}
	}

	rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{
		Name: rolloutName,
	})
	if err != nil {
		return err
	}
	for _, stage := range rollout.Stages {
		var runTasks []string
		for _, task := range stage.Tasks {
			if task.Status == v1pb.Task_NOT_STARTED {
				runTasks = append(runTasks, task.Name)
			}
		}
		if len(runTasks) > 0 {
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, &v1pb.BatchRunTasksRequest{
				Parent: fmt.Sprintf("%s/stages/-", rolloutName),
				Tasks:  runTasks,
			})
			if err != nil {
				return err
			}
		}
	}

	for range ticker.C {
		rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{
			Name: rolloutName,
		})
		if err != nil {
			return err
		}
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
				case v1pb.Task_SKIPPED:
					continue
				case v1pb.Task_FAILED:
					resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, &v1pb.ListTaskRunsRequest{Parent: task.Name, PageSize: 1})
					if err != nil {
						return err
					}
					if len(resp.TaskRuns) > 0 {
						return errors.Errorf(resp.TaskRuns[0].Detail)
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
			break
		}
	}
	return nil
}

// rolloutAndWaitTask rollouts one task in the rollout.
func (ctl *controller) rolloutAndWaitTask(ctx context.Context, issueName, rolloutName string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		issue, err := ctl.issueServiceClient.GetIssue(ctx, &v1pb.GetIssueRequest{Name: issueName})
		if err != nil {
			return err
		}
		if issue.ApprovalFindingError != "" {
			return errors.Errorf("approval finding error: %v", issue.ApprovalFindingError)
		}
		if issue.ApprovalFindingDone {
			break
		}
	}

	rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{
		Name: rolloutName,
	})
	if err != nil {
		return err
	}
	var foundTask string
	for _, stage := range rollout.Stages {
		for _, task := range stage.Tasks {
			if task.Status == v1pb.Task_NOT_STARTED {
				_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, &v1pb.BatchRunTasksRequest{
					Parent: fmt.Sprintf("%s/stages/-", rolloutName),
					Tasks:  []string{task.Name},
				})
				if err != nil {
					return err
				}
				foundTask = task.Name
				break
			}
		}
		if foundTask != "" {
			break
		}
	}
	if foundTask == "" {
		return errors.Errorf("found no task to rollout")
	}

	for range ticker.C {
		rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{
			Name: rolloutName,
		})
		if err != nil {
			return err
		}

		for _, stage := range rollout.Stages {
			for _, task := range stage.Tasks {
				if task.Name != foundTask {
					continue
				}
				switch task.Status {
				case v1pb.Task_DONE:
					return nil
				case v1pb.Task_SKIPPED:
					return nil
				case v1pb.Task_FAILED:
					resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, &v1pb.ListTaskRunsRequest{Parent: task.Name, PageSize: 1})
					if err != nil {
						return err
					}
					if len(resp.TaskRuns) > 0 {
						return errors.Errorf(resp.TaskRuns[0].Detail)
					}
				}
			}
		}
	}
	return nil
}
