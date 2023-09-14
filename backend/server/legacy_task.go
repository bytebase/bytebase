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

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (s *Server) registerTaskRoutes(g *echo.Group) {
	g.PATCH("/pipeline/:pipelineID/task/:taskID", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		taskPatch := &api.TaskPatch{
			ID:        taskID,
			UpdaterID: api.SystemBotID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update task request").SetInternal(err)
		}
		if taskPatch.SheetID != nil {
			schemaVersion := common.DefaultMigrationVersion()
			taskPatch.SchemaVersion = &schemaVersion
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

		if err := patchTask(ctx, s.store, s.activityManager, task, taskPatch, issue); err != nil {
			return err
		}

		// dismiss stale review, re-find the approval template
		if taskPatch.SheetID != nil {
			issue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
				PayloadUpsert: &storepb.IssuePayload{
					Approval: &storepb.IssuePayloadApproval{
						ApprovalFindingDone: false,
					},
				},
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
}

// patchTask patches the statement for a task.
func patchTask(ctx context.Context, stores *store.Store, activityManager *activity.Manager, task *store.TaskMessage, taskPatch *api.TaskPatch, issue *store.IssueMessage) error {
	taskPatched, err := stores.UpdateTaskV2(ctx, taskPatch)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
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
		if _, err := activityManager.CreateActivity(ctx, &store.ActivityMessage{
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
	return nil
}
