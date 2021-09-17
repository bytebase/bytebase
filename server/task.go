package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
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
	g.PATCH("/pipeline/:pipelineId/task/:taskId/status", func(c echo.Context) error {
		taskId, err := strconv.Atoi(c.Param("taskId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskId"))).SetInternal(err)
		}

		taskStatusPatch := &api.TaskStatusPatch{
			ID:        taskId,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update task status request").SetInternal(err)
		}

		taskFind := &api.TaskFind{
			ID: &taskId,
		}
		task, err := s.TaskService.FindTask(context.Background(), taskFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}

		updatedTask, err := s.ChangeTaskStatusWithPatch(context.Background(), task, taskStatusPatch)
		if err != nil {
			if common.ErrorCode(err) == common.EINVALID {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\" status", task.Name)).SetInternal(err)
		}

		if err := s.ComposeTaskRelationship(context.Background(), updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task \"%v\" relationship", updatedTask.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", updatedTask.Name)).SetInternal(err)
		}
		return nil
	})

	g.POST("/pipeline/:pipelineId/task/:taskId/check", func(c echo.Context) error {
		taskId, err := strconv.Atoi(c.Param("taskId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskId"))).SetInternal(err)
		}

		taskFind := &api.TaskFind{
			ID: &taskId,
		}
		task, err := s.TaskService.FindTask(context.Background(), taskFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}

		skipIfAlreadyDone := false
		updatedTask, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(context.Background(), task, c.Get(GetPrincipalIdContextKey()).(int), skipIfAlreadyDone)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to run task check \"%v\"", task.Name)).SetInternal(err)
		}

		if err := s.ComposeTaskRelationship(context.Background(), updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task \"%v\" relationship", updatedTask.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", updatedTask.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeTaskListByPipelineAndStageId(ctx context.Context, pipelineId int, stageId int) ([]*api.Task, error) {
	taskFind := &api.TaskFind{
		PipelineId: &pipelineId,
		StageId:    &stageId,
	}
	taskList, err := s.TaskService.FindTaskList(context.Background(), taskFind)
	if err != nil {
		return nil, err
	}

	for _, task := range taskList {
		if err := s.ComposeTaskRelationship(ctx, task); err != nil {
			return nil, err
		}
	}

	return taskList, nil
}

func (s *Server) ComposeTaskById(ctx context.Context, id int) (*api.Task, error) {
	taskFind := &api.TaskFind{
		ID: &id,
	}
	task, err := s.TaskService.FindTask(context.Background(), taskFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeTaskRelationship(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Server) ComposeTaskRelationship(ctx context.Context, task *api.Task) error {
	var err error

	task.Creator, err = s.ComposePrincipalById(context.Background(), task.CreatorId)
	if err != nil {
		return err
	}

	task.Updater, err = s.ComposePrincipalById(context.Background(), task.UpdaterId)
	if err != nil {
		return err
	}

	for _, taskRun := range task.TaskRunList {
		taskRun.Creator, err = s.ComposePrincipalById(context.Background(), taskRun.CreatorId)
		if err != nil {
			return err
		}

		taskRun.Updater, err = s.ComposePrincipalById(context.Background(), taskRun.UpdaterId)
		if err != nil {
			return err
		}
	}

	for _, taskCheckRun := range task.TaskCheckRunList {
		taskCheckRun.Creator, err = s.ComposePrincipalById(context.Background(), taskCheckRun.CreatorId)
		if err != nil {
			return err
		}

		taskCheckRun.Updater, err = s.ComposePrincipalById(context.Background(), taskCheckRun.UpdaterId)
		if err != nil {
			return err
		}
	}

	task.Instance, err = s.ComposeInstanceById(context.Background(), task.InstanceId)
	if err != nil {
		return err
	}

	if task.DatabaseId != nil {
		databaseFind := &api.DatabaseFind{
			ID: task.DatabaseId,
		}
		task.Database, err = s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) ChangeTaskStatus(ctx context.Context, task *api.Task, newStatus api.TaskStatus, updaterId int) (*api.Task, error) {
	taskStatusPatch := &api.TaskStatusPatch{
		ID:        task.ID,
		UpdaterId: updaterId,
		Status:    newStatus,
	}
	return s.ChangeTaskStatusWithPatch(ctx, task, taskStatusPatch)
}

func (s *Server) ChangeTaskStatusWithPatch(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch) (_ *api.Task, err error) {
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
			Code:    common.EINVALID,
			Message: fmt.Sprintf("Invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status])}
	}

	updatedTask, err := s.TaskService.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to change task %v(%v) status: %w", task.ID, task.Name, err)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issueFind := &api.IssueFind{
		PipelineId: &task.PipelineId,
	}
	issue, err := s.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// Not all pipelines belong to an issue, so it's OK if ENOTFOUND
		if common.ErrorCode(err) != common.ENOTFOUND {
			return nil, fmt.Errorf("failed to fetch containing issue after changing the task status: %v, err: %w", task.Name, err)
		}
	}

	// Create an activity
	issueName := ""
	if issue != nil {
		issueName = issue.Name
	}
	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskId:    task.ID,
		OldStatus: task.Status,
		NewStatus: updatedTask.Status,
		IssueName: issueName,
		TaskName:  task.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity after changing the task status: %v, err: %w", task.Name, err)
	}

	containerId := task.PipelineId
	if issue != nil {
		containerId = issue.ID
	}
	level := api.ACTIVITY_INFO
	if updatedTask.Status == api.TaskFailed {
		level = api.ACTIVITY_ERROR
	}
	activityCreate := &api.ActivityCreate{
		CreatorId:   taskStatusPatch.UpdaterId,
		ContainerId: containerId,
		Type:        api.ActivityPipelineTaskStatusUpdate,
		Comment:     taskStatusPatch.Comment,
		Level:       level,
		Payload:     string(payload),
	}

	activityMeta := ActivityMeta{}
	if issue != nil {
		activityMeta.issue = issue
	}
	_, err = s.ActivityManager.CreateActivity(context.Background(), activityCreate, &activityMeta)
	if err != nil {
		return nil, err
	}

	// Schedule the task if it's being just approved
	if task.Status == api.TaskPendingApproval && updatedTask.Status == api.TaskPending {
		skipIfAlreadyDone := false
		if _, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, updatedTask, api.SYSTEM_BOT_ID, skipIfAlreadyDone); err != nil {
			return nil, fmt.Errorf("failed to schedule task check \"%v\" after approval", updatedTask.Name)
		}

		updatedTask, err = s.TaskScheduler.Schedule(ctx, updatedTask)
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

		z, offset := time.Now().Zone()
		databaseCreate := &api.DatabaseCreate{
			CreatorId:      taskStatusPatch.UpdaterId,
			ProjectId:      payload.ProjectId,
			InstanceId:     task.InstanceId,
			Name:           payload.DatabaseName,
			CharacterSet:   payload.CharacterSet,
			Collation:      payload.Collation,
			TimezoneName:   z,
			TimezoneOffset: offset,
		}
		_, err := s.DatabaseService.CreateDatabase(ctx, databaseCreate)
		if err != nil {
			// Just emits an error instead of failing, since we have another periodic job to sync db info.
			// Though the db will be assigned to the default project instead of the desired project in that case.
			s.l.Error("failed to record database after creating database",
				zap.String("database_name", payload.DatabaseName),
				zap.Int("project_id", payload.ProjectId),
				zap.Int("instance_id", task.InstanceId),
				zap.Error(err),
			)
		}
	}

	// If create database or schema update task completes, we sync the corresponding instance schema immediately.
	if (updatedTask.Type == api.TaskDatabaseCreate || updatedTask.Type == api.TaskDatabaseSchemaUpdate) &&
		updatedTask.Status == api.TaskDone {
		instance, err := s.ComposeInstanceById(context.Background(), task.InstanceId)
		if err != nil {
			return nil, fmt.Errorf("failed to sync instance schema after completing task: %w", err)
		}
		s.SyncSchema(instance)
	}

	// If this is the last task in the pipeline and just completed, and the assignee is system bot:
	// Case 1: If the task is associated with an issue, then we mark the issue (including the pipeline) as DONE.
	// Case 2: If the task is NOT associated with an issue, then we mark the pipeline as DONE.
	if updatedTask.Status == "DONE" && (issue == nil || issue.AssigneeId == api.SYSTEM_BOT_ID) {
		pipeline, err := s.ComposePipelineById(ctx, updatedTask.PipelineId)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pipeline/issue as DONE after completing task %v", updatedTask.Name)
		}
		lastStage := pipeline.StageList[len(pipeline.StageList)-1]
		if lastStage.TaskList[len(lastStage.TaskList)-1].ID == updatedTask.ID {
			if issue == nil {
				status := api.Pipeline_Done
				pipelinePatch := &api.PipelinePatch{
					ID:        pipeline.ID,
					UpdaterId: taskStatusPatch.UpdaterId,
					Status:    &status,
				}
				if _, err := s.PipelineService.PatchPipeline(context.Background(), pipelinePatch); err != nil {
					return nil, fmt.Errorf("failed to mark pipeline %v as DONE after completing task %v: %w", pipeline.Name, updatedTask.Name, err)
				}
			} else {
				issue.Pipeline = pipeline
				_, err := s.ChangeIssueStatus(ctx, issue, api.Issue_Done, taskStatusPatch.UpdaterId, "")
				if err != nil {
					return nil, fmt.Errorf("failed to mark issue %v as DONE after completing task %v: %w", issue.Name, updatedTask.Name, err)
				}
			}
		}
	}

	return updatedTask, nil
}
