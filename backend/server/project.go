package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	vcsPlugin "github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/bitbucket"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

func (s *Server) registerProjectRoutes(g *echo.Group) {
	// When we link the repository with the project, we will also change the project workflow type to VCS
	g.POST("/project/:projectID/repository", func(c echo.Context) error {
		ctx := c.Request().Context()

		setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace setting").SetInternal(err)
		}
		if setting.ExternalUrl == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "external URL isn't setup yet, see https://www.bytebase.com/docs/get-started/install/external-url")
		}

		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", projectID))
		}
		if project.Deleted {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project %d is deleted", projectID))
		}

		repositoryCreate := &api.RepositoryCreate{
			ProjectID:         projectID,
			ProjectResourceID: project.ResourceID,
			CreatorID:         c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, repositoryCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create linked repository request").SetInternal(err)
		}
		if repositoryCreate.BranchFilter == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Branch must be specified.")
		}

		// We need to check the FilePathTemplate in create repository request.
		// This avoids to a certain extent that the creation succeeds but does not work.
		if err := vcsPlugin.IsAsterisksInTemplateValid(path.Join(repositoryCreate.BaseDirectory, repositoryCreate.FilePathTemplate)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, errors.Wrap(err, errors.Wrap(err, "Invalid base directory and filepath template combination").Error()))
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

		// When the branch names doesn't contain wildcards, we should make sure the branch exists in the repo.
		if !strings.Contains(repositoryCreate.BranchFilter, "*") {
			notFound, err := isBranchNotFound(
				ctx,
				vcs,
				s.store,
				repositoryCreate.WebURL,
				repositoryCreate.AccessToken,
				repositoryCreate.RefreshToken,
				repositoryCreate.ExternalID,
				repositoryCreate.BranchFilter,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get branch %q", repositoryCreate.BranchFilter)).SetInternal(err)
			}
			if notFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Branch %q not found in repository %s.", repositoryCreate.BranchFilter, repositoryCreate.Name))
			}
		}

		// For a particular VCS repo, all Bytebase projects share the same webhook.
		repositories, err := s.store.FindRepository(ctx, &api.RepositoryFind{
			WebURL: &repositoryCreate.WebURL,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find repository with web url: %s", repositoryCreate.WebURL)).SetInternal(err)
		}

		repositoryCreate.WebhookURLHost = setting.ExternalUrl
		// If we can find at least one repository with the same web url, we will use the same webhook instead of creating a new one.
		if len(repositories) > 0 {
			repo := repositories[0]
			repositoryCreate.WebhookEndpointID = repo.WebhookEndpointID
			repositoryCreate.WebhookSecretToken = repo.WebhookSecretToken
			repositoryCreate.ExternalWebhookID = repo.ExternalWebhookID
		} else {
			// Bytebase needs to create a webbook in the connecting repository pointing back to the
			// Bytebase address exposed at --external-url.
			if setting.ExternalUrl == "" {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bytebase must start with --external-url to configure GitOps workflow, docs: %s", common.ExternalURLPlaceholder))
			}

			workspaceID, err := s.store.GetWorkspaceID(ctx)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}

			repositoryCreate.WebhookEndpointID = fmt.Sprintf("%s-%d", workspaceID, time.Now().Unix())
			secretToken, err := common.RandomString(gitlab.SecretTokenLength)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate random secret token for VCS").SetInternal(err)
			}
			repositoryCreate.WebhookSecretToken = secretToken

			gitopsWebhookURL := setting.GitopsWebhookUrl
			if gitopsWebhookURL == "" {
				gitopsWebhookURL = setting.ExternalUrl
			}
			webhookID, err := createVCSWebhook(ctx, vcs.Type, repositoryCreate.WebhookEndpointID, secretToken, repositoryCreate.AccessToken, vcs.InstanceURL, repositoryCreate.ExternalID, gitopsWebhookURL)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create webhook for project ID: %v", repositoryCreate.ProjectID)).SetInternal(err)
			}
			repositoryCreate.ExternalWebhookID = webhookID
		}
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

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", projectID))
		}
		if project.Deleted {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project %d is deleted", projectID))
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
		repoPatch.ID = &repo.ID
		newBranchFilter := repo.BranchFilter
		if repoPatch.BranchFilter != nil {
			newBranchFilter = *repoPatch.BranchFilter
		}
		if newBranchFilter == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Branch must be specified.")
		}

		vcs, err := s.store.GetVCSByID(ctx, repo.VCSID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for creating repository: %d", repo.VCSID)).SetInternal(err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS not found with ID: %d", repo.VCSID))
		}

		// When the branch names doesn't contain wildcards, we should make sure the branch exists in the repo.
		if !strings.Contains(newBranchFilter, "*") {
			notFound, err := isBranchNotFound(
				ctx,
				vcs,
				s.store,
				repo.WebURL,
				repo.AccessToken,
				repo.RefreshToken,
				repo.ExternalID,
				newBranchFilter,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get branch %q", newBranchFilter)).SetInternal(err)
			}
			if notFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Branch %q not found in repository %s.", newBranchFilter, repo.Name))
			}
		}

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

		// DO NOT enable the EnableSQLReviewCI field.
		// We will update it through POST /project/:projectID/repository/:repositoryID/sql-review-ci endpoint
		if repoPatch.EnableSQLReviewCI != nil && *repoPatch.EnableSQLReviewCI && !repo.EnableSQLReviewCI {
			repoPatch.EnableSQLReviewCI = &repo.EnableSQLReviewCI
		}

		updatedRepo, err := s.store.PatchRepository(ctx, repoPatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update repository for project ID: %d", projectID)).SetInternal(err)
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
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", projectID))
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
			ProjectID:         projectID,
			ProjectResourceID: project.ResourceID,
			DeleterID:         c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.store.DeleteRepository(ctx, repositoryDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete repository for project ID: %d", projectID)).SetInternal(err)
		}

		// We use one webhook in one repo for at least one Bytebase project, so we only delete the webhook if this project is the last one using this webhook.
		repos, err := s.store.FindRepository(ctx, &api.RepositoryFind{
			WebURL: &repo.WebURL,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find repository for web url: %s", repo.WebURL)).SetInternal(err)
		}
		if len(repos) == 0 {
			// Delete the webhook after we successfully delete the repository.
			// This is because in case the webhook deletion fails, we can still have a cleanup process to cleanup the orphaned webhook.
			// If we delete it before we delete the repository, then if the repository deletion fails, we will have a broken repository with no webhook.
			if err = vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).DeleteWebhook(
				ctx,
				// Need to get ApplicationID, Secret from vcs instead of repository.vcs since the latter is not composed.
				common.OauthContext{
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    refreshTokenNoop(),
				},
				vcs.InstanceURL,
				repo.ExternalID,
				repo.ExternalWebhookID,
			); err != nil {
				// Despite the error here, we have deleted the repository in the database, we still return success.
				log.Error("Failed to delete webhook for project", zap.Int("project", projectID), zap.Int("repo", repo.ID), zap.Error(err))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.PATCH("/project/:id/deployment", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		deploymentConfigUpsert := &api.DeploymentConfigUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, deploymentConfigUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set deployment configuration request").SetInternal(err)
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", projectID))
		}

		apiDeploymentConfig, err := api.ValidateAndGetDeploymentSchedule(deploymentConfigUpsert.Payload)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to validate deployment configuration: %v", err))
		}

		// Remove this when we migrate to V1 API.
		//
		// Convert to store message.
		var schedule store.Schedule
		for _, d := range apiDeploymentConfig.Deployments {
			var labelSelector store.LabelSelector
			for _, spec := range d.Spec.Selector.MatchExpressions {
				operatorTp := store.InOperatorType
				switch spec.Operator {
				case api.InOperatorType:
					operatorTp = store.InOperatorType
				case api.ExistsOperatorType:
					operatorTp = store.ExistsOperatorType
				}
				labelSelector.MatchExpressions = append(labelSelector.MatchExpressions, &store.LabelSelectorRequirement{
					Key:      spec.Key,
					Operator: operatorTp,
					Values:   spec.Values,
				})
			}
			schedule.Deployments = append(schedule.Deployments, &store.Deployment{
				Name: d.Name,
				Spec: &store.DeploymentSpec{
					Selector: &labelSelector,
				},
			})
		}
		storeDeploymentConfig := &store.DeploymentConfigMessage{
			Name:     deploymentConfigUpsert.Name,
			Schedule: &schedule,
		}

		newStoreDeploymentConfig, err := s.store.UpsertDeploymentConfigV2(ctx, projectID, c.Get(getPrincipalIDContextKey()).(int), storeDeploymentConfig)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set deployment configuration").SetInternal(err)
		}
		newAPIDeploymentConfig, err := newStoreDeploymentConfig.ToAPIDeploymentConfig()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert deployment configuration").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, newAPIDeploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal set deployment configuration response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:id/deployment", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found with ID %d", projectID))
		}
		if project.Deleted {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project %d is deleted", projectID))
		}

		// DeploymentConfig is never nil.
		deploymentConfig, err := s.store.GetDeploymentConfigV2(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get deployment configuration for project id: %d", projectID)).SetInternal(err)
		}
		apiDeploymentConfig, err := deploymentConfig.ToAPIDeploymentConfig()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert deployment config to api deployment config").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, apiDeploymentConfig); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get deployment configuration response: %v", projectID)).SetInternal(err)
		}
		return nil
	})

	g.POST("/project/:projectID/sync-sheet", func(c echo.Context) error {
		// TODO(tianzhou): uncomment this after adding the test harness to using Enterprise version.
		// if !s.licenseService.IsFeatureEnabled(api.FeatureVCSSheetSync) {
		// 	return echo.NewHTTPError(http.StatusForbidden, api.FeatureVCSSheetSync.AccessErrorMessage())
		// }

		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Project not found: %d", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found by ID: %d", projectID))
		}
		if project.Deleted {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project %s is deleted", project.Title))
		}
		if project.Workflow != api.VCSWorkflow {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid workflow type: %s, need %s to enable this function", project.Workflow, api.VCSWorkflow))
		}

		repo, err := s.store.GetRepository(ctx, &api.RepositoryFind{ProjectID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find relevant VCS repo, Project ID: %d", projectID)).SetInternal(err)
		}
		if repo == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository not found by project ID: %d", projectID))
		}

		vcs, err := s.store.GetVCSByID(ctx, repo.VCSID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for sync sheet: %d", repo.VCSID)).SetInternal(err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS not found by ID: %d", repo.VCSID))
		}

		basePath := filepath.Dir(repo.SheetPathTemplate)
		// TODO(Steven): The repo.branchFilter could be `test/*` which cannot be the ref value.
		fileList, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).FetchRepositoryFileList(ctx,
			common.OauthContext{
				ClientID:     vcs.ApplicationID,
				ClientSecret: vcs.Secret,
				AccessToken:  repo.AccessToken,
				RefreshToken: repo.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
			},
			vcs.InstanceURL,
			repo.ExternalID,
			repo.BranchFilter,
			basePath,
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository file list from VCS, instance URL: %s", vcs.InstanceURL)).SetInternal(err)
		}

		for _, file := range fileList {
			sheetInfo, err := parseSheetInfo(file.Path, repo.SheetPathTemplate)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse sheet info from template").SetInternal(err)
			}
			if sheetInfo.SheetName == "" {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("sheet name cannot be empty from sheet path %s with template %s", file.Path, repo.SheetPathTemplate)).SetInternal(err)
			}

			fileContent, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).ReadFileContent(ctx,
				common.OauthContext{
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
				},
				vcs.InstanceURL,
				repo.ExternalID,
				file.Path,
				repo.BranchFilter,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch file content from VCS, instance URL: %s, repo ID: %s, file path: %s, branch: %s", vcs.InstanceURL, repo.ExternalID, file.Path, repo.BranchFilter)).SetInternal(err)
			}

			fileMeta, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(ctx,
				common.OauthContext{
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
				},
				vcs.InstanceURL,
				repo.ExternalID,
				file.Path,
				repo.BranchFilter,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch file meta from VCS, instance URL: %s, repo ID: %s, file path: %s, branch: %s", vcs.InstanceURL, repo.ExternalID, file.Path, repo.BranchFilter)).SetInternal(err)
			}

			lastCommit, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).FetchCommitByID(ctx,
				common.OauthContext{
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
				},
				vcs.InstanceURL,
				repo.ExternalID,
				fileMeta.LastCommitID,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch commit data from VCS, instance URL: %s, repo ID: %s, commit ID: %s", vcs.InstanceURL, repo.ExternalID, fileMeta.LastCommitID)).SetInternal(err)
			}

			sheetVCSPayload := &api.SheetVCSPayload{
				FileName:     fileMeta.Name,
				FilePath:     fileMeta.Path,
				Size:         fileMeta.Size,
				Author:       lastCommit.AuthorName,
				LastCommitID: lastCommit.ID,
				LastSyncTs:   time.Now().Unix(),
			}
			payload, err := json.Marshal(sheetVCSPayload)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sheetVCSPayload").SetInternal(err)
			}

			var databaseID *int
			// In non-tenant mode, we can set a databaseId for sheet with ENV_ID and DB_NAME,
			// and ENV_ID and DB_NAME is either both present or neither present.
			if project.TenantMode != api.TenantModeDisabled {
				if sheetInfo.EnvironmentID != "" && sheetInfo.DatabaseName != "" {
					databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID, DatabaseName: &sheetInfo.DatabaseName})
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find database list with name: %s, project ID: %d", sheetInfo.DatabaseName, projectID)).SetInternal(err)
					}
					for _, database := range databases {
						database := database // create a new var "database".
						if database.EnvironmentID == sheetInfo.EnvironmentID {
							databaseID = &database.UID
							break
						}
					}
				}
			}

			var sheetSource api.SheetSource
			switch vcs.Type {
			case vcsPlugin.GitLab:
				sheetSource = api.SheetFromGitLab
			case vcsPlugin.GitHub:
				sheetSource = api.SheetFromGitHub
			case vcsPlugin.Bitbucket:
				sheetSource = api.SheetFromBitbucket
			}
			vscSheetType := api.SheetForSQL
			sheetFind := &api.SheetFind{
				Name:      &sheetInfo.SheetName,
				ProjectID: &project.UID,
				Source:    &sheetSource,
				Type:      &vscSheetType,
			}
			sheet, err := s.store.GetSheet(ctx, sheetFind, currentPrincipalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sheet with name: %s, project ID: %d", sheetInfo.SheetName, projectID)).SetInternal(err)
			}

			if sheet == nil {
				sheetCreate := api.SheetCreate{
					ProjectID:  projectID,
					CreatorID:  currentPrincipalID,
					Name:       sheetInfo.SheetName,
					Statement:  fileContent,
					Visibility: api.ProjectSheet,
					Source:     sheetSource,
					Type:       api.SheetForSQL,
					Payload:    string(payload),
				}
				if databaseID != nil {
					sheetCreate.DatabaseID = databaseID
				}

				if _, err := s.store.CreateSheet(ctx, &sheetCreate); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create sheet from VCS").SetInternal(err)
				}
			} else {
				payloadString := string(payload)
				sheetPatch := api.SheetPatch{
					ID:        sheet.ID,
					UpdaterID: currentPrincipalID,
					Statement: &fileContent,
					Payload:   &payloadString,
				}
				if databaseID != nil {
					sheetPatch.DatabaseID = databaseID
				}

				if _, err := s.store.PatchSheet(ctx, &sheetPatch); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to patch sheet from VCS").SetInternal(err)
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		return nil
	})
}

func createVCSWebhook(ctx context.Context, vcsType vcsPlugin.Type, webhookEndpointID, secretToken, accessToken, instanceURL, externalRepoID, gitopsWebhookURL string) (string, error) {
	// Create a new webhook and retrieve the created webhook ID
	var webhookCreatePayload []byte
	var err error
	switch vcsType {
	case vcsPlugin.GitLab:
		webhookCreate := gitlab.WebhookCreate{
			URL:                   fmt.Sprintf("%s/hook/gitlab/%s", gitopsWebhookURL, webhookEndpointID),
			SecretToken:           secretToken,
			PushEvents:            true,
			EnableSSLVerification: false, // TODO(tianzhou): This is set to false, be lax to not enable_ssl_verification
		}
		webhookCreatePayload, err = json.Marshal(webhookCreate)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal request body for creating webhook")
		}
	case vcsPlugin.GitHub:
		webhookPost := github.WebhookCreateOrUpdate{
			Config: github.WebhookConfig{
				URL:         fmt.Sprintf("%s/hook/github/%s", gitopsWebhookURL, webhookEndpointID),
				ContentType: "json",
				Secret:      secretToken,
				InsecureSSL: 1, // TODO: Allow user to specify this value through api.RepositoryCreate
			},
			Events: []string{"push"},
		}
		webhookCreatePayload, err = json.Marshal(webhookPost)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal request body for creating webhook")
		}
	case vcsPlugin.Bitbucket:
		webhookPost := bitbucket.WebhookCreateOrUpdate{
			Description: "Bytebase GitOps",
			URL:         fmt.Sprintf("%s/hook/bitbucket/%s", gitopsWebhookURL, webhookEndpointID),
			Active:      true,
			Events:      []string{"repo:push"},
		}
		webhookCreatePayload, err = json.Marshal(webhookPost)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal request body for creating webhook")
		}
	}
	webhookID, err := vcsPlugin.Get(vcsType, vcsPlugin.ProviderConfig{}).CreateWebhook(
		ctx,
		common.OauthContext{
			AccessToken: accessToken,
			// We use refreshTokenNoop() because the repository isn't created yet.
			Refresher: refreshTokenNoop(),
		},
		instanceURL,
		externalRepoID,
		webhookCreatePayload,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create webhook")
	}
	return webhookID, nil
}

// refreshToken is a no-op token refresher. It should be used when the repository isn't created yet.
func refreshTokenNoop() common.TokenRefresher {
	return func(newToken, newRefreshToken string, expiresTs int64) error {
		return nil
	}
}

func isBranchNotFound(ctx context.Context, vcs *api.VCS, store *store.Store, webURL, accessToken, refreshToken, externalID, branch string) (bool, error) {
	_, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).GetBranch(ctx,
		common.OauthContext{
			ClientID:     vcs.ApplicationID,
			ClientSecret: vcs.Secret,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			Refresher:    utils.RefreshToken(ctx, store, webURL),
		},
		vcs.InstanceURL, externalID, branch)

	if common.ErrorCode(err) == common.NotFound {
		return true, nil
	}
	return false, err
}

// SheetInfo represents the sheet related information from sheetPathTemplate.
type SheetInfo struct {
	EnvironmentID string
	DatabaseName  string
	SheetName     string
}

// parseSheetInfo matches sheetPath against sheetPathTemplate. If sheetPath matches, then it will derive SheetInfo from the sheetPath.
// Both sheetPath and sheetPathTemplate are the full file path(including the base directory) of the repository.
func parseSheetInfo(sheetPath string, sheetPathTemplate string) (*SheetInfo, error) {
	placeholderList := []string{
		"ENV_ID",
		"DB_NAME",
		"NAME",
	}
	sheetPathRegex := sheetPathTemplate
	for _, placeholder := range placeholderList {
		sheetPathRegex = strings.ReplaceAll(sheetPathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9\\+\\-\\=\\_\\#\\!\\$\\. ]+)", placeholder))
	}
	sheetRegex, err := regexp.Compile(fmt.Sprintf("^%s$", sheetPathRegex))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid sheet path template %q", sheetPathTemplate)
	}
	if !sheetRegex.MatchString(sheetPath) {
		return nil, errors.Errorf("sheet path %q does not match sheet path template %q", sheetPath, sheetPathTemplate)
	}

	matchList := sheetRegex.FindStringSubmatch(sheetPath)
	sheetInfo := &SheetInfo{}
	for _, placeholder := range placeholderList {
		index := sheetRegex.SubexpIndex(placeholder)
		if index >= 0 {
			switch placeholder {
			case "ENV_ID":
				sheetInfo.EnvironmentID = matchList[index]
			case "DB_NAME":
				sheetInfo.DatabaseName = matchList[index]
			case "NAME":
				sheetInfo.SheetName = matchList[index]
			}
		}
	}

	return sheetInfo, nil
}
