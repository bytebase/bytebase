package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/external/gitlab"
	"github.com/google/jsonapi"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerProjectRoutes(g *echo.Group) {
	g.POST("/project", func(c echo.Context) error {
		ctx := context.Background()
		projectCreate := &api.ProjectCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project request").SetInternal(err)
		}
		projectCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		project, err := s.ProjectService.CreateProject(ctx, projectCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project name already exists: %s", projectCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project").SetInternal(err)
		}

		projectMember := &api.ProjectMemberCreate{
			CreatorID:   projectCreate.CreatorID,
			ProjectID:   project.ID,
			Role:        api.ProjectOwner,
			PrincipalID: projectCreate.CreatorID,
		}

		_, err = s.ProjectMemberService.CreateProjectMember(ctx, projectMember)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add owner after creating project").SetInternal(err)
		}

		if err := s.composeProjectRelationship(ctx, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch relationship after creating project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create project response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project", func(c echo.Context) error {
		ctx := context.Background()
		projectFind := &api.ProjectFind{}
		if userIDStr := c.QueryParam("user"); userIDStr != "" {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", userIDStr)).SetInternal(err)
			}
			projectFind.PrincipalID = &userID
		}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			projectFind.RowStatus = &rowStatus
		}
		list, err := s.ProjectService.FindProjectList(ctx, projectFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch project list").SetInternal(err)
		}

		for _, project := range list {
			if err := s.composeProjectRelationship(ctx, project); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project relationship: %v", project.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal project list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:projectID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		project, err := s.composeProjectlByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
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

	g.PATCH("/project/:projectID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		projectPatch := &api.ProjectPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch project request").SetInternal(err)
		}

		project, err := s.ProjectService.PatchProject(ctx, projectPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch project ID: %v", id)).SetInternal(err)
		}

		if err := s.composeProjectRelationship(ctx, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated project relationship: %v", project.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	// When we link the repository with the project, we will also change the project workflow type to VCS
	g.POST("/project/:projectID/repository", func(c echo.Context) error {
		ctx := context.Background()
		repositoryCreate := &api.RepositoryCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, repositoryCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create linked repository request").SetInternal(err)
		}

		if err := validateRepositoryFilePathTemplate(repositoryCreate.FilePathTemplate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted create linked repository request: %s", err.Error()))
		}

		if err := validateRepositorySchemaPathTemplate(repositoryCreate.SchemaPathTemplate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted create linked repository request: %s", err.Error()))
		}

		vcsFind := &api.VCSFind{
			ID: &repositoryCreate.VCSID,
		}
		vcs, err := s.VCSService.FindVCS(ctx, vcsFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", repositoryCreate.VCSID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for creating repository: %d", repositoryCreate.VCSID)).SetInternal(err)
		}

		repositoryCreate.WebhookURLHost = fmt.Sprintf("%s:%d", s.host, s.port)
		repositoryCreate.WebhookEndpointID = uuid.New().String()
		repositoryCreate.WebhookSecretToken = common.RandomString(gitlab.SecretTokenLength)
		switch vcs.Type {
		case "GITLAB_SELF_HOST":
			webhookPost := gitlab.WebhookPost{
				URL:                    fmt.Sprintf("%s:%d/%s/%s", s.host, s.port, gitLabWebhookPath, repositoryCreate.WebhookEndpointID),
				SecretToken:            repositoryCreate.WebhookSecretToken,
				PushEvents:             true,
				PushEventsBranchFilter: repositoryCreate.BranchFilter,
				EnableSSLVerification:  false,
			}
			body, err := json.Marshal(webhookPost)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal post request for creating webhook for project ID: %v", repositoryCreate.ProjectID)).SetInternal(err)
			}
			resourcePath := fmt.Sprintf("projects/%s/hooks", repositoryCreate.ExternalID)
			// We use s.refreshTokenNoop() because the repository isn't created yet.
			resp, err := gitlab.POST(vcs.InstanceURL, resourcePath, &repositoryCreate.AccessToken, bytes.NewBuffer(body), gitlab.OauthContext{}, s.refreshTokenNoop())
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create webhook for project ID: %v", repositoryCreate.ProjectID)).SetInternal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 300 {
				reason := fmt.Sprintf(
					"Failed to create webhook for project ID: %d, status code: %d",
					repositoryCreate.ProjectID,
					resp.StatusCode,
				)
				// Add helper tips if the status code is 422, refer to bytebase#101 for more context.
				if resp.StatusCode == http.StatusUnprocessableEntity {
					reason += ".\n\nIf GitLab and Bytebase are in the same private network, " +
						"please follow the instructions in https://docs.gitlab.com/ee/security/webhooks.html"
				}
				return echo.NewHTTPError(http.StatusInternalServerError, reason)
			}

			webhookInfo := &gitlab.WebhookInfo{}
			if err := json.NewDecoder(resp.Body).Decode(webhookInfo); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to unmarshal create webhook response for project ID: %v", repositoryCreate.ProjectID)).SetInternal(err)
			}
			repositoryCreate.ExternalWebhookID = strconv.Itoa(webhookInfo.ID)
		}

		repositoryCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		// Remove enclosing /
		repositoryCreate.BaseDirectory = strings.Trim(repositoryCreate.BaseDirectory, "/")
		repository, err := s.RepositoryService.CreateRepository(ctx, repositoryCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project %d has already linked repository", repositoryCreate.ProjectID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to link project repository").SetInternal(err)
		}

		if err := s.composeRepositoryRelationship(ctx, repository); err != nil {
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
	g.GET("/project/:projectID/repository", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			ProjectID: &projectID,
		}
		list, err := s.RepositoryService.FindRepositoryList(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(list) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(list), projectID)).SetInternal(err)
		}

		for _, repository := range list {
			if err := s.composeRepositoryRelationship(ctx, repository); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository relationship: %v", repository.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project repository response: %v", projectID)).SetInternal(err)
		}
		return nil
	})

	// When we unlink the repository with the project, we will also change the project workflow type to UI
	g.PATCH("/project/:projectID/repository", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		repositoryPatch := &api.RepositoryPatch{
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, repositoryPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch linked repository request").SetInternal(err)
		}

		if repositoryPatch.FilePathTemplate != nil {
			if err := validateRepositoryFilePathTemplate(*repositoryPatch.FilePathTemplate); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted patch linked repository request: %s", err.Error()))
			}
		}

		if repositoryPatch.SchemaPathTemplate != nil {
			if err := validateRepositorySchemaPathTemplate(*repositoryPatch.SchemaPathTemplate); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted create linked repository request: %s", err.Error()))
			}
		}

		// Remove enclosing /
		if repositoryPatch.BaseDirectory != nil {
			baseDir := strings.Trim(*repositoryPatch.BaseDirectory, "/")
			repositoryPatch.BaseDirectory = &baseDir
		}

		repositoryFind := &api.RepositoryFind{
			ProjectID: &projectID,
		}
		list, err := s.RepositoryService.FindRepositoryList(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(list) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(list), projectID)).SetInternal(err)
		} else if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectID))
		}

		repository := list[0]
		repositoryPatch.ID = repository.ID
		updatedRepository, err := s.RepositoryService.PatchRepository(ctx, repositoryPatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update repository for project ID: %d", projectID)).SetInternal(err)
		}

		if repositoryPatch.BranchFilter != nil {
			vcsFind := &api.VCSFind{
				ID: &repository.VCSID,
			}
			vcs, err := s.VCSService.FindVCS(ctx, vcsFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update repository for project ID: %d", projectID)).SetInternal(err)
			}
			// Updates the webhook after we successfully update the repository.
			// This is because in case the webhook update fails, we can still have a reconcile process to reconcile the webhook state.
			// If we update it before we update the repository, then if the repository update fails, then the reconcile process will reconcile the webhook to the pre-update state which is likely not intended.
			switch vcs.Type {
			case "GITLAB_SELF_HOST":
				webhookPut := gitlab.WebhookPut{
					URL:                    fmt.Sprintf("%s:%d/%s/%s", s.host, s.port, gitLabWebhookPath, updatedRepository.WebhookEndpointID),
					PushEventsBranchFilter: *repositoryPatch.BranchFilter,
				}
				json, err := json.Marshal(webhookPut)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal put request for updating webhook %s for project ID: %v", repository.ExternalWebhookID, projectID)).SetInternal(err)
				}
				resourcePath := fmt.Sprintf("projects/%s/hooks/%s", repository.ExternalID, repository.ExternalWebhookID)
				resp, err := gitlab.PUT(
					vcs.InstanceURL,
					resourcePath,
					&repository.AccessToken,
					bytes.NewBuffer(json),
					gitlab.OauthContext{
						ClientID:     repository.VCS.ApplicationID,
						ClientSecret: repository.VCS.Secret,
						RefreshToken: repository.RefreshToken,
					},
					s.refreshToken(ctx, repository.ID),
				)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update webhook ID %s for project ID: %v", repository.ExternalWebhookID, projectID)).SetInternal(err)
				}
				defer resp.Body.Close()

				// Just emits a warning since we have already updated the repository entry. We will have a separate process to reconcile the state.
				if resp.StatusCode >= 300 {
					s.l.Error(("Failed to update gitlab webhook when updating repository for project"),
						zap.Int("status_code", resp.StatusCode),
						zap.String("status", resp.Status),
						zap.Int("project_id", projectID),
						zap.Int("repository_id", repository.ID),
						zap.String("resource_path", resourcePath),
						zap.String("body", string(json)),
					)
				}
			}
		}

		if err := s.composeRepositoryRelationship(ctx, updatedRepository); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to updating repository for project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedRepository); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project repository response: %v", projectID)).SetInternal(err)
		}
		return nil
	})

	// When we unlink the repository with the project, we will also change the project workflow type to UI
	g.DELETE("/project/:projectID/repository", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			ProjectID: &projectID,
		}
		list, err := s.RepositoryService.FindRepositoryList(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(list) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(list), projectID)).SetInternal(err)
		} else if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectID))
		}

		repository := list[0]
		vcsFind := &api.VCSFind{
			ID: &repository.VCSID,
		}
		vcs, err := s.VCSService.FindVCS(ctx, vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectID)).SetInternal(err)
		}

		repositoryDelete := &api.RepositoryDelete{
			ProjectID: projectID,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.RepositoryService.DeleteRepository(ctx, repositoryDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectID)).SetInternal(err)
		}

		// Deletes the webhook after we successfully delete the repository.
		// This is because in case the webhook deletion fails, we can still have a cleanup process to cleanup the orphaned webhook.
		// If we delete it before we delete the repository, then if the repository deletion fails, we will have a broken repository with no webhook.
		switch vcs.Type {
		case "GITLAB_SELF_HOST":
			resp, err := gitlab.DELETE(
				vcs.InstanceURL,
				fmt.Sprintf("projects/%s/hooks/%s",
					repository.ExternalID,
					repository.ExternalWebhookID),
				&repository.AccessToken,
				gitlab.OauthContext{
					ClientID:     repository.VCS.ApplicationID,
					ClientSecret: repository.VCS.Secret,
					RefreshToken: repository.RefreshToken,
				},
				s.refreshToken(ctx, repository.ID),
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete webhook ID %s for project ID: %v", repository.ExternalWebhookID, projectID)).SetInternal(err)
			}
			defer resp.Body.Close()

			// Just emits a warning since we have already removed the repository entry. We will have a separate process to cleanup the orphaned webhook.
			if resp.StatusCode >= 300 {
				s.l.Error(("Failed to delete gitlab webhook when unlinking repository from project"),
					zap.Int("status_code", resp.StatusCode),
					zap.Int("project_id", projectID),
					zap.Int("repository_id", repository.ID),
					zap.String("gitlab_project_id", repository.ExternalID),
					zap.String("gitlab_webhook_id", repository.ExternalWebhookID))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.PATCH("/project/:id/deployment", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		deploymentConfigUpsert := &api.DeploymentConfigUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, deploymentConfigUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set deployment configuration request").SetInternal(err)
		}
		deploymentConfigUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if _, err := s.composeProjectlByID(ctx, id); err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", id)).SetInternal(err)
		}
		deploymentConfigUpsert.ProjectID = id

		deploymentConfig, err := s.DeploymentConfigService.UpsertDeploymentConfig(ctx, deploymentConfigUpsert)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set deployment configuration").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, deploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal set deployment configuration response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:id/deployment", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		if _, err := s.composeProjectlByID(ctx, id); err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", id)).SetInternal(err)
		}

		deploymentConfigFind := &api.DeploymentConfigFind{
			ProjectID: &id,
		}
		deploymentConfig, err := s.DeploymentConfigService.FindDeploymentConfig(ctx, deploymentConfigFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get deployment configuration for project id: %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, deploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get deployment configuration response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeProjectlByID(ctx context.Context, id int) (*api.Project, error) {
	projectFind := &api.ProjectFind{
		ID: &id,
	}
	project, err := s.ProjectService.FindProject(ctx, projectFind)
	if err != nil {
		return nil, err
	}

	if err := s.composeProjectRelationship(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Server) composeProjectRelationship(ctx context.Context, project *api.Project) error {
	var err error

	project.Creator, err = s.composePrincipalByID(ctx, project.CreatorID)
	if err != nil {
		return err
	}

	project.Updater, err = s.composePrincipalByID(ctx, project.UpdaterID)
	if err != nil {
		return err
	}

	project.ProjectMemberList, err = s.composeProjectMemberListByProjectID(ctx, project.ID)
	if err != nil {
		return err
	}

	return nil
}

func validateRepositoryFilePathTemplate(filePathTemplate string) error {
	if !strings.Contains(filePathTemplate, "{{VERSION}}") {
		return fmt.Errorf("missing {{VERSION}} in file path template")
	}
	if !strings.Contains(filePathTemplate, "{{DB_NAME}}") {
		return fmt.Errorf("missing {{DB_NAME}} in file path template")
	}
	if !strings.Contains(filePathTemplate, "{{TYPE}}") {
		return fmt.Errorf("missing {{TYPE}} in file path template")
	}
	return nil
}

func validateRepositorySchemaPathTemplate(schemaPathTemplate string) error {
	if schemaPathTemplate == "" {
		return nil
	}
	if !strings.Contains(schemaPathTemplate, "{{DB_NAME}}") {
		return fmt.Errorf("missing {{DB_NAME}} in schema path template")
	}
	return nil
}

// refreshToken is a token refresher that stores the latest access token configuration to repository.
func (s *Server) refreshToken(ctx context.Context, repositoryID int) gitlab.TokenRefresher {
	return func(token, refreshToken string, expiresTs int64) error {
		if _, err := s.RepositoryService.PatchRepository(ctx, &api.RepositoryPatch{
			ID:           repositoryID,
			UpdaterID:    api.SystemBotID,
			AccessToken:  &token,
			ExpiresTs:    &expiresTs,
			RefreshToken: &refreshToken,
		}); err != nil {
			return err
		}
		return nil
	}
}

// refreshToken is a no-op token refresher. It should be used when the repository isn't created yet.
func (s *Server) refreshTokenNoop() gitlab.TokenRefresher {
	return func(newToken, newRefreshToken string, expiresTs int64) error {
		return nil
	}
}
