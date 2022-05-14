package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
)

func (s *Server) registerSheetRoutes(g *echo.Group) {
	g.POST("/sheet", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetCreate := &api.SheetCreate{
			CreatorID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create sheet request").SetInternal(err)
		}

		if sheetCreate.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sheet request, missing name")
		}

		// If sheetCreate.DatabaseID is not nil, use its associated ProjectID as the new sheet's ProjectID.
		if sheetCreate.DatabaseID != nil {
			database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: sheetCreate.DatabaseID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %d", *sheetCreate.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", *sheetCreate.DatabaseID))
			}

			sheetCreate.ProjectID = database.ProjectID
		}

		project, err := s.store.GetProjectByID(ctx, sheetCreate.ProjectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %d", sheetCreate.ProjectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", sheetCreate.ProjectID))
		}

		sheetCreate.Source = api.SheetFromBytebase
		sheetCreate.Type = api.SheetForSQL
		sheet, err := s.store.CreateSheet(ctx, sheetCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create sheet").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create sheet response").SetInternal(err)
		}
		return nil
	})

	g.POST("/sheet/project/:projectID/sync", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Project not found: %d", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project not found by ID: %d", projectID))
		}
		if project.WorkflowType != api.VCSWorkflow {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid workflow type: %s, need %s to enable this function", project.WorkflowType, api.VCSWorkflow))
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
					databaseList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
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
			sheetFind := &api.SheetFind{
				Name:      &sheetInfo.SheetName,
				ProjectID: &project.ID,
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

	// Get current user created sheet list.
	g.GET("/sheet/my", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetFind, err := composeCommonSheetFindByQueryParams(c.QueryParams())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bad request: %s", err.Error())).SetInternal(err)
		}

		if rowStatusStr := c.QueryParam("rowStatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			sheetFind.RowStatus = &rowStatus
		}

		sheetFind.CreatorID = &currentPrincipalID

		mySheetList, err := s.store.FindSheet(ctx, sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch sheet list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, mySheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal my sheet list response").SetInternal(err)
		}
		return nil
	})

	// Get sheet list which is shared with current user.
	// The desired sheets would be PROJECT sheets which current user is one of the active member of its project and all public sheets.
	g.GET("/sheet/shared", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetFind, err := composeCommonSheetFindByQueryParams(c.QueryParams())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bad request: %s", err.Error())).SetInternal(err)
		}

		sheetFind.PrincipalID = &currentPrincipalID

		var sheetList []*api.Sheet
		projectSheetVisibility := api.ProjectSheet
		sheetFind.Visibility = &projectSheetVisibility
		projectSheetList, err := s.store.FindSheet(ctx, sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch shared project sheet list").SetInternal(err)
		}
		sheetList = append(sheetList, projectSheetList...)

		publicSheetVisibility := api.PublicSheet
		sheetFind.Visibility = &publicSheetVisibility
		publicSheetList, err := s.store.FindSheet(ctx, sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch shared public sheet list").SetInternal(err)
		}
		sheetList = append(sheetList, publicSheetList...)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal shared sheet list response").SetInternal(err)
		}
		return nil
	})

	// Get current user starred sheet list.
	// The desired sheets would be any visibility but have a record of sheet organizer.
	g.GET("/sheet/starred", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetFind, err := composeCommonSheetFindByQueryParams(c.QueryParams())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bad request: %s", err.Error())).SetInternal(err)
		}

		sheetFind.OrganizerID = &currentPrincipalID

		starredSheetList, err := s.store.FindSheet(ctx, sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch starred sheet list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, starredSheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal starred sheet list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/sheet/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetFind := &api.SheetFind{
			ID: &id,
		}
		sheet, err := s.store.GetSheet(ctx, sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet ID: %v", id)).SetInternal(err)
		}
		if sheet == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal sheet response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/sheet/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetPatch := &api.SheetPatch{
			ID:        id,
			UpdaterID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch sheet request").SetInternal(err)
		}

		sheet, err := s.store.PatchSheet(ctx, sheetPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch sheet with ID: %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal patch sheet response: %d", id)).SetInternal(err)
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
		err = s.store.DeleteSheet(ctx, sheetDelete)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete sheet ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

// composeCommonSheetFindByQueryParams is a common function to compose sheetFind by request query params.
func composeCommonSheetFindByQueryParams(queryParams url.Values) (*api.SheetFind, error) {
	sheetFind := &api.SheetFind{}

	if projectIDStr := queryParams.Get("projectId"); projectIDStr != "" {
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			return nil, fmt.Errorf("project ID is not a number: %s", queryParams.Get("projectId"))
		}
		sheetFind.ProjectID = &projectID
	}
	if databaseIDStr := queryParams.Get("databaseId"); databaseIDStr != "" {
		databaseID, err := strconv.Atoi(databaseIDStr)
		if err != nil {
			return nil, fmt.Errorf("database ID is not a number: %s", queryParams.Get("databaseId"))
		}
		sheetFind.DatabaseID = &databaseID
	}

	return sheetFind, nil
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
