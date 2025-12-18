package taskrun

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// runAutoRolloutScheduler runs in a separate goroutine to schedule auto-rollout tasks.
// This prevents blocking the main scheduler when there are many tasks to auto-rollout.
func (s *Scheduler) runAutoRolloutScheduler(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug("Auto-rollout scheduler started and will run every 10s")
	for {
		select {
		case <-ticker.C:
			if err := s.scheduleAutoRolloutTasks(ctx); err != nil {
				slog.Error("failed to schedule auto rollout tasks", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) scheduleAutoRolloutTasks(ctx context.Context) error {
	environments, err := s.store.GetEnvironment(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to list environments")
	}

	var envs []string
	for _, environment := range environments.GetEnvironments() {
		policy, err := s.store.GetRolloutPolicy(ctx, environment.Id)
		if err != nil {
			return errors.Wrapf(err, "failed to get rollout policy for environment %s", environment.Id)
		}
		if policy.Automatic {
			envs = append(envs, environment.Id)
		}
	}
	taskIDs, err := s.store.ListTasksToAutoRollout(ctx, envs)
	if err != nil {
		return errors.Wrapf(err, "failed to list tasks with zero task run")
	}
	for _, taskID := range taskIDs {
		if err := s.scheduleAutoRolloutTask(ctx, taskID); err != nil {
			slog.Error("failed to schedule auto rollout task", log.BBError(err))
		}
	}

	return nil
}

func (s *Scheduler) scheduleAutoRolloutTask(ctx context.Context, taskUID int) error {
	task, err := s.store.GetTaskByID(ctx, taskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task == nil {
		return nil
	}

	pipeline, err := s.store.GetPipelineByID(ctx, task.PipelineID)
	if err != nil {
		return errors.Wrapf(err, "failed to get pipeline")
	}
	if pipeline == nil {
		return errors.Errorf("pipeline %v not found", task.PipelineID)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &pipeline.ProjectID})
	if err != nil {
		return errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return errors.Errorf("project %v not found", pipeline.ProjectID)
	}

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return err
	}
	if instance.Deleted {
		return nil
	}

	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return err
	}
	if issue != nil {
		if issue.Status != storepb.Issue_OPEN {
			return nil
		}
		// Check if issue approval is required according to the project settings
		if project.Setting.RequireIssueApproval {
			approved, err := utils.CheckIssueApproved(issue)
			if err != nil {
				return errors.Wrapf(err, "failed to check if the issue is approved")
			}
			if !approved {
				return nil
			}
		}
	}

	// Check the latest plan checks based on project settings (error only)
	if project.Setting.RequirePlanCheckNoError {
		pass, err := func() (bool, error) {
			plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{PipelineID: &task.PipelineID})
			if err != nil {
				return false, errors.Wrapf(err, "failed to get plan")
			}
			if plan == nil {
				return true, nil
			}
			latestRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
				PlanUID: &plan.UID,
			})
			if err != nil {
				return false, errors.Wrapf(err, "failed to list latest plan check runs")
			}
			for _, run := range latestRuns {
				if run.Status != store.PlanCheckRunStatusDone {
					return false, nil
				}
				for _, result := range run.Result.Results {
					if result.Status == storepb.Advice_ERROR {
						return false, nil
					}
				}
			}
			return true, nil
		}()
		if err != nil {
			return errors.Wrapf(err, "failed to check if plan check passes")
		}
		if !pass {
			return nil
		}
	}

	create := &store.TaskRunMessage{
		TaskUID: task.ID,
	}
	if task.Payload.GetSheetId() != 0 {
		sheetUID := int(task.Payload.GetSheetId())
		create.SheetUID = &sheetUID
	}

	if err := s.store.CreatePendingTaskRuns(ctx, common.SystemBotEmail, create); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   store.SystemBotUser,
		Type:    storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE,
		Comment: "",
		Issue:   webhook.NewIssue(issue),
		Rollout: webhook.NewRollout(pipeline),
		Project: webhook.NewProject(project),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Title:  task.GetDatabaseName(),
			Status: storepb.TaskRun_PENDING.String(),
		},
	})

	return nil
}
