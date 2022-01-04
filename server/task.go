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
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		api.TaskPending:         {api.TaskRunning},
		api.TaskPendingApproval: {api.TaskPending},
		api.TaskRunning:         {api.TaskDone, api.TaskFailed, api.TaskCanceled},
		api.TaskDone:            {},
		api.TaskFailed:          {api.TaskRunning},
		api.TaskCanceled:        {api.TaskRunning},
	}
)

func (s *Server) registerTaskRoutes(g *echo.Group) {
	g.PATCH("/pipeline/:pipelineID/task/:taskID", func(c echo.Context) error {
		ctx := context.Background()
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

		if taskPatch.Statement != nil {
			if task.Status != api.TaskPending && task.Status != api.TaskPendingApproval && task.Status != api.TaskFailed {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Can not update task in %v state", task.Status))
			}

			if task.Type == api.TaskDatabaseSchemaUpdate {
				payload := &api.TaskDatabaseSchemaUpdatePayload{}
				if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Malformatted database schema update payload").SetInternal(err)
				}
				payload.Statement = *taskPatch.Statement
				bytes, err := json.Marshal(payload)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
				}
				payloadStr := string(bytes)
				taskPatch.Payload = &payloadStr
			}
		}

		updatedTask, err := s.TaskService.PatchTask(ctx, taskPatch)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
		}

		if err := s.composeTaskRelationship(ctx, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task \"%v\" relationship", updatedTask.Name)).SetInternal(err)
		}

		// create an activity and trigger task check for statement update
		if updatedTask.Type == api.TaskDatabaseSchemaUpdate {
			oldPayload := &api.TaskDatabaseSchemaUpdatePayload{}
			newPayload := &api.TaskDatabaseSchemaUpdatePayload{}
			if err := json.Unmarshal([]byte(updatedTask.Payload), newPayload); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task statement: %v", updatedTask.Name).SetInternal(err)
			}
			if err := json.Unmarshal([]byte(task.Payload), oldPayload); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task statement: %v", updatedTask.Name).SetInternal(err)
			}
			if oldPayload.Statement != newPayload.Statement {
				// create an activity
				issueFind := &api.IssueFind{
					PipelineID: &task.PipelineID,
				}
				issue, err := s.IssueService.FindIssue(ctx, issueFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID after updating task statement: %v", task.PipelineID)).SetInternal(err)
				}
				if issue == nil {
					err := fmt.Errorf("issue not found with pipeline ID %v", task.PipelineID)
					return echo.NewHTTPError(http.StatusNotFound, err).SetInternal(err)
				}

				payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
					TaskID:       updatedTask.ID,
					OldStatement: oldPayload.Statement,
					NewStatement: newPayload.Statement,
					TaskName:     task.Name,
					IssueName:    issue.Name,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task statement: %v", updatedTask.Name).SetInternal(err)
				}
				activityCreate := &api.ActivityCreate{
					CreatorID:   updatedTask.CreatorID,
					ContainerID: issue.ID,
					Type:        api.ActivityPipelineTaskStatementUpdate,
					Payload:     string(payload),
					Level:       api.ActivityInfo,
				}
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
					issue: issue,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", updatedTask.Name)).SetInternal(err)
				}

				// For now, we supported MySQL and TiDB dialect check
				if updatedTask.Database.Instance.Engine == db.MySQL || updatedTask.Database.Instance.Engine == db.TiDB {
					payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
						Statement: *taskPatch.Statement,
						DbType:    updatedTask.Database.Instance.Engine,
						Charset:   updatedTask.Database.CharacterSet,
						Collation: updatedTask.Database.Collation,
					})
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err))
					}
					_, err = s.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
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

					_, err = s.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
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
		// create an activity and trigger task check for earliest allowed time update
		if updatedTask.EarliestAllowedTs != task.EarliestAllowedTs {
			// create an activity
			issueFind := &api.IssueFind{
				PipelineID: &task.PipelineID,
			}
			issue, err := s.IssueService.FindIssue(ctx, issueFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID after updating task's earliest allowed time: %v", task.PipelineID)).SetInternal(err)
			}
			if issue == nil {
				err := fmt.Errorf("issue not found with pipeline ID %v", task.PipelineID)
				return echo.NewHTTPError(http.StatusNotFound, err.Error()).SetInternal(err)
			}

			payload, err := json.Marshal(api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
				TaskID:               updatedTask.ID,
				OldEarliestAllowedTs: task.EarliestAllowedTs,
				NewEarliestAllowedTs: updatedTask.EarliestAllowedTs,
				TaskName:             task.Name,
				IssueName:            issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal earliest allowed time activity payload: %v, err: %w", task.Name, err))
			}
			activityCreate := &api.ActivityCreate{
				CreatorID:   updatedTask.CreatorID,
				ContainerID: issue.ID,
				Type:        api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
				Payload:     string(payload),
				Level:       api.ActivityInfo,
			}
			_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
				issue: issue,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", updatedTask.Name)).SetInternal(err)
			}

			// trigger task check
			payload, err = json.Marshal(api.TaskCheckEarliestAllowedTimePayload{
				EarliestAllowedTs: *taskPatch.EarliestAllowedTs,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err))
			}
			_, err = s.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
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
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", updatedTask.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/pipeline/:pipelineID/task/:taskID/status", func(c echo.Context) error {
		ctx := context.Background()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		taskStatusPatch := &api.TaskStatusPatch{
			ID:        taskID,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update task status request").SetInternal(err)
		}

		taskFind := &api.TaskFind{
			ID: &taskID,
		}
		task, err := s.TaskService.FindTask(ctx, taskFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskID))
		}

		updatedTask, err := s.changeTaskStatusWithPatch(ctx, task, taskStatusPatch)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\" status", task.Name)).SetInternal(err)
		}

		if err := s.composeTaskRelationship(ctx, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task \"%v\" relationship", updatedTask.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", updatedTask.Name)).SetInternal(err)
		}
		return nil
	})

	g.POST("/pipeline/:pipelineID/task/:taskID/check", func(c echo.Context) error {
		ctx := context.Background()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		taskFind := &api.TaskFind{
			ID: &taskID,
		}
		task, err := s.TaskService.FindTask(ctx, taskFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskID))
		}

		skipIfAlreadyTerminated := false
		updatedTask, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, c.Get(getPrincipalIDContextKey()).(int), skipIfAlreadyTerminated)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to run task check \"%v\"", task.Name)).SetInternal(err)
		}

		if err := s.composeTaskRelationship(ctx, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task \"%v\" relationship", updatedTask.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", updatedTask.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeTaskListByPipelineAndStageID(ctx context.Context, pipelineID int, stageID int) ([]*api.Task, error) {
	taskFind := &api.TaskFind{
		PipelineID: &pipelineID,
		StageID:    &stageID,
	}
	taskList, err := s.TaskService.FindTaskList(ctx, taskFind)
	if err != nil {
		return nil, err
	}

	for _, task := range taskList {
		if err := s.composeTaskRelationship(ctx, task); err != nil {
			return nil, err
		}
	}

	return taskList, nil
}

func (s *Server) composeTaskRelationship(ctx context.Context, task *api.Task) error {
	var err error

	task.Creator, err = s.composePrincipalByID(ctx, task.CreatorID)
	if err != nil {
		return err
	}

	task.Updater, err = s.composePrincipalByID(ctx, task.UpdaterID)
	if err != nil {
		return err
	}

	for _, taskRun := range task.TaskRunList {
		taskRun.Creator, err = s.composePrincipalByID(ctx, taskRun.CreatorID)
		if err != nil {
			return err
		}

		taskRun.Updater, err = s.composePrincipalByID(ctx, taskRun.UpdaterID)
		if err != nil {
			return err
		}
	}

	for _, taskCheckRun := range task.TaskCheckRunList {
		taskCheckRun.Creator, err = s.composePrincipalByID(ctx, taskCheckRun.CreatorID)
		if err != nil {
			return err
		}

		taskCheckRun.Updater, err = s.composePrincipalByID(ctx, taskCheckRun.UpdaterID)
		if err != nil {
			return err
		}
	}

	task.Instance, err = s.composeInstanceByID(ctx, task.InstanceID)
	if err != nil {
		return err
	}

	if task.DatabaseID != nil {
		databaseFind := &api.DatabaseFind{
			ID: task.DatabaseID,
		}
		task.Database, err = s.composeDatabaseByFind(ctx, databaseFind)
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
	allowTransition := false
	for _, allowedStatus := range applicableTaskStatusTransition[task.Status] {
		if allowedStatus == taskStatusPatch.Status {
			allowTransition = true
			break
		}
	}

	if !allowTransition {
		return nil, &common.Error{
			Code: common.Invalid,
			Err:  fmt.Errorf("invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status])}
	}

	updatedTask, err := s.TaskService.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to change task %v(%v) status: %w", task.ID, task.Name, err)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issueFind := &api.IssueFind{
		PipelineID: &task.PipelineID,
	}
	issue, err := s.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// Not all pipelines belong to an issue, so it's OK if ENOTFOUND
		return nil, fmt.Errorf("failed to fetch containing issue after changing the task status: %v, err: %w", task.Name, err)
	}

	// Create an activity
	issueName := ""
	if issue != nil {
		issueName = issue.Name
	}
	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: task.Status,
		NewStatus: updatedTask.Status,
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
	if updatedTask.Status == api.TaskFailed {
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
	if task.Status == api.TaskPendingApproval && updatedTask.Status == api.TaskPending {
		skipIfAlreadyTerminated := false
		if _, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, updatedTask, api.SystemBotID, skipIfAlreadyTerminated); err != nil {
			return nil, fmt.Errorf("failed to schedule task check \"%v\" after approval", updatedTask.Name)
		}

		updatedTask, err = s.TaskScheduler.ScheduleIfNeeded(ctx, updatedTask)
		if err != nil {
			return nil, fmt.Errorf("failed to schedule task \"%v\" after approval", updatedTask.Name)
		}
	}

	// If create database task completes, then we will create a database entry immediately
	// instead of waiting for the next schema sync cycle to sync over this newly created database.
	// This is for 2 reasons:
	// 1. Assign the proper project to the newly created database. Otherwise, the periodic schema
	//    sync will place the synced db into the default project.
	// 2. Allow user to see the created database right away.
	if (updatedTask.Type == api.TaskDatabaseCreate) &&
		updatedTask.Status == api.TaskDone {
		payload := &api.TaskDatabaseCreatePayload{}
		if err := json.Unmarshal([]byte(updatedTask.Payload), payload); err != nil {
			return nil, fmt.Errorf("invalid create database task payload: %w", err)
		}

		instance, err := s.InstanceService.FindInstance(ctx, &api.InstanceFind{
			ID: &task.InstanceID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to find instance: %v", task.InstanceID)
		}
		if instance == nil {
			return nil, fmt.Errorf("instance ID not found %v", task.InstanceID)
		}
		databaseCreate := &api.DatabaseCreate{
			CreatorID:     taskStatusPatch.UpdaterID,
			ProjectID:     payload.ProjectID,
			InstanceID:    task.InstanceID,
			EnvironmentID: instance.EnvironmentID,
			Name:          payload.DatabaseName,
			CharacterSet:  payload.CharacterSet,
			Collation:     payload.Collation,
			Labels:        &payload.Labels,
			SchemaVersion: payload.SchemaVersion,
		}
		database, err := s.DatabaseService.CreateDatabase(ctx, databaseCreate)
		if err != nil {
			// Just emits an error instead of failing, since we have another periodic job to sync db info.
			// Though the db will be assigned to the default project instead of the desired project in that case.
			s.l.Error("failed to record database after creating database",
				zap.String("database_name", payload.DatabaseName),
				zap.Int("project_id", payload.ProjectID),
				zap.Int("instance_id", task.InstanceID),
				zap.Error(err),
			)
		}
		err = s.composeDatabaseRelationship(ctx, database)
		if err != nil {
			s.l.Error("failed to compose database relationship after creating database",
				zap.String("database_name", payload.DatabaseName),
				zap.Int("project_id", payload.ProjectID),
				zap.Int("instance_id", task.InstanceID),
				zap.Error(err),
			)
		}
		// Set database labels, except bb.environment is immutable and must match instance environment.
		// This needs to be after we compose database relationship.
		if err == nil && databaseCreate.Labels != nil && *databaseCreate.Labels != "" {
			if err := s.setDatabaseLabels(ctx, *databaseCreate.Labels, database, databaseCreate.CreatorID, false /* validateOnly */); err != nil {
				s.l.Error("failed to record database labels after creating database",
					zap.String("database_name", payload.DatabaseName),
					zap.Int("project_id", payload.ProjectID),
					zap.Int("instance_id", task.InstanceID),
					zap.Error(err),
				)
			}
		}
	}

	// If create database or schema update task completes, we sync the corresponding instance schema immediately.
	if (updatedTask.Type == api.TaskDatabaseCreate || updatedTask.Type == api.TaskDatabaseSchemaUpdate) &&
		updatedTask.Status == api.TaskDone {
		instance, err := s.composeInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("failed to sync instance schema after completing task: %w", err)
		}
		s.syncEngineVersionAndSchema(ctx, instance)
	}

	// If this is the last task in the pipeline and just completed, and the assignee is system bot:
	// Case 1: If the task is associated with an issue, then we mark the issue (including the pipeline) as DONE.
	// Case 2: If the task is NOT associated with an issue, then we mark the pipeline as DONE.
	if updatedTask.Status == "DONE" && (issue == nil || issue.AssigneeID == api.SystemBotID) {
		pipeline, err := s.composePipelineByID(ctx, updatedTask.PipelineID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pipeline/issue as DONE after completing task %v", updatedTask.Name)
		}
		if pipeline == nil {
			return nil, fmt.Errorf("pipeline not found for ID %v", updatedTask.PipelineID)
		}
		lastStage := pipeline.StageList[len(pipeline.StageList)-1]
		if lastStage.TaskList[len(lastStage.TaskList)-1].ID == updatedTask.ID {
			if issue == nil {
				status := api.PipelineDone
				pipelinePatch := &api.PipelinePatch{
					ID:        pipeline.ID,
					UpdaterID: taskStatusPatch.UpdaterID,
					Status:    &status,
				}
				if _, err := s.PipelineService.PatchPipeline(ctx, pipelinePatch); err != nil {
					return nil, fmt.Errorf("failed to mark pipeline %v as DONE after completing task %v: %w", pipeline.Name, updatedTask.Name, err)
				}
			} else {
				issue.Pipeline = pipeline
				_, err := s.changeIssueStatus(ctx, issue, api.IssueDone, taskStatusPatch.UpdaterID, "")
				if err != nil {
					return nil, fmt.Errorf("failed to mark issue %v as DONE after completing task %v: %w", issue.Name, updatedTask.Name, err)
				}
			}
		}
	}

	return updatedTask, nil
}
