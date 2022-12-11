package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/utils"
)

var allowedStatementUpdateTaskTypes = map[api.TaskType]bool{
	api.TaskDatabaseCreate:                true,
	api.TaskDatabaseSchemaUpdate:          true,
	api.TaskDatabaseSchemaUpdateSDL:       true,
	api.TaskDatabaseDataUpdate:            true,
	api.TaskDatabaseSchemaUpdateGhostSync: true,
}

func (s *Server) canUpdateTaskStatement(ctx context.Context, task *api.Task) *echo.HTTPError {
	if ok := allowedStatementUpdateTaskTypes[task.Type]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("cannot update statement for task type %q", task.Type))
	}
	// Allow frontend to change the SQL statement of
	// 1. a PendingApproval task which hasn't started yet
	// 2. a Failed task which can be retried
	// 3. a Pending task which can't be scheduled because of failed task checks, task dependency or earliest allowed time
	if task.Status != api.TaskPendingApproval && task.Status != api.TaskFailed && task.Status != api.TaskPending {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("cannot update task in %q state", task.Status))
	}
	if task.Status == api.TaskPending {
		ok, err := s.TaskScheduler.CanSchedule(ctx, task)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to check whether the task can be scheduled").SetInternal(err)
		}
		if ok {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot update the PENDING task because it can be running at any time")
		}
	}
	return nil
}

func (s *Server) registerTaskRoutes(g *echo.Group) {
	g.PATCH("/pipeline/:pipelineID/task/all", func(c echo.Context) error {
		ctx := c.Request().Context()
		pipelineID, err := strconv.Atoi(c.Param("pipelineID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Pipeline ID is not a number: %s", c.Param("pipelineID"))).SetInternal(err)
		}

		taskPatch := &api.TaskPatch{
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update task request").SetInternal(err)
		}

		if taskPatch.EarliestAllowedTs != nil && !s.licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
		}

		issue, err := s.store.GetIssueByPipelineID(ctx, pipelineID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID: %d", pipelineID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipelineID: %d", pipelineID))
		}

		var taskPatchedList []*api.Task
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				taskPatch := *taskPatch
				taskPatch.ID = task.ID
				taskPatched, httpErr := s.patchTask(ctx, task, &taskPatch, issue)
				if httpErr != nil {
					return httpErr
				}
				taskPatchedList = append(taskPatchedList, taskPatched)
			}
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskPatchedList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update pipeline %q tasks response", issue.Pipeline.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/pipeline/:pipelineID/task/:taskID", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		taskPatch := &api.TaskPatch{
			ID:        taskID,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update task request").SetInternal(err)
		}

		if taskPatch.EarliestAllowedTs != nil && !s.licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
		}

		task, err := s.store.GetTaskByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskID))
		}

		issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID: %d", task.PipelineID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipelineID: %d", task.PipelineID))
		}

		if taskPatch.Statement != nil {
			// Tenant mode project don't allow updating SQL statement for a single task.
			if issue.Project.TenantMode == api.TenantModeTenant && (task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseSchemaUpdateSDL) {
				return echo.NewHTTPError(http.StatusBadRequest, "cannot update SQL statement of a single task for projects in tenant mode")
			}
		}

		taskPatched, httpErr := s.patchTask(ctx, task, taskPatch, issue)
		if httpErr != nil {
			return httpErr
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskPatched.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/pipeline/:pipelineID/task/:taskID/status", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		taskStatusPatch := &api.TaskStatusPatch{
			IDList:    []int{taskID},
			UpdaterID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update task status request").SetInternal(err)
		}

		task, err := s.store.GetTaskByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task not found with ID %d", taskID))
		}

		ok, err := s.canPrincipalChangeTaskStatus(ctx, currentPrincipalID, task, taskStatusPatch.Status)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate if the principal can change task status").SetInternal(err)
		}
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not allowed to change task status")
		}

		taskPatched, err := s.TaskScheduler.PatchTaskStatus(ctx, task, taskStatusPatch)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
			}
			if common.ErrorCode(err) == common.NotImplemented {
				return echo.NewHTTPError(http.StatusNotImplemented, common.ErrorMessage(err))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\" status", task.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskPatched.Name)).SetInternal(err)
		}
		return nil
	})

	g.POST("/pipeline/:pipelineID/task/:taskID/check", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		task, err := s.store.GetTaskByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task not found with ID %d", taskID))
		}

		taskUpdated, err := s.TaskCheckScheduler.ScheduleCheck(ctx, task, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to run task check \"%v\"", task.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskUpdated); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskUpdated.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) patchTask(ctx context.Context, task *api.Task, taskPatch *api.TaskPatch, issue *api.Issue) (*api.Task, error) {
	if taskPatch.Statement != nil {
		if httpErr := s.canUpdateTaskStatement(ctx, task); httpErr != nil {
			return nil, httpErr
		}
	}

	schemaVersion := common.DefaultMigrationVersion()
	taskPatch.SchemaVersion = &schemaVersion
	taskPatched, err := s.store.PatchTask(ctx, taskPatch)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
	}

	// create an activity and trigger task check for statement update
	if taskPatch.Statement != nil {
		oldStatement, err := utils.GetTaskStatement(task)
		if err != nil {
			return nil, err
		}
		newStatement := *taskPatch.Statement

		// it's ok to fail.
		if err := s.ApplicationRunner.CancelExternalApproval(ctx, issue.ID, api.ExternalApprovalCancelReasonSQLModified); err != nil {
			log.Error("failed to cancel external approval on SQL modified", zap.Int("issue_id", issue.ID), zap.Error(err))
		}

		if issue.AssigneeNeedAttention && issue.Project.WorkflowType == api.UIWorkflow {
			needAttention := false
			patch := &api.IssuePatch{
				ID:                    issue.ID,
				UpdaterID:             api.SystemBotID,
				AssigneeNeedAttention: &needAttention,
			}
			if _, err := s.store.PatchIssue(ctx, patch); err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to try to patch issue assignee_need_attention after updating task statement").SetInternal(err)
			}
		}

		if taskPatched.Type == api.TaskDatabaseSchemaUpdate || taskPatched.Type == api.TaskDatabaseSchemaUpdateSDL || taskPatched.Type == api.TaskDatabaseDataUpdate || taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
			// create a task statement update activity
			payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
				TaskID:       taskPatched.ID,
				OldStatement: oldStatement,
				NewStatement: newStatement,
				TaskName:     task.Name,
				IssueName:    issue.Name,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task statement: %v", taskPatched.Name).SetInternal(err)
			}

			if _, err = s.ActivityManager.CreateActivity(ctx, &api.ActivityCreate{
				CreatorID:   taskPatched.CreatorID,
				ContainerID: taskPatched.PipelineID,
				Type:        api.ActivityPipelineTaskStatementUpdate,
				Payload:     string(payload),
				Level:       api.ActivityInfo,
			}, &activity.Metadata{
				Issue: issue,
			}); err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
			}

			// updated statement, dismiss stale approvals and transfer the status to PendingApproval for Pending tasks.
			if taskPatched.Status == api.TaskPending {
				t, err := s.TaskScheduler.PatchTaskStatus(ctx, taskPatched, &api.TaskStatusPatch{
					IDList:    []int{taskPatch.ID},
					UpdaterID: taskPatch.UpdaterID,
					Status:    api.TaskPendingApproval,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change task status to PendingApproval after updating task: %v", taskPatched.Name)).SetInternal(err)
				}
				taskPatched = t
			}

			if taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
				_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID: taskPatched.CreatorID,
					TaskID:    task.ID,
					Type:      api.TaskCheckGhostSync,
				})
				if err != nil {
					// It's OK if we failed to trigger a check, just emit an error log
					log.Error("Failed to trigger gh-ost dry run after changing the task statement",
						zap.Int("task_id", task.ID),
						zap.String("task_name", task.Name),
						zap.Error(err),
					)
				}
			}

			if api.IsSyntaxCheckSupported(task.Database.Instance.Engine) {
				payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
					Statement: *taskPatch.Statement,
					DbType:    task.Database.Instance.Engine,
					Charset:   taskPatched.Database.CharacterSet,
					Collation: taskPatched.Database.Collation,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name))
				}
				_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID: api.SystemBotID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementSyntax,
					Payload:   string(payload),
				})
				if err != nil {
					// It's OK if we failed to trigger a check, just emit an error log
					log.Error("Failed to trigger syntax check after changing the task statement",
						zap.Int("task_id", task.ID),
						zap.String("task_name", task.Name),
						zap.Error(err),
					)
				}
			}

			if api.IsSQLReviewSupported(task.Database.Instance.Engine) {
				if err := s.triggerDatabaseStatementAdviseTask(ctx, *taskPatch.Statement, taskPatched); err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to trigger database statement advise task")).SetInternal(err)
				}
			}

			if api.IsStatementTypeCheckSupported(task.Instance.Engine) {
				payload, err := json.Marshal(api.TaskCheckDatabaseStatementTypePayload{
					Statement: *taskPatch.Statement,
					DbType:    task.Instance.Engine,
					Charset:   task.Database.CharacterSet,
					Collation: task.Database.Collation,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal check statement type payload: %v", task.Name))
				}
				if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID: api.SystemBotID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementType,
					Payload:   string(payload),
				}); err != nil {
					// It's OK if we failed to trigger a check, just emit an error log
					log.Error("Failed to trigger statement type check after changing the task statement",
						zap.Int("task_id", task.ID),
						zap.String("task_name", task.Name),
						zap.Error(err),
					)
				}
			}
		}
	}

	// earliest allowed time update.
	// - create an activity.
	// - dismiss stale approval for Pending tasks.
	if taskPatched.EarliestAllowedTs != task.EarliestAllowedTs {
		// create an activity

		payload, err := json.Marshal(api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
			TaskID:               taskPatched.ID,
			OldEarliestAllowedTs: task.EarliestAllowedTs,
			NewEarliestAllowedTs: taskPatched.EarliestAllowedTs,
			TaskName:             task.Name,
			IssueName:            issue.Name,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal earliest allowed time activity payload: %v", task.Name))
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   taskPatched.CreatorID,
			ContainerID: taskPatched.PipelineID,
			Type:        api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
			Payload:     string(payload),
			Level:       api.ActivityInfo,
		}
		if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", taskPatched.Name)).SetInternal(err)
		}

		// updated earliest allowed time, dismiss stale approvals and transfer the status to PendingApproval for Pending tasks.
		if taskPatched.Status == api.TaskPending {
			t, err := s.TaskScheduler.PatchTaskStatus(ctx, taskPatched, &api.TaskStatusPatch{
				IDList:    []int{taskPatch.ID},
				UpdaterID: taskPatch.UpdaterID,
				Status:    api.TaskPendingApproval,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change task status to PendingApproval after updating task: %v", taskPatched.Name)).SetInternal(err)
			}
			taskPatched = t
		}
	}
	return taskPatched, nil
}

// canPrincipalChangeTaskStatus validates if the principal has the privilege to update task status, judging from the principal role and the environment policy.
func (s *Server) canPrincipalChangeTaskStatus(ctx context.Context, principalID int, task *api.Task, toStatus api.TaskStatus) (bool, error) {
	// The creator can cancel task.
	if toStatus == api.TaskCanceled {
		if principalID == task.CreatorID {
			return true, nil
		}
	}
	// the workspace owner and DBA roles can always change task status.
	principal, err := s.store.GetPrincipalByID(ctx, principalID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
	}
	if principal == nil {
		return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
	}
	if principal.Role == api.Owner || principal.Role == api.DBA {
		return true, nil
	}

	issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
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
		member, err := s.store.GetProjectMember(ctx, &api.ProjectMemberFind{
			ProjectID:   &issue.ProjectID,
			PrincipalID: &principalID,
		})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project member by projectID %d, principalID %d", issue.ProjectID, principalID)
		}
		if member != nil && member.Role == string(api.Owner) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) getGroupValueForTask(ctx context.Context, issue *api.Issue, task *api.Task) (*api.AssigneeGroupValue, error) {
	environmentID := api.UnknownID
	for _, stage := range issue.Pipeline.StageList {
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

func (s *Server) triggerDatabaseStatementAdviseTask(ctx context.Context, statement string, task *api.Task) error {
	policyID, err := s.store.GetSQLReviewPolicyIDByEnvID(ctx, task.Instance.EnvironmentID)
	if err != nil {
		// It's OK if we failed to find the SQL review policy, just emit an error log
		log.Error("Failed to found SQL review policy id for task",
			zap.Int("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.Int("environment_id", task.Instance.EnvironmentID),
			zap.Error(err),
		)
		return nil
	}

	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    task.Database.Instance.Engine,
		Charset:   task.Database.CharacterSet,
		Collation: task.Database.Collation,
		PolicyID:  policyID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}

	if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: api.SystemBotID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementAdvise,
		Payload:   string(payload),
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
