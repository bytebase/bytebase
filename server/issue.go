package server

import (
	"bytes"
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

		issue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
		}

		if issuePatch.AssigneeID != nil {
			stage := getActiveStage(issue.Pipeline.StageList)
			if stage == nil {
				// all stages have finished, use the last stage
				stage = issue.Pipeline.StageList[len(issue.Pipeline.StageList)-1]
			}
			ok, err := s.canPrincipalBeAssignee(ctx, *issuePatch.AssigneeID, stage.EnvironmentID, issue.ProjectID, issue.Type)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the assignee can be changed").SetInternal(err)
			}
			if !ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", *issuePatch.AssigneeID)).SetInternal(err)
			}
		}

		updatedIssue, err := s.store.PatchIssue(ctx, issuePatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue with ID %d", id)).SetInternal(err)
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
			_, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
				issue: updatedIssue,
			})
			if err != nil {
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

		updatedIssue, err := s.changeIssueStatus(ctx, issue, issueStatusPatch.Status, issueStatusPatch.UpdaterID, issueStatusPatch.Comment)
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

func (s *Server) createIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Issue, error) {
	// Run pre-condition check first to make sure all tasks are valid, otherwise we will create partial pipelines
	// since we are not creating pipeline/stage list/task list in a single transaction.
	// We may still run into this issue when we actually create those pipeline/stage list/task list, however, that's
	// quite unlikely so we will live with it for now.
	pipelineCreate, err := s.getPipelineCreate(ctx, issueCreate)
	if err != nil {
		return nil, err
	}

	if issueCreate.AssigneeID == api.UnknownID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
	}
	// Try to find a more appropriate assignee if the current assignee is the system bot, indicating that the caller might not be sure about who should be the assignee.
	if issueCreate.AssigneeID == api.SystemBotID {
		assigneeID, err := s.getDefaultAssigneeID(ctx, pipelineCreate.StageList[0].EnvironmentID, issueCreate.ProjectID, issueCreate.Type)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find a default assignee").SetInternal(err)
		}
		issueCreate.AssigneeID = assigneeID
	}
	ok, err := s.canPrincipalBeAssignee(ctx, issueCreate.AssigneeID, pipelineCreate.StageList[0].EnvironmentID, issueCreate.ProjectID, issueCreate.Type)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the assignee can be set for the new issue").SetInternal(err)
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", issueCreate.AssigneeID))
	}

	pipeline, err := s.createPipeline(ctx, issueCreate, pipelineCreate, creatorID)
	if err != nil {
		return nil, err
	}

	if issueCreate.ValidateOnly {
		issue, err := s.store.CreateIssueValidateOnly(ctx, pipeline, issueCreate, creatorID)
		if err != nil {
			return nil, err
		}
		return issue, nil
	}

	issueCreate.CreatorID = creatorID
	issueCreate.PipelineID = pipeline.ID
	issue, err := s.store.CreateIssue(ctx, issueCreate)
	if err != nil {
		return nil, err
	}
	// Create issue subscribers.
	for _, subscriberID := range issueCreate.SubscriberIDList {
		subscriberCreate := &api.IssueSubscriberCreate{
			IssueID:      issue.ID,
			SubscriberID: subscriberID,
		}
		_, err := s.store.CreateIssueSubscriber(ctx, subscriberCreate)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d after creating issue %d", subscriberID, issue.ID)).SetInternal(err)
		}
	}

	if err := s.schedulePipelineTaskCheck(ctx, issue.Pipeline); err != nil {
		return nil, errors.Wrapf(err, "failed to schedule task check after creating the issue: %v", issue.Name)
	}

	if err := s.ScheduleActiveStage(ctx, issue.Pipeline); err != nil {
		return nil, errors.Wrapf(err, "failed to schedule task after creating the issue: %v", issue.Name)
	}

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Name,
	}

	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create activity after creating the issue: %v", issue.Name)
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: issue.ID,
		Type:        api.ActivityIssueCreate,
		Level:       api.ActivityInfo,
		Payload:     string(bytes),
	}
	_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
		issue: issue,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create activity after creating the issue: %v", issue.Name)
	}
	return issue, nil
}

func (s *Server) createPipeline(ctx context.Context, issueCreate *api.IssueCreate, pipelineCreate *api.PipelineCreate, creatorID int) (*api.Pipeline, error) {
	// Return an error if the issue has no task to be executed
	hasTask := false
	for _, stage := range pipelineCreate.StageList {
		if len(stage.TaskList) > 0 {
			hasTask = true
			break
		}
	}
	if !hasTask {
		err := errors.Errorf("issue has no task to be executed")
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}

	// Create the pipeline, stages, and tasks.
	if issueCreate.ValidateOnly {
		return s.store.CreatePipelineValidateOnly(ctx, pipelineCreate, creatorID)
	}

	pipelineCreate.CreatorID = creatorID
	pipelineCreated, err := s.store.CreatePipeline(ctx, pipelineCreate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pipeline for issue")
	}

	// TODO(p0ny): create stages in batch.
	for _, stageCreate := range pipelineCreate.StageList {
		stageCreate.CreatorID = creatorID
		stageCreate.PipelineID = pipelineCreated.ID
		createdStage, err := s.store.CreateStage(ctx, &stageCreate)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create stage for issue")
		}

		var taskCreateList []*api.TaskCreate
		for _, taskCreate := range stageCreate.TaskList {
			c := taskCreate
			c.CreatorID = creatorID
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
	case api.IssueDatabaseSchemaUpdate, api.IssueDatabaseDataUpdate:
		return s.getPipelineCreateForDatabaseSchemaAndDataUpdate(ctx, issueCreate)
	case api.IssueDatabaseSchemaUpdateGhost:
		return s.getPipelineCreateForDatabaseSchemaUpdateGhost(ctx, issueCreate)
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

	// Find instance.
	instance, err := s.store.GetInstanceByID(ctx, c.InstanceID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance ID not found %v", c.InstanceID)
	}
	// Find project.
	project, err := s.store.GetProjectByID(ctx, issueCreate.ProjectID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}
	if project == nil {
		err := errors.Errorf("project ID not found %v", issueCreate.ProjectID)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
	}

	taskCreateList, err := s.createDatabaseCreateTaskList(ctx, c, *instance, *project)
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
			Name: fmt.Sprintf("Pipeline - Create database %v from backup %v", c.DatabaseName, backup.Name),
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
		Name: fmt.Sprintf("Pipeline - Create database %s", c.DatabaseName),
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
	if c.PointInTimeTs != nil && !s.feature(api.FeaturePITR) {
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
		Name: "Database Point-in-time Recovery pipeline",
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
	if !s.feature(api.FeatureTaskScheduleTime) {
		for _, detail := range c.DetailList {
			if detail.EarliestAllowedTs != 0 {
				return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
			}
		}
	}
	create := &api.PipelineCreate{
		Name: "Change database pipeline",
	}

	project, err := s.store.GetProjectByID(ctx, issueCreate.ProjectID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}

	// Tenant mode project pipeline has its own generation.
	if project.TenantMode == api.TenantModeTenant {
		if !s.feature(api.FeatureMultiTenancy) {
			return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
		if len(c.DetailList) == 0 {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Tenant mode project should have at least one update schema detail")
		}
		databaseNameCount, databaseIDCount := 0, 0
		for _, detail := range c.DetailList {
			if detail.MigrationType != db.Migrate && detail.MigrationType != db.Data {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Only Migrate and Data type migration can be performed on tenant mode project")
			}
			if detail.Statement == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
			}
			if detail.DatabaseID > 0 {
				databaseIDCount++
			}
			if detail.DatabaseName != "" {
				databaseNameCount++
			}
		}
		if databaseNameCount > 0 && databaseIDCount > 0 {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Migration detail should set either database name or database ID.")
		}
		if databaseNameCount > 1 {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "There should be at most one migration detail with database name.")
		}

		if databaseIDCount > 0 {
			// Use database IDs in the issue.
			for _, detail := range c.DetailList {
				database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &detail.DatabaseID})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", detail.DatabaseID)).SetInternal(err)
				}
				if database == nil {
					return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", detail.DatabaseID))
				}

				vt, err := getUpdateTask(database, c.VCSPushEvent, detail, getOrDefaultSchemaVersion(detail))
				if err != nil {
					return nil, err
				}

				sameEnvStageFound := false
				for index, stage := range create.StageList {
					if stage.EnvironmentID == database.Instance.Environment.ID {
						stage.TaskList = append(stage.TaskList, *vt.task)
						create.StageList[index] = stage
						sameEnvStageFound = true
						break
					}
				}

				if !sameEnvStageFound {
					create.StageList = append(create.StageList, api.StageCreate{
						Name:          fmt.Sprintf("%s Stage", database.Instance.Environment.Name),
						EnvironmentID: database.Instance.Environment.ID,
						TaskList:      []api.TaskCreate{*vt.task},
					})
				}
			}
		} else {
			dbList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
				ProjectID: &issueCreate.ProjectID,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", issueCreate.ProjectID)).SetInternal(err)
			}

			migrationDetail := c.DetailList[0]
			baseDBName := migrationDetail.DatabaseName
			deployments, matrix, err := s.getTenantDatabaseMatrix(ctx, issueCreate.ProjectID, project.DBNameTemplate, dbList, baseDBName)
			if err != nil {
				return nil, err
			}
			// Convert to pipelineCreate
			for i, databaseList := range matrix {
				// Since environment is required for stage, we use an internal bb system environment for tenant deployments.
				environmentSet := make(map[string]bool)
				var environmentID int
				var taskCreateList []api.TaskCreate
				for _, database := range databaseList {
					environmentSet[database.Instance.Environment.Name] = true
					environmentID = database.Instance.EnvironmentID
					vt, err := getUpdateTask(database, c.VCSPushEvent, migrationDetail, getOrDefaultSchemaVersion(migrationDetail))
					if err != nil {
						return nil, err
					}
					taskCreateList = append(taskCreateList, *vt.task)
				}
				if len(environmentSet) != 1 {
					var environments []string
					for k := range environmentSet {
						environments = append(environments, k)
					}
					err := errors.Errorf("all databases in a stage should have the same environment; got %s", strings.Join(environments, ","))
					return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
				}

				create.StageList = append(create.StageList, api.StageCreate{
					Name:          deployments[i].Name,
					EnvironmentID: environmentID,
					TaskList:      taskCreateList,
				})
			}
		}
	} else {
		maximumTaskLimit := s.getPlanLimitValue(api.PlanLimitMaximumTask)
		if int64(len(c.DetailList)) > maximumTaskLimit {
			return nil, echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Effective plan %s can update up to %d databases, got %d.", s.getEffectivePlan(), maximumTaskLimit, len(c.DetailList)))
		}

		type envKey struct {
			name  string
			id    int
			order int
		}
		envToDatabaseMap := make(map[envKey][]*versionTask)
		for _, d := range c.DetailList {
			if d.MigrationType == db.Migrate && d.Statement == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
			}
			database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &d.DatabaseID})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", d.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", d.DatabaseID))
			}

			vt, err := getUpdateTask(database, c.VCSPushEvent, d, getOrDefaultSchemaVersion(d))
			if err != nil {
				return nil, err
			}

			key := envKey{name: database.Instance.Environment.Name, id: database.Instance.Environment.ID, order: database.Instance.Environment.Order}
			envToDatabaseMap[key] = append(envToDatabaseMap[key], vt)
		}
		// Sort and group by environments.
		var envKeys []envKey
		for k := range envToDatabaseMap {
			envKeys = append(envKeys, k)
		}
		sort.Slice(envKeys, func(i, j int) bool {
			return envKeys[i].order < envKeys[j].order
		})
		for _, env := range envKeys {
			taskList, taskIndexDagList := getTaskAndDagListByVersion(envToDatabaseMap[env])
			create.StageList = append(create.StageList, api.StageCreate{
				Name:             env.name,
				EnvironmentID:    env.id,
				TaskList:         taskList,
				TaskIndexDAGList: taskIndexDagList,
			})
		}
	}
	return create, nil
}

// getTaskAndDagListByVersion adds the task dependencies for tasks belonging to the same database.
// The dependency is by schema version ascending order.
func getTaskAndDagListByVersion(versionTaskList []*versionTask) ([]api.TaskCreate, []api.TaskIndexDAG) {
	var taskCreateList []api.TaskCreate
	var taskIndexDAGList []api.TaskIndexDAG
	databaseMap := make(map[int][]*versionTask)
	for _, vt := range versionTaskList {
		databaseMap[*vt.task.DatabaseID] = append(databaseMap[*vt.task.DatabaseID], vt)
	}
	var databaseList []int
	for k := range databaseMap {
		databaseList = append(databaseList, k)
	}
	sort.Ints(databaseList)

	for _, databaseID := range databaseList {
		list := databaseMap[databaseID]
		sort.Slice(list, func(i, j int) bool {
			return list[i].version < list[j].version
		})
		for i := 0; i < len(list)-1; i++ {
			taskIndexDAGList = append(taskIndexDAGList, api.TaskIndexDAG{FromIndex: len(taskCreateList) + i, ToIndex: len(taskCreateList) + i + 1})
		}
		for _, vt := range list {
			taskCreateList = append(taskCreateList, *vt.task)
		}
	}
	return taskCreateList, taskIndexDAGList
}

func (s *Server) getPipelineCreateForDatabaseSchemaUpdateGhost(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	if !s.feature(api.FeatureOnlineMigration) {
		return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureOnlineMigration.AccessErrorMessage())
	}
	c := api.UpdateSchemaGhostContext{}
	if err := json.Unmarshal([]byte(issueCreate.CreateContext), &c); err != nil {
		return nil, err
	}
	if !s.feature(api.FeatureTaskScheduleTime) {
		for _, detail := range c.DetailList {
			if detail.EarliestAllowedTs != 0 {
				return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
			}
		}
	}

	project, err := s.store.GetProjectByID(ctx, issueCreate.ProjectID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}
	if project.TenantMode == api.TenantModeTenant {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "not implemented yet")
	}

	create := &api.PipelineCreate{}
	create.Name = "Update database schema (gh-ost) pipeline"
	schemaVersion := common.DefaultMigrationVersion()
	for _, detail := range c.DetailList {
		if detail.Statement == "" {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "failed to create issue, sql statement missing")
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &detail.DatabaseID})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch database ID: %v", detail.DatabaseID)).SetInternal(err)
		}
		if database == nil {
			return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("database ID not found: %d", detail.DatabaseID))
		}

		taskCreateList, taskIndexDAGList, err := createGhostTaskList(database, c.VCSPushEvent, detail, schemaVersion)
		if err != nil {
			return nil, err
		}

		create.StageList = append(create.StageList, api.StageCreate{
			Name:             fmt.Sprintf("%s %s", database.Instance.Environment.Name, database.Name),
			EnvironmentID:    database.Instance.Environment.ID,
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

type versionTask struct {
	task    *api.TaskCreate
	version string
}

func getUpdateTask(database *api.Database, vcsPushEvent *vcs.PushEvent, d *api.MigrationDetail, schemaVersion string) (*versionTask, error) {
	var taskName string
	var taskType api.TaskType

	var payloadString string
	switch d.MigrationType {
	case db.Baseline:
		taskName = fmt.Sprintf("Establish baseline for database %q", database.Name)
		taskType = api.TaskDatabaseSchemaBaseline
		payload := api.TaskDatabaseSchemaBaselinePayload{
			Statement:     d.Statement,
			SchemaVersion: schemaVersion,
			VCSPushEvent:  vcsPushEvent,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database schema baseline payload").SetInternal(err)
		}
		payloadString = string(bytes)
	case db.Migrate:
		taskName = fmt.Sprintf("DDL(schema) for database %q", database.Name)
		taskType = api.TaskDatabaseSchemaUpdate
		payload := api.TaskDatabaseSchemaUpdatePayload{
			Statement:     d.Statement,
			SchemaVersion: schemaVersion,
			VCSPushEvent:  vcsPushEvent,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database schema update payload").SetInternal(err)
		}
		payloadString = string(bytes)
	case db.MigrateSDL:
		taskName = fmt.Sprintf("SDL for database %q", database.Name)
		taskType = api.TaskDatabaseSchemaUpdateSDL
		payload := api.TaskDatabaseSchemaUpdateSDLPayload{
			Statement:     d.Statement,
			SchemaVersion: schemaVersion,
			VCSPushEvent:  vcsPushEvent,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database schema update SDL payload").SetInternal(err)
		}
		payloadString = string(bytes)
	case db.Data:
		taskName = fmt.Sprintf("DML(data) for database %q", database.Name)
		taskType = api.TaskDatabaseDataUpdate
		payload := api.TaskDatabaseDataUpdatePayload{
			Statement:     d.Statement,
			SchemaVersion: schemaVersion,
			VCSPushEvent:  vcsPushEvent,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database data update payload").SetInternal(err)
		}
		payloadString = string(bytes)
	default:
		return nil, errors.Errorf("unsupported migration type %q", d.MigrationType)
	}

	return &versionTask{
		task: &api.TaskCreate{
			Name:              taskName,
			InstanceID:        database.Instance.ID,
			DatabaseID:        &database.ID,
			Status:            api.TaskPendingApproval,
			Type:              taskType,
			Statement:         d.Statement,
			EarliestAllowedTs: d.EarliestAllowedTs,
			Payload:           payloadString,
		},
		version: schemaVersion,
	}, nil
}

// createDatabaseCreateTaskList returns the task list for create database.
func (s *Server) createDatabaseCreateTaskList(ctx context.Context, c api.CreateDatabaseContext, instance api.Instance, project api.Project) ([]api.TaskCreate, error) {
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
	// Validate the labels. Labels are set upon task completion.
	if c.Labels != "" {
		if err := s.setDatabaseLabels(ctx, c.Labels, &api.Database{Name: c.DatabaseName, Instance: &instance} /* dummy database */, &project, 0 /* dummy updaterID */, true /* validateOnly */); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid database label %q, error %v", c.Labels, err))
		}
	}

	var schemaVersion, schema string
	// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
	if project.TenantMode == api.TenantModeTenant {
		if !s.feature(api.FeatureMultiTenancy) {
			return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
		baseDatabaseName, err := api.GetBaseDatabaseName(c.DatabaseName, project.DBNameTemplate, c.Labels)
		if err != nil {
			return nil, errors.Wrapf(err, "api.GetBaseDatabaseName(%q, %q, %q) failed", c.DatabaseName, project.DBNameTemplate, c.Labels)
		}
		sv, s, err := s.getSchemaFromPeerTenantDatabase(ctx, &instance, &project, project.ID, baseDatabaseName)
		if err != nil {
			return nil, err
		}
		schemaVersion, schema = sv, s
	}
	if schemaVersion == "" {
		schemaVersion = common.DefaultMigrationVersion()
	}

	// Get admin data source username.
	adminDataSource := api.DataSourceFromInstanceWithType(&instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %d", instance.ID)
	}
	payload := api.TaskDatabaseCreatePayload{
		ProjectID:     project.ID,
		CharacterSet:  c.CharacterSet,
		Collation:     c.Collation,
		Labels:        c.Labels,
		SchemaVersion: schemaVersion,
	}
	payload.DatabaseName, payload.Statement = getDatabaseNameAndStatement(instance.Engine, c, adminDataSource.Username, schema)
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
		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find the project with ID %d", projectID)
		}
		taskList, err := s.createDatabaseCreateTaskList(ctx, *c.CreateDatabaseCtx, *targetInstance, *project)
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

// creates gh-ost TaskCreate list and dependency.
func createGhostTaskList(database *api.Database, vcsPushEvent *vcs.PushEvent, detail *api.UpdateSchemaGhostDetail, schemaVersion string) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
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
	case db.SQLite:
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

func getDatabaseNameAndStatement(dbType db.Type, createDatabaseContext api.CreateDatabaseContext, adminDatasourceUser, schema string) (string, string) {
	databaseName := createDatabaseContext.DatabaseName
	// Snowflake needs to use upper case of DatabaseName.
	if dbType == db.Snowflake {
		databaseName = strings.ToUpper(databaseName)
	}

	var stmt string
	switch dbType {
	case db.MySQL, db.TiDB:
		stmt = fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", databaseName, createDatabaseContext.CharacterSet, createDatabaseContext.Collation)
		if schema != "" {
			stmt = fmt.Sprintf("%s\nUSE `%s`;\n%s", stmt, databaseName, schema)
		}
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
		stmt = fmt.Sprintf("%s\nALTER DATABASE \"%s\" OWNER TO %s;\n", stmt, databaseName, createDatabaseContext.Owner)
		if schema != "" {
			stmt = fmt.Sprintf("%s\n\\connect \"%s\";\n%s", stmt, databaseName, schema)
		}
	case db.ClickHouse:
		clusterPart := ""
		if createDatabaseContext.Cluster != "" {
			clusterPart = fmt.Sprintf(" ON CLUSTER `%s`", createDatabaseContext.Cluster)
		}
		stmt = fmt.Sprintf("CREATE DATABASE `%s`%s;", databaseName, clusterPart)
		if schema != "" {
			stmt = fmt.Sprintf("%s\nUSE `%s`;\n%s", stmt, databaseName, schema)
		}
	case db.Snowflake:
		databaseName = strings.ToUpper(databaseName)
		stmt = fmt.Sprintf("CREATE DATABASE %s;", databaseName)
		if schema != "" {
			stmt = fmt.Sprintf("%s\nUSE DATABASE %s;\n%s", stmt, databaseName, schema)
		}
	case db.SQLite:
		// This is a fake CREATE DATABASE and USE statement since a single SQLite file represents a database. Engine driver will recognize it and establish a connection to create the sqlite file representing the database.
		stmt = fmt.Sprintf("CREATE DATABASE '%s';", databaseName)
		if schema != "" {
			stmt = fmt.Sprintf("%s\nUSE `%s`;\n%s", stmt, databaseName, schema)
		}
	}

	return databaseName, stmt
}

func (s *Server) changeIssueStatus(ctx context.Context, issue *api.Issue, newStatus api.IssueStatus, updaterID int, comment string) (*api.Issue, error) {
	var pipelineStatus api.PipelineStatus
	switch newStatus {
	case api.IssueOpen:
		pipelineStatus = api.PipelineOpen
	case api.IssueDone:
		// Returns error if any of the tasks is not DONE.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status != api.TaskDone {
					return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("failed to resolve issue: %v, task %v has not finished", issue.Name, task.Name)}
				}
			}
		}
		pipelineStatus = api.PipelineDone
	case api.IssueCanceled:
		// If we want to cancel the issue, we find the current running tasks, mark each of them CANCELED.
		// We keep PENDING and FAILED tasks as is since the issue maybe reopened later, and it's better to
		// keep those tasks in the same state before the issue was canceled.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status == api.TaskRunning {
					if _, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
						IDList:    []int{task.ID},
						UpdaterID: updaterID,
						Status:    api.TaskCanceled,
					}); err != nil {
						return nil, errors.Wrapf(err, "failed to cancel issue: %v, failed to cancel task: %v", issue.Name, task.Name)
					}
				}
			}
		}
		pipelineStatus = api.PipelineCanceled
	}

	pipelinePatch := &api.PipelinePatch{
		ID:        issue.PipelineID,
		UpdaterID: updaterID,
		Status:    &pipelineStatus,
	}
	if _, err := s.store.PatchPipeline(ctx, pipelinePatch); err != nil {
		return nil, errors.Wrapf(err, "failed to update issue %q's status, failed to update pipeline status with patch %+v", issue.Name, pipelinePatch)
	}

	issuePatch := &api.IssuePatch{
		ID:        issue.ID,
		UpdaterID: updaterID,
		Status:    &newStatus,
	}
	updatedIssue, err := s.store.PatchIssue(ctx, issuePatch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update issue %q's status with patch %v", issue.Name, issuePatch)
	}

	// Cancel external approval, it's ok if we failed.
	if newStatus != api.IssueOpen {
		if err := s.ApplicationRunner.CancelExternalApproval(ctx, issue); err != nil {
			log.Error("failed to cancel external approval on issue cancellation", zap.Error(err))
		}
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
		IssueName: updatedIssue.Name,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal activity after changing the issue status: %v", issue.Name)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   updaterID,
		ContainerID: issue.ID,
		Type:        api.ActivityIssueStatusUpdate,
		Level:       api.ActivityInfo,
		Comment:     comment,
		Payload:     string(payload),
	}

	_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
		issue: updatedIssue,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create activity after changing the issue status: %v", issue.Name)
	}

	return updatedIssue, nil
}

func (s *Server) postInboxIssueActivity(ctx context.Context, issue *api.Issue, activityID int) error {
	if issue.CreatorID != api.SystemBotID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.CreatorID,
			ActivityID: activityID,
		}
		_, err := s.store.CreateInbox(ctx, inboxCreate)
		if err != nil {
			return errors.Wrapf(err, "failed to post activity to creator inbox: %d", issue.CreatorID)
		}
	}

	if issue.AssigneeID != api.SystemBotID && issue.AssigneeID != issue.CreatorID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.AssigneeID,
			ActivityID: activityID,
		}
		_, err := s.store.CreateInbox(ctx, inboxCreate)
		if err != nil {
			return errors.Wrapf(err, "failed to post activity to assignee inbox: %d", issue.AssigneeID)
		}
	}

	for _, subscriber := range issue.SubscriberList {
		if subscriber.ID != api.SystemBotID && subscriber.ID != issue.CreatorID && subscriber.ID != issue.AssigneeID {
			inboxCreate := &api.InboxCreate{
				ReceiverID: subscriber.ID,
				ActivityID: activityID,
			}
			_, err := s.store.CreateInbox(ctx, inboxCreate)
			if err != nil {
				return errors.Wrapf(err, "failed to post activity to subscriber inbox: %d", subscriber.ID)
			}
		}
	}

	return nil
}

func (s *Server) getTenantDatabaseMatrix(ctx context.Context, projectID int, dbNameTemplate string, dbList []*api.Database, baseDatabaseName string) ([]*api.Deployment, [][]*api.Database, error) {
	deployConfig, err := s.store.GetDeploymentConfigByProjectID(ctx, projectID)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch deployment config for project ID: %v", projectID)).SetInternal(err)
	}
	if deployConfig == nil {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Deployment config missing for project ID: %v", projectID)).SetInternal(err)
	}
	deploySchedule, err := api.ValidateAndGetDeploymentSchedule(deployConfig.Payload)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get deployment schedule").SetInternal(err)
	}

	d, matrix, err := getDatabaseMatrixFromDeploymentSchedule(deploySchedule, baseDatabaseName, dbNameTemplate, dbList)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create deployment pipeline").SetInternal(err)
	}
	return d, matrix, nil
}

// getSchemaFromPeerTenantDatabase gets the schema version and schema from a peer tenant database.
// It's used for creating a database in a tenant mode project.
// When a peer tenant database doesn't exist, we will return an error if there are databases in the project with the same name.
// Otherwise, we will create a blank database without schema.
func (s *Server) getSchemaFromPeerTenantDatabase(ctx context.Context, instance *api.Instance, project *api.Project, projectID int, baseDatabaseName string) (string, string, error) {
	// Find all databases in the project.
	dbList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
		ProjectID: &projectID,
	})
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", projectID)).SetInternal(err)
	}

	_, matrix, err := s.getTenantDatabaseMatrix(ctx, projectID, project.DBNameTemplate, dbList, baseDatabaseName)
	if err != nil {
		return "", "", err
	}
	similarDB := getPeerTenantDatabase(matrix, instance.EnvironmentID)

	// When there is no existing tenant, we will look at all existing databases in the tenant mode project.
	// If there are existing databases with the same name, we will disallow the database creation.
	// Otherwise, we will create a blank new database.
	if similarDB == nil {
		// Ignore the database name conflict if the template is empty.
		if project.DBNameTemplate == "" {
			return "", "", nil
		}

		found := false
		for _, db := range dbList {
			var labelList []*api.DatabaseLabel
			if err := json.Unmarshal([]byte(db.Labels), &labelList); err != nil {
				return "", "", errors.Wrapf(err, "failed to unmarshal labels for database ID %v name %q", db.ID, db.Name)
			}
			labelMap := map[string]string{}
			for _, label := range labelList {
				labelMap[label.Key] = label.Value
			}
			dbName, err := formatDatabaseName(baseDatabaseName, project.DBNameTemplate, labelMap)
			if err != nil {
				return "", "", errors.Wrapf(err, "failed to format database name formatDatabaseName(%q, %q, %+v)", baseDatabaseName, project.DBNameTemplate, labelMap)
			}
			if db.Name == dbName {
				found = true
				break
			}
		}
		if found {
			err := errors.Errorf("conflicting database name, project has existing base database named %q, but it's not from the selected peer tenants", baseDatabaseName)
			return "", "", echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		return "", "", nil
	}

	driver, err := getAdminDatabaseDriver(ctx, similarDB.Instance, similarDB.Name, s.pgInstance.BaseDir, s.profile.DataDir)
	if err != nil {
		return "", "", err
	}
	defer driver.Close(ctx)
	schemaVersion, err := getLatestSchemaVersion(ctx, driver, similarDB.Name)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get migration history for database %q", similarDB.Name)
	}

	var schemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, similarDB.Name, &schemaBuf, true /* schemaOnly */); err != nil {
		return "", "", err
	}
	return schemaVersion, schemaBuf.String(), nil
}

func getPeerTenantDatabase(databaseMatrix [][]*api.Database, environmentID int) *api.Database {
	var similarDB *api.Database
	// We try to use an existing tenant with the same environment, if possible.
	for _, databaseList := range databaseMatrix {
		for _, db := range databaseList {
			if db.Instance.EnvironmentID == environmentID {
				similarDB = db
				break
			}
		}
		if similarDB != nil {
			break
		}
	}
	if similarDB == nil {
		for _, stage := range databaseMatrix {
			if len(stage) > 0 {
				similarDB = stage[0]
				break
			}
		}
	}

	return similarDB
}

func (s *Server) setTaskProgressForIssue(issue *api.Issue) {
	if s.TaskScheduler == nil {
		// readonly server doesn't have a TaskScheduler.
		return
	}
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if progress, ok := s.TaskScheduler.taskProgress.Load(task.ID); ok {
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
