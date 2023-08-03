package taskrun

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SchedulerV2 is the V2 scheduler for task run.
type SchedulerV2 struct {
	store       *store.Store
	stateCfg    *state.State
	executorMap map[api.TaskType]Executor
}

// NewSchedulerV2 will create a new scheduler.
func NewSchedulerV2(store *store.Store, stateCfg *state.State) *SchedulerV2 {
	return &SchedulerV2{
		store:       store,
		stateCfg:    stateCfg,
		executorMap: map[api.TaskType]Executor{},
	}
}

// Register will register a task executor factory.
func (s *SchedulerV2) Register(taskType api.TaskType, executorGetter Executor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executorMap[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executorMap[taskType] = executorGetter
}

// Run will start the scheduler.
func (s *SchedulerV2) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Task scheduler V2 started and will run every %v", taskSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *SchedulerV2) runOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Task scheduler V2 PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	if err := s.scheduleAutoRolloutTasks(ctx); err != nil {
		log.Error("failed to schedule auto rollout tasks", zap.Error(err))
	}

	if err := s.schedulePendingTaskRuns(ctx); err != nil {
		log.Error("failed to schedule pending task runs", zap.Error(err))
	}

	if err := s.scheduleRunningTaskRuns(ctx); err != nil {
		log.Error("failed to schedule running task runs", zap.Error(err))
	}
}

func (s *SchedulerV2) scheduleAutoRolloutTasks(ctx context.Context) error {
	// TODO(p0ny): check if the task can be rolled out.
	taskIDs, err := s.store.ListTasksWithNoTaskRun(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to list tasks with zero task run")
	}
	for _, taskID := range taskIDs {
		if err := s.scheduleAutoRolloutTask(ctx, taskID); err != nil {
			log.Error("failed to schedule auto rollout task", zap.Error(err))
		}
	}
	return nil
}

func (s *SchedulerV2) scheduleAutoRolloutTask(ctx context.Context, taskUID int) error {
	task, err := s.store.GetTaskV2ByID(ctx, taskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return err
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return err
	}
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environment.UID)
	if err != nil {
		return errors.Wrapf(err, "failed to get approval policy for environment ID %d", environment.UID)
	}
	if policy.Value != api.PipelineApprovalValueManualNever {
		return nil
	}

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return err
	}
	if issue != nil {
		if issue.Status != api.IssueOpen {
			return nil
		}
		approved, err := utils.CheckIssueApproved(issue)
		if err != nil {
			return errors.Wrapf(err, "failed to check if the issue is approved")
		}
		if !approved {
			return nil
		}
	}

	create := &store.TaskRunMessage{
		CreatorID: api.SystemBotID,
		TaskUID:   task.ID,
		Name:      fmt.Sprintf("%s %d", task.Name, time.Now().Unix()),
	}

	if err := s.store.CreatePendingTaskRuns(ctx, create); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}
	return nil
}

func (s *SchedulerV2) schedulePendingTaskRuns(ctx context.Context) error {
	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
		Status: &[]api.TaskRunStatus{api.TaskRunPending},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list pending tasks")
	}
	for _, taskRun := range taskRuns {
		if err := s.schedulePendingTaskRun(ctx, taskRun); err != nil {
			log.Error("failed to schedule pending task run", zap.Error(err))
		}
	}

	return nil
}

func (s *SchedulerV2) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	// TODO(p0ny): check blocking tasks
	task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task.EarliestAllowedTs != 0 && time.Now().Before(time.Unix(task.EarliestAllowedTs, 0)) {
		return nil
	}
	if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:        taskRun.ID,
		UpdaterID: api.SystemBotID,
		Status:    api.TaskRunRunning,
	}); err != nil {
		return errors.Wrapf(err, "failed to update task run status to running")
	}
	return nil
}

func (s *SchedulerV2) scheduleRunningTaskRuns(ctx context.Context) error {
	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
		Status: &[]api.TaskRunStatus{api.TaskRunRunning},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list pending tasks")
	}

	for _, taskRun := range taskRuns {
		// Skip the task run if it is already executing.
		if _, ok := s.stateCfg.RunningTaskRuns.Load(taskRun.ID); ok {
			continue
		}
		task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
		if err != nil {
			log.Error("failed to get task", zap.Int("task id", taskRun.TaskUID), zap.Error(err))
			continue
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			continue
		}
		if instance.Deleted {
			continue
		}
		executor, ok := s.executorMap[task.Type]
		if !ok {
			log.Error("Skip running task with unknown type",
				zap.Int("id", task.ID),
				zap.String("name", task.Name),
				zap.String("type", string(task.Type)),
			)
			continue
		}
		s.stateCfg.Lock()
		if s.stateCfg.InstanceOutstandingConnections[task.InstanceID] >= state.InstanceMaximumConnectionNumber {
			s.stateCfg.Unlock()
			continue
		}
		s.stateCfg.InstanceOutstandingConnections[task.InstanceID]++
		s.stateCfg.Unlock()

		s.stateCfg.RunningTaskRuns.Store(taskRun.ID, true)
		go s.runTaskRunOnce(ctx, taskRun, task, executor)
	}

	return nil
}

func (s *SchedulerV2) runTaskRunOnce(ctx context.Context, taskRun *store.TaskRunMessage, task *store.TaskMessage, executor Executor) {
	defer func() {
		s.stateCfg.RunningTaskRuns.Delete(taskRun.ID)
		s.stateCfg.RunningTaskRunsCancelFunc.Delete(taskRun.ID)
		s.stateCfg.Lock()
		s.stateCfg.InstanceOutstandingConnections[task.InstanceID]--
		s.stateCfg.Unlock()
	}()

	executorCtx, cancel := context.WithCancel(ctx)
	s.stateCfg.RunningTaskRunsCancelFunc.Store(taskRun.ID, cancel)

	done, result, err := RunExecutorOnce(executorCtx, executor, task)

	if !done && err != nil {
		log.Debug("Encountered transient error running task, will retry",
			zap.Int("id", task.ID),
			zap.String("name", task.Name),
			zap.String("type", string(task.Type)),
			zap.Error(err),
		)
		return
	}

	if done && err != nil && errors.Is(err, context.Canceled) {
		log.Warn("task run is canceled")
		return
	}

	if done && err != nil {
		log.Warn("task run failed",
			zap.Int("id", task.ID),
			zap.String("name", task.Name),
			zap.String("type", string(task.Type)),
			zap.Error(err),
		)

		resultBytes, marshalErr := protojson.Marshal(&storepb.TaskRunResult{
			Detail:        err.Error(),
			ChangeHistory: "",
			Version:       "",
		})
		if marshalErr != nil {
			log.Error("Failed to marshal task run result",
				zap.Int("task_id", task.ID),
				zap.String("type", string(task.Type)),
				zap.Error(marshalErr),
			)
			return
		}
		code := common.ErrorCode(err)
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:        taskRun.ID,
			UpdaterID: api.SystemBotID,
			Status:    api.TaskRunFailed,
			Code:      &code,
			Result:    &result,
		}

		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			log.Error("Failed to mark task as FAILED",
				zap.Int("id", task.ID),
				zap.String("name", task.Name),
				zap.Error(err),
			)
			return
		}
		return
	}

	if done && err == nil {
		resultBytes, marshalErr := protojson.Marshal(&storepb.TaskRunResult{
			Detail:        result.Detail,
			ChangeHistory: result.ChangeHistory,
			Version:       result.Version,
		})
		if marshalErr != nil {
			log.Error("Failed to marshal task run result",
				zap.Int("task_id", task.ID),
				zap.String("type", string(task.Type)),
				zap.Error(marshalErr),
			)
			return
		}
		code := common.Ok
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:        taskRun.ID,
			UpdaterID: api.SystemBotID,
			Status:    api.TaskRunDone,
			Code:      &code,
			Result:    &result,
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			log.Error("Failed to mark task as DONE",
				zap.Int("id", task.ID),
				zap.String("name", task.Name),
				zap.Error(err),
			)
			return
		}
		return
	}
}

// CanUserRunStageTasks returns if a user can run the tasks in a stage.
func (s *SchedulerV2) CanUserRunStageTasks(ctx context.Context, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int) (bool, error) {
	// the workspace owner and DBA roles can always run tasks.
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}
	groupValue, err := s.getGroupValueForIssueTypeEnvironment(ctx, issue.Type, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get assignee group value for issueID %d", issue.UID)
	}
	// as the policy says, the project owner has the privilege to run.
	if groupValue == api.AssigneeGroupValueProjectOwner {
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == user.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// CanUserCancelStageTaskRun returns if a user can cancel the task runs in a stage.
func (s *SchedulerV2) CanUserCancelStageTaskRun(ctx context.Context, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int) (bool, error) {
	// the workspace owner and DBA roles can always cancel task runs.
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}
	// The creator can cancel task runs.
	if user.ID == issue.Creator.ID {
		return true, nil
	}
	groupValue, err := s.getGroupValueForIssueTypeEnvironment(ctx, issue.Type, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get assignee group value for issueID %d", issue.UID)
	}
	// as the policy says, the project owner has the privilege to cancel.
	if groupValue == api.AssigneeGroupValueProjectOwner {
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == user.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (s *SchedulerV2) getGroupValueForIssueTypeEnvironment(ctx context.Context, issueType api.IssueType, environmentID int) (api.AssigneeGroupValue, error) {
	defaultGroupValue := api.AssigneeGroupValueWorkspaceOwnerOrDBA
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return defaultGroupValue, errors.Wrapf(err, "failed to get pipeline approval policy by environmentID %d", environmentID)
	}
	if policy == nil {
		return defaultGroupValue, nil
	}

	for _, assigneeGroup := range policy.AssigneeGroupList {
		if assigneeGroup.IssueType == issueType {
			return assigneeGroup.Value, nil
		}
	}
	return defaultGroupValue, nil
}
