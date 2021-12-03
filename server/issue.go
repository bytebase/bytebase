package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
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

		// Run pre-condition check first to make sure all tasks are valid, otherwise we will create partial pipelines
		// since we are not creating pipeline/stage list/task list in a single transaction.
		// We may still run into this issue when we actually create those pipeline/stage list/task list, however, that's
		// quite unlikely so we will live with it for now.
		if issueCreate.AssigneeID == api.UnknownID {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
		}

		for _, stageCreate := range issueCreate.Pipeline.StageList {
			for _, taskCreate := range stageCreate.TaskList {
				if taskCreate.Type == api.TaskDatabaseCreate {
					if taskCreate.Statement != "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement should not be set.")
					}
					if taskCreate.DatabaseName == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
					}
					instanceFind := &api.InstanceFind{
						ID: &taskCreate.InstanceID,
					}
					instance, err := s.InstanceService.FindInstance(ctx, instanceFind)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue.").SetInternal(err)
					}
					// ClickHouse does not support character set and collation at the database level.
					if instance.Engine == db.ClickHouse {
						if taskCreate.CharacterSet != "" {
							return echo.NewHTTPError(
								http.StatusBadRequest,
								fmt.Sprintf("Failed to create issue, ClickHouse does not support character set, got %s\n", taskCreate.CharacterSet),
							)
						}
						if taskCreate.Collation != "" {
							return echo.NewHTTPError(
								http.StatusBadRequest,
								fmt.Sprintf("Failed to create issue, ClickHouse does not support collation, got %s\n", taskCreate.Collation),
							)
						}
					} else if instance.Engine == db.Snowflake {
						if taskCreate.CharacterSet != "" {
							return echo.NewHTTPError(
								http.StatusBadRequest,
								fmt.Sprintf("Failed to create issue, Snowflake does not support character set, got %s\n", taskCreate.CharacterSet),
							)
						}
						if taskCreate.Collation != "" {
							return echo.NewHTTPError(
								http.StatusBadRequest,
								fmt.Sprintf("Failed to create issue, Snowflake does not support collation, got %s\n", taskCreate.Collation),
							)
						}
					} else {
						if taskCreate.CharacterSet == "" {
							return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, character set missing")
						}
						// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
						// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
						// install it.
						if instance.Engine != db.Postgres && taskCreate.Collation == "" {
							return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, collation missing")
						}
					}
				} else if taskCreate.Type == api.TaskDatabaseSchemaUpdate {
					if taskCreate.Statement == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
					}
				} else if taskCreate.Type == api.TaskDatabaseRestore {
					if taskCreate.DatabaseName == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
					}
					if taskCreate.BackupID == nil {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, backup missing")
					}
				}
			}
		}

		issue, err := s.createIssue(ctx, issueCreate, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
		}

		for _, subscriberID := range issueCreate.SubscriberIDList {
			subscriberCreate := &api.IssueSubscriberCreate{
				IssueID:      issue.ID,
				SubscriberID: subscriberID,
			}
			_, err := s.IssueSubscriberService.CreateIssueSubscriber(ctx, subscriberCreate)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d after creating issue %d", subscriberID, issue.ID)).SetInternal(err)
			}
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
		list, err := s.IssueService.FindIssueList(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch issue list").SetInternal(err)
		}

		for _, issue := range list {
			if err := s.composeIssueRelationship(ctx, issue); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
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
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
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

		issueFind := &api.IssueFind{
			ID: &id,
		}
		issue, err := s.IssueService.FindIssue(ctx, issueFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
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
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
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

	if err := s.composeIssueRelationship(ctx, issue); err != nil {
		return nil, err
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
	list, err := s.IssueSubscriberService.FindIssueSubscriberList(ctx, issueSubscriberFind)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber list for issue %d", issue.ID)).SetInternal(err)
	}

	issue.SubscriberIDList = []int{}
	for _, subscriber := range list {
		issue.SubscriberIDList = append(issue.SubscriberIDList, subscriber.SubscriberID)
	}

	issue.Project, err = s.composeProjectlByID(ctx, issue.ProjectID)
	if err != nil {
		return err
	}

	issue.Pipeline, err = s.composePipelineByID(ctx, issue.PipelineID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) createIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Issue, error) {
	issueCreate.Pipeline.CreatorID = creatorID
	createdPipeline, err := s.PipelineService.CreatePipeline(ctx, &issueCreate.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline for issue. Error %w", err)
	}

	for _, stageCreate := range issueCreate.Pipeline.StageList {
		stageCreate.CreatorID = creatorID
		stageCreate.PipelineID = createdPipeline.ID
		createdStage, err := s.StageService.CreateStage(ctx, &stageCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create stage for issue. Error %w", err)
		}

		for _, taskCreate := range stageCreate.TaskList {
			taskCreate.CreatorID = creatorID
			taskCreate.PipelineID = createdPipeline.ID
			taskCreate.StageID = createdStage.ID
			instanceFind := &api.InstanceFind{
				ID: &taskCreate.InstanceID,
			}
			instance, err := s.InstanceService.FindInstance(ctx, instanceFind)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch instance in issue creation: %v", err)
			}
			if taskCreate.Type == api.TaskDatabaseCreate {
				// Snowflake needs to use upper case of DatabaseName.
				if instance.Engine == db.Snowflake {
					taskCreate.DatabaseName = strings.ToUpper(taskCreate.DatabaseName)
				}
				payload := api.TaskDatabaseCreatePayload{}
				payload.ProjectID = issueCreate.ProjectID
				payload.DatabaseName = taskCreate.DatabaseName
				payload.CharacterSet = taskCreate.CharacterSet
				payload.Collation = taskCreate.Collation

				switch instance.Engine {
				case db.MySQL, db.TiDB:
					payload.Statement = fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s", taskCreate.DatabaseName, taskCreate.CharacterSet, taskCreate.Collation)
				case db.Postgres:
					if taskCreate.Collation == "" {
						payload.Statement = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q", taskCreate.DatabaseName, taskCreate.CharacterSet)
					} else {
						payload.Statement = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q", taskCreate.DatabaseName, taskCreate.CharacterSet, taskCreate.Collation)
					}
				case db.ClickHouse:
					payload.Statement = fmt.Sprintf("CREATE DATABASE `%s`", taskCreate.DatabaseName)
				case db.Snowflake:
					payload.Statement = fmt.Sprintf("CREATE DATABASE %s", taskCreate.DatabaseName)
				}
				bytes, err := json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("failed to create database creation task, unable to marshal payload %w", err)
				}
				taskCreate.Payload = string(bytes)
			} else if taskCreate.Type == api.TaskDatabaseSchemaUpdate {
				payload := api.TaskDatabaseSchemaUpdatePayload{}
				payload.MigrationType = taskCreate.MigrationType
				payload.Statement = taskCreate.Statement
				if taskCreate.RollbackStatement != "" {
					payload.RollbackStatement = taskCreate.RollbackStatement
				}
				if taskCreate.VCSPushEvent != nil {
					payload.VCSPushEvent = taskCreate.VCSPushEvent
				}
				bytes, err := json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("failed to create schema update task, unable to marshal payload %w", err)
				}
				taskCreate.Payload = string(bytes)
			} else if taskCreate.Type == api.TaskDatabaseRestore {
				// Snowflake needs to use upper case of DatabaseName.
				if instance.Engine == db.Snowflake {
					taskCreate.DatabaseName = strings.ToUpper(taskCreate.DatabaseName)
				}
				payload := api.TaskDatabaseRestorePayload{}
				payload.DatabaseName = taskCreate.DatabaseName
				payload.BackupID = *taskCreate.BackupID
				bytes, err := json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("failed to create restore database task, unable to marshal payload %w", err)
				}
				taskCreate.Payload = string(bytes)
			}
			if _, err = s.TaskService.CreateTask(ctx, &taskCreate); err != nil {
				return nil, fmt.Errorf("failed to create task for issue. Error %w", err)
			}
		}
	}

	issueCreate.CreatorID = creatorID
	issueCreate.PipelineID = createdPipeline.ID
	issue, err := s.IssueService.CreateIssue(ctx, issueCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue. Error %w", err)
	}

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Name,
	}
	if issueCreate.RollbackIssueID != nil {
		createActivityPayload.RollbackIssueID = *issueCreate.RollbackIssueID
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

	// If we are creating a rollback issue, then we will also post a comment on the original issue
	if issueCreate.RollbackIssueID != nil {
		issueFind := &api.IssueFind{
			ID: issueCreate.RollbackIssueID,
		}
		rollbackIssue, err := s.IssueService.FindIssue(ctx, issueFind)
		if err != nil {
			return nil, fmt.Errorf("failed to create activity after creating the rollback issue: %v. Error %w", issue.Name, err)
		}
		bytes, err := json.Marshal(api.ActivityIssueCommentCreatePayload{
			IssueName: rollbackIssue.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create activity after creating the rollback issue: %v. Error %w", issue.Name, err)
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   creatorID,
			ContainerID: *issueCreate.RollbackIssueID,
			Type:        api.ActivityIssueCommentCreate,
			Level:       api.ActivityInfo,
			Comment:     fmt.Sprintf("Created rollback issue %q", issue.Name),
			Payload:     string(bytes),
		}
		_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
			issue: rollbackIssue,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create activity after creating the rollback issue: %v. Error %w", issue.Name, err)
		}
	}

	if err := s.composeIssueRelationship(ctx, issue); err != nil {
		return nil, err
	}

	if _, err := s.ScheduleNextTaskIfNeeded(ctx, issue.Pipeline); err != nil {
		return nil, fmt.Errorf("failed to schedule task after creating the issue: %v. Error %w", issue.Name, err)
	}

	return issue, nil
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

	for _, subscriberID := range issue.SubscriberIDList {
		if subscriberID != api.SystemBotID && subscriberID != issue.CreatorID && subscriberID != issue.AssigneeID {
			inboxCreate := &api.InboxCreate{
				ReceiverID: subscriberID,
				ActivityID: activityID,
			}
			_, err := s.InboxService.CreateInbox(ctx, inboxCreate)
			if err != nil {
				return fmt.Errorf("failed to post activity to subscriber inbox: %d, error: %w", subscriberID, err)
			}
		}
	}

	return nil
}
