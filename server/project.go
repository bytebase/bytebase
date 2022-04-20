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

	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/google/jsonapi"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerProjectRoutes(g *echo.Group) {
	g.POST("/project", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectCreate := &api.ProjectCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project request").SetInternal(err)
		}
		if projectCreate.TenantMode == api.TenantModeTenant && !s.feature(api.FeatureMultiTenancy) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
		projectCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		if projectCreate.TenantMode == "" {
			projectCreate.TenantMode = api.TenantModeDisabled
		}
		if err := api.ValidateProjectDBNameTemplate(projectCreate.DBNameTemplate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted create project request: %s", err.Error()))
		}
		if projectCreate.TenantMode != api.TenantModeTenant && projectCreate.DBNameTemplate != "" {
			return echo.NewHTTPError(http.StatusBadRequest, "database name template can only be set for tenant mode project")
		}
		projectRaw, err := s.ProjectService.CreateProject(ctx, projectCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project name already exists: %s", projectCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project").SetInternal(err)
		}

		projectMember := &api.ProjectMemberCreate{
			CreatorID:   projectCreate.CreatorID,
			ProjectID:   projectRaw.ID,
			Role:        common.ProjectOwner,
			PrincipalID: projectCreate.CreatorID,
		}

		if _, err = s.ProjectMemberService.CreateProjectMember(ctx, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add owner after creating project").SetInternal(err)
		}

		project, err := s.composeProjectRelationship(ctx, projectRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch relationship after creating project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create project response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project", func(c echo.Context) error {
		ctx := c.Request().Context()
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
		projectRawList, err := s.ProjectService.FindProjectList(ctx, projectFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch project list").SetInternal(err)
		}

		var activeProjectList []*api.Project
		var projectList []*api.Project
		for _, projectRaw := range projectRawList {
			project, err := s.composeProjectRelationship(ctx, projectRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project relationship: %v", project.Name)).SetInternal(err)
			}
			projectList = append(projectList, project)
			// We will filter those project with the current principle as an inactive member (the role provider differs from that of the project)
			// TODO(dragonly): move this if-branch out of the for loop to optimize access pattern
			if projectFind.PrincipalID != nil {
				principalID := *projectFind.PrincipalID
				roleProvider := project.RoleProvider
				for _, projectMember := range project.ProjectMemberList {
					if projectMember.PrincipalID == principalID && projectMember.RoleProvider == roleProvider {
						activeProjectList = append(activeProjectList, project)
						break
					}
				}
			}
		}

		// if principalID is not passed, we will disable the filter logic
		if projectFind.PrincipalID == nil {
			activeProjectList = projectList
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activeProjectList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal project list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:projectID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		project, err := s.composeProjectByID(ctx, id)
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
		ctx := c.Request().Context()
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

		projectRaw, err := s.ProjectService.PatchProject(ctx, projectPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch project ID: %v", id)).SetInternal(err)
		}

		project, err := s.composeProjectRelationship(ctx, projectRaw)
		if err != nil {
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
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		repositoryCreate := &api.RepositoryCreate{
			ProjectID: projectID,
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, repositoryCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create linked repository request").SetInternal(err)
		}

		project, err := s.composeProjectByID(ctx, projectID)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", projectID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}

		if err := api.ValidateRepositoryFilePathTemplate(repositoryCreate.FilePathTemplate, project.TenantMode); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted create linked repository request: %s", err.Error()))
		}

		if err := api.ValidateRepositorySchemaPathTemplate(repositoryCreate.SchemaPathTemplate, project.TenantMode); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted create linked repository request: %s", err.Error()))
		}

		vcs, err := s.store.GetVCSByID(ctx, repositoryCreate.VCSID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for creating repository: %d", repositoryCreate.VCSID)).SetInternal(err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS not found with ID: %d", repositoryCreate.VCSID))
		}

		repositoryCreate.WebhookURLHost = fmt.Sprintf("%s:%d", s.host, s.port)
		repositoryCreate.WebhookEndpointID = uuid.New().String()
		repositoryCreate.WebhookSecretToken = common.RandomString(gitlab.SecretTokenLength)

		// Create webhook and retrieve the created webhook id
		var webhookCreatePayload []byte
		switch vcs.Type {
		case "GITLAB_SELF_HOST":
			webhookPost := gitlab.WebhookPost{
				URL:                    fmt.Sprintf("%s:%d/%s/%s", s.host, s.port, gitLabWebhookPath, repositoryCreate.WebhookEndpointID),
				SecretToken:            repositoryCreate.WebhookSecretToken,
				PushEvents:             true,
				PushEventsBranchFilter: repositoryCreate.BranchFilter,
				EnableSSLVerification:  false,
			}
			webhookCreatePayload, err = json.Marshal(webhookPost)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal post request for creating webhook for project ID: %v", repositoryCreate.ProjectID)).SetInternal(err)
			}
		}

		webhookID, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).CreateWebhook(
			ctx,
			common.OauthContext{
				AccessToken: repositoryCreate.AccessToken,
				// We use s.refreshTokenNoop() because the repository isn't created yet.
				Refresher: s.refreshTokenNoop(),
			},
			vcs.InstanceURL,
			repositoryCreate.ExternalID,
			webhookCreatePayload,
		)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create webhook for project ID: %v", repositoryCreate.ProjectID)).SetInternal(err)
		}
		repositoryCreate.ExternalWebhookID = webhookID

		// Remove enclosing /
		repositoryCreate.BaseDirectory = strings.Trim(repositoryCreate.BaseDirectory, "/")
		repoRaw, err := s.RepositoryService.CreateRepository(ctx, repositoryCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project %d has already linked repository", repositoryCreate.ProjectID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to link project repository").SetInternal(err)
		}

		repo, err := s.composeRepositoryRelationship(ctx, repoRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, repo); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal link project repository response").SetInternal(err)
		}
		return nil
	})

	// Requires a separate API to return the repository, we do this because
	// 1. repository also contains project, which would cause circular dependency when composing it.
	// 2. repository info is only needed when fetching a particular project by id, thus it's unnecessary to include it in the project list response.
	g.GET("/project/:projectID/repository", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		repoFind := &api.RepositoryFind{
			ProjectID: &projectID,
		}
		repoRawList, err := s.RepositoryService.FindRepositoryList(ctx, repoFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(repoRawList) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(repoRawList), projectID)).SetInternal(err)
		}

		var repoList []*api.Repository
		for _, repoRaw := range repoRawList {
			repo, err := s.composeRepositoryRelationship(ctx, repoRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository relationship: %v", repoRaw.Name)).SetInternal(err)
			}
			repoList = append(repoList, repo)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, repoList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project repository response: %v", projectID)).SetInternal(err)
		}
		return nil
	})

	// When we unlink the repository with the project, we will also change the project workflow type to UI
	g.PATCH("/project/:projectID/repository", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		repoPatch := &api.RepositoryPatch{
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, repoPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch linked repository request").SetInternal(err)
		}
		project, err := s.composeProjectByID(ctx, projectID)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", projectID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}

		if repoPatch.FilePathTemplate != nil {
			if err := api.ValidateRepositoryFilePathTemplate(*repoPatch.FilePathTemplate, project.TenantMode); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted patch linked repository request: %s", err.Error()))
			}
		}

		if repoPatch.SchemaPathTemplate != nil {
			if err := api.ValidateRepositorySchemaPathTemplate(*repoPatch.SchemaPathTemplate, project.TenantMode); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformatted create linked repository request: %s", err.Error()))
			}
		}

		// Remove enclosing /
		if repoPatch.BaseDirectory != nil {
			baseDir := strings.Trim(*repoPatch.BaseDirectory, "/")
			repoPatch.BaseDirectory = &baseDir
		}

		repoFind := &api.RepositoryFind{
			ProjectID: &projectID,
		}
		repoRawList, err := s.RepositoryService.FindRepositoryList(ctx, repoFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(repoRawList) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(repoRawList), projectID)).SetInternal(err)
		} else if len(repoRawList) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectID))
		}

		repo := repoRawList[0]
		repoPatch.ID = repo.ID
		updatedRepoRaw, err := s.RepositoryService.PatchRepository(ctx, repoPatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update repository for project ID: %d", projectID)).SetInternal(err)
		}

		if repoPatch.BranchFilter != nil {
			vcs, err := s.store.GetVCSByID(ctx, repo.VCSID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update repository for project ID: %d", projectID)).SetInternal(err)
			}
			if vcs == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS not found with ID: %d", repo.VCSID))
			}
			// Update the webhook after we successfully update the repository.
			// This is because in case the webhook update fails, we can still have a reconcile process to reconcile the webhook state.
			// If we update it before we update the repository, then if the repository update fails, then the reconcile process will reconcile the webhook to the pre-update state which is likely not intended.
			var webhookPatchPayload []byte
			switch vcs.Type {
			case "GITLAB_SELF_HOST":
				webhookPut := gitlab.WebhookPut{
					URL:                    fmt.Sprintf("%s:%d/%s/%s", s.host, s.port, gitLabWebhookPath, updatedRepoRaw.WebhookEndpointID),
					PushEventsBranchFilter: *repoPatch.BranchFilter,
				}
				webhookPatchPayload, err = json.Marshal(webhookPut)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal put request for updating webhook %s for project ID: %v", repo.ExternalWebhookID, projectID)).SetInternal(err)
				}
			}

			err = vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).PatchWebhook(
				ctx,
				common.OauthContext{
					// Need to get ApplicationID, Secret from vcs instead of repository.vcs since the latter is not composed.
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    s.refreshToken(ctx, repo.ID),
				},
				vcs.InstanceURL,
				repo.ExternalID,
				repo.ExternalWebhookID,
				webhookPatchPayload,
			)

			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update webhook ID %s for project ID: %v", repo.ExternalWebhookID, projectID)).SetInternal(err)
			}
		}

		updatedRepo, err := s.composeRepositoryRelationship(ctx, updatedRepoRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to updating repository for project").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedRepo); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project repository response: %v", projectID)).SetInternal(err)
		}
		return nil
	})

	// When we unlink the repository with the project, we will also change the project workflow type to UI
	g.DELETE("/project/:projectID/repository", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			ProjectID: &projectID,
		}
		repoRawList, err := s.RepositoryService.FindRepositoryList(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(repoRawList) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(repoRawList), projectID)).SetInternal(err)
		} else if len(repoRawList) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectID))
		}

		repo := repoRawList[0]
		vcs, err := s.store.GetVCSByID(ctx, repo.VCSID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectID)).SetInternal(err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS not found with ID: %d", repo.VCSID))
		}

		repositoryDelete := &api.RepositoryDelete{
			ProjectID: projectID,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.RepositoryService.DeleteRepository(ctx, repositoryDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectID)).SetInternal(err)
		}

		// Delete the webhook after we successfully delete the repository.
		// This is because in case the webhook deletion fails, we can still have a cleanup process to cleanup the orphaned webhook.
		// If we delete it before we delete the repository, then if the repository deletion fails, we will have a broken repository with no webhook.
		err = vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).DeleteWebhook(
			ctx,
			// Need to get ApplicationID, Secret from vcs instead of repository.vcs since the latter is not composed.
			common.OauthContext{
				ClientID:     vcs.ApplicationID,
				ClientSecret: vcs.Secret,
				AccessToken:  repo.AccessToken,
				RefreshToken: repo.RefreshToken,
				Refresher:    s.refreshToken(ctx, repo.ID),
			},
			vcs.InstanceURL,
			repo.ExternalID,
			repo.ExternalWebhookID,
		)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete webhook ID %s for project ID: %v", repo.ExternalWebhookID, projectID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.PATCH("/project/:id/deployment", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		deploymentConfigUpsert := &api.DeploymentConfigUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, deploymentConfigUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set deployment configuration request").SetInternal(err)
		}
		deploymentConfigUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if _, err := s.composeProjectByID(ctx, id); err != nil {
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
		if err := s.composeDeploymentConfigRelationship(ctx, deploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose deployment configuration relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, deploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal set deployment configuration response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:id/deployment", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		if _, err := s.composeProjectByID(ctx, id); err != nil {
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

		// We should return empty deployment config when it doesn't exist.
		if deploymentConfig == nil {
			deploymentConfig = &api.DeploymentConfig{}
		} else {
			if err := s.composeDeploymentConfigRelationship(ctx, deploymentConfig); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose deployment configuration relationship").SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, deploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get deployment configuration response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeProjectByID(ctx context.Context, id int) (*api.Project, error) {
	projectFind := &api.ProjectFind{
		ID: &id,
	}
	projectRaw, err := s.ProjectService.FindProject(ctx, projectFind)
	if err != nil {
		return nil, err
	}
	if projectRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project not found with ID %d", id)}
	}

	project, err := s.composeProjectRelationship(ctx, projectRaw)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Server) composeProjectRelationship(ctx context.Context, raw *api.ProjectRaw) (*api.Project, error) {
	project := raw.ToProject()

	creator, err := s.store.GetPrincipalByID(ctx, project.CreatorID)
	if err != nil {
		return nil, err
	}
	project.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, project.UpdaterID)
	if err != nil {
		return nil, err
	}
	project.Updater = updater

	memberList, err := s.composeProjectMemberListByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	project.ProjectMemberList = memberList

	return project, nil
}

func (s *Server) composeDeploymentConfigRelationship(ctx context.Context, deploymentConfig *api.DeploymentConfig) error {
	var err error
	deploymentConfig.Creator, err = s.store.GetPrincipalByID(ctx, deploymentConfig.CreatorID)
	if err != nil {
		return err
	}
	deploymentConfig.Updater, err = s.store.GetPrincipalByID(ctx, deploymentConfig.UpdaterID)
	if err != nil {
		return err
	}
	deploymentConfig.Project, err = s.composeProjectByID(ctx, deploymentConfig.ProjectID)
	if err != nil {
		return err
	}
	return nil
}

// refreshToken is a token refresher that stores the latest access token configuration to repository.
func (s *Server) refreshToken(ctx context.Context, repositoryID int) common.TokenRefresher {
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
func (s *Server) refreshTokenNoop() common.TokenRefresher {
	return func(newToken, newRefreshToken string, expiresTs int64) error {
		return nil
	}
}
