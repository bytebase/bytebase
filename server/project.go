package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/external/gitlab"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

func (s *Server) registerProjectRoutes(g *echo.Group) {
	g.POST("/project", func(c echo.Context) error {
		projectCreate := &api.ProjectCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project request").SetInternal(err)
		}
		projectCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		project, err := s.ProjectService.CreateProject(context.Background(), projectCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project name already exists: %s", projectCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project").SetInternal(err)
		}

		projectMember := &api.ProjectMemberCreate{
			CreatorId:   projectCreate.CreatorId,
			ProjectId:   project.ID,
			Role:        api.ProjectOwner,
			PrincipalId: projectCreate.CreatorId,
		}

		_, err = s.ProjectMemberService.CreateProjectMember(context.Background(), projectMember)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add owner after creating project").SetInternal(err)
		}

		if err := s.ComposeProjectRelationship(context.Background(), project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch relationship after creating project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create project response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project", func(c echo.Context) error {
		projectFind := &api.ProjectFind{}
		if userIdStr := c.QueryParam("user"); userIdStr != "" {
			userId, err := strconv.Atoi(userIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", userIdStr)).SetInternal(err)
			}
			projectFind.PrincipalId = &userId
		}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			projectFind.RowStatus = &rowStatus
		}
		list, err := s.ProjectService.FindProjectList(context.Background(), projectFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch project list").SetInternal(err)
		}

		for _, project := range list {
			if err := s.ComposeProjectRelationship(context.Background(), project); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project relationship: %v", project.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal project list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:projectId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		project, err := s.ComposeProjectlById(context.Background(), id)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		projectPatch := &api.ProjectPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch project request").SetInternal(err)
		}

		project, err := s.ProjectService.PatchProject(context.Background(), projectPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch project ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeProjectRelationship(context.Background(), project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated project relationship: %v", project.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	// When we link the repository with the project, we will also change the project workflow type to VCS
	g.POST("/project/:projectId/repository", func(c echo.Context) error {
		repositoryCreate := &api.RepositoryCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, repositoryCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create linked repository request").SetInternal(err)
		}

		vcsFind := &api.VCSFind{
			ID: &repositoryCreate.VCSId,
		}
		vcs, err := s.VCSService.FindVCS(context.Background(), vcsFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", repositoryCreate.VCSId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for creating repository: %d", repositoryCreate.VCSId)).SetInternal(err)
		}

		repositoryCreate.WebhookURLHost = fmt.Sprintf("%s:%d", s.host, s.port)
		repositoryCreate.WebhookEndpointId = uuid.NewV4().String()
		repositoryCreate.WebhookSecretToken = bytebase.RandomString(gitlab.SECRET_TOKEN_LENGTH)
		switch vcs.Type {
		case "GITLAB_SELF_HOST":
			webhookPost := gitlab.WebhookPost{
				URL:                    fmt.Sprintf("%s:%d/%s/%s", s.host, s.port, gitLabWebhookPath, repositoryCreate.WebhookEndpointId),
				SecretToken:            repositoryCreate.WebhookSecretToken,
				PushEvents:             true,
				PushEventsBranchFilter: repositoryCreate.BranchFilter,
				EnableSSLVerification:  false,
			}
			body, err := json.Marshal(webhookPost)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal post request for creating webhook for project ID: %v", repositoryCreate.ProjectId)).SetInternal(err)
			}
			resourcePath := fmt.Sprintf("projects/%s/hooks", repositoryCreate.ExternalId)
			resp, err := gitlab.POST(vcs.InstanceURL, resourcePath, repositoryCreate.AccessToken, bytes.NewBuffer(body))
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create webhook for project ID: %v", repositoryCreate.ProjectId)).SetInternal(err)
			}
			defer resp.Body.Close()

			// Just emits a warning since we have already updated the repository entry. We will have a separate process to reconcile the state.
			if resp.StatusCode >= 300 {
				return echo.NewHTTPError(http.StatusInternalServerError,
					fmt.Sprintf("Failed to create webhook for project ID: %v, status code: %d, status: %s",
						repositoryCreate.ProjectId,
						resp.StatusCode,
						resp.Status,
					))
			}

			webhookInfo := &gitlab.WebhookInfo{}
			if err := json.NewDecoder(resp.Body).Decode(webhookInfo); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to unmarshal create webhook response for project ID: %v", repositoryCreate.ProjectId)).SetInternal(err)
			}
			repositoryCreate.ExternalWebhookId = strconv.Itoa(webhookInfo.ID)
		}

		repositoryCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		// Remove enclosing /
		repositoryCreate.BaseDirectory = strings.Trim(repositoryCreate.BaseDirectory, "/")
		repository, err := s.RepositoryService.CreateRepository(context.Background(), repositoryCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project %d has already linked repository", repositoryCreate.ProjectId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to link project repository").SetInternal(err)
		}

		if err := s.ComposeRepositoryRelationship(context.Background(), repository); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, repository); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal link project repository response").SetInternal(err)
		}
		return nil
	})

	// Requires a separate API to return the repository, we do this because
	// 1. repository also contains project, which would cause circular dependency when composing it.
	// 2. repository info is only needed when fetching a particular project by id, thus it's unnecessary to include it in the project list response.
	g.GET("/project/:projectId/repository", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			ProjectId: &projectId,
		}
		list, err := s.RepositoryService.FindRepositoryList(context.Background(), repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectId)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(list) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(list), projectId)).SetInternal(err)
		}

		for _, repository := range list {
			if err := s.ComposeRepositoryRelationship(context.Background(), repository); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository relationship: %v", repository.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project repository response: %v", projectId)).SetInternal(err)
		}
		return nil
	})

	// When we unlink the repository with the project, we will also change the project workflow type to UI
	g.PATCH("/project/:projectId/repository", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		repositoryPatch := &api.RepositoryPatch{
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, repositoryPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch repository request").SetInternal(err)
		}
		// Remove enclosing /
		if repositoryPatch.BaseDirectory != nil {
			baseDir := strings.Trim(*repositoryPatch.BaseDirectory, "/")
			repositoryPatch.BaseDirectory = &baseDir
		}

		repositoryFind := &api.RepositoryFind{
			ProjectId: &projectId,
		}
		list, err := s.RepositoryService.FindRepositoryList(context.Background(), repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectId)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(list) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(list), projectId)).SetInternal(err)
		} else if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectId))
		}

		repository := list[0]
		repositoryPatch.ID = repository.ID
		updatedRepository, err := s.RepositoryService.PatchRepository(context.Background(), repositoryPatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update repository for project ID: %d", projectId)).SetInternal(err)
		}

		if repositoryPatch.BranchFilter != nil {
			vcsFind := &api.VCSFind{
				ID: &repository.VCSId,
			}
			vcs, err := s.VCSService.FindVCS(context.Background(), vcsFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update repository for project ID: %d", projectId)).SetInternal(err)
			}
			// Updates the webhook after we successfully update the repository.
			// This is because in case the webhook update fails, we can still have a reconcile process to reconcile the webhook state.
			// If we update it before we update the repository, then if the repository update fails, then the reconcile process will reconcile the webhook to the pre-update state which is likely not intended.
			switch vcs.Type {
			case "GITLAB_SELF_HOST":
				webhookPut := gitlab.WebhookPut{
					URL:                    fmt.Sprintf("%s:%d/%s/%s", s.host, s.port, gitLabWebhookPath, updatedRepository.WebhookEndpointId),
					PushEventsBranchFilter: *repositoryPatch.BranchFilter,
				}
				json, err := json.Marshal(webhookPut)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal put request for updating webhook %s for project ID: %v", repository.ExternalWebhookId, projectId)).SetInternal(err)
				}
				resourcePath := fmt.Sprintf("projects/%s/hooks/%s", repository.ExternalId, repository.ExternalWebhookId)
				resp, err := gitlab.PUT(vcs.InstanceURL, resourcePath, vcs.AccessToken, bytes.NewBuffer(json))
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update webhook ID %s for project ID: %v", repository.ExternalWebhookId, projectId)).SetInternal(err)
				}
				defer resp.Body.Close()

				// Just emits a warning since we have already updated the repository entry. We will have a separate process to reconcile the state.
				if resp.StatusCode >= 300 {
					s.l.Error(("Failed to update gitlab webhook when updating repository for project"),
						zap.Int("status_code", resp.StatusCode),
						zap.String("status", resp.Status),
						zap.Int("project_id", projectId),
						zap.Int("repository_id", repository.ID),
						zap.String("resource_path", resourcePath),
						zap.String("body", string(json)),
					)
				}
			}
		}

		if err := s.ComposeRepositoryRelationship(context.Background(), updatedRepository); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to updating repository for project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedRepository); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project repository response: %v", projectId)).SetInternal(err)
		}
		return nil
	})

	// When we unlink the repository with the project, we will also change the project workflow type to UI
	g.DELETE("/project/:projectId/repository", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			ProjectId: &projectId,
		}
		list, err := s.RepositoryService.FindRepositoryList(context.Background(), repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectId)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(list) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(list), projectId)).SetInternal(err)
		} else if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectId))
		}

		repository := list[0]
		vcsFind := &api.VCSFind{
			ID: &repository.VCSId,
		}
		vcs, err := s.VCSService.FindVCS(context.Background(), vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectId)).SetInternal(err)
		}

		repositoryDelete := &api.RepositoryDelete{
			ProjectId: projectId,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := s.RepositoryService.DeleteRepository(context.Background(), repositoryDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectId)).SetInternal(err)
		}

		// Deletes the webhook after we successfully delete the repository.
		// This is because in case the webhook deletion fails, we can still have a cleanup process to cleanup the orphaned webhook.
		// If we delete it before we delete the repository, then if the repository deletion fails, we will have a broken repository with no webhook.
		switch vcs.Type {
		case "GITLAB_SELF_HOST":
			resp, err := gitlab.DELETE(vcs.InstanceURL, fmt.Sprintf("projects/%s/hooks/%s", repository.ExternalId, repository.ExternalWebhookId), vcs.AccessToken)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete webhook ID %s for project ID: %v", repository.ExternalWebhookId, projectId)).SetInternal(err)
			}
			defer resp.Body.Close()

			// Just emits a warning since we have already removed the repository entry. We will have a separate process to cleanup the orphaned webhook.
			if resp.StatusCode >= 300 {
				s.l.Error(("Failed to delete gitlab webhook when unlinking repository from project"),
					zap.Int("status_code", resp.StatusCode),
					zap.Int("project_id", projectId),
					zap.Int("repository_id", repository.ID),
					zap.String("gitlab_project_id", repository.ExternalId),
					zap.String("gitlab_webhook_id", repository.ExternalWebhookId))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) ComposeProjectlById(ctx context.Context, id int) (*api.Project, error) {
	projectFind := &api.ProjectFind{
		ID: &id,
	}
	project, err := s.ProjectService.FindProject(ctx, projectFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeProjectRelationship(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Server) ComposeProjectRelationship(ctx context.Context, project *api.Project) error {
	var err error

	project.Creator, err = s.ComposePrincipalById(context.Background(), project.CreatorId)
	if err != nil {
		return err
	}

	project.Updater, err = s.ComposePrincipalById(context.Background(), project.UpdaterId)
	if err != nil {
		return err
	}

	project.ProjectMemberList, err = s.ComposeProjectMemberListByProjectId(ctx, project.ID)
	if err != nil {
		return err
	}

	return nil
}
