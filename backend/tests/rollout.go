package tests

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func (ctl *controller) changeDatabase(ctx context.Context, project *v1pb.Project, database *v1pb.Database, sheet *v1pb.Sheet, enableGhost bool) error {
	spec := &v1pb.Plan_Spec{
		Id: uuid.NewString(),
		Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
				Targets:     []string{database.Name},
				Sheet:       sheet.Name,
				EnableGhost: enableGhost,
			},
		},
	}
	_, _, _, err := ctl.changeDatabaseWithConfig(ctx, project, spec)
	return err
}

func (ctl *controller) changeDatabaseWithConfig(ctx context.Context, project *v1pb.Project, spec *v1pb.Plan_Spec) (*v1pb.Plan, *v1pb.Rollout, *v1pb.Issue, error) {
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{spec},
		},
	}))
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to create plan")
	}
	plan := planResp.Msg
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       "change database",
			Description: "change database",
			Plan:        plan.Name,
		},
	}))
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to create issue")
	}
	issue := issueResp.Msg
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: plan.Name}))
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to create rollout")
	}
	rollout := rolloutResp.Msg
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

	// Add timeout to prevent infinite loops
	timeout := time.After(2 * time.Minute)
	startTime := time.Now()

	// Wait for approval
waitApproval:
	for {
		select {
		case <-timeout:
			// Timeout - fetch current state for debugging
			issueResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{Name: issueName}))
			if err != nil {
				return errors.Wrapf(err, "timeout after %v waiting for approval (failed to fetch issue state)", time.Since(startTime))
			}
			issue := issueResp.Msg
			return errors.Errorf("timeout after %v waiting for approval to complete, current approval status: %s",
				time.Since(startTime), issue.ApprovalStatus.String())

		case <-ticker.C:
			issueResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{Name: issueName}))
			if err != nil {
				return err
			}
			issue := issueResp.Msg
			if issue.ApprovalStatus == v1pb.Issue_APPROVED || issue.ApprovalStatus == v1pb.Issue_SKIPPED {
				break waitApproval
			}
			// If the issue is pending approval, approve it (test user should be project owner)
			if issue.ApprovalStatus == v1pb.Issue_PENDING {
				_, err := ctl.issueServiceClient.ApproveIssue(ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{
					Name: issueName,
				}))
				if err != nil {
					return errors.Wrapf(err, "failed to approve issue")
				}
			}
		}
	}

	rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
		Name: rolloutName,
	}))
	if err != nil {
		return err
	}
	rollout := rolloutResp.Msg
	for _, stage := range rollout.Stages {
		var runTasks []string
		for _, task := range stage.Tasks {
			if task.Status == v1pb.Task_NOT_STARTED {
				runTasks = append(runTasks, task.Name)
			}
		}
		if len(runTasks) > 0 {
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
				Parent: fmt.Sprintf("%s/stages/-", rolloutName),
				Tasks:  runTasks,
			}))
			if err != nil {
				return err
			}
		}
	}

	for range ticker.C {
		rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
			Name: rolloutName,
		}))
		if err != nil {
			return err
		}
		rollout := rolloutResp.Msg
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
					resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{Parent: task.Name}))
					if err != nil {
						return err
					}
					if len(resp.Msg.TaskRuns) > 0 {
						return errors.New(resp.Msg.TaskRuns[0].Detail)
					}
				default:
					completed = false
				}
			}
		}

		// Rollout tasks.
		if len(runTasks) > 0 {
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
				Parent: fmt.Sprintf("%s/stages/-", rolloutName),
				Tasks:  runTasks,
			}))
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

// waitRolloutWithoutApproval waits for a rollout to complete without going through the issue approval flow.
// This is used for test scenarios where issues are not created (e.g., direct rollout creation).
func (ctl *controller) waitRolloutWithoutApproval(ctx context.Context, rolloutName string) error {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	// Add timeout to prevent infinite loops
	timeout := time.After(2 * time.Minute)
	startTime := time.Now()

	rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
		Name: rolloutName,
	}))
	if err != nil {
		return err
	}
	rollout := rolloutResp.Msg
	for _, stage := range rollout.Stages {
		var runTasks []string
		for _, task := range stage.Tasks {
			if task.Status == v1pb.Task_NOT_STARTED {
				runTasks = append(runTasks, task.Name)
			}
		}
		if len(runTasks) > 0 {
			_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
				Parent: fmt.Sprintf("%s/stages/-", rolloutName),
				Tasks:  runTasks,
			}))
			if err != nil {
				return err
			}
		}
	}

	for {
		select {
		case <-timeout:
			// Timeout - fetch current state for debugging
			rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
				Name: rolloutName,
			}))
			if err != nil {
				return errors.Wrapf(err, "timeout after %v waiting for rollout (failed to fetch rollout state)", time.Since(startTime))
			}
			rollout := rolloutResp.Msg
			var taskStatuses []string
			for _, stage := range rollout.Stages {
				for _, task := range stage.Tasks {
					taskStatuses = append(taskStatuses, fmt.Sprintf("%s: %s", task.Name, task.Status.String()))
				}
			}
			return errors.Errorf("timeout after %v waiting for rollout to complete, task statuses: %s",
				time.Since(startTime), strings.Join(taskStatuses, ", "))

		case <-ticker.C:
			rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
				Name: rolloutName,
			}))
			if err != nil {
				return err
			}
			rollout := rolloutResp.Msg
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
						resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{Parent: task.Name}))
						if err != nil {
							return err
						}
						if len(resp.Msg.TaskRuns) > 0 {
							return errors.New(resp.Msg.TaskRuns[0].Detail)
						}
					default:
						completed = false
					}
				}
			}

			// Rollout tasks.
			if len(runTasks) > 0 {
				_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
					Parent: fmt.Sprintf("%s/stages/-", rolloutName),
					Tasks:  runTasks,
				}))
				if err != nil {
					return err
				}
			}

			if completed {
				return nil
			}
		}
	}
}
