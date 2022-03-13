package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		ctx := context.Background()
		issueCreate := &api.IssueCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create issue request").SetInternal(err)
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
		ctx := context.Background()
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
		issueList, err := s.IssueService.FindIssueList(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch issue list").SetInternal(err)
		}

		for _, issue := range issueList {
			if err := s.composeIssueRelationship(ctx, issue); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issue, err := s.composeIssueByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issuePatch := &api.IssuePatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issuePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue request").SetInternal(err)
		}

		if issuePatch.AssigneeID != nil {
			if err := s.validateAssigneeRoleByID(ctx, *issuePatch.AssigneeID); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", *issuePatch.AssigneeID)).SetInternal(err)
			}
		}

		issueFind := &api.IssueFind{
			ID: &id,
		}
		issue, err := s.IssueService.FindIssue(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
		}

		updatedIssue, err := s.IssueService.PatchIssue(ctx, issuePatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue ID: %v", id)).SetInternal(err)
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

		if err := s.composeIssueRelationship(ctx, updatedIssue); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update issue response: %v", updatedIssue.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID/status", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issueStatusPatch := &api.IssueStatusPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue status request").SetInternal(err)
		}

		issue, err := s.composeIssueByID(ctx, id)
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

		if err := s.composeIssueRelationship(ctx, updatedIssue); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeIssueByID(ctx context.Context, id int) (*api.Issue, error) {
	issueFind := &api.IssueFind{
		ID: &id,
	}
	issue, err := s.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		return nil, err
	}
	if id > 0 && issue == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("Issue not found with ID %v", id)}
	}

	if issue != nil {
		if err := s.composeIssueRelationship(ctx, issue); err != nil {
			return nil, err
		}
	}

	return issue, nil
}

func (s *Server) composeIssueRelationship(ctx context.Context, issue *api.Issue) error {
	var err error

	issue.Creator, err = s.composePrincipalByID(ctx, issue.CreatorID)
	if err != nil {
		return err
	}

	issue.Updater, err = s.composePrincipalByID(ctx, issue.UpdaterID)
	if err != nil {
		return err
	}

	issue.Assignee, err = s.composePrincipalByID(ctx, issue.AssigneeID)
	if err != nil {
		return err
	}

	issueSubscriberFind := &api.IssueSubscriberFind{
		IssueID: &issue.ID,
	}
	issueSubscriberList, err := s.IssueSubscriberService.FindIssueSubscriberList(ctx, issueSubscriberFind)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber list for issue %d", issue.ID)).SetInternal(err)
	}
	for _, issueSubscriber := range issueSubscriberList {
		if err := s.composeIssueSubscriberRelationship(ctx, issueSubscriber); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber %d relationship for issue %d", issueSubscriber.SubscriberID, issueSubscriber.IssueID)).SetInternal(err)
		}
		issue.SubscriberList = append(issue.SubscriberList, issueSubscriber.Subscriber)
	}

	issue.Project, err = s.composeProjectByID(ctx, issue.ProjectID)
	if err != nil {
		return err
	}

	if issue.Pipeline == nil {
		issue.Pipeline, err = s.composePipelineByID(ctx, issue.PipelineID)
		if err != nil {
			return err
		}
	} else {
		if err := s.composePipelineRelationship(ctx, issue.Pipeline); err != nil {
			return err
		}
	}

	return nil
}

// Only allow Bot/Owner/DBA as the assignee, not Developer.
func (s *Server) validateAssigneeRoleByID(ctx context.Context, assigneeID int) error {
	assignee, err := s.PrincipalService.FindPrincipal(ctx, &api.PrincipalFind{
		ID: &assigneeID,
	})
	if err != nil {
		return err
	}
	if assignee == nil {
		return fmt.Errorf("Principal ID not found: %d", assigneeID)
	}
	if err = s.composePrincipalRole(ctx, assignee); err != nil {
		return err
	}
	if assignee.Role != api.Owner && assignee.Role != api.DBA {
		return fmt.Errorf("%s is not allowed as assignee", assignee.Role)
	}

	return nil
}

func (s *Server) createIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Issue, error) {
	// Run pre-condition check first to make sure all tasks are valid, otherwise we will create partial pipelines
	// since we are not creating pipeline/stage list/task list in a single transaction.
	// We may still run into this issue when we actually create those pipeline/stage list/task list, however, that's
	// quite unlikely so we will live with it for now.
	if issueCreate.AssigneeID == api.UnknownID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
	}

	if err := s.validateAssigneeRoleByID(ctx, issueCreate.AssigneeID); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot set assignee with user id %d", issueCreate.AssigneeID)).SetInternal(err)
	}

	// If frontend does not pass the stageList, we will generate it from backend.
	pipeline, err := s.createPipelineFromIssue(ctx, issueCreate, creatorID, issueCreate.ValidateOnly)
	if err != nil {
		return nil, err
	}

	var issue *api.Issue
	if issueCreate.ValidateOnly {
		issue = &api.Issue{
			CreatorID:   creatorID,
			CreatedTs:   time.Now().Unix(),
			UpdaterID:   creatorID,
			UpdatedTs:   time.Now().Unix(),
			ProjectID:   issueCreate.ProjectID,
			Name:        issueCreate.Name,
			Status:      api.IssueOpen,
			Type:        issueCreate.Type,
			Description: issueCreate.Description,
			AssigneeID:  issueCreate.AssigneeID,
			PipelineID:  pipeline.ID,
			Pipeline:    pipeline,
		}
	} else {
		issueCreate.CreatorID = creatorID
		issueCreate.PipelineID = pipeline.ID
		issueCreated, err := s.IssueService.CreateIssue(ctx, issueCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create issue. Error %w", err)
		}
		issue = issueCreated
		// Create issue subscribers.
		for _, subscriberID := range issueCreate.SubscriberIDList {
			subscriberCreate := &api.IssueSubscriberCreate{
				IssueID:      issue.ID,
				SubscriberID: subscriberID,
			}
			_, err := s.IssueSubscriberService.CreateIssueSubscriber(ctx, subscriberCreate)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d after creating issue %d", subscriberID, issue.ID)).SetInternal(err)
			}
		}
	}
	if err := s.composeIssueRelationship(ctx, issue); err != nil {
		return nil, err
	}

	// Return early if this is a validate only request.
	if issueCreate.ValidateOnly {
		return issue, nil
	}

	task, err := s.ScheduleNextTaskIfNeeded(ctx, issue.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule task after creating the issue: %v. Error %w", issue.Name, err)
	}
	// We need to re-compose task relationship because the one in issue is modified by ScheduleNextTaskIfNeeded.
	if task != nil {
		if err := s.composeTaskRelationship(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to compose task %v, error %w", task.Name, err)
		}
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

func createPipelineValidateOnly(ctx context.Context, pc *api.PipelineCreate, creatorID int) (*api.Pipeline, error) {
	// We cannot emit ID or use default zero by following https://google.aip.dev/163, otherwise
	// jsonapi resource relationships will collide different resources into the same bucket.
	id := 0
	ts := time.Now().Unix()
	pipeline := &api.Pipeline{
		ID:        id,
		Name:      pc.Name,
		Status:    api.PipelineOpen,
		CreatorID: creatorID,
		CreatedTs: ts,
		UpdaterID: creatorID,
		UpdatedTs: ts,
	}
	for _, sc := range pc.StageList {
		id++
		stage := &api.Stage{
			ID:            id,
			Name:          sc.Name,
			CreatorID:     creatorID,
			CreatedTs:     ts,
			UpdaterID:     creatorID,
			UpdatedTs:     ts,
			PipelineID:    sc.PipelineID,
			EnvironmentID: sc.EnvironmentID,
		}
		for _, tc := range sc.TaskList {
			id++
			task := &api.Task{
				ID:                id,
				Name:              tc.Name,
				Status:            tc.Status,
				CreatorID:         creatorID,
				CreatedTs:         ts,
				UpdaterID:         creatorID,
				UpdatedTs:         ts,
				Type:              tc.Type,
				Payload:           tc.Payload,
				EarliestAllowedTs: tc.EarliestAllowedTs,
				PipelineID:        pipeline.ID,
				StageID:           stage.ID,
				InstanceID:        tc.InstanceID,
				DatabaseID:        tc.DatabaseID,
			}
			stage.TaskList = append(stage.TaskList, task)
		}
		pipeline.StageList = append(pipeline.StageList, stage)
	}
	return pipeline, nil
}

func (s *Server) createPipelineFromIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int, validateOnly bool) (*api.Pipeline, error) {
	var pipelineCreate *api.PipelineCreate
	switch {
	case issueCreate.Type == api.IssueDatabaseCreate:
		m := api.CreateDatabaseContext{}
		if err := json.Unmarshal([]byte(issueCreate.CreateContext), &m); err != nil {
			return nil, err
		}
		if m.DatabaseName == "" {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
		}

		// Find instance.
		instance, err := s.composeInstanceByID(ctx, m.InstanceID)
		if err != nil {
			return nil, err
		}
		if instance == nil {
			return nil, fmt.Errorf("instance ID not found %v", m.InstanceID)
		}
		// Find project
		project, err := s.composeProjectByID(ctx, issueCreate.ProjectID)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", issueCreate.ProjectID)).SetInternal(err)
		}
		if project == nil {
			err := fmt.Errorf("project ID not found %v", issueCreate.ProjectID)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
		}

		switch instance.Engine {
		case db.ClickHouse:
			// ClickHouse does not support character set and collation at the database level.
			if m.CharacterSet != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, ClickHouse does not support character set, got %s\n", m.CharacterSet),
				)
			}
			if m.Collation != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, ClickHouse does not support collation, got %s\n", m.Collation),
				)
			}
		case db.Snowflake:
			if m.CharacterSet != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, Snowflake does not support character set, got %s\n", m.CharacterSet),
				)
			}
			if m.Collation != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, Snowflake does not support collation, got %s\n", m.Collation),
				)
			}

			// Snowflake needs to use upper case of DatabaseName.
			m.DatabaseName = strings.ToUpper(m.DatabaseName)
		case db.SQLite:
			// no-op.
		default:
			if m.CharacterSet == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, character set missing")
			}
			// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
			// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
			// install it.
			if instance.Engine != db.Postgres && m.Collation == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, collation missing")
			}
		}

		// Validate the labels. Labels are set upon task completion.
		if m.Labels != "" {
			if err := s.setDatabaseLabels(ctx, m.Labels, &api.Database{Name: m.DatabaseName, Instance: instance} /* dummy database */, project, 0 /* dummy updaterID */, true /* validateOnly */); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid database label %q, error %v", m.Labels, err))
			}
		}

		var schemaVersion, schema string
		// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
		if project.TenantMode == api.TenantModeTenant {
			if !s.feature(api.FeatureMultiTenancy) {
				return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
			}
			baseDatabaseName, err := api.GetBaseDatabaseName(m.DatabaseName, project.DBNameTemplate, m.Labels)
			if err != nil {
				return nil, fmt.Errorf("api.GetBaseDatabaseName(%q, %q, %q) failed, error: %v", m.DatabaseName, project.DBNameTemplate, m.Labels, err)
			}
			sv, s, err := s.getSchemaFromPeerTenantDatabase(ctx, instance, project, issueCreate.ProjectID, baseDatabaseName)
			if err != nil {
				return nil, err
			}
			schemaVersion, schema = sv, s
		}
		if schemaVersion == "" {
			schemaVersion = defaultMigrationVersionFromTaskID()
		}

		payload := api.TaskDatabaseCreatePayload{}
		payload.ProjectID = issueCreate.ProjectID
		payload.CharacterSet = m.CharacterSet
		payload.Collation = m.Collation
		payload.Labels = m.Labels
		payload.SchemaVersion = schemaVersion
		payload.DatabaseName, payload.Statement = getDatabaseNameAndStatement(instance.Engine, m.DatabaseName, m.CharacterSet, m.Collation, schema)
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create database creation task, unable to marshal payload %w", err)
		}

		taskStatus := api.TaskPendingApproval
		policy, err := s.PolicyService.GetPipelineApprovalPolicy(ctx, instance.EnvironmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get approval policy for environment ID %v, error %v", instance.EnvironmentID, err)
		}
		if policy.Value == api.PipelineApprovalValueManualNever {
			taskStatus = api.TaskPending
		}

		if m.BackupID != 0 {
			backupRaw, err := s.BackupService.FindBackup(ctx, &api.BackupFind{ID: &m.BackupID})
			if err != nil {
				return nil, fmt.Errorf("failed to find backup %v", m.BackupID)
			}
			restorePayload := api.TaskDatabaseRestorePayload{}
			restorePayload.DatabaseName = m.DatabaseName
			restorePayload.BackupID = m.BackupID
			restoreBytes, err := json.Marshal(restorePayload)
			if err != nil {
				return nil, fmt.Errorf("failed to create restore database task, unable to marshal payload %w", err)
			}

			pipelineCreate = &api.PipelineCreate{
				Name: fmt.Sprintf("Pipeline - Create database %v from backup %v", payload.DatabaseName, backupRaw.Name),
				StageList: []api.StageCreate{
					{
						Name:          "Create database",
						EnvironmentID: instance.EnvironmentID,
						TaskList: []api.TaskCreate{
							{
								InstanceID:   m.InstanceID,
								Name:         fmt.Sprintf("Create database %v", payload.DatabaseName),
								Status:       taskStatus,
								Type:         api.TaskDatabaseCreate,
								DatabaseName: payload.DatabaseName,
								Payload:      string(bytes),
							},
						},
					},
					{
						Name:          "Restore backup",
						EnvironmentID: instance.EnvironmentID,
						TaskList: []api.TaskCreate{
							{
								InstanceID:   m.InstanceID,
								Name:         fmt.Sprintf("Restore backup %v", backupRaw.Name),
								Status:       api.TaskPending,
								Type:         api.TaskDatabaseRestore,
								DatabaseName: payload.DatabaseName,
								BackupID:     &m.BackupID,
								Payload:      string(restoreBytes),
							},
						},
					},
				},
			}
		} else {
			pipelineCreate = &api.PipelineCreate{
				Name: fmt.Sprintf("Pipeline - Create database %v", payload.DatabaseName),
				StageList: []api.StageCreate{
					{
						Name:          "Create database",
						EnvironmentID: instance.EnvironmentID,
						TaskList: []api.TaskCreate{
							{
								InstanceID:   m.InstanceID,
								Name:         fmt.Sprintf("Create database %v", payload.DatabaseName),
								Status:       taskStatus,
								Type:         api.TaskDatabaseCreate,
								DatabaseName: payload.DatabaseName,
								Payload:      string(bytes),
							},
						},
					},
				},
			}
		}
	case issueCreate.Type == api.IssueDatabaseSchemaUpdate || issueCreate.Type == api.IssueDatabaseDataUpdate:
		m := api.UpdateSchemaContext{}
		if err := json.Unmarshal([]byte(issueCreate.CreateContext), &m); err != nil {
			return nil, err
		}
		if !s.feature(api.FeatureTaskScheduleTime) {
			for _, detail := range m.UpdateSchemaDetailList {
				if detail.EarliestAllowedTs != 0 {
					return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
				}
			}
		}
		pc := &api.PipelineCreate{}
		switch m.MigrationType {
		case db.Baseline:
			pc.Name = "Establish database baseline pipeline"
		case db.Migrate:
			pc.Name = "Update database schema pipeline"
		case db.Data:
			pc.Name = "Update database data pipeline"
		default:
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid migration type %q", m.MigrationType))
		}
		project, err := s.composeProjectByID(ctx, issueCreate.ProjectID)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", issueCreate.ProjectID)).SetInternal(err)
		}

		schemaVersion := defaultMigrationVersionFromTaskID()
		// Tenant mode project pipeline has its own generation.
		if project.TenantMode == api.TenantModeTenant {
			if !s.feature(api.FeatureMultiTenancy) {
				return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
			}
			if !(m.MigrationType == db.Migrate || m.MigrationType == db.Data) {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Only Migrate and Data type migration can be performed on tenant mode project")
			}
			if len(m.UpdateSchemaDetailList) != 1 {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Tenant mode project should have exactly one update schema detail")
			}
			d := m.UpdateSchemaDetailList[0]
			if d.Statement == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
			}

			databaseList, err := s.DatabaseService.FindDatabaseList(ctx, &api.DatabaseFind{
				ProjectID: &issueCreate.ProjectID,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", issueCreate.ProjectID)).SetInternal(err)
			}
			baseDatabaseName := d.DatabaseName
			if err != nil {
				return nil, fmt.Errorf("api.GetBaseDatabaseName(%q, %q) failed, error: %v", d.DatabaseName, project.DBNameTemplate, err)
			}
			deployments, matrix, err := s.getTenantDatabaseMatrix(ctx, issueCreate.ProjectID, project.DBNameTemplate, databaseList, baseDatabaseName)
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
					taskStatus := api.TaskPendingApproval
					policy, err := s.PolicyService.GetPipelineApprovalPolicy(ctx, environmentID)
					if err != nil {
						return nil, fmt.Errorf("failed to get approval policy for environment ID %v, error %v", environmentID, err)
					}
					if policy.Value == api.PipelineApprovalValueManualNever {
						taskStatus = api.TaskPending
					}
					taskCreate, err := getUpdateTask(database, m.MigrationType, m.VCSPushEvent, d, schemaVersion, taskStatus)
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

				pc.StageList = append(pc.StageList, api.StageCreate{
					Name:          fmt.Sprintf("Deployment: %s", deployments[i].Name),
					EnvironmentID: environmentID,
					TaskList:      taskCreateList,
				})
			}
			pipelineCreate = pc
		} else {
			for _, d := range m.UpdateSchemaDetailList {
				if m.MigrationType == db.Migrate && d.Statement == "" {
					return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
				}
				databaseFind := &api.DatabaseFind{
					ID: &d.DatabaseID,
				}
				database, err := s.composeDatabaseByFind(ctx, databaseFind)
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", d.DatabaseID)).SetInternal(err)
				}
				if database == nil {
					return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", d.DatabaseID))
				}

				taskStatus := api.TaskPendingApproval
				policy, err := s.PolicyService.GetPipelineApprovalPolicy(ctx, database.Instance.EnvironmentID)
				if err != nil {
					return nil, fmt.Errorf("failed to get approval policy for environment ID %v, error %v", database.Instance.EnvironmentID, err)
				}
				if policy.Value == api.PipelineApprovalValueManualNever {
					taskStatus = api.TaskPending
				}

				taskCreate, err := getUpdateTask(database, m.MigrationType, m.VCSPushEvent, d, schemaVersion, taskStatus)
				if err != nil {
					return nil, err
				}

				pc.StageList = append(pc.StageList, api.StageCreate{
					Name:          fmt.Sprintf("%s %s", database.Instance.Environment.Name, database.Name),
					EnvironmentID: database.Instance.Environment.ID,
					TaskList:      []api.TaskCreate{*taskCreate},
				})
			}
			pipelineCreate = pc
		}
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid issue type %q", issueCreate.Type))
	}

	// Return an error if the issue has no task to be executed
	hasTask := false
	for _, stage := range pipelineCreate.StageList {
		if len(stage.TaskList) > 0 {
			hasTask = true
			break
		}
	}
	if !hasTask {
		err := fmt.Errorf("issue has no task to be executed")
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}

	// Create the pipeline, stages, and tasks.
	if validateOnly {
		return createPipelineValidateOnly(ctx, pipelineCreate, creatorID)
	}

	pipelineCreate.CreatorID = creatorID
	createdPipeline, err := s.PipelineService.CreatePipeline(ctx, pipelineCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline for issue, error %v", err)
	}

	for _, stageCreate := range pipelineCreate.StageList {
		stageCreate.CreatorID = creatorID
		stageCreate.PipelineID = createdPipeline.ID
		createdStage, err := s.StageService.CreateStage(ctx, &stageCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create stage for issue, error %v", err)
		}

		for _, taskCreate := range stageCreate.TaskList {
			taskCreate.CreatorID = creatorID
			taskCreate.PipelineID = createdPipeline.ID
			taskCreate.StageID = createdStage.ID
			if _, err = s.TaskService.CreateTask(ctx, &taskCreate); err != nil {
				return nil, fmt.Errorf("failed to create task for issue, error %v", err)
			}
		}
	}
	return createdPipeline, nil
}

func getUpdateTask(database *api.Database, migrationType db.MigrationType, vcsPushEvent *vcs.PushEvent, d *api.UpdateSchemaDetail, schemaVersion string, taskStatus api.TaskStatus) (*api.TaskCreate, error) {
	taskName := fmt.Sprintf("Establish %q baseline", database.Name)
	if migrationType == db.Migrate {
		taskName = fmt.Sprintf("Update %q schema", database.Name)
	} else if migrationType == db.Data {
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
		Status:            taskStatus,
		Type:              taskType,
		Statement:         d.Statement,
		EarliestAllowedTs: d.EarliestAllowedTs,
		MigrationType:     migrationType,
		Payload:           string(bytes),
	}, nil
}

func getDatabaseNameAndStatement(dbType db.Type, databaseName, characterSet, collation, schema string) (string, string) {
	// Snowflake needs to use upper case of DatabaseName.
	if dbType == db.Snowflake {
		databaseName = strings.ToUpper(databaseName)
	}

	var stmt string
	switch dbType {
	case db.MySQL, db.TiDB:
		stmt = fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", databaseName, characterSet, collation)
		if schema != "" {
			stmt = fmt.Sprintf("%s\nUSE `%s`;\n%s", stmt, databaseName, schema)
		}
	case db.Postgres:
		if collation == "" {
			stmt = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q;", databaseName, characterSet)
		} else {
			stmt = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q;", databaseName, characterSet, collation)
		}
		if schema != "" {
			stmt = fmt.Sprintf("%s\n\\connect \"%s\";\n%s", stmt, databaseName, schema)
		}
	case db.ClickHouse:
		stmt = fmt.Sprintf("CREATE DATABASE `%s`;", databaseName)
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
		// This is a fake CREATE DATABASE statement since a single SQLite file represents a database. Engine driver will recognize it and establish a connection to create the sqlite file representing the database.
		stmt = fmt.Sprintf("CREATE DATABASE '%s';", databaseName)
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
					if _, err := s.changeTaskStatus(ctx, task, api.TaskCanceled, updaterID); err != nil {
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
	if _, err := s.PipelineService.PatchPipeline(ctx, pipelinePatch); err != nil {
		return nil, fmt.Errorf("failed to update issue status: %v, failed to update pipeline status: %w", issue.Name, err)
	}

	issuePatch := &api.IssuePatch{
		ID:        issue.ID,
		UpdaterID: updaterID,
		Status:    &newStatus,
	}
	updatedIssue, err := s.IssueService.PatchIssue(ctx, issuePatch)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, fmt.Errorf("failed to update issue status: %v, error: %w", issue.Name, err)
		}
		return nil, fmt.Errorf("failed update issue status: %v, error: %w", issue.Name, err)
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
		_, err := s.InboxService.CreateInbox(ctx, inboxCreate)
		if err != nil {
			return fmt.Errorf("failed to post activity to creator inbox: %d, error: %w", issue.CreatorID, err)
		}
	}

	if issue.AssigneeID != api.SystemBotID && issue.AssigneeID != issue.CreatorID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.AssigneeID,
			ActivityID: activityID,
		}
		_, err := s.InboxService.CreateInbox(ctx, inboxCreate)
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
			_, err := s.InboxService.CreateInbox(ctx, inboxCreate)
			if err != nil {
				return fmt.Errorf("failed to post activity to subscriber inbox: %d, error: %w", subscriber.ID, err)
			}
		}
	}

	return nil
}

func (s *Server) getTenantDatabaseMatrix(ctx context.Context, projectID int, dbNameTemplate string, databaseList []*api.Database, baseDatabaseName string) ([]*api.Deployment, [][]*api.Database, error) {
	deployConfig, err := s.DeploymentConfigService.FindDeploymentConfig(ctx, &api.DeploymentConfigFind{
		ProjectID: &projectID,
	})
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
	for _, database := range databaseList {
		if err := s.composeDatabaseRelationship(ctx, database); err != nil {
			return nil, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose database relationship for database ID %v", database.ID)).SetInternal(err)
		}
	}
	d, matrix, err := getDatabaseMatrixFromDeploymentSchedule(deploySchedule, baseDatabaseName, dbNameTemplate, databaseList)
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
	databaseList, err := s.DatabaseService.FindDatabaseList(ctx, &api.DatabaseFind{
		ProjectID: &projectID,
	})
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", projectID)).SetInternal(err)
	}

	_, matrix, err := s.getTenantDatabaseMatrix(ctx, projectID, project.DBNameTemplate, databaseList, baseDatabaseName)
	if err != nil {
		return "", "", err
	}
	similarDB := getPeerTenantDatabase(matrix, instance.EnvironmentID)

	// When there is no existing tenant, we will look at all existing databases in the tenant mode project.
	// If there are existing databases with the same name, we will disallow the database creation.
	// Otherwise, we will create a blank new database.
	if similarDB == nil {
		found := false
		for _, db := range databaseList {
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
			err := fmt.Errorf("Conflicting database name, project has existing base database named %q, but it's not from the selected peer tenants", baseDatabaseName)
			return "", "", echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		return "", "", nil
	}

	driver, err := getAdminDatabaseDriver(ctx, similarDB.Instance, similarDB.Name, s.l)
	if err != nil {
		return "", "", err
	}
	defer driver.Close(ctx)
	schemaVersion, err := getLatestSchemaVersion(ctx, driver, similarDB.Name)
	if err != nil {
		return "", "", fmt.Errorf("failed to get migration history for database %q: %w", similarDB.Name, err)
	}

	var schemaBuf bytes.Buffer
	if err := driver.Dump(ctx, similarDB.Name, &schemaBuf, true /* schemaOnly */); err != nil {
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
