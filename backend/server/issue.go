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

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueCreate := &api.IssueCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create issue request").SetInternal(err)
		}

		issue, err := s.createIssue(ctx, issueCreate, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
		}

		if s.MetricReporter != nil {
			s.MetricReporter.Report(&metric.Metric{
				Name:  metricAPI.IssueCreateMetricName,
				Value: 1,
				Labels: map[string]interface{}{
					"type": issue.Type,
				},
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueFind := &store.FindIssueMessage{}

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
			issueFind.ProjectUID = &projectID
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
		updaterID := c.Get(getPrincipalIDContextKey()).(int)

		issuePatch := &api.IssuePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issuePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update issue request").SetInternal(err)
		}

		if issuePatch.AssigneeNeedAttention != nil && !*issuePatch.AssigneeNeedAttention {
			return echo.NewHTTPError(http.StatusBadRequest, "Cannot set assigneeNeedAttention to false")
		}

		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
		}
		updateIssueMessage := &store.UpdateIssueMessage{
			Title:         issuePatch.Name,
			Description:   issuePatch.Description,
			NeedAttention: issuePatch.AssigneeNeedAttention,
			Payload:       issuePatch.Payload,
		}

		if issuePatch.AssigneeID != nil {
			assignee, err := s.store.GetUserByID(ctx, *issuePatch.AssigneeID)
			if err != nil {
				return err
			}
			updateIssueMessage.Assignee = assignee
			stages, err := s.store.ListStageV2(ctx, issue.PipelineUID)
			if err != nil {
				return err
			}
			activeStage := utils.GetActiveStage(stages)
			// When all stages have finished, assignee can be anyone such as creator.
			if activeStage != nil {
				ok, err := s.TaskScheduler.CanPrincipalBeAssignee(ctx, assignee.ID, activeStage.EnvironmentID, issue.Project.UID, issue.Type)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the assignee can be changed").SetInternal(err)
				}
				if !ok {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", *issuePatch.AssigneeID)).SetInternal(err)
				}
			}
			// set AssigneeNeedAttention to false on assignee change
			if issue.Project.Workflow == api.UIWorkflow {
				needAttention := false
				updateIssueMessage.NeedAttention = &needAttention
			}
		}

		updatedIssue, err := s.store.UpdateIssueV2(ctx, id, updateIssueMessage, updaterID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue with ID %d", id)).SetInternal(err)
		}
		// cancel external approval on assignee change
		if issuePatch.AssigneeID != nil {
			if err := s.ApplicationRunner.CancelExternalApproval(ctx, issue.UID, api.ExternalApprovalCancelReasonReassigned); err != nil {
				log.Error("failed to cancel external approval on assignee change", zap.Int("issue_id", issue.UID), zap.Error(err))
			}
		}

		payloadList := [][]byte{}
		if issuePatch.Name != nil && *issuePatch.Name != issue.Title {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldName,
				OldValue:  issue.Title,
				NewValue:  *issuePatch.Name,
				IssueName: issue.Title,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue name: %v", updatedIssue.Title)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.Description != nil && *issuePatch.Description != issue.Description {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldDescription,
				OldValue:  issue.Description,
				NewValue:  *issuePatch.Description,
				IssueName: issue.Title,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue description: %v", updatedIssue.Title)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.AssigneeID != nil && *issuePatch.AssigneeID != issue.Assignee.ID {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldAssignee,
				OldValue:  strconv.Itoa(issue.Assignee.ID),
				NewValue:  strconv.Itoa(*issuePatch.AssigneeID),
				IssueName: issue.Title,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue assignee: %v", updatedIssue.Title)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}

		for _, payload := range payloadList {
			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: issue.UID,
				Type:        api.ActivityIssueFieldUpdate,
				Level:       api.ActivityInfo,
				Payload:     string(payload),
			}
			if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
				Issue: updatedIssue,
			}); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating issue: %v", updatedIssue.Title)).SetInternal(err)
			}
		}

		composedIssue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update issue response: %v", updatedIssue.Title)).SetInternal(err)
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

		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
		}

		if err := s.TaskScheduler.ChangeIssueStatus(ctx, issue, issueStatusPatch.Status, issueStatusPatch.UpdaterID, issueStatusPatch.Comment); err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound).SetInternal(err)
			} else if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		updatedComposedIssue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedComposedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) createIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Issue, error) {
	if issueCreate.ProjectID == api.DefaultProjectUID {
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
		assignee, err := s.TaskScheduler.GetDefaultAssignee(ctx, firstEnvironmentID, issueCreate.ProjectID, issueCreate.Type)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find a default assignee").SetInternal(err)
		}
		issueCreate.AssigneeID = assignee.ID
	}
	ok, err := s.TaskScheduler.CanPrincipalBeAssignee(ctx, issueCreate.AssigneeID, firstEnvironmentID, issueCreate.ProjectID, issueCreate.Type)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the assignee can be set for the new issue").SetInternal(err)
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", issueCreate.AssigneeID))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &issueCreate.ProjectID})
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project %d not found", issueCreate.ProjectID))
	}
	assignee, err := s.store.GetUserByID(ctx, issueCreate.AssigneeID)
	if err != nil {
		return nil, err
	}
	if assignee == nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("assignee %d not found", issueCreate.AssigneeID))
	}
	issueCreateMessage := &store.IssueMessage{
		Project:       project,
		Title:         issueCreate.Name,
		Type:          issueCreate.Type,
		Description:   issueCreate.Description,
		Assignee:      assignee,
		NeedAttention: issueCreate.AssigneeNeedAttention,
		Payload:       issueCreate.Payload,
	}

	if issueCreate.ValidateOnly {
		issue, err := s.store.CreateIssueValidateOnly(ctx, pipelineCreate, issueCreateMessage, creatorID)
		if err != nil {
			return nil, err
		}
		return issue, nil
	}

	pipeline, err := s.createPipeline(ctx, creatorID, pipelineCreate)
	if err != nil {
		return nil, err
	}
	issueCreateMessage.PipelineUID = pipeline.ID
	issue, err := s.store.CreateIssueV2(ctx, issueCreateMessage, creatorID)
	if err != nil {
		return nil, err
	}
	composedIssue, err := s.store.GetIssueByID(ctx, issue.UID)
	if err != nil {
		return nil, err
	}

	if err := s.TaskCheckScheduler.SchedulePipelineTaskCheck(ctx, project, pipeline.ID); err != nil {
		return nil, errors.Wrapf(err, "failed to schedule task check after creating the issue: %v", issue.Title)
	}

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Title,
	}

	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: issue.UID,
		Type:        api.ActivityIssueCreate,
		Level:       api.ActivityInfo,
		Payload:     string(bytes),
	}
	if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: issue,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}

	if len(composedIssue.Pipeline.StageList) > 0 {
		stage := composedIssue.Pipeline.StageList[0]
		createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
			StageID:               stage.ID,
			StageStatusUpdateType: api.StageStatusUpdateTypeBegin,
			IssueName:             issue.Title,
			StageName:             stage.Name,
		}
		bytes, err := json.Marshal(createActivityPayload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create ActivityPipelineStageStatusUpdate activity after creating the issue: %v", issue.Title)
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: issue.PipelineUID,
			Type:        api.ActivityPipelineStageStatusUpdate,
			Level:       api.ActivityInfo,
			Payload:     string(bytes),
		}
		if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to create ActivityPipelineStageStatusUpdate activity after creating the issue: %v", issue.Title)
		}
	}

	return composedIssue, nil
}

func (s *Server) createPipeline(ctx context.Context, creatorID int, pipelineCreate *api.PipelineCreate) (*store.PipelineMessage, error) {
	pipelineCreated, err := s.store.CreatePipelineV2(ctx, &store.PipelineMessage{Name: pipelineCreate.Name}, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pipeline for issue")
	}

	var stageCreates []*store.StageMessage
	for _, stage := range pipelineCreate.StageList {
		stageCreates = append(stageCreates, &store.StageMessage{
			Name:          stage.Name,
			EnvironmentID: stage.EnvironmentID,
			PipelineID:    pipelineCreated.ID,
		})
	}
	createdStages, err := s.store.CreateStageV2(ctx, stageCreates, creatorID)
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
			c.CreatorID = creatorID
			c.PipelineID = pipelineCreated.ID
			c.StageID = createdStage.ID
			taskCreateList = append(taskCreateList, &c)
		}
		tasks, err := s.store.CreateTasksV2(ctx, taskCreateList...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create tasks for issue")
		}

		// TODO(p0ny): create task dags in batch.
		for _, indexDAG := range stageCreate.TaskIndexDAGList {
			if err := s.store.CreateTaskDAGV2(ctx, &store.TaskDAGMessage{
				FromTaskID: tasks[indexDAG.FromIndex].ID,
				ToTaskID:   tasks[indexDAG.ToIndex].ID,
			}); err != nil {
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
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid issue type %q", issueCreate.Type))
	}
}

func (s *Server) getPipelineCreateForDatabaseCreate(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	c := api.CreateDatabaseContext{}
	if err := json.Unmarshal([]byte(issueCreate.CreateContext), &c); err != nil {
		return nil, err
	}
	if c.DatabaseName == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &c.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance ID not found %v", c.InstanceID)
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment ID not found %v", instance.EnvironmentID)
	}

	if instance.Engine == db.MongoDB && c.TableName == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, collection name missing for MongoDB")
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &issueCreate.ProjectID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}
	if project == nil {
		err := errors.Errorf("project ID not found %v", issueCreate.ProjectID)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
	}

	taskCreateList, err := s.createDatabaseCreateTaskList(c, instance, project)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create task list of creating database")
	}

	if c.BackupID != 0 {
		backup, err := s.store.GetBackupV2(ctx, c.BackupID)
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
			Name: fmt.Sprintf("Pipeline - Create database %v from backup %v", c.DatabaseName, backup.Name),
			StageList: []api.StageCreate{
				{
					Name:          environment.Title,
					EnvironmentID: environment.UID,
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
		Name: fmt.Sprintf("Pipeline - Create database %s", c.DatabaseName),
		StageList: []api.StageCreate{
			{
				Name:          environment.Title,
				EnvironmentID: environment.UID,
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
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &issueCreate.ProjectID})
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("project %d not found", issueCreate.ProjectID))
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &c.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("database %d not found", c.DatabaseID))
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", database.InstanceID)
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("environment %q not found", database.EnvironmentID))
	}
	if database.ProjectID != project.ResourceID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The issue project %d must be the same as the database project %q.", issueCreate.ProjectID, database.ProjectID))
	}

	taskCreateList, taskIndexDAGList, err := s.createPITRTaskList(ctx, database, instance, issueCreate.ProjectID, c)
	if err != nil {
		return nil, err
	}

	return &api.PipelineCreate{
		Name: "Database Point-in-time Recovery pipeline",
		StageList: []api.StageCreate{
			{
				Name:             environment.Title,
				EnvironmentID:    environment.UID,
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
	deploymentConfig, err := s.store.GetDeploymentConfigV2(ctx, project.UID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch deployment config for project ID: %v", project.UID)).SetInternal(err)
	}
	apiDeploymentConfig, err := deploymentConfig.ToAPIDeploymentConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to convert deployment config for project ID: %v", project.UID)
	}

	deploySchedule, err := api.ValidateAndGetDeploymentSchedule(apiDeploymentConfig.Payload)
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
	aggregatedMatrix := make([][]*store.DatabaseMessage, len(deploySchedule.Deployments))
	databaseToMigrationList := make(map[int][]*api.MigrationDetail)

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", issueCreate.ProjectID)).SetInternal(err)
	}
	databaseMap := make(map[int]*store.DatabaseMessage)
	for _, database := range databases {
		databaseMap[database.UID] = database
	}

	if databaseIDCount == 0 {
		// Deploy to all tenant databases.
		migrationDetail := c.DetailList[0]
		matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, databases)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to build deployment pipeline").SetInternal(err)
		}
		aggregatedMatrix = matrix
		for _, databaseList := range matrix {
			for _, database := range databaseList {
				// There should be only one migration per database for tenant mode deployment.
				databaseToMigrationList[database.UID] = []*api.MigrationDetail{migrationDetail}
			}
		}
	} else {
		for _, d := range c.DetailList {
			database, ok := databaseMap[d.DatabaseID]
			if !ok {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID %d not found in project %d", d.DatabaseID, issueCreate.ProjectID))
			}
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instance not found for database %v", d.DatabaseID))
			}
			if instance.Engine == db.MongoDB && d.MigrationType != db.Data && d.MigrationType != db.Baseline {
				// We disallow user to create non-data migration for MongoDB.
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Cannot create non-data migration for MongoDB, consider using data migration(DML) instead.")
			}
			matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, []*store.DatabaseMessage{database})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to build deployment pipeline").SetInternal(err)
			}
			for i, databaseList := range matrix {
				if len(databaseList) == 0 {
					continue
				} else if len(databaseList) > 1 {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "should be at most one database in the matrix stage")
				}
				database := databaseList[0]
				found := false
				for _, v := range aggregatedMatrix[i] {
					if v == database {
						found = true
						break
					}
				}
				if !found {
					aggregatedMatrix[i] = append(aggregatedMatrix[i], database)
				}
			}
			databaseToMigrationList[database.UID] = append(databaseToMigrationList[database.UID], d)
		}
	}

	if issueCreate.Type == api.IssueDatabaseSchemaUpdateGhost {
		create := &api.PipelineCreate{
			Name: "Update database schema (gh-ost) pipeline",
		}
		for i, databaseList := range aggregatedMatrix {
			// Skip the stage if the stage includes no database.
			if len(databaseList) == 0 {
				continue
			}
			var environmentID string
			var taskCreateLists [][]api.TaskCreate
			var taskIndexDAGLists [][]api.TaskIndexDAG
			for _, database := range databaseList {
				if environmentID != "" && environmentID != database.EnvironmentID {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "all databases in a stage should have the same environment")
				}
				environmentID = database.EnvironmentID

				schemaVersion := common.DefaultMigrationVersion()
				migrationDetailList := databaseToMigrationList[database.UID]
				for _, migrationDetail := range migrationDetailList {
					instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
					if err != nil {
						return nil, err
					}
					taskCreateList, taskIndexDAGList, err := createGhostTaskList(database, instance, c.VCSPushEvent, migrationDetail, schemaVersion)
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
			environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
			if err != nil {
				return nil, err
			}
			create.StageList = append(create.StageList, api.StageCreate{
				Name:             deploySchedule.Deployments[i].Name,
				EnvironmentID:    environment.UID,
				TaskList:         taskCreateList,
				TaskIndexDAGList: taskIndexDAGList,
			})
		}
		return create, nil
	}
	create := &api.PipelineCreate{
		Name: "Change database pipeline",
	}
	for i, databaseList := range aggregatedMatrix {
		// Skip the stage if the stage includes no database.
		if len(databaseList) == 0 {
			continue
		}
		var environmentID string
		var taskCreateList []api.TaskCreate
		var taskIndexDAGList []api.TaskIndexDAG
		for _, database := range databaseList {
			if environmentID != "" && environmentID != database.EnvironmentID {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "all databases in a stage should have the same environment")
			}
			environmentID = database.EnvironmentID
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instance not found for database %v", database.UID))
			}

			migrationDetailList := databaseToMigrationList[database.UID]
			sort.Slice(migrationDetailList, func(i, j int) bool {
				return migrationDetailList[i].SchemaVersion < migrationDetailList[j].SchemaVersion
			})
			for i := 0; i < len(migrationDetailList)-1; i++ {
				taskIndexDAGList = append(taskIndexDAGList, api.TaskIndexDAG{FromIndex: len(taskCreateList) + i, ToIndex: len(taskCreateList) + i + 1})
			}
			for _, migrationDetail := range migrationDetailList {
				taskCreate, err := getUpdateTask(database, instance, c.VCSPushEvent, migrationDetail, getOrDefaultSchemaVersion(migrationDetail))
				if err != nil {
					return nil, err
				}
				taskCreateList = append(taskCreateList, taskCreate)
			}
		}

		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
		if err != nil {
			return nil, err
		}
		create.StageList = append(create.StageList, api.StageCreate{
			Name:             deploySchedule.Deployments[i].Name,
			EnvironmentID:    environment.UID,
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

func getUpdateTask(database *store.DatabaseMessage, instance *store.InstanceMessage, vcsPushEvent *vcs.PushEvent, d *api.MigrationDetail, schemaVersion string) (api.TaskCreate, error) {
	var taskName string
	var taskType api.TaskType

	var payloadString string
	switch d.MigrationType {
	case db.Baseline:
		taskName = fmt.Sprintf("Establish baseline for database %q", database.DatabaseName)
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
		taskName = fmt.Sprintf("DDL(schema) for database %q", database.DatabaseName)
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
		taskName = fmt.Sprintf("SDL for database %q", database.DatabaseName)
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
		taskName = fmt.Sprintf("DML(data) for database %q", database.DatabaseName)
		taskType = api.TaskDatabaseDataUpdate
		payload := api.TaskDatabaseDataUpdatePayload{
			Statement:         d.Statement,
			SheetID:           d.SheetID,
			SchemaVersion:     schemaVersion,
			VCSPushEvent:      vcsPushEvent,
			RollbackEnabled:   d.RollbackEnabled,
			RollbackSQLStatus: api.RollbackSQLStatusPending,
		}
		if d.RollbackDetail != nil {
			payload.RollbackFromIssueID = d.RollbackDetail.IssueID
			payload.RollbackFromTaskID = d.RollbackDetail.TaskID
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
		InstanceID:        instance.UID,
		DatabaseID:        &database.UID,
		Status:            api.TaskPendingApproval,
		Type:              taskType,
		Statement:         d.Statement,
		EarliestAllowedTs: d.EarliestAllowedTs,
		Payload:           payloadString,
	}, nil
}

// createDatabaseCreateTaskList returns the task list for create database.
func (s *Server) createDatabaseCreateTaskList(c api.CreateDatabaseContext, instance *store.InstanceMessage, project *store.ProjectMessage) ([]api.TaskCreate, error) {
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
	if _, err := convertDatabaseLabels(c.Labels); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid database label %q, error: %v", c.Labels, err))
	}

	// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
	if project.TenantMode == api.TenantModeTenant {
		if !s.licenseService.IsFeatureEnabled(api.FeatureMultiTenancy) {
			return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
	}

	// Get admin data source username.
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.Title)
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

func (s *Server) createPITRTaskList(ctx context.Context, originDatabase *store.DatabaseMessage, instance *store.InstanceMessage, projectID int, c api.PITRContext) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	var taskCreateList []api.TaskCreate
	// Restore payload
	payloadRestore := api.TaskDatabasePITRRestorePayload{
		ProjectID: projectID,
	}

	// PITR to new db: task 1
	if c.CreateDatabaseCtx != nil {
		targetInstance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &c.CreateDatabaseCtx.InstanceID})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find the instance with ID %d", c.CreateDatabaseCtx.InstanceID)
		}
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find the project with ID %d", projectID)
		}
		taskList, err := s.createDatabaseCreateTaskList(*c.CreateDatabaseCtx, targetInstance, project)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create the database create task list")
		}
		taskCreateList = append(taskCreateList, taskList...)

		payloadRestore.TargetInstanceID = &targetInstance.UID
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
		InstanceID: instance.UID,
		DatabaseID: &originDatabase.UID,
		Payload:    string(bytesRestore),
		BackupID:   c.BackupID,
	}

	if payloadRestore.TargetInstanceID != nil {
		// PITR to new database: task 2
		restoreTaskCreate.Name = fmt.Sprintf("Restore to new database %q", *payloadRestore.DatabaseName)
		restoreTaskCreate.DatabaseName = c.CreateDatabaseCtx.DatabaseName
	} else {
		// PITR in place: task 1
		restoreTaskCreate.Name = fmt.Sprintf("Restore to PITR database %q", originDatabase.DatabaseName)
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
			Name:       fmt.Sprintf("Swap PITR and the original database %q", originDatabase.DatabaseName),
			InstanceID: instance.UID,
			DatabaseID: &originDatabase.UID,
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
		return fmt.Sprintf("%s\nALTER DATABASE \"%s\" OWNER TO \"%s\";", stmt, databaseName, createDatabaseContext.Owner), nil
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
	case db.Spanner:
		return fmt.Sprintf("CREATE DATABASE %s", databaseName), nil
	}
	return "", errors.Errorf("unsupported database type %s", dbType)
}

// creates gh-ost TaskCreate list and dependency.
func createGhostTaskList(database *store.DatabaseMessage, instance *store.InstanceMessage, vcsPushEvent *vcs.PushEvent, detail *api.MigrationDetail, schemaVersion string) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
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
		Name:              fmt.Sprintf("Update schema gh-ost sync for database %q", database.DatabaseName),
		InstanceID:        instance.UID,
		DatabaseID:        &database.UID,
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
		Name:              fmt.Sprintf("Update schema gh-ost cutover for database %q", database.DatabaseName),
		InstanceID:        instance.UID,
		DatabaseID:        &database.UID,
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
	case db.Spanner:
		// Spanner does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("Spanner does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Spanner does not support collation, but got %s", collation)
		}
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
func convertDatabaseLabels(labelsJSON string) ([]*api.DatabaseLabel, error) {
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
	return labels, nil
}
