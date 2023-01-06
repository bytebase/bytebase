package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/utils"
	"github.com/bytebase/bytebase/store"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueCreate := &api.IssueCreate{
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create issue request").SetInternal(err)
		}

		issue, err := s.createIssue(ctx, issueCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueFind := &api.IssueFind{}

		pageToken := c.QueryParams().Get("token")
		// We use descending order by default for issues.
		sinceID, err := unmarshalPageToken(pageToken, api.DESC)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed page token").SetInternal(err)
		}
		issueFind.SinceID = &sinceID

		projectIDStr := c.QueryParams().Get("project")
		if projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIDStr)).SetInternal(err)
			}
			issueFind.ProjectID = &projectID
		}
		if issueStatusListStr := c.QueryParam("status"); issueStatusListStr != "" {
			statusList := []api.IssueStatus{}
			for _, status := range strings.Split(issueStatusListStr, ",") {
				statusList = append(statusList, api.IssueStatus(status))
			}
			issueFind.StatusList = statusList
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
			}
			issueFind.Limit = &limit
		} else {
			limit := api.DefaultPageSize
			issueFind.Limit = &limit
		}

		if userIDStr := c.QueryParam("user"); userIDStr != "" {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("user query parameter is not a number: %s", userIDStr)).SetInternal(err)
			}
			issueFind.PrincipalID = &userID
		}
		if creatorIDStr := c.QueryParam("creator"); creatorIDStr != "" {
			creatorID, err := strconv.Atoi(creatorIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("creator query parameter is not a number: %s", creatorIDStr)).SetInternal(err)
			}
			issueFind.CreatorID = &creatorID
		}
		if assigneeIDStr := c.QueryParam("assignee"); assigneeIDStr != "" {
			assigneeID, err := strconv.Atoi(assigneeIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("assignee query parameter is not a number: %s", assigneeIDStr)).SetInternal(err)
			}
			issueFind.AssigneeID = &assigneeID
		}
		if subscriberIDStr := c.QueryParam("subscriber"); subscriberIDStr != "" {
			subscriberID, err := strconv.Atoi(subscriberIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("subscriber query parameter is not a number: %s", subscriberIDStr)).SetInternal(err)
			}
			issueFind.SubscriberID = &subscriberID
		}

		issueList, err := s.store.FindIssueStripped(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch issue list").SetInternal(err)
		}

		for _, issue := range issueList {
			s.setTaskProgressForIssue(issue)
		}

		issueResponse := &api.IssueResponse{}
		issueResponse.Issues = issueList

		nextSinceID := sinceID
		if len(issueList) > 0 {
			// Decrement the ID as we use decreasing order by default.
			nextSinceID = issueList[len(issueList)-1].ID - 1
		}
		if issueResponse.NextToken, err = marshalPageToken(nextSinceID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal page token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueResponse); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
		}

		s.setTaskProgressForIssue(issue)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issuePatch := &api.IssuePatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issuePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update issue request").SetInternal(err)
		}

		if issuePatch.AssigneeNeedAttention != nil && !*issuePatch.AssigneeNeedAttention {
			return echo.NewHTTPError(http.StatusBadRequest, "Cannot set assigneeNeedAttention to false")
		}

		issue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
		}

		if issuePatch.AssigneeID != nil {
			if *issuePatch.AssigneeID == issue.AssigneeID {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d because it's already the case", *issuePatch.AssigneeID))
			}
			stage := utils.GetActiveStage(issue.Pipeline)
			if stage == nil {
				// all stages have finished, use the last stage
				stage = issue.Pipeline.StageList[len(issue.Pipeline.StageList)-1]
			}
			ok, err := s.TaskScheduler.CanPrincipalBeAssignee(ctx, *issuePatch.AssigneeID, stage.EnvironmentID, issue.ProjectID, issue.Type)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the assignee can be changed").SetInternal(err)
			}
			if !ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", *issuePatch.AssigneeID)).SetInternal(err)
			}

			// set AssigneeNeedAttention to false on assignee change
			if issue.Project.WorkflowType == api.UIWorkflow {
				needAttention := false
				issuePatch.AssigneeNeedAttention = &needAttention
			}
		}

		updatedIssue, err := s.store.PatchIssue(ctx, issuePatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue with ID %d", id)).SetInternal(err)
		}

		// cancel external approval on assignee change
		if issuePatch.AssigneeID != nil {
			if err := s.ApplicationRunner.CancelExternalApproval(ctx, issue.ID, api.ExternalApprovalCancelReasonReassigned); err != nil {
				log.Error("failed to cancel external approval on assignee change", zap.Int("issue_id", issue.ID), zap.Error(err))
			}
		}

		payloadList := [][]byte{}
		if issuePatch.Name != nil && *issuePatch.Name != issue.Name {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldName,
				OldValue:  issue.Name,
				NewValue:  *issuePatch.Name,
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue name: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.Description != nil && *issuePatch.Description != issue.Description {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldDescription,
				OldValue:  issue.Description,
				NewValue:  *issuePatch.Description,
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue description: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.AssigneeID != nil && *issuePatch.AssigneeID != issue.AssigneeID {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldAssignee,
				OldValue:  strconv.Itoa(issue.AssigneeID),
				NewValue:  strconv.Itoa(*issuePatch.AssigneeID),
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue assignee: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}

		for _, payload := range payloadList {
			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: issue.ID,
				Type:        api.ActivityIssueFieldUpdate,
				Level:       api.ActivityInfo,
				Payload:     string(payload),
			}
			if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
				Issue: updatedIssue,
			}); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating issue: %v", updatedIssue.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update issue response: %v", updatedIssue.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID/status", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issueStatusPatch := &api.IssueStatusPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update issue status request").SetInternal(err)
		}

		issue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
		}

		updatedIssue, err := s.TaskScheduler.ChangeIssueStatus(ctx, issue, issueStatusPatch.Status, issueStatusPatch.UpdaterID, issueStatusPatch.Comment)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound).SetInternal(err)
			} else if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) createIssue(ctx context.Context, issueCreate *api.IssueCreate) (*api.Issue, error) {
	if issueCreate.ProjectID == api.DefaultProjectID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Cannot create a new issue in the default project")
	}

	// Run pre-condition check first to make sure all tasks are valid, otherwise we will create partial pipelines
	// since we are not creating pipeline/stage list/task list in a single transaction.
	// We may still run into this issue when we actually create those pipeline/stage list/task list, however, that's
	// quite unlikely so we will live with it for now.
	pipelineCreate, err := s.getPipelineCreate(ctx, issueCreate)
	if err != nil {
		return nil, err
	}
	if len(pipelineCreate.StageList) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "no database matched for deployment")
	}
	firstEnvironmentID := pipelineCreate.StageList[0].EnvironmentID

	if issueCreate.AssigneeID == api.UnknownID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
	}
	// Try to find a more appropriate assignee if the current assignee is the system bot, indicating that the caller might not be sure about who should be the assignee.
	if issueCreate.AssigneeID == api.SystemBotID {
		assigneeID, err := s.TaskScheduler.GetDefaultAssigneeID(ctx, firstEnvironmentID, issueCreate.ProjectID, issueCreate.Type)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find a default assignee").SetInternal(err)
		}
		issueCreate.AssigneeID = assigneeID
	}
	ok, err := s.TaskScheduler.CanPrincipalBeAssignee(ctx, issueCreate.AssigneeID, firstEnvironmentID, issueCreate.ProjectID, issueCreate.Type)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the assignee can be set for the new issue").SetInternal(err)
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", issueCreate.AssigneeID))
	}

	if issueCreate.ValidateOnly {
		issue, err := s.store.CreateIssueValidateOnly(ctx, pipelineCreate, issueCreate)
		if err != nil {
			return nil, err
		}
		return issue, nil
	}

	pipeline, err := s.createPipeline(ctx, issueCreate, pipelineCreate)
	if err != nil {
		return nil, err
	}
	issueCreate.PipelineID = pipeline.ID
	issue, err := s.store.CreateIssue(ctx, issueCreate)
	if err != nil {
		return nil, err
	}
	// Create issue subscribers.
	// TODO(p0ny): create subscriber in batch.
	for _, subscriberID := range issueCreate.SubscriberIDList {
		subscriberCreate := &api.IssueSubscriberCreate{
			IssueID:      issue.ID,
			SubscriberID: subscriberID,
		}
		if _, err := s.store.CreateIssueSubscriber(ctx, subscriberCreate); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d after creating issue %d", subscriberID, issue.ID)).SetInternal(err)
		}
	}

	if err := s.TaskCheckScheduler.SchedulePipelineTaskCheck(ctx, issue.Pipeline); err != nil {
		return nil, errors.Wrapf(err, "failed to schedule task check after creating the issue: %v", issue.Name)
	}

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Name,
	}

	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Name)
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   issueCreate.CreatorID,
		ContainerID: issue.ID,
		Type:        api.ActivityIssueCreate,
		Level:       api.ActivityInfo,
		Payload:     string(bytes),
	}
	if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: issue,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Name)
	}

	if len(issue.Pipeline.StageList) > 0 {
		stage := issue.Pipeline.StageList[0]
		createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
			StageID:               stage.ID,
			StageStatusUpdateType: api.StageStatusUpdateTypeBegin,
			IssueName:             issue.Name,
			StageName:             stage.Name,
		}
		bytes, err := json.Marshal(createActivityPayload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create ActivityPipelineStageStatusUpdate activity after creating the issue: %v", issue.Name)
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: issue.PipelineID,
			Type:        api.ActivityPipelineStageStatusUpdate,
			Level:       api.ActivityInfo,
			Payload:     string(bytes),
		}
		if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to create ActivityPipelineStageStatusUpdate activity after creating the issue: %v", issue.Name)
		}
	}

	return issue, nil
}

func (s *Server) createPipeline(ctx context.Context, issueCreate *api.IssueCreate, pipelineCreate *api.PipelineCreate) (*api.Pipeline, error) {
	pipelineCreated, err := s.store.CreatePipeline(ctx, pipelineCreate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pipeline for issue")
	}

	var stageCreates []*api.StageCreate
	for i := range pipelineCreate.StageList {
		pipelineCreate.StageList[i].CreatorID = issueCreate.CreatorID
		pipelineCreate.StageList[i].PipelineID = pipelineCreated.ID
		stageCreates = append(stageCreates, &pipelineCreate.StageList[i])
	}
	createdStages, err := s.store.CreateStage(ctx, stageCreates)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stages for issue")
	}
	if len(createdStages) != len(stageCreates) {
		return nil, errors.Errorf("failed to create stages, expect to have created %d stages, got %d", len(stageCreates), len(createdStages))
	}

	for i, stageCreate := range pipelineCreate.StageList {
		createdStage := createdStages[i]

		var taskCreateList []*api.TaskCreate
		for _, taskCreate := range stageCreate.TaskList {
			c := taskCreate
			c.CreatorID = issueCreate.CreatorID
			c.PipelineID = pipelineCreated.ID
			c.StageID = createdStage.ID
			taskCreateList = append(taskCreateList, &c)
		}
		taskList, err := s.store.BatchCreateTask(ctx, taskCreateList)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create tasks for issue")
		}

		// TODO(p0ny): create task dags in batch.
		for _, indexDAG := range stageCreate.TaskIndexDAGList {
			taskDAGCreate := api.TaskDAGCreate{
				FromTaskID: taskList[indexDAG.FromIndex].ID,
				ToTaskID:   taskList[indexDAG.ToIndex].ID,
				Payload:    "{}",
			}
			if _, err := s.store.CreateTaskDAG(ctx, &taskDAGCreate); err != nil {
				return nil, errors.Wrap(err, "failed to create task DAG for issue")
			}
		}
	}

	return pipelineCreated, nil
}

func (s *Server) getPipelineCreate(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	switch issueCreate.Type {
	case api.IssueDatabaseCreate:
		return s.getPipelineCreateForDatabaseCreate(ctx, issueCreate)
	case api.IssueDatabaseRestorePITR:
		return s.getPipelineCreateForDatabasePITR(ctx, issueCreate)
	case api.IssueDatabaseSchemaUpdate, api.IssueDatabaseDataUpdate, api.IssueDatabaseSchemaUpdateGhost:
		return s.getPipelineCreateForDatabaseSchemaAndDataUpdate(ctx, issueCreate)
	case api.IssueDatabaseRollback:
		return s.getPipelineCreateForDatabaseRollback(ctx, issueCreate)
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid issue type %q", issueCreate.Type))
	}
}

func (s *Server) getPipelineCreateForDatabaseRollback(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	c := api.RollbackContext{}
	if err := json.Unmarshal([]byte(issueCreate.CreateContext), &c); err != nil {
		return nil, err
	}

	issueID := c.IssueID
	if len(c.TaskIDList) != 1 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "The task ID list must have exactly one element")
	}
	taskID := c.TaskIDList[0]
	issue, err := s.store.GetIssueByID(ctx, issueID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with ID %d", issueID)).SetInternal(err)
	}
	if issue == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", issueID))
	}
	task, err := s.store.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get task with ID %d", taskID)).SetInternal(err)
	}
	if task == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task not found with ID %d", taskID))
	}
	if task.Type != api.TaskDatabaseDataUpdate {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task type must be %s, but got %s", api.TaskDatabaseDataUpdate, task.Type))
	}
	if task.Database.Instance.Engine != db.MySQL {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Only support rollback for MySQL now, but got %s", task.Database.Instance.Engine))
	}
	if task.PipelineID != issue.PipelineID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task %d is not in issue %d", taskID, issue.ID))
	}
	if task.Status != api.TaskDone && task.Status != api.TaskFailed {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task %d has status %s, must be %s or %s", taskID, task.Status, api.TaskDone, api.TaskFailed))
	}

	taskPayload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to unmarshal the task payload with ID %d", taskID)).SetInternal(err)
	}
	switch {
	case taskPayload.RollbackStatement == "" && taskPayload.RollbackError == "":
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Rollback SQL generation for task %d is still in progress", taskID))
	case taskPayload.RollbackStatement == "" && taskPayload.RollbackError != "":
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Rollback SQL generation for task %d has already failed: %s", taskID, taskPayload.RollbackError))
	case taskPayload.RollbackStatement != "" && taskPayload.RollbackError != "":
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Invalid task payload: RollbackStatement=%q, RollbackError=%q", taskPayload.RollbackStatement, taskPayload.RollbackError))
	}

	issueCreateContext := &api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    *task.DatabaseID,
				Statement:     taskPayload.RollbackStatement,
			},
		},
	}
	bytes, err := json.Marshal(issueCreateContext)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue create context for rollback issue")
	}
	issueCreate.CreateContext = string(bytes)
	issueCreate.Type = api.IssueDatabaseDataUpdate
	pipelineCreate, err := s.getPipelineCreateForDatabaseSchemaAndDataUpdate(ctx, issueCreate)
	if err != nil {
		return nil, err
	}

	if len(pipelineCreate.StageList) != 1 {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Must have one stage for a rollback task")
	}
	if len(pipelineCreate.StageList[0].TaskList) != 1 {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Must have one task for a rollback task")
	}
	rollbackTaskPayload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(pipelineCreate.StageList[0].TaskList[0].Payload), rollbackTaskPayload); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal the rollback task create payload").SetInternal(err)
	}
	rollbackTaskPayload.RollbackFromIssueID = issueID
	rollbackTaskPayload.RollbackFromTaskID = taskID
	buf, err := json.Marshal(rollbackTaskPayload)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal rollback task payload").SetInternal(err)
	}
	pipelineCreate.StageList[0].TaskList[0].Payload = string(buf)

	return pipelineCreate, nil
}

func (s *Server) getPipelineCreateForDatabaseCreate(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	c := api.CreateDatabaseContext{}
	if err := json.Unmarshal([]byte(issueCreate.CreateContext), &c); err != nil {
		return nil, err
	}
	if c.DatabaseName == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
	}

	// Find instance.
	instance, err := s.store.GetInstanceByID(ctx, c.InstanceID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance ID not found %v", c.InstanceID)
	}

	if instance.Engine == db.MongoDB && c.TableName == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, collection name missing for MongoDB")
	}

	// Find project.
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &issueCreate.ProjectID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}
	if project == nil {
		err := errors.Errorf("project ID not found %v", issueCreate.ProjectID)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
	}

	taskCreateList, err := s.createDatabaseCreateTaskList(c, *instance, project)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create task list of creating database")
	}

	if c.BackupID != 0 {
		backup, err := s.store.GetBackupByID(ctx, c.BackupID)
		if err != nil {
			return nil, errors.Errorf("failed to find backup %v", c.BackupID)
		}
		if backup == nil {
			return nil, errors.Errorf("backup not found with ID %d", c.BackupID)
		}
		restorePayload := api.TaskDatabasePITRRestorePayload{
			ProjectID:        issueCreate.ProjectID,
			TargetInstanceID: &c.InstanceID,
			DatabaseName:     &c.DatabaseName,
			BackupID:         &c.BackupID,
		}
		restoreBytes, err := json.Marshal(restorePayload)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create restore database task, unable to marshal payload")
		}

		taskCreateList = append(taskCreateList, api.TaskCreate{
			InstanceID:   c.InstanceID,
			Name:         fmt.Sprintf("Restore backup %v", backup.Name),
			Status:       api.TaskPendingApproval,
			Type:         api.TaskDatabaseRestorePITRRestore,
			DatabaseName: c.DatabaseName,
			BackupID:     &c.BackupID,
			Payload:      string(restoreBytes),
		})

		return &api.PipelineCreate{
			Name:      fmt.Sprintf("Pipeline - Create database %v from backup %v", c.DatabaseName, backup.Name),
			CreatorID: issueCreate.CreatorID,
			StageList: []api.StageCreate{
				{
					Name:          instance.Environment.Name,
					EnvironmentID: instance.EnvironmentID,
					TaskList:      taskCreateList,
					// TODO(zp): Find a common way to merge taskCreateList and TaskIndexDAGList.
					TaskIndexDAGList: []api.TaskIndexDAG{
						{
							FromIndex: 0,
							ToIndex:   1,
						},
					},
				},
			},
		}, nil
	}

	return &api.PipelineCreate{
		Name:      fmt.Sprintf("Pipeline - Create database %s", c.DatabaseName),
		CreatorID: issueCreate.CreatorID,
		StageList: []api.StageCreate{
			{
				Name:          instance.Environment.Name,
				EnvironmentID: instance.EnvironmentID,
				TaskList:      taskCreateList,
			},
		},
	}, nil
}

func (s *Server) getPipelineCreateForDatabasePITR(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	c := api.PITRContext{}
	if err := json.Unmarshal([]byte(issueCreate.CreateContext), &c); err != nil {
		return nil, err
	}
	if c.PointInTimeTs != nil && !s.licenseService.IsFeatureEnabled(api.FeaturePITR) {
		return nil, echo.NewHTTPError(http.StatusForbidden, api.FeaturePITR.AccessErrorMessage())
	}

	database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &c.DatabaseID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", c.DatabaseID)).SetInternal(err)
	}
	if database == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", c.DatabaseID))
	}
	if database.ProjectID != issueCreate.ProjectID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The issue project %d must be the same as the database project %d.", issueCreate.ProjectID, database.ProjectID))
	}

	taskCreateList, taskIndexDAGList, err := s.createPITRTaskList(ctx, database, issueCreate.ProjectID, c)
	if err != nil {
		return nil, err
	}

	return &api.PipelineCreate{
		Name:      "Database Point-in-time Recovery pipeline",
		CreatorID: issueCreate.CreatorID,
		StageList: []api.StageCreate{
			{
				Name:             database.Instance.Environment.Name,
				EnvironmentID:    database.Instance.Environment.ID,
				TaskList:         taskCreateList,
				TaskIndexDAGList: taskIndexDAGList,
			},
		},
	}, nil
}

func (s *Server) getPipelineCreateForDatabaseSchemaAndDataUpdate(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	c := api.MigrationContext{}
	if err := json.Unmarshal([]byte(issueCreate.CreateContext), &c); err != nil {
		return nil, err
	}
	if !s.licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime) {
		for _, detail := range c.DetailList {
			if detail.EarliestAllowedTs != 0 {
				return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
			}
		}
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &issueCreate.ProjectID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}
	deployConfig, err := s.store.GetDeploymentConfigByProjectID(ctx, project.UID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch deployment config for project ID: %v", project.UID)).SetInternal(err)
	}
	deploySchedule, err := api.ValidateAndGetDeploymentSchedule(deployConfig.Payload)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get deployment schedule").SetInternal(err)
	}

	// Validate issue detail list.
	if len(c.DetailList) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "migration detail list should not be empty")
	}
	emptyDatabaseIDCount, databaseIDCount := 0, 0
	for _, detail := range c.DetailList {
		if detail.MigrationType != db.Baseline && detail.MigrationType != db.Migrate && detail.MigrationType != db.MigrateSDL && detail.MigrationType != db.Data {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "support migrate, migrateSDL and data type migration only")
		}
		if detail.MigrationType != db.Baseline && (detail.Statement == "" && detail.SheetID == 0) {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "require sql statement or sheet ID to create an issue")
		}
		if detail.Statement != "" && detail.SheetID > 0 {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Cannot set both statement and sheet ID to create an issue")
		}
		// TODO(d): validate sheet ID.
		if detail.DatabaseID > 0 {
			databaseIDCount++
		} else {
			emptyDatabaseIDCount++
		}
	}
	if emptyDatabaseIDCount > 0 && databaseIDCount > 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Migration detail should set either database name or database ID.")
	}
	if emptyDatabaseIDCount > 1 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "There should be at most one migration detail with empty database ID.")
	}
	if project.TenantMode == api.TenantModeTenant && !s.licenseService.IsFeatureEnabled(api.FeatureMultiTenancy) {
		return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
	}
	maximumTaskLimit := s.licenseService.GetPlanLimitValue(api.PlanLimitMaximumTask)
	if int64(databaseIDCount) > maximumTaskLimit {
		return nil, echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Current plan can update up to %d databases, got %d.", maximumTaskLimit, databaseIDCount))
	}

	// aggregatedMatrix is the aggregated matrix by deployments.
	// databaseToMigrationList is the mapping from database ID to migration detail.
	aggregatedMatrix := make([][]*api.Database, len(deploySchedule.Deployments))
	databaseToMigrationList := make(map[int][]*api.MigrationDetail)

	dbList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
		ProjectID: &issueCreate.ProjectID,
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", issueCreate.ProjectID)).SetInternal(err)
	}
	if databaseIDCount == 0 {
		// Deploy to all tenant databases.
		migrationDetail := c.DetailList[0]
		matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, dbList)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to build deployment pipeline").SetInternal(err)
		}
		aggregatedMatrix = matrix
		for _, databaseList := range matrix {
			for _, database := range databaseList {
				// There should be only one migration per database for tenant mode deployment.
				databaseToMigrationList[database.ID] = []*api.MigrationDetail{migrationDetail}
			}
		}
	} else {
		databaseMap := make(map[int]*api.Database)
		for _, db := range dbList {
			databaseMap[db.ID] = db
		}
		for _, d := range c.DetailList {
			database, ok := databaseMap[d.DatabaseID]
			if !ok {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID %d not found in project %d", d.DatabaseID, issueCreate.ProjectID))
			}
			if database.Instance.Engine == db.MongoDB && d.MigrationType != db.Data && d.MigrationType != db.Baseline {
				// We disallow user to create non-data migration for MongoDB.
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Cannot create non-data migration for MongoDB, consider using data migration(DML) instead.")
			}
			matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, []*api.Database{database})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to build deployment pipeline").SetInternal(err)
			}
			for i, databaseList := range matrix {
				aggregatedMatrix[i] = append(aggregatedMatrix[i], databaseList...)
			}
			databaseToMigrationList[database.ID] = append(databaseToMigrationList[database.ID], d)
		}
	}

	if issueCreate.Type == api.IssueDatabaseSchemaUpdateGhost {
		create := &api.PipelineCreate{
			Name:      "Update database schema (gh-ost) pipeline",
			CreatorID: issueCreate.CreatorID,
		}
		for i, databaseList := range aggregatedMatrix {
			// Skip the stage if the stage includes no database.
			if len(databaseList) == 0 {
				continue
			}
			var environmentID int
			var taskCreateLists [][]api.TaskCreate
			var taskIndexDAGLists [][]api.TaskIndexDAG
			for _, database := range databaseList {
				if environmentID > 0 && environmentID != database.Instance.EnvironmentID {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "all databases in a stage should have the same environment")
				}
				environmentID = database.Instance.EnvironmentID

				schemaVersion := common.DefaultMigrationVersion()
				migrationDetailList := databaseToMigrationList[database.ID]
				for _, migrationDetail := range migrationDetailList {
					taskCreateList, taskIndexDAGList, err := createGhostTaskList(database, c.VCSPushEvent, migrationDetail, schemaVersion)
					if err != nil {
						return nil, err
					}
					taskCreateLists = append(taskCreateLists, taskCreateList)
					taskIndexDAGLists = append(taskIndexDAGLists, taskIndexDAGList)
				}
			}

			taskCreateList, taskIndexDAGList, err := utils.MergeTaskCreateLists(taskCreateLists, taskIndexDAGLists)
			if err != nil {
				return nil, err
			}
			create.StageList = append(create.StageList, api.StageCreate{
				Name:             deploySchedule.Deployments[i].Name,
				EnvironmentID:    environmentID,
				TaskList:         taskCreateList,
				TaskIndexDAGList: taskIndexDAGList,
			})
		}
		return create, nil
	}
	create := &api.PipelineCreate{
		Name:      "Change database pipeline",
		CreatorID: issueCreate.CreatorID,
	}
	for i, databaseList := range aggregatedMatrix {
		// Skip the stage if the stage includes no database.
		if len(databaseList) == 0 {
			continue
		}
		var environmentID int
		var taskCreateList []api.TaskCreate
		var taskIndexDAGList []api.TaskIndexDAG
		for _, database := range databaseList {
			if environmentID > 0 && environmentID != database.Instance.EnvironmentID {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "all databases in a stage should have the same environment")
			}
			environmentID = database.Instance.EnvironmentID

			migrationDetailList := databaseToMigrationList[database.ID]
			sort.Slice(migrationDetailList, func(i, j int) bool {
				return migrationDetailList[i].SchemaVersion < migrationDetailList[j].SchemaVersion
			})
			for i := 0; i < len(migrationDetailList)-1; i++ {
				taskIndexDAGList = append(taskIndexDAGList, api.TaskIndexDAG{FromIndex: len(taskCreateList) + i, ToIndex: len(taskCreateList) + i + 1})
			}
			for _, migrationDetail := range migrationDetailList {
				taskCreate, err := getUpdateTask(database, c.VCSPushEvent, migrationDetail, getOrDefaultSchemaVersion(migrationDetail))
				if err != nil {
					return nil, err
				}
				taskCreateList = append(taskCreateList, taskCreate)
			}
		}

		create.StageList = append(create.StageList, api.StageCreate{
			Name:             deploySchedule.Deployments[i].Name,
			EnvironmentID:    environmentID,
			TaskList:         taskCreateList,
			TaskIndexDAGList: taskIndexDAGList,
		})
	}
	return create, nil
}

func getOrDefaultSchemaVersion(detail *api.MigrationDetail) string {
	if detail.SchemaVersion != "" {
		return detail.SchemaVersion
	}
	return common.DefaultMigrationVersion()
}

func getUpdateTask(database *api.Database, vcsPushEvent *vcs.PushEvent, d *api.MigrationDetail, schemaVersion string) (api.TaskCreate, error) {
	var taskName string
	var taskType api.TaskType

	var payloadString string
	switch d.MigrationType {
	case db.Baseline:
		taskName = fmt.Sprintf("Establish baseline for database %q", database.Name)
		taskType = api.TaskDatabaseSchemaBaseline
		payload := api.TaskDatabaseSchemaBaselinePayload{
			SchemaVersion: schemaVersion,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return api.TaskCreate{}, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database schema baseline payload").SetInternal(err)
		}
		payloadString = string(bytes)
	case db.Migrate:
		taskName = fmt.Sprintf("DDL(schema) for database %q", database.Name)
		taskType = api.TaskDatabaseSchemaUpdate
		payload := api.TaskDatabaseSchemaUpdatePayload{
			Statement:     d.Statement,
			SheetID:       d.SheetID,
			SchemaVersion: schemaVersion,
			VCSPushEvent:  vcsPushEvent,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return api.TaskCreate{}, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database schema update payload").SetInternal(err)
		}
		payloadString = string(bytes)
	case db.MigrateSDL:
		taskName = fmt.Sprintf("SDL for database %q", database.Name)
		taskType = api.TaskDatabaseSchemaUpdateSDL
		payload := api.TaskDatabaseSchemaUpdateSDLPayload{
			Statement:     d.Statement,
			SheetID:       d.SheetID,
			SchemaVersion: schemaVersion,
			VCSPushEvent:  vcsPushEvent,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return api.TaskCreate{}, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database schema update SDL payload").SetInternal(err)
		}
		payloadString = string(bytes)
	case db.Data:
		taskName = fmt.Sprintf("DML(data) for database %q", database.Name)
		taskType = api.TaskDatabaseDataUpdate
		payload := api.TaskDatabaseDataUpdatePayload{
			Statement:     d.Statement,
			SheetID:       d.SheetID,
			SchemaVersion: schemaVersion,
			VCSPushEvent:  vcsPushEvent,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return api.TaskCreate{}, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database data update payload").SetInternal(err)
		}
		payloadString = string(bytes)
	default:
		return api.TaskCreate{}, errors.Errorf("unsupported migration type %q", d.MigrationType)
	}

	return api.TaskCreate{
		Name:              taskName,
		InstanceID:        database.Instance.ID,
		DatabaseID:        &database.ID,
		Status:            api.TaskPendingApproval,
		Type:              taskType,
		Statement:         d.Statement,
		EarliestAllowedTs: d.EarliestAllowedTs,
		Payload:           payloadString,
	}, nil
}

// createDatabaseCreateTaskList returns the task list for create database.
func (s *Server) createDatabaseCreateTaskList(c api.CreateDatabaseContext, instance api.Instance, project *store.ProjectMessage) ([]api.TaskCreate, error) {
	if err := checkCharacterSetCollationOwner(instance.Engine, c.CharacterSet, c.Collation, c.Owner); err != nil {
		return nil, err
	}
	if c.DatabaseName == "" {
		return nil, util.FormatError(common.Errorf(common.Invalid, "Failed to create issue, database name missing"))
	}
	if instance.Engine == db.Snowflake {
		// Snowflake needs to use upper case of DatabaseName.
		c.DatabaseName = strings.ToUpper(c.DatabaseName)
	}
	if instance.Engine == db.MongoDB && c.TableName == "" {
		return nil, util.FormatError(common.Errorf(common.Invalid, "Failed to create issue, collection name missing for MongoDB"))
	}
	// Validate the labels. Labels are set upon task completion.
	if _, err := convertDatabaseLabels(c.Labels, &api.Database{Name: c.DatabaseName, Instance: &instance} /* dummy database */); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid database label %q, error %v", c.Labels, err))
	}

	// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
	if project.TenantMode == api.TenantModeTenant {
		if !s.licenseService.IsFeatureEnabled(api.FeatureMultiTenancy) {
			return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
	}

	// Get admin data source username.
	adminDataSource := api.DataSourceFromInstanceWithType(&instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %d", instance.ID)
	}
	// Snowflake needs to use upper case of DatabaseName.
	databaseName := c.DatabaseName
	if instance.Engine == db.Snowflake {
		databaseName = strings.ToUpper(databaseName)
	}
	statement, err := getCreateDatabaseStatement(instance.Engine, c, databaseName, adminDataSource.Username)
	if err != nil {
		return nil, err
	}
	payload := api.TaskDatabaseCreatePayload{
		ProjectID:    project.UID,
		CharacterSet: c.CharacterSet,
		TableName:    c.TableName,
		Collation:    c.Collation,
		Labels:       c.Labels,
		DatabaseName: databaseName,
		Statement:    statement,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create database creation task, unable to marshal payload")
	}

	return []api.TaskCreate{
		{
			InstanceID:   c.InstanceID,
			Name:         fmt.Sprintf("Create database %v", payload.DatabaseName),
			Status:       api.TaskPendingApproval,
			Type:         api.TaskDatabaseCreate,
			DatabaseName: payload.DatabaseName,
			Payload:      string(bytes),
		},
	}, nil
}

func (s *Server) createPITRTaskList(ctx context.Context, originDatabase *api.Database, projectID int, c api.PITRContext) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	var taskCreateList []api.TaskCreate
	// Restore payload
	payloadRestore := api.TaskDatabasePITRRestorePayload{
		ProjectID: projectID,
	}

	// PITR to new db: task 1
	if c.CreateDatabaseCtx != nil {
		targetInstance, err := s.store.GetInstanceByID(ctx, c.CreateDatabaseCtx.InstanceID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find the instance with ID %d", c.CreateDatabaseCtx.InstanceID)
		}
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find the project with ID %d", projectID)
		}
		taskList, err := s.createDatabaseCreateTaskList(*c.CreateDatabaseCtx, *targetInstance, project)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create the database create task list")
		}
		taskCreateList = append(taskCreateList, taskList...)

		payloadRestore.TargetInstanceID = &targetInstance.ID
		payloadRestore.DatabaseName = &c.CreateDatabaseCtx.DatabaseName
	}

	if c.BackupID != nil {
		payloadRestore.BackupID = c.BackupID
	}

	payloadRestore.PointInTimeTs = c.PointInTimeTs
	bytesRestore, err := json.Marshal(payloadRestore)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create PITR restore task, unable to marshal payload")
	}

	restoreTaskCreate := api.TaskCreate{
		Status:     api.TaskPendingApproval,
		Type:       api.TaskDatabaseRestorePITRRestore,
		InstanceID: originDatabase.InstanceID,
		DatabaseID: &originDatabase.ID,
		Payload:    string(bytesRestore),
		BackupID:   c.BackupID,
	}

	if payloadRestore.TargetInstanceID != nil {
		// PITR to new database: task 2
		restoreTaskCreate.Name = fmt.Sprintf("Restore to new database %q", *payloadRestore.DatabaseName)
		restoreTaskCreate.DatabaseName = c.CreateDatabaseCtx.DatabaseName
	} else {
		// PITR in place: task 1
		restoreTaskCreate.Name = fmt.Sprintf("Restore to PITR database %q", originDatabase.Name)
	}
	taskCreateList = append(taskCreateList, restoreTaskCreate)

	// PITR in place: task 2
	if payloadRestore.TargetInstanceID == nil {
		payloadCutover := api.TaskDatabasePITRCutoverPayload{}
		bytesCutover, err := json.Marshal(payloadCutover)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create PITR cutover task, unable to marshal payload")
		}
		taskCreateList = append(taskCreateList, api.TaskCreate{
			Name:       fmt.Sprintf("Swap PITR and the original database %q", originDatabase.Name),
			InstanceID: originDatabase.InstanceID,
			DatabaseID: &originDatabase.ID,
			Status:     api.TaskPendingApproval,
			Type:       api.TaskDatabaseRestorePITRCutover,
			Payload:    string(bytesCutover),
		})
	}
	// We make sure that createPITRTaskList will always return 2 tasks.
	taskIndexDAGList := []api.TaskIndexDAG{
		{
			FromIndex: 0,
			ToIndex:   1,
		},
	}
	return taskCreateList, taskIndexDAGList, nil
}

func getCreateDatabaseStatement(dbType db.Type, createDatabaseContext api.CreateDatabaseContext, databaseName, adminDatasourceUser string) (string, error) {
	var stmt string
	switch dbType {
	case db.MySQL, db.TiDB:
		return fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", databaseName, createDatabaseContext.CharacterSet, createDatabaseContext.Collation), nil
	case db.Postgres:
		// On Cloud RDS, the data source role isn't the actual superuser with sudo privilege.
		// We need to grant the database owner role to the data source admin so that Bytebase can have permission for the database using the data source admin.
		if adminDatasourceUser != "" && createDatabaseContext.Owner != adminDatasourceUser {
			stmt = fmt.Sprintf("GRANT \"%s\" TO \"%s\";\n", createDatabaseContext.Owner, adminDatasourceUser)
		}
		if createDatabaseContext.Collation == "" {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q;", stmt, databaseName, createDatabaseContext.CharacterSet)
		} else {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q;", stmt, databaseName, createDatabaseContext.CharacterSet, createDatabaseContext.Collation)
		}
		// Set the database owner.
		// We didn't use CREATE DATABASE WITH OWNER because RDS requires the current role to be a member of the database owner.
		// However, people can still use ALTER DATABASE to change the owner afterwards.
		// Error string below:
		// query: CREATE DATABASE h1 WITH OWNER hello;
		// ERROR:  must be member of role "hello"
		//
		// For tenant project, the schema for the newly created database will belong to the same owner.
		// TODO(d): alter schema "public" owner to the database owner.
		return fmt.Sprintf("%s\nALTER DATABASE \"%s\" OWNER TO %s;", stmt, databaseName, createDatabaseContext.Owner), nil
	case db.ClickHouse:
		clusterPart := ""
		if createDatabaseContext.Cluster != "" {
			clusterPart = fmt.Sprintf(" ON CLUSTER `%s`", createDatabaseContext.Cluster)
		}
		return fmt.Sprintf("CREATE DATABASE `%s`%s;", databaseName, clusterPart), nil
	case db.Snowflake:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.SQLite:
		// This is a fake CREATE DATABASE and USE statement since a single SQLite file represents a database. Engine driver will recognize it and establish a connection to create the sqlite file representing the database.
		return fmt.Sprintf("CREATE DATABASE '%s';", databaseName), nil
	case db.MongoDB:
		// We just run createCollection in mongosh instead of execute `use <database>` first, because we execute the
		// mongodb statement in mongosh with --file flag, and it doesn't support `use <database>` statement in the file.
		// And we pass the database name to Bytebase engine driver, which will be used to build the connection string.
		return fmt.Sprintf(`db.createCollection("%s");`, createDatabaseContext.TableName), nil
	}
	return "", errors.Errorf("unsupported database type %s", dbType)
}

// creates gh-ost TaskCreate list and dependency.
func createGhostTaskList(database *api.Database, vcsPushEvent *vcs.PushEvent, detail *api.MigrationDetail, schemaVersion string) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	var taskCreateList []api.TaskCreate
	// task "sync"
	payloadSync := api.TaskDatabaseSchemaUpdateGhostSyncPayload{
		Statement:     detail.Statement,
		SchemaVersion: schemaVersion,
		VCSPushEvent:  vcsPushEvent,
	}
	bytesSync, err := json.Marshal(payloadSync)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to marshal database schema update gh-ost sync payload, error: %v", err))
	}
	taskCreateList = append(taskCreateList, api.TaskCreate{
		Name:              fmt.Sprintf("Update schema gh-ost sync for database %q", database.Name),
		InstanceID:        database.InstanceID,
		DatabaseID:        &database.ID,
		Status:            api.TaskPendingApproval,
		Type:              api.TaskDatabaseSchemaUpdateGhostSync,
		Statement:         detail.Statement,
		EarliestAllowedTs: detail.EarliestAllowedTs,
		Payload:           string(bytesSync),
	})

	// task "cutover"
	payloadCutover := api.TaskDatabaseSchemaUpdateGhostCutoverPayload{}
	bytesCutover, err := json.Marshal(payloadCutover)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to marshal database schema update ghost cutover payload, error: %v", err))
	}
	taskCreateList = append(taskCreateList, api.TaskCreate{
		Name:              fmt.Sprintf("Update schema gh-ost cutover for database %q", database.Name),
		InstanceID:        database.InstanceID,
		DatabaseID:        &database.ID,
		Status:            api.TaskPendingApproval,
		Type:              api.TaskDatabaseSchemaUpdateGhostCutover,
		EarliestAllowedTs: detail.EarliestAllowedTs,
		Payload:           string(bytesCutover),
	})

	// The below list means that taskCreateList[0] blocks taskCreateList[1].
	// In other words, task "sync" blocks task "cutover".
	taskIndexDAGList := []api.TaskIndexDAG{
		{FromIndex: 0, ToIndex: 1},
	}
	return taskCreateList, taskIndexDAGList, nil
}

// checkCharacterSetCollationOwner checks if the character set, collation and owner are legal according to the dbType.
func checkCharacterSetCollationOwner(dbType db.Type, characterSet, collation, owner string) error {
	switch dbType {
	case db.ClickHouse:
		// ClickHouse does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("ClickHouse does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("ClickHouse does not support collation, but got %s", collation)
		}
	case db.Snowflake:
		if characterSet != "" {
			return errors.Errorf("Snowflake does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Snowflake does not support collation, but got %s", collation)
		}
	case db.Postgres:
		if owner == "" {
			return errors.Errorf("database owner is required for PostgreSQL")
		}
	case db.SQLite, db.MongoDB:
		// no-op.
	default:
		if characterSet == "" {
			return errors.Errorf("character set missing for %s", string(dbType))
		}
		// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
		// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
		// install it.
		if collation == "" {
			return errors.Errorf("collation missing for %s", string(dbType))
		}
	}
	return nil
}

func (s *Server) setTaskProgressForIssue(issue *api.Issue) {
	if s.TaskScheduler == nil {
		// readonly server doesn't have a TaskScheduler.
		return
	}
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if progress, ok := s.stateCfg.TaskProgress.Load(task.ID); ok {
				task.Progress = progress.(api.Progress)
			}
		}
	}
}

func marshalPageToken(id int) (string, error) {
	b, err := json.Marshal(id)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal page token")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// unmarshalPageToken unmarshals the page token, and returns the last issue ID. If the page token nil empty, it returns MaxInt32.
func unmarshalPageToken(pageToken string, sortOrder api.SortOrder) (int, error) {
	if pageToken == "" {
		if sortOrder == api.ASC {
			return 0, nil
		}
		return math.MaxInt32, nil
	}

	bs, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return 0, errors.Wrap(err, "failed to decode page token")
	}

	var id int
	if err := json.Unmarshal(bs, &id); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal page token")
	}

	return id, nil
}

// convertDatabaseLabels cnverts the json labels.
func convertDatabaseLabels(labelsJSON string, database *api.Database) ([]*api.DatabaseLabel, error) {
	if labelsJSON == "" {
		return nil, nil
	}
	var labels []*api.DatabaseLabel
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		return nil, err
	}

	// For scalability, each database can have up to four labels for now.
	if len(labels) > api.DatabaseLabelSizeMax {
		err := errors.Errorf("database labels are up to a maximum of %d", api.DatabaseLabelSizeMax)
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	if err := validateDatabaseLabelList(labels, database.Instance.Environment.Name); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to validate database labels").SetInternal(err)
	}

	return labels, nil
}

func validateDatabaseLabelList(labelList []*api.DatabaseLabel, environmentName string) error {
	var environmentValue *string

	// check label key & value availability
	for _, label := range labelList {
		if label.Key == api.EnvironmentLabelKey {
			environmentValue = &label.Value
			continue
		}
	}

	// Environment label must exist and is immutable.
	if environmentValue == nil {
		return common.Errorf(common.NotFound, "database label key %v not found", api.EnvironmentLabelKey)
	}
	if environmentName != *environmentValue {
		return common.Errorf(common.Invalid, "cannot mutate database label key %v from %v to %v", api.EnvironmentLabelKey, environmentName, *environmentValue)
	}

	return nil
}
