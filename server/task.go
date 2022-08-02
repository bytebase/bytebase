package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
)

var (
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		api.TaskPendingApproval: {api.TaskPending},
		api.TaskPending:         {api.TaskRunning, api.TaskPendingApproval},
		api.TaskRunning:         {api.TaskDone, api.TaskFailed, api.TaskCanceled},
		api.TaskDone:            {},
		api.TaskFailed:          {api.TaskRunning, api.TaskPendingApproval},
		api.TaskCanceled:        {api.TaskRunning},
	}
)

func isTaskStatusTransitionAllowed(fromStatus, toStatus api.TaskStatus) bool {
	for _, allowedStatus := range applicableTaskStatusTransition[fromStatus] {
		if allowedStatus == toStatus {
			return true
		}
	}
	return false
}

func (s *Server) canUpdateTaskStatement(ctx context.Context, task *api.Task) *echo.HTTPError {
	// Allow frontend to change the SQL statement of
	// 1. a PendingApproval task which hasn't started yet
	// 2. a Failed task which can be retried
	// 3. a Pending task which can't be scheduled because of failed task checks
	if task.Status != api.TaskPendingApproval && task.Status != api.TaskFailed && task.Status != api.TaskPending {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("can not update task in %q state", task.Status))
	}
	if task.Status == api.TaskPending {
		ok, err := s.TaskScheduler.canTransitTaskStatus(ctx, task, api.TaskCheckStatusWarn)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to check whether the task can be scheduled").SetInternal(err)
		}
		if ok {
			return echo.NewHTTPError(http.StatusBadRequest, "can not update the PENDING task because it can be running at any time")
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

		if taskPatch.EarliestAllowedTs != nil && !s.feature(api.FeatureTaskScheduleTime) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
		}

		issue, err := s.store.GetIssueByPipelineID(ctx, pipelineID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID: %d", pipelineID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipelineID: %d", pipelineID))
		}

		if taskPatch.Statement != nil {
			// check if all tasks can update statement
			for _, stage := range issue.Pipeline.StageList {
				for _, task := range stage.TaskList {
					if httpErr := s.canUpdateTaskStatement(ctx, task); httpErr != nil {
						return httpErr
					}
				}
			}
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

		if taskPatch.EarliestAllowedTs != nil && !s.feature(api.FeatureTaskScheduleTime) {
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
			project, err := s.store.GetProjectByID(ctx, issue.ProjectID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID: %d", issue.ProjectID)).SetInternal(err)
			}
			if project.TenantMode == api.TenantModeTenant && task.Type == api.TaskDatabaseSchemaUpdate {
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
			ID:        taskID,
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

		if err := s.validateIssueAssignee(ctx, currentPrincipalID, task.PipelineID); err != nil {
			return err
		}

		taskPatched, err := s.changeTaskStatusWithPatch(ctx, task, taskStatusPatch)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
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

		skipIfAlreadyTerminated := false
		taskUpdated, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, c.Get(getPrincipalIDContextKey()).(int), skipIfAlreadyTerminated)
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

func (s *Server) patchTask(ctx context.Context, task *api.Task, taskPatch *api.TaskPatch, issue *api.Issue) (*api.Task, *echo.HTTPError) {
	oldStatement := ""
	newStatement := ""
	if taskPatch.Statement != nil {
		if httpErr := s.canUpdateTaskStatement(ctx, task); httpErr != nil {
			return nil, httpErr
		}
		newStatement = *taskPatch.Statement

		switch task.Type {
		case api.TaskDatabaseSchemaUpdate:
			payload := &api.TaskDatabaseSchemaUpdatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database schema update payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// We should update the schema version if we've updated the SQL, otherwise we will
			// get migration history version conflict if the previous task has been attempted.
			payload.SchemaVersion = common.DefaultMigrationVersion()
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		case api.TaskDatabaseDataUpdate:
			payload := &api.TaskDatabaseDataUpdatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database data update payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// We should update the schema version if we've updated the SQL, otherwise we will
			// get migration history version conflict if the previous task has been attempted.
			payload.SchemaVersion = common.DefaultMigrationVersion()
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		case api.TaskDatabaseCreate:
			payload := &api.TaskDatabaseCreatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database create payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// We should update the schema version if we've updated the SQL, otherwise we will
			// get migration history version conflict if the previous task has been attempted.
			payload.SchemaVersion = common.DefaultMigrationVersion()
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		case api.TaskDatabaseSchemaUpdateGhostSync:
			payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database data update payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// We should update the schema version if we've updated the SQL, otherwise we will
			// get migration history version conflict if the previous task has been attempted.
			payload.SchemaVersion = common.DefaultMigrationVersion()
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		}
	}

	taskPatched, err := s.store.PatchTask(ctx, taskPatch)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
	}

	// create an activity and trigger task check for statement update
	if taskPatched.Type == api.TaskDatabaseSchemaUpdate || taskPatched.Type == api.TaskDatabaseDataUpdate || taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		if oldStatement != newStatement {
			if issue == nil {
				err := fmt.Errorf("issue not found with pipeline ID %v", task.PipelineID)
				return nil, echo.NewHTTPError(http.StatusNotFound, err).SetInternal(err)
			}

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
				ContainerID: issue.ID,
				Type:        api.ActivityPipelineTaskStatementUpdate,
				Payload:     string(payload),
				Level:       api.ActivityInfo,
			}, &ActivityMeta{
				issue: issue,
			}); err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
			}

			// dismiss stale reviews and transfer the status to PendingApproval.
			if taskPatched.Status != api.TaskPendingApproval {
				t, err := s.changeTaskStatusWithPatch(ctx, taskPatched, &api.TaskStatusPatch{
					ID:        taskPatch.ID,
					UpdaterID: taskPatch.UpdaterID,
					Status:    api.TaskPendingApproval,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change task status to PendingApproval after updating task statement: %v", taskPatched.Name)).SetInternal(err)
				}
				taskPatched = t
			}

			if taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
				_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID:               taskPatched.CreatorID,
					TaskID:                  task.ID,
					Type:                    api.TaskCheckGhostSync,
					SkipIfAlreadyTerminated: false,
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

			if api.IsSyntaxCheckSupported(task.Database.Instance.Engine, s.profile.Mode) {
				payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
					Statement: *taskPatch.Statement,
					DbType:    task.Database.Instance.Engine,
					Charset:   taskPatched.Database.CharacterSet,
					Collation: taskPatched.Database.Collation,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err))
				}
				_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID:               api.SystemBotID,
					TaskID:                  task.ID,
					Type:                    api.TaskCheckDatabaseStatementSyntax,
					Payload:                 string(payload),
					SkipIfAlreadyTerminated: false,
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

			if s.feature(api.FeatureSQLReviewPolicy) && api.IsSQLReviewSupported(task.Database.Instance.Engine, s.profile.Mode) {
				if err := s.triggerDatabaseStatementAdviseTask(ctx, *taskPatch.Statement, taskPatched); err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to trigger database statement advise task, err: %w", err)).SetInternal(err)
				}
			}
		}
	}

	// create an activity and trigger task check for earliest allowed time update
	if taskPatched.EarliestAllowedTs != task.EarliestAllowedTs {
		// create an activity
		if issue == nil {
			err := fmt.Errorf("issue not found with pipeline ID %v", task.PipelineID)
			return nil, echo.NewHTTPError(http.StatusNotFound, err.Error()).SetInternal(err)
		}

		payload, err := json.Marshal(api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
			TaskID:               taskPatched.ID,
			OldEarliestAllowedTs: task.EarliestAllowedTs,
			NewEarliestAllowedTs: taskPatched.EarliestAllowedTs,
			TaskName:             task.Name,
			IssueName:            issue.Name,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal earliest allowed time activity payload: %v, err: %w", task.Name, err))
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   taskPatched.CreatorID,
			ContainerID: issue.ID,
			Type:        api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
			Payload:     string(payload),
			Level:       api.ActivityInfo,
		}
		_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
			issue: issue,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", taskPatched.Name)).SetInternal(err)
		}

		// trigger task check
		payload, err = json.Marshal(api.TaskCheckEarliestAllowedTimePayload{
			EarliestAllowedTs: *taskPatch.EarliestAllowedTs,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err))
		}
		_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
			CreatorID:               api.SystemBotID,
			TaskID:                  task.ID,
			Type:                    api.TaskCheckGeneralEarliestAllowedTime,
			Payload:                 string(payload),
			SkipIfAlreadyTerminated: false,
		})
		if err != nil {
			// It's OK if we failed to trigger a check, just emit an error log
			log.Error("Failed to trigger timing check after changing task earliest allowed time",
				zap.Int("task_id", task.ID),
				zap.String("task_name", task.Name),
				zap.Error(err),
			)
		}
	}
	return taskPatched, nil
}

func (s *Server) validateIssueAssignee(ctx context.Context, currentPrincipalID, pipelineID int) error {
	issue, err := s.store.GetIssueByPipelineID(ctx, pipelineID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find issue").SetInternal(err)
	}
	if issue == nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue not found by pipeline ID: %d", pipelineID))
	}
	if issue.AssigneeID == api.SystemBotID {
		currentPrincipal, err := s.store.GetPrincipalByID(ctx, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find principal").SetInternal(err)
		}
		if currentPrincipal.Role != api.Owner && currentPrincipal.Role != api.DBA {
			return echo.NewHTTPError(http.StatusUnauthorized, "Only allow Owner/DBA system account to update task status")
		}
	} else if issue.AssigneeID != currentPrincipalID {
		return echo.NewHTTPError(http.StatusUnauthorized, "Only allow the assignee to update task status")
	}
	return nil
}

// TODO(p0ny): remove this function because it adds yet another layer of indirection when traveling in our codebase while doesn't seem useful to me.
func (s *Server) changeTaskStatus(ctx context.Context, task *api.Task, newStatus api.TaskStatus, updaterID int) (*api.Task, error) {
	taskStatusPatch := &api.TaskStatusPatch{
		ID:        task.ID,
		UpdaterID: updaterID,
		Status:    newStatus,
	}
	return s.changeTaskStatusWithPatch(ctx, task, taskStatusPatch)
}

func (s *Server) changeTaskStatusWithPatch(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch) (_ *api.Task, err error) {
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
		return nil, &common.Error{
			Code: common.Invalid,
			Err:  fmt.Errorf("invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status])}
	}

	taskPatched, err := s.store.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to change task %v(%v) status: %w", task.ID, task.Name, err)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch containing issue after changing the task status: %v, err: %w", task.Name, err)
	}
	// Not all pipelines belong to an issue, so it's OK if issue is not found.
	if issue == nil {
		log.Debug("Pipeline has no linking issue",
			zap.Int("pipelineID", task.PipelineID),
			zap.String("task", task.Name))
	}

	// Create an activity
	issueName := ""
	if issue != nil {
		issueName = issue.Name
	}
	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: task.Status,
		NewStatus: taskPatched.Status,
		IssueName: issueName,
		TaskName:  task.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity after changing the task status: %v, err: %w", task.Name, err)
	}

	containerID := task.PipelineID
	if issue != nil {
		containerID = issue.ID
	}
	level := api.ActivityInfo
	if taskPatched.Status == api.TaskFailed {
		level = api.ActivityError
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   taskStatusPatch.UpdaterID,
		ContainerID: containerID,
		Type:        api.ActivityPipelineTaskStatusUpdate,
		Level:       level,
		Payload:     string(payload),
	}
	if taskStatusPatch.Comment != nil {
		activityCreate.Comment = *taskStatusPatch.Comment
	}

	activityMeta := ActivityMeta{}
	if issue != nil {
		activityMeta.issue = issue
	}
	_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activityMeta)
	if err != nil {
		return nil, err
	}

	// If create database, schema update and gh-ost cutover task completes, we sync the corresponding instance schema immediately.
	if (taskPatched.Type == api.TaskDatabaseCreate || taskPatched.Type == api.TaskDatabaseSchemaUpdate || taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostCutover) && taskPatched.Status == api.TaskDone {
		instance, err := s.store.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("failed to sync instance schema after completing task: %w", err)
		}
		if err := s.syncDatabaseSchema(ctx, instance, taskPatched.Database.Name); err != nil {
			log.Error("failed to sync database schema",
				zap.String("instance", instance.Name),
				zap.String("databaseName", taskPatched.Database.Name),
			)
		}
	}

	// If this is the last task in the pipeline and just completed, and the assignee is system bot:
	// Case 1: If the task is associated with an issue, then we mark the issue (including the pipeline) as DONE.
	// Case 2: If the task is NOT associated with an issue, then we mark the pipeline as DONE.
	if taskPatched.Status == "DONE" && (issue == nil || issue.AssigneeID == api.SystemBotID) {
		pipeline, err := s.store.GetPipelineByID(ctx, taskPatched.PipelineID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pipeline/issue as DONE after completing task %v", taskPatched.Name)
		}
		if pipeline == nil {
			return nil, fmt.Errorf("pipeline not found for ID %v", taskPatched.PipelineID)
		}
		lastStage := pipeline.StageList[len(pipeline.StageList)-1]
		if lastStage.TaskList[len(lastStage.TaskList)-1].ID == taskPatched.ID {
			if issue == nil {
				status := api.PipelineDone
				pipelinePatch := &api.PipelinePatch{
					ID:        pipeline.ID,
					UpdaterID: taskStatusPatch.UpdaterID,
					Status:    &status,
				}
				if _, err := s.store.PatchPipeline(ctx, pipelinePatch); err != nil {
					return nil, fmt.Errorf("failed to mark pipeline %v as DONE after completing task %v: %w", pipeline.Name, taskPatched.Name, err)
				}
			} else {
				issue.Pipeline = pipeline
				_, err := s.changeIssueStatus(ctx, issue, api.IssueDone, taskStatusPatch.UpdaterID, "")
				if err != nil {
					return nil, fmt.Errorf("failed to mark issue %v as DONE after completing task %v: %w", issue.Name, taskPatched.Name, err)
				}
			}
		}
	}

	return taskPatched, nil
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
		return fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err)
	}

	if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID:               api.SystemBotID,
		TaskID:                  task.ID,
		Type:                    api.TaskCheckDatabaseStatementAdvise,
		Payload:                 string(payload),
		SkipIfAlreadyTerminated: false,
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
