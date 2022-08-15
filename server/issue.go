package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
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
			issueFind.StatusList = &statusList
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
			}
			issueFind.Limit = &limit
		}
		userIDStr := c.QueryParams().Get("user")
		if userIDStr != "" {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("user query parameter is not a number: %s", userIDStr)).SetInternal(err)
			}
			issueFind.PrincipalID = &userID
		}

		issueList, err := s.store.FindIssue(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch issue list").SetInternal(err)
		}

		for _, issue := range issueList {
			s.setTaskProgressForIssue(issue)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueList); err != nil {
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
			environmentID := getActiveTaskEnvironmentID(issue.Pipeline)
			ok, err := s.canPrincipalBeAssignee(ctx, *issuePatch.AssigneeID, environmentID, issue.ProjectID, issue.Type)
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
				issue: issue,
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
	pipeline, err := s.createPipelineFromIssue(ctx, issueCreate, creatorID)
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

	if _, err := s.ScheduleNextTaskIfNeeded(ctx, issue.Pipeline); err != nil {
		return nil, fmt.Errorf("failed to schedule task after creating the issue: %v. Error %w", issue.Name, err)
	}

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Name,
	}

	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after creating the issue: %v. Error %w", issue.Name, err)
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
		return nil, fmt.Errorf("failed to create activity after creating the issue: %v. Error %w", issue.Name, err)
	}
	return issue, nil
}

func (s *Server) createPipelineFromIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Pipeline, error) {
	// Run pre-condition check first to make sure all tasks are valid, otherwise we will create partial pipelines
	// since we are not creating pipeline/stage list/task list in a single transaction.
	// We may still run into this issue when we actually create those pipeline/stage list/task list, however, that's
	// quite unlikely so we will live with it for now.
	if issueCreate.AssigneeID == api.UnknownID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
	}

	pipelineCreate, err := s.getPipelineCreate(ctx, issueCreate)
	if err != nil {
		return nil, err
	}
	var environmentID int
	// Return an error if the issue has no task to be executed
	hasTask := false
	for _, stage := range pipelineCreate.StageList {
		if len(stage.TaskList) > 0 {
			hasTask = true
			environmentID = stage.EnvironmentID
			break
		}
	}
	if !hasTask {
		err := fmt.Errorf("issue has no task to be executed")
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}

	// Create the pipeline, stages, and tasks.
	if issueCreate.ValidateOnly {
		return s.store.CreatePipelineValidateOnly(ctx, pipelineCreate, creatorID)
	}

	// check the assignee if it's NOT ValidateOnly
	ok, err := s.canPrincipalBeAssignee(ctx, issueCreate.AssigneeID, environmentID, issueCreate.ProjectID, issueCreate.Type)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the assignee can be set for the new issue").SetInternal(err)
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", issueCreate.AssigneeID)).SetInternal(err)
	}

	pipelineCreate.CreatorID = creatorID
	pipelineCreated, err := s.store.CreatePipeline(ctx, pipelineCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline for issue, error %v", err)
	}

	for _, stageCreate := range pipelineCreate.StageList {
		stageCreate.CreatorID = creatorID
		stageCreate.PipelineID = pipelineCreated.ID
		createdStage, err := s.store.CreateStage(ctx, &stageCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create stage for issue, error %v", err)
		}

		taskID := make(map[int]int)

		for index, taskCreate := range stageCreate.TaskList {
			taskCreate.CreatorID = creatorID
			taskCreate.PipelineID = pipelineCreated.ID
			taskCreate.StageID = createdStage.ID
			task, err := s.store.CreateTask(ctx, &taskCreate)
			if err != nil {
				return nil, fmt.Errorf("failed to create task for issue, error %v", err)
			}
			taskID[index] = task.ID
		}

		for _, indexDAG := range stageCreate.TaskIndexDAGList {
			taskDAGCreate := api.TaskDAGCreate{
				FromTaskID: taskID[indexDAG.FromIndex],
				ToTaskID:   taskID[indexDAG.ToIndex],
				Payload:    "{}",
			}
			if _, err := s.store.CreateTaskDAG(ctx, &taskDAGCreate); err != nil {
				return nil, fmt.Errorf("failed to create task DAG for issue, error %w", err)
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
		return nil, fmt.Errorf("instance ID not found %v", c.InstanceID)
	}
	// Find project.
	project, err := s.store.GetProjectByID(ctx, issueCreate.ProjectID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}
	if project == nil {
		err := fmt.Errorf("project ID not found %v", issueCreate.ProjectID)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
	}

	if err := checkCharacterSetCollationOwner(instance.Engine, c.CharacterSet, c.Collation, c.Owner); err != nil {
		return nil, err
	}

	if instance.Engine == db.Snowflake {
		// Snowflake needs to use upper case of DatabaseName.
		c.DatabaseName = strings.ToUpper(c.DatabaseName)
	}

	taskCreateList, err := s.createDatabaseCreateTaskList(ctx, c, *instance, *project)
	if err != nil {
		return nil, fmt.Errorf("failed to create task list of creating database, error: %w", err)
	}

	if c.BackupID != 0 {
		backup, err := s.store.GetBackupByID(ctx, c.BackupID)
		if err != nil {
			return nil, fmt.Errorf("failed to find backup %v", c.BackupID)
		}
		if backup == nil {
			return nil, fmt.Errorf("backup not found with ID %d", c.BackupID)
		}
		restorePayload := api.TaskDatabaseRestorePayload{}
		restorePayload.DatabaseName = c.DatabaseName
		restorePayload.BackupID = c.BackupID
		restoreBytes, err := json.Marshal(restorePayload)
		if err != nil {
			return nil, fmt.Errorf("failed to create restore database task, unable to marshal payload %w", err)
		}

		taskCreateList = append(taskCreateList, api.TaskCreate{
			InstanceID:   c.InstanceID,
			Name:         fmt.Sprintf("Restore backup %v", backup.Name),
			Status:       api.TaskPending,
			Type:         api.TaskDatabaseRestore,
			DatabaseName: c.DatabaseName,
			BackupID:     &c.BackupID,
			Payload:      string(restoreBytes),
		})

		return &api.PipelineCreate{
			Name: fmt.Sprintf("Pipeline - Create database %v from backup %v", c.DatabaseName, backup.Name),
			StageList: []api.StageCreate{
				{
					Name:          "Restore backup",
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
				Name:          "Create database",
				EnvironmentID: instance.EnvironmentID,
				TaskList:      taskCreateList,
			},
		},
	}, nil
}

func (s *Server) getPipelineCreateForDatabasePITR(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	if !s.feature(api.FeaturePITR) {
		return nil, echo.NewHTTPError(http.StatusForbidden, api.FeaturePITR.AccessErrorMessage())
	}
	c := api.PITRContext{}
	if err := json.Unmarshal([]byte(issueCreate.CreateContext), &c); err != nil {
		return nil, err
	}

	database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &c.DatabaseID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", c.DatabaseID)).SetInternal(err)
	}
	if database == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", c.DatabaseID))
	}

	taskCreateList, taskIndexDAGList, err := createPITRTaskList(database, issueCreate.ProjectID, *c.PointInTimeTs)
	if err != nil {
		return nil, err
	}

	return &api.PipelineCreate{
		Name: "Database Point-in-time Recovery pipeline",
		StageList: []api.StageCreate{
			{
				Name:             "PITR",
				EnvironmentID:    database.Instance.Environment.ID,
				TaskList:         taskCreateList,
				TaskIndexDAGList: taskIndexDAGList,
			},
		},
	}, nil
}

func (s *Server) getPipelineCreateForDatabaseSchemaAndDataUpdate(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	c := api.UpdateSchemaContext{}
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
	create := &api.PipelineCreate{}
	switch c.MigrationType {
	case db.Baseline:
		create.Name = "Establish database baseline pipeline"
	case db.Migrate:
		create.Name = "Update database schema pipeline"
	case db.Data:
		create.Name = "Update database data pipeline"
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid migration type %q", c.MigrationType))
	}
	project, err := s.store.GetProjectByID(ctx, issueCreate.ProjectID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID %d", issueCreate.ProjectID)).SetInternal(err)
	}

	// If migration type is establishing baseline and project's workflow is VCS,
	// then the context must contain a VCS push event field. We need this VCS
	// context to identify the write-back destination after migration.
	if c.MigrationType == db.Baseline && project.WorkflowType == api.VCSWorkflow {
		if c.VCSPushEvent == nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create establishing baseline issue for GitOps workflow project, vcs context missing")
		}
	}

	schemaVersion := common.DefaultMigrationVersion()
	// Tenant mode project pipeline has its own generation.
	if project.TenantMode == api.TenantModeTenant {
		if !s.feature(api.FeatureMultiTenancy) {
			return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
		if c.MigrationType != db.Migrate && c.MigrationType != db.Data {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Only Migrate and Data type migration can be performed on tenant mode project")
		}
		if len(c.DetailList) != 1 {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Tenant mode project should have exactly one update schema detail")
		}
		d := c.DetailList[0]
		if d.Statement == "" {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
		}

		if d.DatabaseName == "" && d.DatabaseID > 0 {
			database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &d.DatabaseID})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", d.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", d.DatabaseID))
			}

			taskCreate, err := getUpdateTask(database, c.MigrationType, c.VCSPushEvent, d, schemaVersion)
			if err != nil {
				return nil, err
			}

			create.StageList = append(create.StageList, api.StageCreate{
				Name:          fmt.Sprintf("%s %s", database.Instance.Environment.Name, database.Name),
				EnvironmentID: database.Instance.Environment.ID,
				TaskList:      []api.TaskCreate{*taskCreate},
			})
		} else {
			dbList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
				ProjectID: &issueCreate.ProjectID,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", issueCreate.ProjectID)).SetInternal(err)
			}

			baseDBName := d.DatabaseName
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

					taskCreate, err := getUpdateTask(database, c.MigrationType, c.VCSPushEvent, d, schemaVersion)
					if err != nil {
						return nil, err
					}
					taskCreateList = append(taskCreateList, *taskCreate)
				}
				if len(environmentSet) != 1 {
					var environments []string
					for k := range environmentSet {
						environments = append(environments, k)
					}
					err := fmt.Errorf("all databases in a stage should have the same environment; got %s", strings.Join(environments, ","))
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
		envToDatabaseMap := make(map[envKey][]api.TaskCreate)
		for _, d := range c.DetailList {
			if c.MigrationType == db.Migrate && d.Statement == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
			}
			database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &d.DatabaseID})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", d.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", d.DatabaseID))
			}

			taskCreate, err := getUpdateTask(database, c.MigrationType, c.VCSPushEvent, d, schemaVersion)
			if err != nil {
				return nil, err
			}

			key := envKey{name: database.Instance.Environment.Name, id: database.Instance.Environment.ID, order: database.Instance.Environment.Order}
			envToDatabaseMap[key] = append(envToDatabaseMap[key], *taskCreate)
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
			create.StageList = append(create.StageList, api.StageCreate{
				Name:          env.name,
				EnvironmentID: env.id,
				TaskList:      envToDatabaseMap[env],
			})
		}
	}
	return create, nil
}

func (s *Server) getPipelineCreateForDatabaseSchemaUpdateGhost(ctx context.Context, issueCreate *api.IssueCreate) (*api.PipelineCreate, error) {
	if !s.feature(api.FeatureGhost) {
		return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureGhost.AccessErrorMessage())
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

func getUpdateTask(database *api.Database, migrationType db.MigrationType, vcsPushEvent *vcs.PushEvent, d *api.UpdateSchemaDetail, schemaVersion string) (*api.TaskCreate, error) {
	taskName := fmt.Sprintf("Establish %q baseline", database.Name)
	switch migrationType {
	case db.Migrate:
		taskName = fmt.Sprintf("Update %q schema", database.Name)
	case db.Data:
		taskName = fmt.Sprintf("Update %q data", database.Name)
	}
	payload := api.TaskDatabaseSchemaUpdatePayload{}
	payload.MigrationType = migrationType
	payload.Statement = d.Statement
	payload.SchemaVersion = schemaVersion
	if vcsPushEvent != nil {
		payload.VCSPushEvent = vcsPushEvent
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal database schema update payload: %v", err)
		if migrationType == db.Data {
			errMsg = fmt.Sprintf("Failed to marshal database data update payload: %v", err)
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, errMsg)
	}

	taskType := api.TaskDatabaseSchemaUpdate
	if migrationType == db.Data {
		taskType = api.TaskDatabaseDataUpdate
	}
	return &api.TaskCreate{
		Name:              taskName,
		InstanceID:        database.Instance.ID,
		DatabaseID:        &database.ID,
		Status:            api.TaskPendingApproval,
		Type:              taskType,
		Statement:         d.Statement,
		EarliestAllowedTs: d.EarliestAllowedTs,
		MigrationType:     migrationType,
		Payload:           string(bytes),
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
			return nil, fmt.Errorf("api.GetBaseDatabaseName(%q, %q, %q) failed, error: %v", c.DatabaseName, project.DBNameTemplate, c.Labels, err)
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

	payload := api.TaskDatabaseCreatePayload{
		ProjectID:     project.ID,
		CharacterSet:  c.CharacterSet,
		Collation:     c.Collation,
		Labels:        c.Labels,
		SchemaVersion: schemaVersion,
	}
	payload.DatabaseName, payload.Statement = getDatabaseNameAndStatement(instance.Engine, c, schema)
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create database creation task, unable to marshal payload %w", err)
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

// creates PITR TaskCreate list and dependency.
func createPITRTaskList(database *api.Database, projectID int, targetTs int64) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	var taskCreateList []api.TaskCreate

	// task: create and restore to PITR database
	payloadRestore := api.TaskDatabasePITRRestorePayload{
		ProjectID:     projectID,
		PointInTimeTs: &targetTs,
	}
	bytesRestore, err := json.Marshal(payloadRestore)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create PITR restore task, unable to marshal payload, error: %w", err)
	}

	taskCreateList = append(taskCreateList, api.TaskCreate{
		Name:       fmt.Sprintf("Restore PITR database %s", database.Name),
		InstanceID: database.InstanceID,
		DatabaseID: &database.ID,
		Status:     api.TaskPendingApproval,
		Type:       api.TaskDatabaseRestorePITRRestore,
		Payload:    string(bytesRestore),
	})

	// task: swap PITR and the original database
	payloadCutover := api.TaskDatabasePITRCutoverPayload{}
	bytesCutover, err := json.Marshal(payloadCutover)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create PITR cutover task, unable to marshal payload, error: %w", err)
	}

	taskCreateList = append(taskCreateList, api.TaskCreate{
		Name:       fmt.Sprintf("Swap PITR and the original database %s", database.Name),
		InstanceID: database.InstanceID,
		DatabaseID: &database.ID,
		Status:     api.TaskPendingApproval,
		Type:       api.TaskDatabaseRestorePITRCutover,
		Payload:    string(bytesCutover),
	})

	taskIndexDAGList := []api.TaskIndexDAG{
		{FromIndex: 0, ToIndex: 1},
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
		Name:              fmt.Sprintf("Update %q schema gh-ost sync", database.Name),
		InstanceID:        database.InstanceID,
		DatabaseID:        &database.ID,
		Status:            api.TaskPendingApproval,
		Type:              api.TaskDatabaseSchemaUpdateGhostSync,
		Statement:         detail.Statement,
		EarliestAllowedTs: detail.EarliestAllowedTs,
		MigrationType:     db.Migrate,
		Payload:           string(bytesSync),
	})

	// task "cutover"
	payloadCutover := api.TaskDatabaseSchemaUpdateGhostCutoverPayload{}
	bytesCutover, err := json.Marshal(payloadCutover)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to marshal database schema update ghost cutover payload, error: %v", err))
	}
	taskCreateList = append(taskCreateList, api.TaskCreate{
		Name:              fmt.Sprintf("Update %q schema gh-ost cutover", database.Name),
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
			return fmt.Errorf("ClickHouse does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return fmt.Errorf("ClickHouse does not support collation, but got %s", collation)
		}
	case db.Snowflake:
		if characterSet != "" {
			return fmt.Errorf("Snowflake does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return fmt.Errorf("Snowflake does not support collation, but got %s", collation)
		}
	case db.Postgres:
		if owner == "" {
			return fmt.Errorf("database owner is required for PostgreSQL")
		}
	case db.SQLite:
		// no-op.
	default:
		if characterSet == "" {
			return fmt.Errorf("character set missing for %s", string(dbType))
		}
		// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
		// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
		// install it.
		if collation == "" {
			return fmt.Errorf("collation missing for %s", string(dbType))
		}
	}
	return nil
}

func getDatabaseNameAndStatement(dbType db.Type, createDatabaseContext api.CreateDatabaseContext, schema string) (string, string) {
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
		if createDatabaseContext.Collation == "" {
			stmt = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q;", databaseName, createDatabaseContext.CharacterSet)
		} else {
			stmt = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q;", databaseName, createDatabaseContext.CharacterSet, createDatabaseContext.Collation)
		}
		// Set the database owner.
		// We didn't use CREATE DATABASE WITH OWNER because RDS requires the current role to be a member of the database owner.
		// However, people can still use ALTER DATABASE to change the owner afterwards.
		// Error string below:
		// query: CREATE DATABASE h1 WITH OWNER hello;
		// ERROR:  must be member of role "hello"
		//
		// For tenant project, the schema for the newly created database will belong to the same owner.
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
					return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("failed to resolve issue: %v, task %v has not finished", issue.Name, task.Name)}
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
						ID:        task.ID,
						UpdaterID: updaterID,
						Status:    api.TaskCanceled,
					}); err != nil {
						return nil, fmt.Errorf("failed to cancel issue: %v, failed to cancel task: %v, error: %w", issue.Name, task.Name, err)
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
		return nil, fmt.Errorf("failed to update issue %q's status, failed to update pipeline status with patch %+v, error: %w", issue.Name, pipelinePatch, err)
	}

	issuePatch := &api.IssuePatch{
		ID:        issue.ID,
		UpdaterID: updaterID,
		Status:    &newStatus,
	}
	updatedIssue, err := s.store.PatchIssue(ctx, issuePatch)
	if err != nil {
		return nil, fmt.Errorf("failed to update issue %q's status with patch %v, error: %w", issue.Name, issuePatch, err)
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
		IssueName: updatedIssue.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity after changing the issue status: %v, error: %w", issue.Name, err)
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
		return nil, fmt.Errorf("failed to create activity after changing the issue status: %v, error: %w", issue.Name, err)
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
			return fmt.Errorf("failed to post activity to creator inbox: %d, error: %w", issue.CreatorID, err)
		}
	}

	if issue.AssigneeID != api.SystemBotID && issue.AssigneeID != issue.CreatorID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.AssigneeID,
			ActivityID: activityID,
		}
		_, err := s.store.CreateInbox(ctx, inboxCreate)
		if err != nil {
			return fmt.Errorf("failed to post activity to assignee inbox: %d, error: %w", issue.AssigneeID, err)
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
				return fmt.Errorf("failed to post activity to subscriber inbox: %d, error: %w", subscriber.ID, err)
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
		found := false
		for _, db := range dbList {
			var labelList []*api.DatabaseLabel
			if err := json.Unmarshal([]byte(db.Labels), &labelList); err != nil {
				return "", "", fmt.Errorf("failed to unmarshal labels for database ID %v name %q, error: %v", db.ID, db.Name, err)
			}
			labelMap := map[string]string{}
			for _, label := range labelList {
				labelMap[label.Key] = label.Value
			}
			dbName, err := formatDatabaseName(baseDatabaseName, project.DBNameTemplate, labelMap)
			if err != nil {
				return "", "", fmt.Errorf("failed to format database name formatDatabaseName(%q, %q, %+v), error: %v", baseDatabaseName, project.DBNameTemplate, labelMap, err)
			}
			if db.Name == dbName {
				found = true
				break
			}
		}
		if found {
			err := fmt.Errorf("conflicting database name, project has existing base database named %q, but it's not from the selected peer tenants", baseDatabaseName)
			return "", "", echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		return "", "", nil
	}

	driver, err := s.getAdminDatabaseDriver(ctx, similarDB.Instance, similarDB.Name)
	if err != nil {
		return "", "", err
	}
	defer driver.Close(ctx)
	schemaVersion, err := getLatestSchemaVersion(ctx, driver, similarDB.Name)
	if err != nil {
		return "", "", fmt.Errorf("failed to get migration history for database %q: %w", similarDB.Name, err)
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

func getActiveTaskEnvironmentID(pipeline *api.Pipeline) int {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval ||
				task.Status == api.TaskPending ||
				task.Status == api.TaskRunning ||
				task.Status == api.TaskCanceled {
				return stage.EnvironmentID
			}
		}
	}
	// use the last stage if all done
	return pipeline.StageList[len(pipeline.StageList)-1].EnvironmentID
}
