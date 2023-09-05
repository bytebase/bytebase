// Package taskrun implements a runner for executing tasks.
package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
	taskSchedulerInterval = time.Duration(1) * time.Second
)

var (
	taskCancellationImplemented = map[api.TaskType]bool{
		api.TaskDatabaseSchemaUpdateGhostSync: true,
	}
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		api.TaskPendingApproval: {api.TaskPending, api.TaskDone},
		api.TaskPending:         {api.TaskCanceled, api.TaskRunning, api.TaskPendingApproval, api.TaskDone},
		api.TaskRunning:         {api.TaskDone, api.TaskFailed, api.TaskCanceled},
		api.TaskDone:            {},
		api.TaskFailed:          {api.TaskPendingApproval, api.TaskDone},
		api.TaskCanceled:        {api.TaskPendingApproval, api.TaskDone},
	}
	allowedSkippedTaskStatus = map[api.TaskStatus]bool{
		api.TaskPendingApproval: true,
		api.TaskFailed:          true,
	}
	terminatedTaskStatus = map[api.TaskStatus]bool{
		api.TaskDone:     true,
		api.TaskCanceled: true,
		api.TaskFailed:   true,
	}
)

// NewScheduler creates a new task scheduler.
func NewScheduler(
	store *store.Store,
	schemaSyncer *schemasync.Syncer,
	activityManager *activity.Manager,
	licenseService enterpriseAPI.LicenseService,
	stateCfg *state.State,
	profile config.Profile,
	metricReporter *metricreport.Reporter) *Scheduler {
	return &Scheduler{
		store:           store,
		schemaSyncer:    schemaSyncer,
		activityManager: activityManager,
		licenseService:  licenseService,
		profile:         profile,
		stateCfg:        stateCfg,
		executorMap:     make(map[api.TaskType]Executor),
		metricReporter:  metricReporter,
	}
}

// Scheduler is the task scheduler.
type Scheduler struct {
	store           *store.Store
	schemaSyncer    *schemasync.Syncer
	activityManager *activity.Manager
	licenseService  enterpriseAPI.LicenseService
	stateCfg        *state.State
	profile         config.Profile
	executorMap     map[api.TaskType]Executor
	metricReporter  *metricreport.Reporter
}

// Register will register a task executor factory.
func (s *Scheduler) Register(taskType api.TaskType, executorGetter Executor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executorMap[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executorMap[taskType] = executorGetter
}

// PatchTask patches the statement, earliest allowed time and rollbackEnabled for a task.
func (s *Scheduler) PatchTask(ctx context.Context, task *store.TaskMessage, taskPatch *api.TaskPatch, issue *store.IssueMessage) error {
	if taskPatch.SheetID != nil {
		if err := canUpdateTaskStatement(task); err != nil {
			return err
		}
	}

	// Reset because we are trying to build
	// the RollbackSheetID again and there could be previous runs.
	if taskPatch.RollbackEnabled != nil && *taskPatch.RollbackEnabled {
		empty := ""
		pending := api.RollbackSQLStatusPending
		taskPatch.RollbackSQLStatus = &pending
		taskPatch.RollbackSheetID = nil
		taskPatch.RollbackError = &empty
	}
	// if *taskPatch.RollbackEnabled == false, we don't reset
	// 1. they are meaningless anyway if rollback is disabled
	// 2. they will be reset when enabling
	// 3. we cancel the generation after writing to db. The rollback sql generation may finish and write to db, too.

	taskPatched, err := s.store.UpdateTaskV2(ctx, taskPatch)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
	}
	if issue.NeedAttention && issue.Project.Workflow == api.UIWorkflow {
		needAttention := false
		if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{NeedAttention: &needAttention}, api.SystemBotID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to try to patch issue assignee_need_attention after updating task statement").SetInternal(err)
		}
	}

	// enqueue or cancel after it's written to the database.
	if taskPatch.RollbackEnabled != nil {
		// Enqueue the rollback sql generation if the task done.
		if *taskPatch.RollbackEnabled && taskPatched.Status == api.TaskDone {
			s.stateCfg.RollbackGenerate.Store(taskPatched.ID, taskPatched)
		} else {
			// Cancel running rollback sql generation.
			if v, ok := s.stateCfg.RollbackCancel.Load(taskPatched.ID); ok {
				if cancel, ok := v.(context.CancelFunc); ok {
					cancel()
				}
			}
			// We don't erase the keys for RollbackCancel and RollbackGenerate here because they will eventually be erased by the rollback runner.
		}
	}

	// Trigger task checks.
	if taskPatch.SheetID != nil {
		dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
		if err != nil {
			return err
		}
		if dbSchema == nil {
			return errors.Errorf("database schema ID not found %v", task.DatabaseID)
		}
		// it's ok to fail.
		s.stateCfg.IssueExternalApprovalRelayCancelChan <- issue.UID
		if taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
			if err := s.store.CreateTaskCheckRun(ctx, &store.TaskCheckRunMessage{
				CreatorID: taskPatched.CreatorID,
				TaskID:    task.ID,
				Type:      api.TaskCheckGhostSync,
			}); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger gh-ost dry run after changing the task statement",
					zap.Int("task_id", task.ID),
					zap.String("task_name", task.Name),
					zap.Error(err),
				)
			}
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			return err
		}

		if api.IsSQLReviewSupported(instance.Engine) {
			if err := s.triggerDatabaseStatementAdviseTask(ctx, taskPatched); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to trigger database statement advise task")).SetInternal(err)
			}
		}

		if api.IsStatementTypeCheckSupported(instance.Engine) {
			if err := s.store.CreateTaskCheckRun(ctx, &store.TaskCheckRunMessage{
				CreatorID: taskPatched.CreatorID,
				TaskID:    task.ID,
				Type:      api.TaskCheckDatabaseStatementType,
			}); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger statement type check after changing the task statement",
					zap.Int("task_id", task.ID),
					zap.String("task_name", task.Name),
					zap.Error(err),
				)
			}
		}

		if api.IsTaskCheckReportSupported(instance.Engine) && api.IsTaskCheckReportNeededForTaskType(task.Type) {
			if err := s.store.CreateTaskCheckRun(ctx,
				&store.TaskCheckRunMessage{
					CreatorID: taskPatched.CreatorID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementAffectedRowsReport,
				},
				&store.TaskCheckRunMessage{
					CreatorID: taskPatched.CreatorID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementTypeReport,
				},
			); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger task report check after changing the task statement", zap.Int("task_id", task.ID), zap.String("task_name", task.Name), zap.Error(err))
			}
		}
	}

	if taskPatch.SheetID != nil {
		oldSheetID, err := utils.GetTaskSheetID(task.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to get old sheet ID")
		}
		newSheetID := *taskPatch.SheetID

		// create a task sheet update activity
		payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
			TaskID:     taskPatched.ID,
			OldSheetID: oldSheetID,
			NewSheetID: newSheetID,
			TaskName:   task.Name,
			IssueName:  issue.Title,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task sheet: %v", taskPatched.Name).SetInternal(err)
		}
		if _, err := s.activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:   taskPatch.UpdaterID,
			ContainerUID: taskPatched.PipelineID,
			Type:         api.ActivityPipelineTaskStatementUpdate,
			Payload:      string(payload),
			Level:        api.ActivityInfo,
		}, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
		}
	}

	// Earliest allowed time update activity.
	if taskPatch.EarliestAllowedTs != nil {
		// create an activity
		payload, err := json.Marshal(api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
			TaskID:               taskPatched.ID,
			OldEarliestAllowedTs: task.EarliestAllowedTs,
			NewEarliestAllowedTs: taskPatched.EarliestAllowedTs,
			TaskName:             task.Name,
			IssueName:            issue.Title,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal earliest allowed time activity payload: %v", task.Name))
		}
		activityCreate := &store.ActivityMessage{
			CreatorUID:   taskPatch.UpdaterID,
			ContainerUID: taskPatched.PipelineID,
			Type:         api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
			Payload:      string(payload),
			Level:        api.ActivityInfo,
		}
		if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", taskPatched.Name)).SetInternal(err)
		}
	}
	return nil
}

func (s *Scheduler) triggerDatabaseStatementAdviseTask(ctx context.Context, task *store.TaskMessage) error {
	dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
	if err != nil {
		return err
	}
	if dbSchema == nil {
		return errors.Errorf("database schema ID not found %v", task.DatabaseID)
	}

	if err := s.store.CreateTaskCheckRun(ctx, &store.TaskCheckRunMessage{
		CreatorID: api.SystemBotID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementAdvise,
	}); err != nil {
		// It's OK if we failed to trigger a check, just emit an error log
		log.Error("Failed to trigger statement advise task after changing task statement",
			zap.Int("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.Error(err),
		)
	}

	return nil
}

// CanPrincipalChangeTaskStatus validates if the principal has the privilege to update task status, judging from the principal role and the environment policy.
func (s *Scheduler) CanPrincipalChangeTaskStatus(ctx context.Context, principalID int, task *store.TaskMessage, toStatus api.TaskStatus) (bool, error) {
	// The creator can cancel task.
	if toStatus == api.TaskCanceled {
		if principalID == task.CreatorID {
			return true, nil
		}
	}
	// the workspace owner and DBA roles can always change task status.
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
	}
	if user == nil {
		return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
	}
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to find issue")
	}
	if issue == nil {
		return false, common.Errorf(common.NotFound, "issue not found by pipeline ID: %d", task.PipelineID)
	}
	groupValue, err := s.getGroupValueForTask(ctx, issue, task)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get assignee group value for taskID %d", task.ID)
	}
	if groupValue == nil {
		return false, nil
	}
	// as the policy says, the project owner has the privilege to change task status.
	if *groupValue == api.AssigneeGroupValueProjectOwner {
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == principalID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (s *Scheduler) getGroupValueForTask(ctx context.Context, issue *store.IssueMessage, task *store.TaskMessage) (*api.AssigneeGroupValue, error) {
	environmentID := api.UnknownID
	stages, err := s.store.ListStageV2(ctx, task.PipelineID)
	if err != nil {
		return nil, err
	}
	for _, stage := range stages {
		if stage.ID == task.StageID {
			environmentID = stage.EnvironmentID
			break
		}
	}
	if environmentID == api.UnknownID {
		return nil, common.Errorf(common.NotFound, "failed to find environmentID by task.StageID %d", task.StageID)
	}

	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to get pipeline approval policy by environmentID %d", environmentID)
	}

	for _, assigneeGroup := range policy.AssigneeGroupList {
		if assigneeGroup.IssueType == issue.Type {
			return &assigneeGroup.Value, nil
		}
	}
	return nil, nil
}

var (
	allowedStatementUpdateTaskTypes = map[api.TaskType]bool{
		api.TaskDatabaseCreate:                true,
		api.TaskDatabaseSchemaUpdate:          true,
		api.TaskDatabaseSchemaUpdateSDL:       true,
		api.TaskDatabaseDataUpdate:            true,
		api.TaskDatabaseSchemaUpdateGhostSync: true,
	}
	allowedPatchStatementStatus = map[api.TaskStatus]bool{
		api.TaskPendingApproval: true,
		api.TaskFailed:          true,
	}
)

func canUpdateTaskStatement(task *store.TaskMessage) *echo.HTTPError {
	if ok := allowedStatementUpdateTaskTypes[task.Type]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("cannot update statement for task type %q", task.Type))
	}
	// Allow frontend to change the SQL statement of
	// 1. a PendingApproval task which hasn't started yet;
	// 2. a Failed task which can be retried.
	if !allowedPatchStatementStatus[task.Status] {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("cannot update task in %q status", task.Status))
	}
	return nil
}

// PatchTaskStatus patches a single task.
func (s *Scheduler) PatchTaskStatus(ctx context.Context, task *store.TaskMessage, taskStatusPatch *api.TaskStatusPatch) (err error) {
	defer func() {
		if err != nil {
			log.Error("Failed to change task status.",
				zap.Int("id", task.ID),
				zap.String("name", task.Name),
				zap.String("old_status", string(task.Status)),
				zap.String("new_status", string(taskStatusPatch.Status)),
				zap.Error(err))
		}
	}()

	if !isTaskStatusTransitionAllowed(task.Status, taskStatusPatch.Status) {
		return &common.Error{
			Code: common.Invalid,
			Err:  errors.Errorf("invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status]),
		}
	}

	if taskStatusPatch.Skipped != nil && *taskStatusPatch.Skipped && !allowedSkippedTaskStatus[task.Status] {
		return &common.Error{
			Code: common.Invalid,
			Err:  errors.Errorf("cannot skip task whose status is %v", task.Status),
		}
	}

	if taskStatusPatch.Status == api.TaskCanceled {
		if !taskCancellationImplemented[task.Type] {
			return common.Errorf(common.NotImplemented, "Canceling task type %s is not supported", task.Type)
		}
		cancelAny, ok := s.stateCfg.RunningTasksCancel.Load(task.ID)
		if !ok {
			return errors.New("failed to cancel task")
		}
		cancel := cancelAny.(context.CancelFunc)
		cancel()
		result, err := json.Marshal(api.TaskRunResultPayload{
			Detail: "Task cancellation requested.",
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal TaskRunResultPayload")
		}
		resultStr := string(result)
		taskStatusPatch.Result = &resultStr
	}

	taskPatched, err := s.store.UpdateTaskStatusV2(ctx, taskStatusPatch)
	if err != nil {
		return errors.Wrapf(err, "failed to change task %v(%v) status", task.ID, task.Name)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch containing issue after changing the task status: %v", task.Name)
	}

	// Create an activity
	if err := s.createTaskStatusUpdateActivity(ctx, task, taskStatusPatch, issue); err != nil {
		return err
	}

	// Cancel every task depending on the canceled task.
	if taskPatched.Status == api.TaskCanceled {
		if err := s.cancelDependingTasks(ctx, taskPatched); err != nil {
			return errors.Wrapf(err, "failed to cancel depending tasks for task %d", taskPatched.ID)
		}
	}

	if issue != nil {
		if err := s.onTaskStatusPatched(ctx, issue, taskPatched); err != nil {
			return err
		}
	}

	return nil
}

func isTaskStatusTransitionAllowed(fromStatus, toStatus api.TaskStatus) bool {
	for _, allowedStatus := range applicableTaskStatusTransition[fromStatus] {
		if allowedStatus == toStatus {
			return true
		}
	}
	return false
}

func (s *Scheduler) createTaskStatusUpdateActivity(ctx context.Context, task *store.TaskMessage, taskStatusPatch *api.TaskStatusPatch, issue *store.IssueMessage) error {
	var issueName string
	if issue != nil {
		issueName = issue.Title
	}
	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: task.Status,
		NewStatus: taskStatusPatch.Status,
		IssueName: issueName,
		TaskName:  task.Name,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
	}

	level := api.ActivityInfo
	if taskStatusPatch.Status == api.TaskFailed {
		level = api.ActivityError
	}
	activityCreate := &store.ActivityMessage{
		CreatorUID:   taskStatusPatch.UpdaterID,
		ContainerUID: task.PipelineID,
		Type:         api.ActivityPipelineTaskStatusUpdate,
		Level:        level,
		Payload:      string(payload),
	}
	if taskStatusPatch.Comment != nil {
		activityCreate.Comment = *taskStatusPatch.Comment
	}

	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue}); err != nil {
		return err
	}
	return nil
}

func (s *Scheduler) cancelDependingTasks(ctx context.Context, task *store.TaskMessage) error {
	queue := []int{task.ID}
	seen := map[int]bool{task.ID: true}
	var idList []int
	dags, err := s.store.ListTaskDags(ctx, &store.TaskDAGFind{PipelineID: &task.PipelineID, StageID: &task.StageID})
	if err != nil {
		return err
	}
	for len(queue) != 0 {
		fromTaskID := queue[0]
		queue = queue[1:]
		for _, dag := range dags {
			if dag.FromTaskID != fromTaskID {
				continue
			}
			if seen[dag.ToTaskID] {
				return errors.Errorf("found a cycle in task dag, visit task %v twice", dag.ToTaskID)
			}
			seen[dag.ToTaskID] = true
			idList = append(idList, dag.ToTaskID)
			queue = append(queue, dag.ToTaskID)
		}
	}
	if err := s.store.BatchPatchTaskStatus(ctx, idList, api.TaskCanceled, api.SystemBotID); err != nil {
		return errors.Wrapf(err, "failed to change task %v's status to %s", idList, api.TaskCanceled)
	}
	return nil
}

// GetDefaultAssignee gets the default assignee for an issue.
func (s *Scheduler) GetDefaultAssignee(ctx context.Context, environmentID int, projectID int, issueType api.IssueType) (*store.UserMessage, error) {
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to GetPipelineApprovalPolicy for environmentID %d", environmentID)
	}
	if policy.Value == api.PipelineApprovalValueManualNever {
		// use SystemBot for auto approval tasks.
		systemBot, err := s.store.GetUserByID(ctx, api.SystemBotID)
		if err != nil {
			return nil, err
		}
		return systemBot, nil
	}

	var groupValue *api.AssigneeGroupValue
	for i, group := range policy.AssigneeGroupList {
		if group.IssueType == issueType {
			groupValue = &policy.AssigneeGroupList[i].Value
			break
		}
	}
	if groupValue == nil || *groupValue == api.AssigneeGroupValueWorkspaceOwnerOrDBA {
		user, err := s.getAnyWorkspaceOwner(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a workspace owner or DBA")
		}
		return user, nil
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		projectOwner, err := s.getAnyProjectOwner(ctx, projectID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a project owner")
		}
		return projectOwner, nil
	}
	// never reached
	return nil, errors.New("invalid assigneeGroupValue")
}

// getAnyWorkspaceOwner finds a default assignee from the workspace owners.
func (s *Scheduler) getAnyWorkspaceOwner(ctx context.Context) (*store.UserMessage, error) {
	// There must be at least one non-systembot owner in the workspace.
	owner := api.Owner
	users, err := s.store.ListUsers(ctx, &store.FindUserMessage{Role: &owner})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get role %v", api.Owner)
	}
	for _, user := range users {
		if user.ID == api.SystemBotID {
			continue
		}
		return user, nil
	}
	return nil, errors.New("failed to get a workspace owner or DBA")
}

// getAnyProjectOwner gets a default assignee from the project owners.
func (s *Scheduler) getAnyProjectOwner(ctx context.Context, projectID int) (*store.UserMessage, error) {
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectID})
	if err != nil {
		return nil, err
	}
	// Find the project member that is not workspace owner or DBA first.
	for _, binding := range policy.Bindings {
		if binding.Role != api.Owner || len(binding.Members) == 0 {
			continue
		}
		for _, user := range binding.Members {
			if user.Role != api.Owner && user.Role != api.DBA {
				return user, nil
			}
		}
	}
	for _, binding := range policy.Bindings {
		if binding.Role == api.Owner && len(binding.Members) > 0 {
			return binding.Members[0], nil
		}
	}
	return nil, errors.New("failed to get a project owner")
}

// CanPrincipalBeAssignee checks if a principal could be the assignee of an issue, judging by the principal role and the environment policy.
func (s *Scheduler) CanPrincipalBeAssignee(ctx context.Context, principalID int, environmentID int, projectID int, issueType api.IssueType) (bool, error) {
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return false, err
	}
	var groupValue *api.AssigneeGroupValue
	for i, group := range policy.AssigneeGroupList {
		if group.IssueType == issueType {
			groupValue = &policy.AssigneeGroupList[i].Value
			break
		}
	}
	if groupValue == nil || *groupValue == api.AssigneeGroupValueWorkspaceOwnerOrDBA {
		// no value is set, fallback to default.
		// the assignee group is the workspace owner or DBA.
		user, err := s.store.GetUserByID(ctx, principalID)
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
		}
		if user == nil {
			return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
		}
		if s.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
			user.Role = api.Owner
		}
		if user.Role == api.Owner || user.Role == api.DBA {
			return true, nil
		}
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		// the assignee group is the project owner.
		if s.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
			// nolint:nilerr
			return true, nil
		}
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", projectID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == principalID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (s *Scheduler) onTaskStatusPatched(ctx context.Context, issue *store.IssueMessage, taskPatched *store.TaskMessage) error {
	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricAPI.TaskStatusMetricName,
		Value: 1,
		Labels: map[string]any{
			"type":  taskPatched.Type,
			"value": taskPatched.Status,
		},
	})

	stages, err := s.store.ListStageV2(ctx, taskPatched.PipelineID)
	if err != nil {
		return errors.Wrap(err, "failed to list stages")
	}
	var taskStage, nextStage *store.StageMessage
	for i, stage := range stages {
		if stage.ID == taskPatched.StageID {
			taskStage = stages[i]
			if i+1 < len(stages) {
				nextStage = stages[i+1]
			}
			break
		}
	}
	if taskStage == nil {
		return errors.New("failed to find corresponding stage of the task in the issue pipeline")
	}

	if !taskStage.Active && nextStage != nil {
		// Every task in this stage has finished, we are moving to the next stage.
		// The current assignee doesn't fit in the new assignee group, we will reassign a new one based on the new assignee group.
		func() {
			environmentID := nextStage.EnvironmentID
			ok, err := s.CanPrincipalBeAssignee(ctx, issue.Assignee.ID, environmentID, issue.Project.UID, issue.Type)
			if err != nil {
				log.Error("failed to check if the current assignee still fits in the new assignee group", zap.Error(err))
				return
			}
			if !ok {
				// reassign the issue to a new assignee if the current one doesn't fit.
				assignee, err := s.GetDefaultAssignee(ctx, environmentID, issue.Project.UID, issue.Type)
				if err != nil {
					log.Error("failed to get a default assignee", zap.Error(err))
					return
				}
				if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{Assignee: assignee}, api.SystemBotID); err != nil {
					log.Error("failed to update the issue assignee", zap.Error(err))
					return
				}
			}
		}()
	}
	if !taskStage.Active && nextStage == nil {
		// Every task in the pipeline has finished.
		// Resolve the issue automatically for the user.
		if err := utils.ChangeIssueStatus(ctx, s.store, s.activityManager, issue, api.IssueDone, api.SystemBotID, ""); err != nil {
			log.Error("failed to change the issue status to done automatically after completing every task", zap.Error(err))
		}
	}

	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &taskPatched.PipelineID, StageID: &taskPatched.StageID})
	if err != nil {
		return errors.Wrap(err, "failed to list tasks")
	}
	stageTaskHasPendingApproval := false
	stageTaskAllTerminated := true
	for _, task := range tasks {
		if task.Status == api.TaskPendingApproval {
			stageTaskHasPendingApproval = true
		}
		if !terminatedTaskStatus[task.Status] {
			stageTaskAllTerminated = false
		}
	}

	// every task in the stage terminated
	// create "stage ends" activity.
	if stageTaskAllTerminated {
		if err := func() error {
			createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
				StageID:               taskStage.ID,
				StageStatusUpdateType: api.StageStatusUpdateTypeEnd,
				IssueName:             issue.Title,
				StageName:             taskStage.Name,
			}
			bytes, err := json.Marshal(createActivityPayload)
			if err != nil {
				return errors.Wrap(err, "failed to marshal ActivityPipelineStageStatusUpdate payload")
			}
			activityCreate := &store.ActivityMessage{
				CreatorUID:   api.SystemBotID,
				ContainerUID: *issue.PipelineUID,
				Type:         api.ActivityPipelineStageStatusUpdate,
				Level:        api.ActivityInfo,
				Payload:      string(bytes),
			}
			if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
				Issue: issue,
			}); err != nil {
				return errors.Wrap(err, "failed to create activity")
			}
			return nil
		}(); err != nil {
			log.Error("failed to create ActivityPipelineStageStatusUpdate activity", zap.Error(err))
		}
	}

	// every task in the stage completes and this is not the last stage.
	// create "stage begins" activity.
	if !taskStage.Active && nextStage != nil {
		if err := func() error {
			createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
				StageID:               nextStage.ID,
				StageStatusUpdateType: api.StageStatusUpdateTypeBegin,
				IssueName:             issue.Title,
				StageName:             nextStage.Name,
			}
			bytes, err := json.Marshal(createActivityPayload)
			if err != nil {
				return errors.Wrap(err, "failed to marshal ActivityPipelineStageStatusUpdate payload")
			}
			activityCreate := &store.ActivityMessage{
				CreatorUID:   api.SystemBotID,
				ContainerUID: *issue.PipelineUID,
				Type:         api.ActivityPipelineStageStatusUpdate,
				Level:        api.ActivityInfo,
				Payload:      string(bytes),
			}
			if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
				Issue: issue,
			}); err != nil {
				return errors.Wrap(err, "failed to create activity")
			}
			return nil
		}(); err != nil {
			log.Error("failed to create ActivityPipelineStageStatusUpdate activity", zap.Error(err))
		}
	}

	// there isn't a pendingApproval task
	// we need to set issue.AssigneeNeedAttention to false for UI workflow.
	if taskPatched.Status == api.TaskPending && issue.Project.Workflow == api.UIWorkflow && !stageTaskHasPendingApproval {
		needAttention := false
		if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{NeedAttention: &needAttention}, api.SystemBotID); err != nil {
			log.Error("failed to patch issue assigneeNeedAttention after finding out that there isn't any pendingApproval task in the stage", zap.Error(err))
		}
	}

	return nil
}

// CanPrincipalChangeIssueStageTaskStatus returns whether the principal can change the task status.
func (s *Scheduler) CanPrincipalChangeIssueStageTaskStatus(ctx context.Context, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int, toStatus api.TaskStatus) (bool, error) {
	// the workspace owner and DBA roles can always change task status.
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}
	// The creator can cancel task.
	if toStatus == api.TaskCanceled {
		if user.ID == issue.Creator.ID {
			return true, nil
		}
	}
	groupValue, err := s.getGroupValueForIssueTypeEnvironment(ctx, issue.Type, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get assignee group value for issueID %d", issue.UID)
	}
	// as the policy says, the project owner has the privilege to change task status.
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

func (s *Scheduler) getGroupValueForIssueTypeEnvironment(ctx context.Context, issueType api.IssueType, environmentID int) (api.AssigneeGroupValue, error) {
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
