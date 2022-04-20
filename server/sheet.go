package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerSheetRoutes(g *echo.Group) {
	g.POST("/sheet", func(c echo.Context) error {
		ctx := c.Request().Context()
		sheetCreate := &api.SheetCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create sheet request").SetInternal(err)
		}

		if sheetCreate.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sheet request, missing name")
		}

		// If sheetCreate.DatabaseID is not nil, use its associated ProjectID as the new sheet's ProjectID.
		if sheetCreate.DatabaseID != nil {
			database, err := s.DatabaseService.FindDatabase(ctx, &api.DatabaseFind{
				ID: sheetCreate.DatabaseID,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %d", *sheetCreate.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", *sheetCreate.DatabaseID))
			}

			sheetCreate.ProjectID = database.ProjectID
		}

		project, err := s.ProjectService.FindProject(ctx, &api.ProjectFind{
			ID: &sheetCreate.ProjectID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %d", sheetCreate.ProjectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", sheetCreate.ProjectID))
		}

		sheetCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		sheetRaw, err := s.SheetService.CreateSheet(ctx, sheetCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create sheet").SetInternal(err)
		}
		sheet, err := s.composeSheetRelationship(ctx, sheetRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose sheet with ID %d", sheetRaw.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create sheet response").SetInternal(err)
		}
		return nil
	})

	g.POST("/sheet/project/:projectID/sync", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		project, err := s.ProjectService.FindProject(ctx, &api.ProjectFind{ID: &projectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Project not found: %d", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found by ID: %d", projectID))
		}
		if project.WorkflowType != api.VCSWorkflow {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid workflow type: %s, need %s to enable this function", project.WorkflowType, api.VCSWorkflow))
		}

		repo, err := s.RepositoryService.FindRepository(ctx, &api.RepositoryFind{ProjectID: &projectID})
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
		fileList, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).FetchRepositoryFileList(ctx,
			common.OauthContext{
				ClientID:     vcs.ApplicationID,
				ClientSecret: vcs.Secret,
				AccessToken:  repo.AccessToken,
				RefreshToken: repo.RefreshToken,
				Refresher:    s.refreshToken(ctx, repo.ID),
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

			fileContent, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).ReadFileContent(ctx,
				common.OauthContext{
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    s.refreshToken(ctx, repo.ID),
				},
				vcs.InstanceURL,
				repo.ExternalID,
				file.Path,
				repo.BranchFilter,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch file content from VCS, instance URL: %s, repo ID: %s, file path: %s, branch: %s", vcs.InstanceURL, repo.ExternalID, file.Path, repo.BranchFilter)).SetInternal(err)
			}

			fileMeta, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).ReadFileMeta(ctx,
				common.OauthContext{
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    s.refreshToken(ctx, repo.ID),
				},
				vcs.InstanceURL,
				repo.ExternalID,
				file.Path,
				repo.BranchFilter,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch file meta from VCS, instance URL: %s, repo ID: %s, file path: %s, branch: %s", vcs.InstanceURL, repo.ExternalID, file.Path, repo.BranchFilter)).SetInternal(err)
			}

			lastCommit, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).FetchCommitByID(ctx,
				common.OauthContext{
					ClientID:     vcs.ApplicationID,
					ClientSecret: vcs.Secret,
					AccessToken:  repo.AccessToken,
					RefreshToken: repo.RefreshToken,
					Refresher:    s.refreshToken(ctx, repo.ID),
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
			// In non-tenant mode, we can set a databaseId for sheet with ENV_NAME and DB_NAME,
			// and ENV_NAME and DB_NAME is either both present or neither present.
			if project.TenantMode != api.TenantModeDisabled {
				if sheetInfo.EnvironmentName != "" && sheetInfo.DatabaseName != "" {
					databaseList, err := s.composeDatabaseListByFind(ctx, &api.DatabaseFind{
						Name:      &sheetInfo.DatabaseName,
						ProjectID: &projectID,
					})
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find database list with name: %s, project ID: %d", sheetInfo.DatabaseName, projectID)).SetInternal(err)
					}

					for _, database := range databaseList {
						if database.Instance.Environment.Name == sheetInfo.EnvironmentName {
							databaseID = &database.ID
							break
						}
					}
				}
			}

			var sheetSource api.SheetSource
			switch vcs.Type {
			case vcsPlugin.GitLabSelfHost:
				sheetSource = api.SheetFromGitLabSelfHost
			case vcsPlugin.GitHubCom:
				sheetSource = api.SheetFromGitHubCom
			}
			vscSheetType := api.SheetForSQL
			sheet, err := s.SheetService.FindSheet(ctx, &api.SheetFind{
				Name:      &sheetInfo.SheetName,
				ProjectID: &project.ID,
				Source:    &sheetSource,
				Type:      &vscSheetType,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sheet with name: %s, project ID: %d", sheetInfo.SheetName, projectID)).SetInternal(err)
			}

			if sheet == nil {
				sheetCreate := api.SheetCreate{
					ProjectID:  projectID,
					CreatorID:  c.Get(getPrincipalIDContextKey()).(int),
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

				if _, err := s.SheetService.CreateSheet(ctx, &sheetCreate); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create sheet from VCS").SetInternal(err)
				}
			} else {
				payloadString := string(payload)
				sheetPatch := api.SheetPatch{
					ID:        sheet.ID,
					UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
					Statement: &fileContent,
					Payload:   &payloadString,
				}
				if databaseID != nil {
					sheetPatch.DatabaseID = databaseID
				}

				if _, err := s.SheetService.PatchSheet(ctx, &sheetPatch); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to patch sheet from VCS").SetInternal(err)
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		return nil
	})

	g.GET("/sheet", func(c echo.Context) error {
		ctx := c.Request().Context()
		sheetFind := &api.SheetFind{}

		if rowStatusStr := c.QueryParam("rowStatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			sheetFind.RowStatus = &rowStatus
		}
		if projectIDStr := c.QueryParam("projectId"); projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.QueryParam("projectId"))).SetInternal(err)
			}
			sheetFind.ProjectID = &projectID
		}
		if databaseIDStr := c.QueryParam("databaseId"); databaseIDStr != "" {
			databaseID, err := strconv.Atoi(databaseIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.QueryParam("databaseId"))).SetInternal(err)
			}
			sheetFind.DatabaseID = &databaseID
		}
		if visibility := api.SheetVisibility(c.QueryParam("visibility")); visibility != "" {
			sheetFind.Visibility = &visibility
		}
		if creatorIDStr := c.QueryParam("creatorId"); creatorIDStr != "" {
			creatorID, err := strconv.Atoi(creatorIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("creatorID ID is not a number: %s", c.QueryParam("creatorId"))).SetInternal(err)
			}
			sheetFind.CreatorID = &creatorID
		}
		// When getting private/project sheet list, we should set the PrincipalID to ensure only
		// the sheet which related project containing PrincipalID as an active member could be found.
		if sheetFind.Visibility != nil && (*sheetFind.Visibility == api.PrivateSheet || *sheetFind.Visibility == api.ProjectSheet) {
			principalID := c.Get(getPrincipalIDContextKey()).(int)
			sheetFind.PrincipalID = &principalID
		}

		sheetRawList, err := s.SheetService.FindSheetList(ctx, sheetFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch sheet list").SetInternal(err)
		}
		var sheetList []*api.Sheet
		for _, sheetRaw := range sheetRawList {
			sheet, err := s.composeSheetRelationship(ctx, sheetRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet relationship: %v", sheet.Name)).SetInternal(err)
			}
			sheetList = append(sheetList, sheet)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sheet list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/sheet/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetFind := &api.SheetFind{
			ID: &id,
		}

		sheetRaw, err := s.SheetService.FindSheet(ctx, sheetFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet ID: %v", id)).SetInternal(err)
		}
		if sheetRaw == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
		}

		sheet, err := s.composeSheetRelationship(ctx, sheetRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet relationship: %v", sheet.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal sheet response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/sheet/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetPatch := &api.SheetPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch sheet request").SetInternal(err)
		}

		sheetRaw, err := s.SheetService.PatchSheet(ctx, sheetPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch sheet ID: %v", id)).SetInternal(err)
		}

		sheet, err := s.composeSheetRelationship(ctx, sheetRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated sheet relationship: %v", sheet.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal patch sheet response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/sheet/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetDelete := &api.SheetDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		err = s.SheetService.DeleteSheet(ctx, sheetDelete)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete sheet ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) composeSheetRelationship(ctx context.Context, raw *api.SheetRaw) (*api.Sheet, error) {
	sheet := raw.ToSheet()

	creator, err := s.store.GetPrincipalByID(ctx, sheet.CreatorID)
	if err != nil {
		return nil, err
	}
	sheet.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, sheet.UpdaterID)
	if err != nil {
		return nil, err
	}
	sheet.Updater = updater

	project, err := s.composeProjectByID(ctx, sheet.ProjectID)
	if err != nil {
		return nil, err
	}
	sheet.Project = project

	if sheet.DatabaseID != nil {
		database, err := s.composeDatabaseByFind(ctx, &api.DatabaseFind{
			ID: sheet.DatabaseID,
		})
		if err != nil {
			return nil, err
		}
		sheet.Database = database
	}

	return sheet, nil
}

// SheetInfo represents the sheet related information from sheetPathTemplate.
type SheetInfo struct {
	EnvironmentName string
	DatabaseName    string
	SheetName       string
}

// parseSheetInfo matches sheetPath against sheetPathTemplate. If sheetPath matches, then it will derive SheetInfo from the sheetPath.
// Both sheetPath and sheetPathTemplate are the full file path(including the base directory) of the repository.
func parseSheetInfo(sheetPath string, sheetPathTemplate string) (*SheetInfo, error) {
	placeholderList := []string{
		"ENV_NAME",
		"DB_NAME",
		"NAME",
	}
	sheetPathRegex := sheetPathTemplate
	for _, placeholder := range placeholderList {
		sheetPathRegex = strings.ReplaceAll(sheetPathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9\\+\\-\\=\\_\\#\\!\\$\\. ]+)", placeholder))
	}
	sheetRegex, err := regexp.Compile(fmt.Sprintf("^%s$", sheetPathRegex))
	if err != nil {
		return nil, fmt.Errorf("invalid sheet path template: %q, err: %v", sheetPathTemplate, err)
	}
	if !sheetRegex.MatchString(sheetPath) {
		return nil, fmt.Errorf("sheet path %q does not match sheet path template %q", sheetPath, sheetPathTemplate)
	}

	matchList := sheetRegex.FindStringSubmatch(sheetPath)
	sheetInfo := &SheetInfo{}
	for _, placeholder := range placeholderList {
		index := sheetRegex.SubexpIndex(placeholder)
		if index >= 0 {
			switch placeholder {
			case "ENV_NAME":
				sheetInfo.EnvironmentName = matchList[index]
			case "DB_NAME":
				sheetInfo.DatabaseName = matchList[index]
			case "NAME":
				sheetInfo.SheetName = matchList[index]
			}
		}
	}

	return sheetInfo, nil
}
