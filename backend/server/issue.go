package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		issueCreate := &api.IssueCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create issue request").SetInternal(err)
		}

		issue, err := s.CreateIssue(context.Background(), issueCreate, c.Get(GetPrincipalIdContextKey()).(int))
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
		issueFind := &api.IssueFind{}
		projectIdStr := c.QueryParams().Get("project")
		if projectIdStr != "" {
			projectId, err := strconv.Atoi(projectIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIdStr)).SetInternal(err)
			}
			issueFind.ProjectId = &projectId
		}
		userIdStr := c.QueryParams().Get("user")
		if userIdStr != "" {
			userId, err := strconv.Atoi(userIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("user query parameter is not a number: %s", userIdStr)).SetInternal(err)
			}
			issueFind.PrincipalId = &userId
		}
		list, err := s.IssueService.FindIssueList(context.Background(), issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch issue list").SetInternal(err)
		}

		for _, issue := range list {
			if err := s.ComposeIssueRelationship(context.Background(), issue, c.Get(getIncludeKey()).([]string)); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("issueId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		issue, err := s.ComposeIssueById(context.Background(), id, c.Get(getIncludeKey()).([]string))
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
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

	g.PATCH("/issue/:issueId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("issueId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		issuePatch := &api.IssuePatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issuePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue request").SetInternal(err)
		}

		issueFind := &api.IssueFind{
			ID: &id,
		}
		issue, err := s.IssueService.FindIssue(context.Background(), issueFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}

		updatedIssue, err := s.IssueService.PatchIssue(context.Background(), issuePatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue ID: %v", id)).SetInternal(err)
		}

		payloadList := [][]byte{}
		if issuePatch.Name != nil && *issuePatch.Name != issue.Name {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldId:  api.IssueFieldName,
				OldValue: issue.Name,
				NewValue: *issuePatch.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue name: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.Description != nil && *issuePatch.Description != issue.Description {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldId:  api.IssueFieldDescription,
				OldValue: issue.Description,
				NewValue: *issuePatch.Description,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue description: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.AssigneeId != nil && *issuePatch.AssigneeId != issue.AssigneeId {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldId:  api.IssueFieldAssignee,
				OldValue: strconv.Itoa(issue.AssigneeId),
				NewValue: strconv.Itoa(*issuePatch.AssigneeId),
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue assignee: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.SubscriberIdListStr != nil {
			oldSubscriberIdList := []string{}
			for _, item := range issue.SubscriberIdList {
				oldSubscriberIdList = append(oldSubscriberIdList, strconv.Itoa(item))
			}
			oldSubscriberIdStr := strings.Join(oldSubscriberIdList, ",")

			if *issuePatch.SubscriberIdListStr != oldSubscriberIdStr {
				payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
					FieldId:  api.IssueFieldSubscriberList,
					OldValue: oldSubscriberIdStr,
					NewValue: *issuePatch.SubscriberIdListStr,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue subscribers: %v", updatedIssue.Name)).SetInternal(err)
				}
				payloadList = append(payloadList, payload)
			}
		}

		for _, payload := range payloadList {
			_, err = s.ActivityService.CreateActivity(context.Background(), &api.ActivityCreate{
				CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
				ContainerId: issue.ID,
				Type:        api.ActivityIssueFieldUpdate,
				Payload:     string(payload),
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating issue: %v", updatedIssue.Name)).SetInternal(err)
			}
		}

		if err := s.ComposeIssueRelationship(context.Background(), updatedIssue, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update issue response: %v", updatedIssue.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueId/status", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("issueId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		issueStatusPatch := &api.IssueStatusPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue status request").SetInternal(err)
		}

		issue, err := s.ComposeIssueById(context.Background(), id, c.Get(getIncludeKey()).([]string))
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}

		updatedIssue, err := s.ChangeIssueStatus(context.Background(), issue, issueStatusPatch.Status, issueStatusPatch.UpdaterId, issueStatusPatch.Comment)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound).SetInternal(err)
			} else if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		if err := s.ComposeIssueRelationship(context.Background(), updatedIssue, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeIssueById(ctx context.Context, id int, includeList []string) (*api.Issue, error) {
	issueFind := &api.IssueFind{
		ID: &id,
	}
	issue, err := s.IssueService.FindIssue(context.Background(), issueFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeIssueRelationship(ctx, issue, includeList); err != nil {
		return nil, err
	}

	return issue, nil
}

func (s *Server) ComposeIssueRelationship(ctx context.Context, issue *api.Issue, includeList []string) error {
	var err error

	issue.Creator, err = s.ComposePrincipalById(context.Background(), issue.CreatorId, includeList)
	if err != nil {
		return err
	}

	issue.Updater, err = s.ComposePrincipalById(context.Background(), issue.UpdaterId, includeList)
	if err != nil {
		return err
	}

	issue.Assignee, err = s.ComposePrincipalById(context.Background(), issue.AssigneeId, includeList)
	if err != nil {
		return err
	}

	subscriberList := []*api.Principal{}
	for _, subscriberId := range issue.SubscriberIdList {
		oneSubscriber, err := s.ComposePrincipalById(context.Background(), subscriberId, includeList)
		if err != nil {
			return err
		}
		subscriberList = append(subscriberList, oneSubscriber)
	}
	issue.SubscriberList = subscriberList

	issue.Project, err = s.ComposeProjectlById(context.Background(), issue.ProjectId, includeList)
	if err != nil {
		return err
	}

	issue.Pipeline, err = s.ComposePipelineById(context.Background(), issue.PipelineId, includeList)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) CreateIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorId int) (*api.Issue, error) {
	issueCreate.Pipeline.CreatorId = creatorId
	createdPipeline, err := s.PipelineService.CreatePipeline(ctx, &issueCreate.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline for issue. Error %w", err)
	}

	for _, stageCreate := range issueCreate.Pipeline.StageList {
		stageCreate.CreatorId = creatorId
		stageCreate.PipelineId = createdPipeline.ID
		createdStage, err := s.StageService.CreateStage(context.Background(), &stageCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create stage for issue. Error %w", err)
		}

		for _, taskCreate := range stageCreate.TaskList {
			taskCreate.CreatorId = creatorId
			taskCreate.PipelineId = createdPipeline.ID
			taskCreate.StageId = createdStage.ID
			if taskCreate.Type == api.TaskDatabaseCreate {
				payload := api.TaskDatabaseCreatePayload{}
				if taskCreate.Statement == "" {
					return nil, fmt.Errorf("failed to create database creation task, sql statement missing")
				}
				if taskCreate.DatabaseName == "" {
					return nil, fmt.Errorf("failed to create database creation task, database name missing")
				}
				if taskCreate.CharacterSet == "" {
					return nil, fmt.Errorf("failed to create database creation task, character set missing")
				}
				if taskCreate.Collation == "" {
					return nil, fmt.Errorf("failed to create database creation task, collation missing")
				}
				payload.Statement = taskCreate.Statement
				payload.DatabaseName = taskCreate.DatabaseName
				payload.CharacterSet = taskCreate.CharacterSet
				payload.Collation = taskCreate.Collation
				bytes, err := json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("failed to create database creation task, unable to marshal payload %w", err)
				}
				taskCreate.Payload = string(bytes)
			} else if taskCreate.Type == api.TaskDatabaseSchemaUpdate {
				payload := api.TaskDatabaseSchemaUpdatePayload{}
				if taskCreate.Statement == "" {
					return nil, fmt.Errorf("failed to create schema update task, sql statement missing")
				}
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
			}
			_, err := s.TaskService.CreateTask(context.Background(), &taskCreate)
			if err != nil {
				return nil, fmt.Errorf("failed to create task for issue. Error %w", err)
			}
		}
	}

	issueCreate.CreatorId = creatorId
	issueCreate.PipelineId = createdPipeline.ID
	issue, err := s.IssueService.CreateIssue(context.Background(), issueCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue. Error %w", err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorId:   creatorId,
		ContainerId: issue.ID,
		Type:        api.ActivityIssueCreate,
	}
	_, err = s.ActivityService.CreateActivity(context.Background(), activityCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after creating the issue: %v. Error %w", issue.Name, err)
	}

	if err := s.ComposeIssueRelationship(context.Background(), issue, []string{}); err != nil {
		return nil, err
	}

	if err := s.ScheduleNextTaskIfNeeded(context.Background(), issue.Pipeline); err != nil {
		return nil, fmt.Errorf("failed to schedule task after creating the issue: %v. Error %w", issue.Name, err)
	}

	return issue, nil
}

func (s *Server) ChangeIssueStatus(ctx context.Context, issue *api.Issue, newStatus api.IssueStatus, updatorId int, comment string) (*api.Issue, error) {
	var pipelineStatus api.PipelineStatus
	switch newStatus {
	case api.Issue_Open:
		pipelineStatus = api.Pipeline_Open
	case api.Issue_Done:
		// Returns error if any of the tasks is not DONE.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status != api.TaskDone {
					return nil, &bytebase.Error{Code: bytebase.ECONFLICT, Message: fmt.Sprintf("failed to resolve issue: %v, task %v has not finished", issue.Name, task.Name)}
				}
			}
		}
		pipelineStatus = api.Pipeline_Done
	case api.Issue_Canceled:
		// If we want to cancel the issue, we find the current running tasks, mark each of them CANCELED.
		// We keep PENDING and FAILED tasks as is since the issue maybe reopened later, and it's better to
		// keep those tasks in the same state before the issue was canceled.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status == api.TaskRunning {
					if _, err := s.ChangeTaskStatus(context.Background(), task, api.TaskCanceled, updatorId); err != nil {
						return nil, fmt.Errorf("failed to cancel issue: %v, failed to cancel task: %v, error: %w", issue.Name, task.Name, err)
					}
				}
			}
		}
		pipelineStatus = api.Pipeline_Canceled
	}

	pipelinePatch := &api.PipelinePatch{
		ID:        issue.PipelineId,
		UpdaterId: updatorId,
		Status:    &pipelineStatus,
	}
	if _, err := s.PipelineService.PatchPipeline(context.Background(), pipelinePatch); err != nil {
		return nil, fmt.Errorf("failed to update issue status: %v, failed to update pipeline status: %w", issue.Name, err)
	}

	issuePatch := &api.IssuePatch{
		ID:        issue.ID,
		UpdaterId: updatorId,
		Status:    &newStatus,
	}
	updatedIssue, err := s.IssueService.PatchIssue(context.Background(), issuePatch)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
			return nil, fmt.Errorf("failed to update issue status: %v, error: %w", issue.Name, err)
		}
		return nil, fmt.Errorf("failed update issue status: %v, error: %w", issue.Name, err)
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity after changing the issue status: %v, error: %w", issue.Name, err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorId:   updatorId,
		ContainerId: issue.ID,
		Type:        api.ActivityIssueStatusUpdate,
		Comment:     comment,
		Payload:     string(payload),
	}
	_, err = s.ActivityService.CreateActivity(context.Background(), activityCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after changing the issue status: %v, error: %w", issue.Name, err)
	}

	return updatedIssue, nil
}
