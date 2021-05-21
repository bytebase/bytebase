package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		issueCreate := &api.IssueCreate{WorkspaceId: api.DEFAULT_WORKPSACE_ID}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create issue request").SetInternal(err)
		}

		issueCreate.Pipeline.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		issueCreate.Pipeline.WorkspaceId = api.DEFAULT_WORKPSACE_ID
		createdPipeline, err := s.PipelineService.CreatePipeline(context.Background(), &issueCreate.Pipeline)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create pipeline for issue").SetInternal(err)
		}

		for _, stageCreate := range issueCreate.Pipeline.StageList {
			stageCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
			stageCreate.WorkspaceId = api.DEFAULT_WORKPSACE_ID
			stageCreate.PipelineId = createdPipeline.ID
			createdStage, err := s.StageService.CreateStage(context.Background(), &stageCreate)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create stage for issue").SetInternal(err)
			}

			for _, taskCreate := range stageCreate.TaskList {
				taskCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
				taskCreate.WorkspaceId = api.DEFAULT_WORKPSACE_ID
				taskCreate.PipelineId = createdPipeline.ID
				taskCreate.StageId = createdStage.ID
				_, err := s.TaskService.CreateTask(context.Background(), &taskCreate)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create task for issue").SetInternal(err)
				}
			}
		}

		issueCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		issueCreate.PipelineId = createdPipeline.ID
		issue, err := s.IssueService.CreateIssue(context.Background(), issueCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
		}

		if err := s.ComposeIssueRelationship(context.Background(), issue, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue", func(c echo.Context) error {
		workspaceId := api.DEFAULT_WORKPSACE_ID
		issueFind := &api.IssueFind{
			WorkspaceId: &workspaceId,
		}
		projectIdStr := c.QueryParams().Get("project")
		if projectIdStr != "" {
			projectId, err := strconv.Atoi(projectIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIdStr)).SetInternal(err)
			}
			issueFind.ProjectId = &projectId
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
			return err
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
			ID:          id,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issuePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch issue request").SetInternal(err)
		}

		issue, err := s.IssueService.PatchIssue(context.Background(), issuePatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch issue ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeIssueRelationship(context.Background(), issue, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueId/status", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("issueId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		issueStatusPatch := &api.IssueStatusPatch{
			ID:          id,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue status request").SetInternal(err)
		}

		if issueStatusPatch.Status != nil {
			issue, err := s.ComposeIssueById(context.Background(), id, c.Get(getIncludeKey()).([]string))
			if err != nil {
				return err
			}

			var pipelineStatus api.PipelineStatus
			pipelinePatch := &api.PipelinePatch{
				ID:          issue.PipelineId,
				WorkspaceId: api.DEFAULT_WORKPSACE_ID,
				UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
			}
			switch api.IssueStatus(*issueStatusPatch.Status) {
			case api.Issue_Open:
				pipelineStatus = api.Pipeline_Open
			case api.Issue_Done:
				// Returns error if any of the tasks is not in the end status.
				for _, stage := range issue.Pipeline.StageList {
					for _, task := range stage.TaskList {
						if task.Status.IsEndStatus() {
							return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Failed to resolve issue: %v. Task %v is in %v status.", issue.Name, task.Name, task.Status))
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
							taskStatus := api.TaskCanceled
							taskPatch := &api.TaskPatch{
								ID:          id,
								WorkspaceId: api.DEFAULT_WORKPSACE_ID,
								UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
								Status:      &taskStatus,
							}
							if _, err := s.TaskService.PatchTask(context.Background(), taskPatch); err != nil {
								return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to cancel issue: %v. Failed to cancel task: %v.", issue.Name, task.Name)).SetInternal(err)
							}
						}
					}
				}
				pipelineStatus = api.Pipeline_Canceled
			}

			pipelinePatch.Status = &pipelineStatus
			if _, err := s.PipelineService.PatchPipeline(context.Background(), pipelinePatch); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue: %v", id)).SetInternal(err)
			}
		}

		issueStatus := api.IssueStatus(*issueStatusPatch.Status)
		issuePatch := &api.IssuePatch{
			ID:          id,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
			Status:      &issueStatus,
		}
		updatedIssue, err := s.IssueService.PatchIssue(context.Background(), issuePatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue ID: %v", id)).SetInternal(err)
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
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
			return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
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
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch creator for issue: %v", issue.Name)).SetInternal(err)
	}

	issue.Updater, err = s.ComposePrincipalById(context.Background(), issue.UpdaterId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updater for issue: %v", issue.Name)).SetInternal(err)
	}

	issue.Assignee, err = s.ComposePrincipalById(context.Background(), issue.AssigneeId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch assignee for issue: %v", issue.Name)).SetInternal(err)
	}

	issue.Project, err = s.ComposeProjectlById(context.Background(), issue.ProjectId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project for issue: %v", issue.Name)).SetInternal(err)
	}

	issue.Pipeline, err = s.ComposePipelineById(context.Background(), issue.PipelineId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch pipeline for issue: %v", issue.Name)).SetInternal(err)
	}

	return nil
}
