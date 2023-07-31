package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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

		if taskPatch.EarliestAllowedTs != nil {
			if err := s.licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime); err != nil {
				return echo.NewHTTPError(http.StatusForbidden, err.Error())
			}
		}

		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &pipelineID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID: %d", pipelineID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipelineID: %d", pipelineID))
		}

		tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: issue.PipelineUID})
		if err != nil {
			return err
		}
		for _, task := range tasks {
			if taskPatch.EarliestAllowedTs != nil {
				instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find instance %d", task.InstanceID)).SetInternal(err)
				}
				if instance == nil {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Cannot found the find instance %d", task.InstanceID))
				}
				if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureTaskScheduleTime, instance); err != nil {
					return echo.NewHTTPError(http.StatusForbidden, err.Error())
				}
			}
			// Skip gh-ost cutover task as this task has no statement.
			if task.Type == api.TaskDatabaseSchemaUpdateGhostCutover {
				continue
			}
			taskPatch := *taskPatch
			taskPatch.ID = task.ID
			// TODO(d): patch tasks in batch.
			if err := s.TaskScheduler.PatchTask(ctx, task, &taskPatch, issue); err != nil {
				return err
			}
		}

		// dismiss stale review, re-find the approval template
		if taskPatch.SheetID != nil {
			payload := &storepb.IssuePayload{}
			if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to unmarshal").SetInternal(err)
			}
			payload.Approval = &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
			}
			payloadBytes, err := protojson.Marshal(payload)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue payload").SetInternal(err)
			}
			payloadStr := string(payloadBytes)
			issue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
				Payload: &payloadStr,
			}, api.SystemBotID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update issue").SetInternal(err)
			}
			s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
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

		task, err := s.store.GetTaskV2ByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID not found: %d", taskID))
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find instance %d", task.InstanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Cannot found the find instance %d", task.InstanceID))
		}
		if taskPatch.EarliestAllowedTs != nil {
			if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureTaskScheduleTime, instance); err != nil {
				return echo.NewHTTPError(http.StatusForbidden, err.Error())
			}
		}

		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID: %d", task.PipelineID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipelineID: %d", task.PipelineID))
		}

		if taskPatch.RollbackEnabled != nil && task.Type != api.TaskDatabaseDataUpdate {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot generate rollback SQL statement for a non-DML task")
		}

		if err := s.TaskScheduler.PatchTask(ctx, task, taskPatch, issue); err != nil {
			return err
		}

		// dismiss stale review, re-find the approval template
		if taskPatch.SheetID != nil && task.Status == api.TaskPendingApproval {
			payload := &storepb.IssuePayload{}
			if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to unmarshal").SetInternal(err)
			}
			payload.Approval = &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
			}
			payloadBytes, err := protojson.Marshal(payload)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue payload").SetInternal(err)
			}
			payloadStr := string(payloadBytes)
			issue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
				Payload: &payloadStr,
			}, api.SystemBotID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update issue").SetInternal(err)
			}
			s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
		}

		composedTaskPatched, err := s.store.GetTaskByID(ctx, task.ID)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedTaskPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", task.Name)).SetInternal(err)
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

		task, err := s.store.GetTaskV2ByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task not found with ID %d", taskID))
		}

		ok, err := s.TaskScheduler.CanPrincipalChangeTaskStatus(ctx, currentPrincipalID, task, taskStatusPatch.Status)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate if the principal can change task status").SetInternal(err)
		}
		if !ok {
			return echo.NewHTTPError(http.StatusForbidden, "Not allowed to change task status")
		}

		if taskStatusPatch.Status == api.TaskPending {
			issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID %d", task.PipelineID)).SetInternal(err)
			}
			if issue == nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipeline ID %d", task.PipelineID))
			}
			approved, err := utils.CheckIssueApproved(issue)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the issue is approved").SetInternal(err)
			}
			if !approved {
				return echo.NewHTTPError(http.StatusBadRequest, "Cannot patch task status because the issue is not approved")
			}

			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
			if err != nil {
				return err
			}
			taskCheckRuns, err := s.store.ListTaskCheckRuns(ctx, &store.TaskCheckRunFind{TaskID: &task.ID})
			if err != nil {
				return err
			}
			ok, err = utils.PassAllCheck(task, api.TaskCheckStatusWarn, taskCheckRuns, instance.Engine)
			if err != nil {
				return err
			}
			if !ok {
				return echo.NewHTTPError(http.StatusBadRequest, "The task has not passed all the checks yet")
			}
			stages, err := s.store.ListStageV2(ctx, task.PipelineID)
			if err != nil {
				return err
			}
			activeStage := utils.GetActiveStage(stages)
			if activeStage == nil {
				return echo.NewHTTPError(http.StatusBadRequest, "All tasks are done already")
			}
			if task.StageID != activeStage.ID {
				return echo.NewHTTPError(http.StatusBadRequest, "Tasks in the prior stage are not done yet")
			}
		}

		if taskStatusPatch.Status == api.TaskDone {
			// the user marks the task as DONE, set Skipped to true and SkippedReason to Comment.
			skipped := true
			taskStatusPatch.Skipped = &skipped
			taskStatusPatch.SkippedReason = taskStatusPatch.Comment
		}

		if err := s.TaskScheduler.PatchTaskStatus(ctx, task, taskStatusPatch); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
			}
			if common.ErrorCode(err) == common.NotImplemented {
				return echo.NewHTTPError(http.StatusNotImplemented, common.ErrorMessage(err))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\" status", task.Name)).SetInternal(err)
		}

		composedTask, err := s.store.GetTaskByID(ctx, task.ID)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", task.Name)).SetInternal(err)
		}
		return nil
	})

	g.POST("/pipeline/:pipelineID/task/:taskID/check", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		task, err := s.store.GetTaskV2ByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task not found with ID %d", taskID))
		}

		if err := s.TaskCheckScheduler.ScheduleCheck(ctx, task, c.Get(getPrincipalIDContextKey()).(int)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to run task check \"%v\"", task.Name)).SetInternal(err)
		}
		composedTask, err := s.store.GetTaskByID(ctx, task.ID)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", task.Name)).SetInternal(err)
		}
		return nil
	})
}
