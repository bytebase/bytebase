package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/store"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		api.TaskPendingApproval: {api.TaskPending},
		api.TaskPending:         {api.TaskRunning},
		api.TaskRunning:         {api.TaskDone, api.TaskFailed, api.TaskCanceled},
		api.TaskDone:            {},
		api.TaskFailed:          {api.TaskRunning},
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

func (s *Server) registerTaskRoutes(g *echo.Group) {
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update task request").SetInternal(err)
		}

		if taskPatch.EarliestAllowedTs != nil && !s.feature(api.FeatureTaskScheduleTime) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
		}

		taskFind := &api.TaskFind{
			ID: &taskID,
		}
		task, err := s.TaskService.FindTask(ctx, taskFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskID))
		}

		issueFind := &api.IssueFind{
			PipelineID: &task.PipelineID,
		}
		issueRaw, err := s.IssueService.FindIssue(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID %v", task.PipelineID)).SetInternal(err)
		}
		issue, err := s.composeIssueRelationship(ctx, issueRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose issue relationship with pipeline ID %v", task.PipelineID)).SetInternal(err)
		}

		oldStatement := ""
		newStatement := ""
		if taskPatch.Statement != nil {
			// Tenant mode project don't allow updating SQL statement.
			project, err := s.store.GetProjectByID(ctx, issue.ProjectID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID[%d]", issue.ProjectID)).SetInternal(err)
			}
			if project.TenantMode == api.TenantModeTenant {
				err := fmt.Errorf("cannot update SQL statement for projects in tenant mode")
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}

			if task.Status != api.TaskPending && task.Status != api.TaskPendingApproval && task.Status != api.TaskFailed {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Can not update task in %v state", task.Status))
			}
			newStatement = *taskPatch.Statement

			switch task.Type {
			case api.TaskDatabaseSchemaUpdate:
				payload := &api.TaskDatabaseSchemaUpdatePayload{}
				if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Malformatted database schema update payload").SetInternal(err)
				}
				oldStatement = payload.Statement
				payload.Statement = *taskPatch.Statement
				// We should update the schema version if we've updated the SQL, otherwise we will
				// get migration history version conflict if the previous task has been attempted.
				payload.SchemaVersion = common.DefaultMigrationVersion()
				bytes, err := json.Marshal(payload)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
				}
				payloadStr := string(bytes)
				taskPatch.Payload = &payloadStr
			case api.TaskDatabaseDataUpdate:

				payload := &api.TaskDatabaseDataUpdatePayload{}
				if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Malformatted database data update payload").SetInternal(err)
				}
				oldStatement = payload.Statement
				payload.Statement = *taskPatch.Statement
				// We should update the schema version if we've updated the SQL, otherwise we will
				// get migration history version conflict if the previous task has been attempted.
				payload.SchemaVersion = common.DefaultMigrationVersion()
				bytes, err := json.Marshal(payload)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
				}
				payloadStr := string(bytes)
				taskPatch.Payload = &payloadStr

			}
		}

		taskPatchedRaw, err := s.TaskService.PatchTask(ctx, taskPatch)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
		}

		taskPatched, err := s.composeTaskRelationship(ctx, taskPatchedRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task \"%v\" relationship", taskPatchedRaw.Name)).SetInternal(err)
		}

		// create an activity and trigger task check for statement update
		if taskPatched.Type == api.TaskDatabaseSchemaUpdate || taskPatched.Type == api.TaskDatabaseDataUpdate {
			if oldStatement != newStatement {
				// create an activity
				if issue == nil {
					err := fmt.Errorf("issue not found with pipeline ID %v", task.PipelineID)
					return echo.NewHTTPError(http.StatusNotFound, err).SetInternal(err)
				}

				payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
					TaskID:       taskPatched.ID,
					OldStatement: oldStatement,
					NewStatement: newStatement,
					TaskName:     task.Name,
					IssueName:    issue.Name,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task statement: %v", taskPatched.Name).SetInternal(err)
				}
				activityCreate := &api.ActivityCreate{
					CreatorID:   taskPatched.CreatorID,
					ContainerID: issue.ID,
					Type:        api.ActivityPipelineTaskStatementUpdate,
					Payload:     string(payload),
					Level:       api.ActivityInfo,
				}
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
					issue: issue,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
				}

				// For now, we supported MySQL and TiDB dialect check
				if taskPatched.Database.Instance.Engine == db.MySQL || taskPatched.Database.Instance.Engine == db.TiDB {
					payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
						Statement: *taskPatch.Statement,
						DbType:    taskPatched.Database.Instance.Engine,
						Charset:   taskPatched.Database.CharacterSet,
						Collation: taskPatched.Database.Collation,
					})
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err))
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
						s.l.Error("Failed to trigger syntax check after changing task statement",
							zap.Int("task_id", task.ID),
							zap.String("task_name", task.Name),
							zap.Error(err),
						)
					}

					if s.feature(api.FeatureBackwardCompatibility) {
						_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
							CreatorID:               api.SystemBotID,
							TaskID:                  task.ID,
							Type:                    api.TaskCheckDatabaseStatementCompatibility,
							Payload:                 string(payload),
							SkipIfAlreadyTerminated: false,
						})
						if err != nil {
							// It's OK if we failed to trigger a check, just emit an error log
							s.l.Error("Failed to trigger compatibility check after changing task statement",
								zap.Int("task_id", task.ID),
								zap.String("task_name", task.Name),
								zap.Error(err),
							)
						}
					}
				}
			}

		}
		// create an activity and trigger task check for earliest allowed time update
		if taskPatched.EarliestAllowedTs != task.EarliestAllowedTs {
			// create an activity
			if issue == nil {
				err := fmt.Errorf("issue not found with pipeline ID %v", task.PipelineID)
				return echo.NewHTTPError(http.StatusNotFound, err.Error()).SetInternal(err)
			}

			payload, err := json.Marshal(api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
				TaskID:               taskPatched.ID,
				OldEarliestAllowedTs: task.EarliestAllowedTs,
				NewEarliestAllowedTs: taskPatched.EarliestAllowedTs,
				TaskName:             task.Name,
				IssueName:            issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal earliest allowed time activity payload: %v, err: %w", task.Name, err))
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
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", taskPatched.Name)).SetInternal(err)
			}

			// trigger task check
			payload, err = json.Marshal(api.TaskCheckEarliestAllowedTimePayload{
				EarliestAllowedTs: *taskPatch.EarliestAllowedTs,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err))
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
				s.l.Error("Failed to trigger timing check after changing task earliest allowed time",
					zap.Int("task_id", task.ID),
					zap.String("task_name", task.Name),
					zap.Error(err),
				)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskPatchedRaw.Name)).SetInternal(err)
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update task status request").SetInternal(err)
		}

		taskFind := &api.TaskFind{
			ID: &taskID,
		}
		taskRaw, err := s.TaskService.FindTask(ctx, taskFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if taskRaw == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task not found with ID %d", taskID))
		}
		task, err := s.composeTaskRelationship(ctx, taskRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose task relationship with ID %d", taskID)).SetInternal(err)
		}

		issue, err := s.IssueService.FindIssue(ctx, &api.IssueFind{
			PipelineID: &taskRaw.PipelineID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find issue").SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue not found by pipeline ID: %d", task.PipelineID))
		}
		if issue.AssigneeID == api.SystemBotID {
			currentPrincipal, err := s.store.GetPrincipalByID(ctx, currentPrincipalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find principal").SetInternal(err)
			}
			if currentPrincipal.Role != api.Owner && currentPrincipal.Role != api.DBA {
				return echo.NewHTTPError(http.StatusUnauthorized, "Only allow Owner/DBA system account to update this task status")
			}
		} else {
			if issue.AssigneeID != currentPrincipalID {
				return echo.NewHTTPError(http.StatusUnauthorized, "Only allow the assignee to update task status")
			}
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

		taskFind := &api.TaskFind{
			ID: &taskID,
		}
		taskRaw, err := s.TaskService.FindTask(ctx, taskFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if taskRaw == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task not found with ID %d", taskID))
		}
		task, err := s.composeTaskRelationship(ctx, taskRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose task %v(%v) relationship", taskRaw.ID, taskRaw.Name)).SetInternal(err)
		}

		skipIfAlreadyTerminated := false
		taskUpdated, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, c.Get(getPrincipalIDContextKey()).(int), skipIfAlreadyTerminated)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to run task check \"%v\"", taskRaw.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskUpdated); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskUpdated.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeTaskListByPipelineAndStageID(ctx context.Context, pipelineID int, stageID int) ([]*api.Task, error) {
	taskFind := &api.TaskFind{
		PipelineID: &pipelineID,
		StageID:    &stageID,
	}
	taskRawList, err := s.TaskService.FindTaskList(ctx, taskFind)
	if err != nil {
		return nil, err
	}

	var taskList []*api.Task
	for _, taskRaw := range taskRawList {
		task, err := s.composeTaskRelationship(ctx, taskRaw)
		if err != nil {
			return nil, err
		}
		taskList = append(taskList, task)
	}

	return taskList, nil
}

func (s *Server) composeTaskRelationship(ctx context.Context, raw *store.TaskRaw) (*api.Task, error) {
	task := raw.ToTask()

	creator, err := s.store.GetPrincipalByID(ctx, task.CreatorID)
	if err != nil {
		return nil, err
	}
	task.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, task.UpdaterID)
	if err != nil {
		return nil, err
	}
	task.Updater = updater

	for _, taskRun := range task.TaskRunList {
		creator, err := s.store.GetPrincipalByID(ctx, taskRun.CreatorID)
		if err != nil {
			return nil, err
		}
		taskRun.Creator = creator

		updater, err := s.store.GetPrincipalByID(ctx, taskRun.UpdaterID)
		if err != nil {
			return nil, err
		}
		taskRun.Updater = updater
	}

	for _, taskCheckRun := range task.TaskCheckRunList {
		creator, err := s.store.GetPrincipalByID(ctx, taskCheckRun.CreatorID)
		if err != nil {
			return nil, err
		}
		taskCheckRun.Creator = creator

		updater, err := s.store.GetPrincipalByID(ctx, taskCheckRun.UpdaterID)
		if err != nil {
			return nil, err
		}
		taskCheckRun.Updater = updater
	}

	blockedBy := []string{}
	taskDAGList, err := s.store.FindTaskDAGList(ctx, &api.TaskDAGFind{ToTaskID: raw.ID})
	if err != nil {
		return nil, err
	}
	for _, taskDAG := range taskDAGList {
		blockedBy = append(blockedBy, strconv.Itoa(taskDAG.FromTaskID))
	}
	task.BlockedBy = blockedBy

	instance, err := s.store.GetInstanceByID(ctx, task.InstanceID)
	if err != nil {
		return nil, err
	}
	task.Instance = instance

	if task.DatabaseID != nil {
		database, err := s.store.GetDatabaseByID(ctx, *task.DatabaseID)
		if err != nil {
			return nil, err
		}
		if database == nil {
			return nil, fmt.Errorf("database not found with ID %v", task.DatabaseID)
		}
		task.Database = database
	}

	return task, nil
}

// TODO(dragonly): remove this hack
func (s *Server) composeTaskRelationshipValidateOnly(ctx context.Context, task *api.Task) error {
	var err error

	task.Creator, err = s.store.GetPrincipalByID(ctx, task.CreatorID)
	if err != nil {
		return err
	}

	task.Updater, err = s.store.GetPrincipalByID(ctx, task.UpdaterID)
	if err != nil {
		return err
	}

	for _, taskRun := range task.TaskRunList {
		taskRun.Creator, err = s.store.GetPrincipalByID(ctx, taskRun.CreatorID)
		if err != nil {
			return err
		}

		taskRun.Updater, err = s.store.GetPrincipalByID(ctx, taskRun.UpdaterID)
		if err != nil {
			return err
		}
	}

	for _, taskCheckRun := range task.TaskCheckRunList {
		taskCheckRun.Creator, err = s.store.GetPrincipalByID(ctx, taskCheckRun.CreatorID)
		if err != nil {
			return err
		}

		taskCheckRun.Updater, err = s.store.GetPrincipalByID(ctx, taskCheckRun.UpdaterID)
		if err != nil {
			return err
		}
	}

	instance, err := s.store.GetInstanceByID(ctx, task.InstanceID)
	if err != nil {
		return err
	}
	task.Instance = instance

	if task.DatabaseID != nil {
		task.Database, err = s.store.GetDatabaseByID(ctx, *task.DatabaseID)
		if err != nil {
			return err
		}
		if task.Database == nil {
			return fmt.Errorf("database ID not found %v", task.DatabaseID)
		}
	}

	return nil
}

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
			s.l.Error("Failed to change task status.",
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

	taskPatchedRaw, err := s.TaskService.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to change task %v(%v) status: %w", task.ID, task.Name, err)
	}

	taskPatched, err := s.composeTaskRelationship(ctx, taskPatchedRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose task %v(%v) relationship: %w", task.ID, task.Name, err)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issueFind := &api.IssueFind{
		PipelineID: &task.PipelineID,
	}
	issueRaw, err := s.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// Not all pipelines belong to an issue, so it's OK if ENOTFOUND
		return nil, fmt.Errorf("failed to fetch containing issue after changing the task status: %v, err: %w", task.Name, err)
	}
	if issueRaw == nil {
		return nil, fmt.Errorf("failed to find issue with pipeline ID %d, task: %s", task.PipelineID, task.Name)
	}
	issue, err := s.composeIssueRelationship(ctx, issueRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose issue relationship with ID %d, err: %w", issueRaw.ID, err)
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

	// Schedule the task if it's being just approved
	if task.Status == api.TaskPendingApproval && taskPatched.Status == api.TaskPending {
		skipIfAlreadyTerminated := false
		if _, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, taskPatched, api.SystemBotID, skipIfAlreadyTerminated); err != nil {
			return nil, fmt.Errorf("failed to schedule task check \"%v\" after approval", taskPatched.Name)
		}

		scheduledTask, err := s.TaskScheduler.ScheduleIfNeeded(ctx, taskPatched)
		if err != nil {
			return nil, fmt.Errorf("failed to schedule task \"%v\" after approval", taskPatched.Name)
		}
		taskPatched = scheduledTask
	}

	// If create database or schema update task completes, we sync the corresponding instance schema immediately.
	if (taskPatched.Type == api.TaskDatabaseCreate || taskPatched.Type == api.TaskDatabaseSchemaUpdate) &&
		taskPatched.Status == api.TaskDone {
		instance, err := s.store.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("failed to sync instance schema after completing task: %w", err)
		}
		s.syncEngineVersionAndSchema(ctx, instance)
	}

	// If this is the last task in the pipeline and just completed, and the assignee is system bot:
	// Case 1: If the task is associated with an issue, then we mark the issue (including the pipeline) as DONE.
	// Case 2: If the task is NOT associated with an issue, then we mark the pipeline as DONE.
	if taskPatched.Status == "DONE" && (issue == nil || issue.AssigneeID == api.SystemBotID) {
		pipeline, err := s.composePipelineByID(ctx, taskPatched.PipelineID)
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
				if _, err := s.PipelineService.PatchPipeline(ctx, pipelinePatch); err != nil {
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
