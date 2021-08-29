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
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		issueCreate := &api.IssueCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create issue request").SetInternal(err)
		}

		// Run pre-condition check first to make sure all tasks are valid, otherwise we will create partial pipelines
		// since we are not creating pipeline/stage list/task list in a single transaction.
		// We may still run into this issue when we actually create those pipeline/stage list/task list, however, that's
		// quite unlikely so we will live with it for now.
		if issueCreate.AssigneeId == api.UNKNOWN_ID {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
		}

		for _, stageCreate := range issueCreate.Pipeline.StageList {
			for _, taskCreate := range stageCreate.TaskList {
				if taskCreate.Type == api.TaskDatabaseCreate {
					if taskCreate.Statement == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
					}
					if taskCreate.DatabaseName == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
					}
					if taskCreate.CharacterSet == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, character set missing")
					}
					if taskCreate.Collation == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, collation missing")
					}
				} else if taskCreate.Type == api.TaskDatabaseSchemaUpdate {
					if taskCreate.Statement == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
					}
				} else if taskCreate.Type == api.TaskDatabaseRestore {
					if taskCreate.DatabaseName == "" {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
					}
					if taskCreate.BackupId == nil {
						return echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, backup missing")
					}
				}
			}
		}

		issue, err := s.CreateIssue(context.Background(), issueCreate, c.Get(GetPrincipalIdContextKey()).(int))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
		}

		for _, subscriberId := range issueCreate.SubscriberIdList {
			subscriberCreate := &api.IssueSubscriberCreate{
				IssueId:      issue.ID,
				SubscriberId: subscriberId,
			}
			_, err := s.IssueSubscriberService.CreateIssueSubscriber(context.Background(), subscriberCreate)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d after creating issue %d", subscriberId, issue.ID)).SetInternal(err)
			}
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
			if err := s.ComposeIssueRelationship(context.Background(), issue); err != nil {
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
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueId"))).SetInternal(err)
		}

		issue, err := s.ComposeIssueById(context.Background(), id)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
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
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueId"))).SetInternal(err)
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
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}

		updatedIssue, err := s.IssueService.PatchIssue(context.Background(), issuePatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue ID: %v", id)).SetInternal(err)
		}

		payloadList := [][]byte{}
		if issuePatch.Name != nil && *issuePatch.Name != issue.Name {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldId:   api.IssueFieldName,
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
				FieldId:   api.IssueFieldDescription,
				OldValue:  issue.Description,
				NewValue:  *issuePatch.Description,
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue description: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.AssigneeId != nil && *issuePatch.AssigneeId != issue.AssigneeId {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldId:   api.IssueFieldAssignee,
				OldValue:  strconv.Itoa(issue.AssigneeId),
				NewValue:  strconv.Itoa(*issuePatch.AssigneeId),
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue assignee: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}

		for _, payload := range payloadList {
			activityCreate := &api.ActivityCreate{
				CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
				ContainerId: issue.ID,
				Type:        api.ActivityIssueFieldUpdate,
				Level:       api.ACTIVITY_INFO,
				Payload:     string(payload),
			}
			_, err := s.ActivityManager.CreateActivity(context.Background(), activityCreate, &ActivityMeta{
				issue: issue,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating issue: %v", updatedIssue.Name)).SetInternal(err)
			}
		}

		if err := s.ComposeIssueRelationship(context.Background(), updatedIssue); err != nil {
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
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueId"))).SetInternal(err)
		}

		issueStatusPatch := &api.IssueStatusPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue status request").SetInternal(err)
		}

		issue, err := s.ComposeIssueById(context.Background(), id)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}

		updatedIssue, err := s.ChangeIssueStatus(context.Background(), issue, issueStatusPatch.Status, issueStatusPatch.UpdaterId, issueStatusPatch.Comment)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound).SetInternal(err)
			} else if common.ErrorCode(err) == common.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		if err := s.ComposeIssueRelationship(context.Background(), updatedIssue); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeIssueById(ctx context.Context, id int) (*api.Issue, error) {
	issueFind := &api.IssueFind{
		ID: &id,
	}
	issue, err := s.IssueService.FindIssue(context.Background(), issueFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeIssueRelationship(ctx, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

func (s *Server) ComposeIssueRelationship(ctx context.Context, issue *api.Issue) error {
	var err error

	issue.Creator, err = s.ComposePrincipalById(context.Background(), issue.CreatorId)
	if err != nil {
		return err
	}

	issue.Updater, err = s.ComposePrincipalById(context.Background(), issue.UpdaterId)
	if err != nil {
		return err
	}

	issue.Assignee, err = s.ComposePrincipalById(context.Background(), issue.AssigneeId)
	if err != nil {
		return err
	}

	issueSubscriberFind := &api.IssueSubscriberFind{
		IssueId: &issue.ID,
	}
	list, err := s.IssueSubscriberService.FindIssueSubscriberList(context.Background(), issueSubscriberFind)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber list for issue %d", issue.ID)).SetInternal(err)
	}

	issue.SubscriberIdList = []int{}
	for _, subscriber := range list {
		issue.SubscriberIdList = append(issue.SubscriberIdList, subscriber.SubscriberId)
	}

	issue.Project, err = s.ComposeProjectlById(context.Background(), issue.ProjectId)
	if err != nil {
		return err
	}

	issue.Pipeline, err = s.ComposePipelineById(context.Background(), issue.PipelineId)
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
				payload.ProjectId = issueCreate.ProjectId
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
				payload := api.TaskDatabaseRestorePayload{}
				payload.DatabaseName = taskCreate.DatabaseName
				payload.BackupId = *taskCreate.BackupId
				bytes, err := json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("failed to create restore database task, unable to marshal payload %w", err)
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

	bytes, err := json.Marshal(api.ActivityIssueCreatePayload{
		IssueName: issue.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after creating the issue: %v. Error %w", issue.Name, err)
	}
	activityCreate := &api.ActivityCreate{
		CreatorId:   creatorId,
		ContainerId: issue.ID,
		Type:        api.ActivityIssueCreate,
		Level:       api.ACTIVITY_INFO,
		Payload:     string(bytes),
	}
	_, err = s.ActivityManager.CreateActivity(context.Background(), activityCreate, &ActivityMeta{
		issue: issue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after creating the issue: %v. Error %w", issue.Name, err)
	}

	if err := s.ComposeIssueRelationship(context.Background(), issue); err != nil {
		return nil, err
	}

	if err := s.ScheduleNextTaskIfNeeded(context.Background(), issue.Pipeline); err != nil {
		return nil, fmt.Errorf("failed to schedule task after creating the issue: %v. Error %w", issue.Name, err)
	}

	return issue, nil
}

func (s *Server) ChangeIssueStatus(ctx context.Context, issue *api.Issue, newStatus api.IssueStatus, updaterId int, comment string) (*api.Issue, error) {
	var pipelineStatus api.PipelineStatus
	switch newStatus {
	case api.Issue_Open:
		pipelineStatus = api.Pipeline_Open
	case api.Issue_Done:
		// Returns error if any of the tasks is not DONE.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status != api.TaskDone {
					return nil, &common.Error{Code: common.ECONFLICT, Message: fmt.Sprintf("failed to resolve issue: %v, task %v has not finished", issue.Name, task.Name)}
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
					if _, err := s.ChangeTaskStatus(context.Background(), task, api.TaskCanceled, updaterId); err != nil {
						return nil, fmt.Errorf("failed to cancel issue: %v, failed to cancel task: %v, error: %w", issue.Name, task.Name, err)
					}
				}
			}
		}
		pipelineStatus = api.Pipeline_Canceled
	}

	pipelinePatch := &api.PipelinePatch{
		ID:        issue.PipelineId,
		UpdaterId: updaterId,
		Status:    &pipelineStatus,
	}
	if _, err := s.PipelineService.PatchPipeline(context.Background(), pipelinePatch); err != nil {
		return nil, fmt.Errorf("failed to update issue status: %v, failed to update pipeline status: %w", issue.Name, err)
	}

	issuePatch := &api.IssuePatch{
		ID:        issue.ID,
		UpdaterId: updaterId,
		Status:    &newStatus,
	}
	updatedIssue, err := s.IssueService.PatchIssue(context.Background(), issuePatch)
	if err != nil {
		if common.ErrorCode(err) == common.ENOTFOUND {
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
		CreatorId:   updaterId,
		ContainerId: issue.ID,
		Type:        api.ActivityIssueStatusUpdate,
		Level:       api.ACTIVITY_INFO,
		Comment:     comment,
		Payload:     string(payload),
	}

	_, err = s.ActivityManager.CreateActivity(context.Background(), activityCreate, &ActivityMeta{
		issue: updatedIssue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after changing the issue status: %v, error: %w", issue.Name, err)
	}

	return updatedIssue, nil
}

func (s *Server) PostInboxIssueActivity(ctx context.Context, issue *api.Issue, activity_id int) error {
	if issue.CreatorId != api.SYSTEM_BOT_ID {
		inboxCreate := &api.InboxCreate{
			ReceiverId: issue.CreatorId,
			ActivityId: activity_id,
		}
		_, err := s.InboxService.CreateInbox(context.Background(), inboxCreate)
		if err != nil {
			return fmt.Errorf("failed to post activity to creator inbox: %d, error: %w", issue.CreatorId, err)
		}
	}

	if issue.AssigneeId != api.SYSTEM_BOT_ID && issue.AssigneeId != issue.CreatorId {
		inboxCreate := &api.InboxCreate{
			ReceiverId: issue.AssigneeId,
			ActivityId: activity_id,
		}
		_, err := s.InboxService.CreateInbox(context.Background(), inboxCreate)
		if err != nil {
			return fmt.Errorf("failed to post activity to assignee inbox: %d, error: %w", issue.AssigneeId, err)
		}
	}

	for _, subscriberId := range issue.SubscriberIdList {
		if subscriberId != api.SYSTEM_BOT_ID && subscriberId != issue.CreatorId && subscriberId != issue.AssigneeId {
			inboxCreate := &api.InboxCreate{
				ReceiverId: subscriberId,
				ActivityId: activity_id,
			}
			_, err := s.InboxService.CreateInbox(context.Background(), inboxCreate)
			if err != nil {
				return fmt.Errorf("failed to post activity to subscriber inbox: %d, error: %w", subscriberId, err)
			}
		}
	}

	return nil
}
