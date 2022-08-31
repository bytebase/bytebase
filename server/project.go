package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/github"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
)

func (s *Server) registerProjectRoutes(g *echo.Group) {
	g.POST("/project", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectCreate := &api.ProjectCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create project request").SetInternal(err)
		}
		if projectCreate.TenantMode == api.TenantModeTenant && !s.feature(api.FeatureMultiTenancy) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
		projectCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		if projectCreate.TenantMode == "" {
			projectCreate.TenantMode = api.TenantModeDisabled
		}
		if err := api.ValidateProjectDBNameTemplate(projectCreate.DBNameTemplate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed create project request: %s", err.Error()))
		}
		if projectCreate.TenantMode != api.TenantModeTenant && projectCreate.DBNameTemplate != "" {
			return echo.NewHTTPError(http.StatusBadRequest, "database name template can only be set for tenant mode project")
		}
		project, err := s.store.CreateProject(ctx, projectCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project name already exists: %s", projectCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project").SetInternal(err)
		}

		projectMember := &api.ProjectMemberCreate{
			CreatorID:   projectCreate.CreatorID,
			ProjectID:   project.ID,
			Role:        common.ProjectOwner,
			PrincipalID: projectCreate.CreatorID,
		}

		if _, err = s.store.CreateProjectMember(ctx, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add owner after creating project").SetInternal(err)
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

		// Only Owner and DBA can fetch all projects from all users.
		if projectFind.PrincipalID == nil {
			role := c.Get(getRoleContextKey()).(api.Role)
			if role != api.Owner && role != api.DBA {
				return echo.NewHTTPError(http.StatusForbidden, "Not allowed to fetch all project list")
			}
		}

		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			projectFind.RowStatus = &rowStatus
		}
		projectList, err := s.store.FindProject(ctx, projectFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch project list").SetInternal(err)
		}

		var activeProjectList []*api.Project
		// if principalID is passed, we will enable the filter logic
		if projectFind.PrincipalID != nil {
			principalID := *projectFind.PrincipalID
			for _, project := range projectList {
				// We will filter those project with the current principal as an inactive member (the role provider differs from that of the project)
				roleProvider := project.RoleProvider
				for _, projectMember := range project.ProjectMemberList {
					if projectMember.PrincipalID == principalID && projectMember.RoleProvider == roleProvider {
						activeProjectList = append(activeProjectList, project)
						break
					}
				}
			}
		} else {
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

		project, err := s.store.GetProjectByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", id)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", id))
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch project request").SetInternal(err)
		}

		// Verify before archiving the project:
		// 1. the project has no database.
		// 2. the issue status of this project should be canceled or done.
		if v := projectPatch.RowStatus; v != nil && *v == string(api.Archived) {
			databases, err := s.store.FindDatabase(ctx, &api.DatabaseFind{ProjectID: &id})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, errors.Errorf("failed to find databases in the project %d", id)).SetInternal(err)
			}
			if len(databases) > 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "Please transfer all databases under the project before archiving the project.")
			}

			issueList, err := s.store.FindIssueStripped(ctx, &api.IssueFind{ProjectID: &id, StatusList: []api.IssueStatus{api.IssueOpen}})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, errors.Errorf("failed to find issues in the project %d", id)).SetInternal(err)
			}
			if len(issueList) > 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "Please resolve all the issues in it before archiving the project.")
			}
		}

		project, err := s.store.PatchProject(ctx, projectPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch project ID: %v", id)).SetInternal(err)
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create linked repository request").SetInternal(err)
		}

		if strings.Contains(repositoryCreate.BranchFilter, "*") {
			return echo.NewHTTPError(http.StatusBadRequest, "Wildcard isn't supported for branch setting")
		}

		// We need to check the FilePathTemplate in create repository request.
		// This avoids to a certain extent that the creation succeeds but does not work.
		if err := vcsPlugin.IsAsterisksInTemplateValid(path.Join(repositoryCreate.BaseDirectory, repositoryCreate.FilePathTemplate)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, errors.Wrap(err, errors.Wrap(err, "Invalid base directory and filepath template combination").Error()))
		}

		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", projectID))
		}

		if err := api.ValidateRepositoryFilePathTemplate(repositoryCreate.FilePathTemplate, project.TenantMode); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed create linked repository request: %s", err.Error()))
		}

		if err := api.ValidateRepositorySchemaPathTemplate(repositoryCreate.SchemaPathTemplate, project.TenantMode); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed create linked repository request: %s", err.Error()))
		}

		vcs, err := s.store.GetVCSByID(ctx, repositoryCreate.VCSID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for creating repository: %d", repositoryCreate.VCSID)).SetInternal(err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS not found with ID: %d", repositoryCreate.VCSID))
		}

		repositoryCreate.WebhookURLHost = fmt.Sprintf("%s:%d", s.profile.BackendHost, s.profile.BackendPort)
		repositoryCreate.WebhookEndpointID = uuid.New().String()
		secretToken, err := common.RandomString(gitlab.SecretTokenLength)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate random secret token for GitLab").SetInternal(err)
		}
		repositoryCreate.WebhookSecretToken = secretToken

		// Create a new webhook and retrieve the created webhook ID
		var webhookCreatePayload []byte
		switch vcs.Type {
		case vcsPlugin.GitLabSelfHost:
			webhookCreate := gitlab.WebhookCreate{
				URL:                    fmt.Sprintf("%s:%d/%s/%s", s.profile.BackendHost, s.profile.BackendPort, gitlabWebhookPath, repositoryCreate.WebhookEndpointID),
				SecretToken:            repositoryCreate.WebhookSecretToken,
				PushEvents:             true,
				PushEventsBranchFilter: repositoryCreate.BranchFilter,
				EnableSSLVerification:  false, // TODO(tianzhou): This is set to false, be lax to not enable_ssl_verification
			}
			webhookCreatePayload, err = json.Marshal(webhookCreate)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal request body for creating webhook for project ID: %d", repositoryCreate.ProjectID)).SetInternal(err)
			}
		case vcsPlugin.GitHubCom:
			webhookHost := s.profile.BackendHost
			if s.profile.BackendHost == "http://localhost" {
				webhookHost = fmt.Sprintf("%s:%d", s.profile.BackendHost, s.profile.BackendPort)
			}
			webhookPost := github.WebhookCreateOrUpdate{
				Config: github.WebhookConfig{
					URL:         fmt.Sprintf("%s/%s/%s", webhookHost, githubWebhookPath, repositoryCreate.WebhookEndpointID),
					ContentType: "json",
					Secret:      repositoryCreate.WebhookSecretToken,
					InsecureSSL: 1, // TODO: Allow user to specify this value through api.RepositoryCreate
				},
				Events: []string{"push"},
			}
			webhookCreatePayload, err = json.Marshal(webhookPost)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal request body for creating webhook for project ID: %d", repositoryCreate.ProjectID)).SetInternal(err)
			}
		}

		webhookID, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).CreateWebhook(
			ctx,
			common.OauthContext{
				AccessToken: repositoryCreate.AccessToken,
				// We use refreshTokenNoop() because the repository isn't created yet.
				Refresher: refreshTokenNoop(),
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
		repository, err := s.store.CreateRepository(ctx, repositoryCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project %d has already linked repository", repositoryCreate.ProjectID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to link project repository").SetInternal(err)
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
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		repoFind := &api.RepositoryFind{
			ProjectID: &projectID,
		}
		repoList, err := s.store.FindRepository(ctx, repoFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(repoList) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(repoList), projectID)).SetInternal(err)
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch linked repository request").SetInternal(err)
		}
		if repoPatch.BranchFilter != nil && strings.Contains(*repoPatch.BranchFilter, "*") {
			return echo.NewHTTPError(http.StatusBadRequest, "Wildcard isn't supported for branch setting")
		}

		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", projectID))
		}

		if repoPatch.FilePathTemplate != nil {
			if err := api.ValidateRepositoryFilePathTemplate(*repoPatch.FilePathTemplate, project.TenantMode); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed patch linked repository request: %s", err.Error()))
			}
		}

		if repoPatch.SchemaPathTemplate != nil {
			if err := api.ValidateRepositorySchemaPathTemplate(*repoPatch.SchemaPathTemplate, project.TenantMode); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed create linked repository request: %s", err.Error()))
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
		repoList, err := s.store.FindRepository(ctx, repoFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(repoList) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(repoList), projectID)).SetInternal(err)
		} else if len(repoList) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectID))
		}

		repo := repoList[0]
		repoPatch.ID = repo.ID

		// We need to check the FilePathTemplate in create repository request.
		// This avoids to a certain extent that the creation succeeds but does not work.
		newBaseDirectory, newFilePathTemplate := repo.BaseDirectory, repo.FilePathTemplate
		if repoPatch.BaseDirectory != nil {
			newBaseDirectory = *repoPatch.BaseDirectory
		}
		if repoPatch.FilePathTemplate != nil {
			newFilePathTemplate = *repoPatch.FilePathTemplate
		}

		if err := vcsPlugin.IsAsterisksInTemplateValid(path.Join(newBaseDirectory, newFilePathTemplate)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, errors.Wrap(err, "Invalid base directory and filepath template combination").Error())
		}

		updatedRepo, err := s.store.PatchRepository(ctx, repoPatch)
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
			var webhookUpdatePayload []byte
			switch vcs.Type {
			case vcsPlugin.GitLabSelfHost:
				webhookUpdate := gitlab.WebhookUpdate{
					URL:                    fmt.Sprintf("%s:%d/%s/%s", s.profile.BackendHost, s.profile.BackendPort, gitlabWebhookPath, updatedRepo.WebhookEndpointID),
					PushEventsBranchFilter: *repoPatch.BranchFilter,
				}
				webhookUpdatePayload, err = json.Marshal(webhookUpdate)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal request body for updating webhook %s for project ID: %v", repo.ExternalWebhookID, projectID)).SetInternal(err)
				}
			case vcsPlugin.GitHubCom:
				webhookUpdate := github.WebhookCreateOrUpdate{
					Config: github.WebhookConfig{
						URL:         fmt.Sprintf("%s:%d/%s/%s", s.profile.BackendHost, s.profile.BackendPort, githubWebhookPath, updatedRepo.WebhookEndpointID),
						ContentType: "json",
						Secret:      updatedRepo.WebhookSecretToken,
						InsecureSSL: 1, // TODO: Allow user to specify this value through api.RepositoryPatch
					},
					Events: []string{"push"},
				}
				webhookUpdatePayload, err = json.Marshal(webhookUpdate)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal request body for updating webhook %s for project ID: %v", repo.ExternalWebhookID, projectID)).SetInternal(err)
				}
			}

			err = vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).PatchWebhook(
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
				webhookUpdatePayload,
			)

			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update webhook ID %s for project ID: %v", repo.ExternalWebhookID, projectID)).SetInternal(err)
			}
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
		repoList, err := s.store.FindRepository(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for project ID: %d", projectID)).SetInternal(err)
		}

		// Just be defensive, this shouldn't happen because we set UNIQUE constraint on project_id
		if len(repoList) > 1 {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Retrieved %d repository list for project ID: %d, expect at most 1", len(repoList), projectID)).SetInternal(err)
		} else if len(repoList) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found for project ID: %d", projectID))
		}

		repo := repoList[0]
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
		if err := s.store.DeleteRepository(ctx, repositoryDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectID)).SetInternal(err)
		}

		// Delete the webhook after we successfully delete the repository.
		// This is because in case the webhook deletion fails, we can still have a cleanup process to cleanup the orphaned webhook.
		// If we delete it before we delete the repository, then if the repository deletion fails, we will have a broken repository with no webhook.
		err = vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).DeleteWebhook(
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
			// Despite the error here, we have deleted the repository in the database, we still return success.
			log.Error("Failed to delete webhook for project", zap.Int("project", projectID), zap.Int("repo", repo.ID), zap.Error(err))
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set deployment configuration request").SetInternal(err)
		}
		deploymentConfigUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		project, err := s.store.GetProjectByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", id)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", id))
		}
		deploymentConfigUpsert.ProjectID = id

		deploymentConfig, err := s.store.UpsertDeploymentConfig(ctx, deploymentConfigUpsert)
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
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		project, err := s.store.GetProjectByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", id)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", id))
		}

		deploymentConfig, err := s.store.GetDeploymentConfigByProjectID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get deployment configuration for project id: %d", id)).SetInternal(err)
		}

		// We should return empty deployment config when it doesn't exist.
		if deploymentConfig == nil {
			deploymentConfig = &api.DeploymentConfig{}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, deploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get deployment configuration response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

// refreshToken is a token refresher that stores the latest access token configuration to repository.
func (s *Server) refreshToken(ctx context.Context, repositoryID int) common.TokenRefresher {
	return func(token, refreshToken string, expiresTs int64) error {
		if _, err := s.store.PatchRepository(ctx, &api.RepositoryPatch{
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
func refreshTokenNoop() common.TokenRefresher {
	return func(newToken, newRefreshToken string, expiresTs int64) error {
		return nil
	}
}
