package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	roleContextKey = "role"
)

func getRoleContextKey() string {
	return roleContextKey
}

func aclMiddleware(s *Server, pathPrefix string, ce *casbin.Enforcer, next echo.HandlerFunc, readonly bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		path := strings.TrimPrefix(c.Request().URL.Path, pathPrefix)

		// Skips auth
		if common.HasPrefixes(path, "/auth", "/oauth") {
			return next(c)
		}

		// Skips OpenAPI SQL endpoint
		if common.HasPrefixes(c.Path(), fmt.Sprintf("%s/sql", openAPIPrefix)) {
			return next(c)
		}

		method := c.Request().Method

		// Skip GET /feature request
		if common.HasPrefixes(path, "/feature") && method == "GET" {
			return next(c)
		}

		if readonly && method != "GET" {
			return echo.NewHTTPError(http.StatusMethodNotAllowed, "Server is in readonly mode")
		}

		// Gets principal id from the context.
		principalID := c.Get(getPrincipalIDContextKey()).(int)

		user, err := s.store.GetUserByID(ctx, principalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User ID is not a member: %d", principalID))
		}
		if user.MemberDeleted {
			return echo.NewHTTPError(http.StatusUnauthorized, "This user has been deactivated by the admin")
		}

		role := user.Role
		// If admin feature is not enabled, then we treat all user as OWNER.
		if !s.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
			role = api.Owner
		}

		// Performs the ACL check.
		pass, err := ce.Enforce(string(role), path, method)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		if !pass {
			// If the request is trying to GET/PATCH/DELETE itself, we will change the method signature to
			// XXX_SELF and try again. Because XXX is a superset of XXX_SELF, thus we only try XXX_SELF after
			// XXX fails.
			if method == "GET" || method == "PATCH" || method == "DELETE" {
				if isSelf, err := isOperatingSelf(ctx, c, s, principalID, method, path); err != nil {
					return err
				} else if isSelf {
					method += "_SELF"

					// Performs the ACL check with _SELF.
					pass, err = ce.Enforce(string(role), path, method)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
					}
				}
			}
		}

		if !pass {
			return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(
				errors.Errorf("rejected by the ACL policy; %s %s u%d/%s", method, path, principalID, role))
		}

		// Workspace Owner or DBA assumes project Owner role for all projects, so will
		// pass any project ACL.
		if role != api.Owner && role != api.DBA {
			var aclErr *echo.HTTPError
			projectRolesFinder := func(projectID int, principalID int) (map[common.ProjectRole]bool, error) {
				policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectID})
				if err != nil {
					return nil, err
				}
				projectRoles := make(map[common.ProjectRole]bool)
				for _, binding := range policy.Bindings {
					for _, member := range binding.Members {
						if member.ID == principalID {
							projectRoles[common.ProjectRole(binding.Role)] = true
							break
						}
					}
				}
				return projectRoles, nil
			}

			sheetFinder := func(sheetID int) (*api.Sheet, error) {
				sheetFind := &api.SheetFind{
					ID: &sheetID,
				}
				return s.store.GetSheet(ctx, sheetFind, principalID)
			}

			if strings.HasPrefix(path, "/project") {
				aclErr = enforceWorkspaceDeveloperProjectRouteACL(s.licenseService.GetEffectivePlan(), path, method, principalID, projectRolesFinder)
			} else if strings.HasPrefix(path, "/sheet") {
				aclErr = enforceWorkspaceDeveloperSheetRouteACL(s.licenseService.GetEffectivePlan(), path, method, principalID, projectRolesFinder, sheetFinder)
			} else if strings.HasPrefix(path, "/database") {
				// We need to copy the body because it will be consumed by the next middleware.
				// And TeeReader require us the write must complete before the read completes.
				// The body under the /issue route is a JSON object, and always not too large, so using ioutil.ReadAll is fine here.
				bodyBytes, err := io.ReadAll(c.Request().Body)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read request body.").SetInternal(err)
				}
				if err := c.Request().Body.Close(); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to close request body.").SetInternal(err)
				}
				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				aclErr = enforceWorkspaceDeveloperDatabaseRouteACL(s.licenseService.GetEffectivePlan(), path, method, string(bodyBytes), principalID, getRetrieveDatabaseProjectID(ctx, s.store), projectRolesFinder)
			} else if strings.HasPrefix(path, "/issue") {
				// We need to copy the body because it will be consumed by the next middleware.
				// And TeeReader require us the write must complete before the read completes.
				// The body under the /issue route is a JSON object, and always not too large, so using ioutil.ReadAll is fine here.
				bodyBytes, err := io.ReadAll(c.Request().Body)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read request body.").SetInternal(err)
				}
				if err := c.Request().Body.Close(); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to close request body.").SetInternal(err)
				}
				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				aclErr = enforceWorkspaceDeveloperIssueRouteACL(s.licenseService.GetEffectivePlan(), path, method, string(bodyBytes), c.QueryParams(), principalID, getRetrieveIssueProjectID(ctx, s.store), projectRolesFinder)
			}
			if aclErr != nil {
				return aclErr
			}
		}

		// Stores role into context.
		c.Set(getRoleContextKey(), role)

		return next(c)
	}
}

var projectGeneralRouteRegex = regexp.MustCompile(`^/project/(?P<projectID>\d+)`)
var projectMemberRouteRegex = regexp.MustCompile(`^/project/(?P<projectID>\d+)/member`)
var projectSyncSheetRouteRegex = regexp.MustCompile(`^/project/(?P<projectID>\d+)/sync-sheet`)

func enforceWorkspaceDeveloperProjectRouteACL(plan api.PlanType, path string, method string, principalID int, projectRolesFinder func(projectID int, principalID int) (map[common.ProjectRole]bool, error)) *echo.HTTPError {
	var projectID int
	var permission api.ProjectPermissionType
	var permissionErrMsg string
	if method != "GET" {
		if matches := projectMemberRouteRegex.FindStringSubmatch(path); matches != nil {
			projectID, _ = strconv.Atoi(matches[1])
			permission = api.ProjectPermissionManageMember
			permissionErrMsg = "not have permission to manage the project member"
		} else if matches := projectSyncSheetRouteRegex.FindStringSubmatch(path); matches != nil {
			projectID, _ = strconv.Atoi(matches[1])
			permission = api.ProjectPermissionSyncSheet
			permissionErrMsg = "not have permission to sync sheet for project"
		} else if matches := projectGeneralRouteRegex.FindStringSubmatch(path); matches != nil {
			projectID, _ = strconv.Atoi(matches[1])
			permission = api.ProjectPermissionManageGeneral
			permissionErrMsg = "not have permission to manage the project general setting"
		}
	}

	if projectID > 0 {
		projectRoles, err := projectRolesFinder(projectID, principalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		if len(projectRoles) == 0 {
			return echo.NewHTTPError(http.StatusUnauthorized, "is not a member of the project")
		}

		if !api.ProjectPermission(permission, plan, projectRoles) {
			return echo.NewHTTPError(http.StatusUnauthorized, permissionErrMsg)
		}
	}

	return nil
}

var databaseGeneralRouteRegex = regexp.MustCompile(`^/database/(?P<databaseID>\d+)$`)

func enforceWorkspaceDeveloperDatabaseRouteACL(plan api.PlanType, path string, method string, body string, principalID int, projectIDOfDatabase func(databaseID int) (int, error), projectRolesFinder func(projectID int, principalID int) (map[common.ProjectRole]bool, error)) *echo.HTTPError {
	switch method {
	case http.MethodGet:
	case http.MethodPatch:
		// PATCH /database/xxx
		if matches := databaseGeneralRouteRegex.FindStringSubmatch(path); len(matches) > 0 {
			databaseID := matches[1]
			databaseIDInt, err := strconv.Atoi(databaseID)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid database id").SetInternal(err)
			}
			oldProjectID, err := projectIDOfDatabase(databaseIDInt)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Cannot find project id for database %d", databaseIDInt)).SetInternal(err)
			}
			oldProjectRoles, err := projectRolesFinder(oldProjectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Cannot find project member ids for project %d", oldProjectID)).SetInternal(err)
			}
			if plan == api.ENTERPRISE {
				if _, ok := oldProjectRoles[common.ProjectOwner]; !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("user is not project owner of project owns the database %d", databaseIDInt))
				}
			} else {
				if len(oldProjectRoles) == 0 {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("user is not a project member of project owns the database %d", databaseIDInt))
				}
			}

			// Workspace developer can only modify the database belongs to the project which he is a member of.
			var databasePatch api.DatabasePatch
			if err := jsonapi.UnmarshalPayload(strings.NewReader(body), &databasePatch); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch database request").SetInternal(err)
			}
			// Workspace developer cannot transfer the project to the project that he is not a member of.
			if databasePatch.ProjectID != nil {
				newProjectRoles, err := projectRolesFinder(*databasePatch.ProjectID, principalID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Cannot find project member ids for project %d", *databasePatch.ProjectID)).SetInternal(err)
				}
				if plan == api.ENTERPRISE {
					if _, ok := newProjectRoles[common.ProjectOwner]; !ok {
						return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("user is not project owner of project want owns the database %d", databaseIDInt))
					}
				} else {
					if len(newProjectRoles) == 0 {
						return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("user is not a project member of project want to owns the database %d", databaseIDInt))
					}
				}
			}
		}
	}
	return nil
}

var sheetRouteRegex = regexp.MustCompile(`^/sheet/(?P<sheetID>\d+)`)
var sheetOrganizeRouteRegex = regexp.MustCompile(`^/sheet/(?P<projectID>\d+)/organize`)

func enforceWorkspaceDeveloperSheetRouteACL(plan api.PlanType, path string, method string, principalID int, projectRolesFinder func(projectID int, principalID int) (map[common.ProjectRole]bool, error), sheetFinder func(sheetID int) (*api.Sheet, error)) *echo.HTTPError {
	if matches := sheetOrganizeRouteRegex.FindStringSubmatch(path); matches != nil {
		sheetID, _ := strconv.Atoi(matches[1])
		sheet, err := sheetFinder(sheetID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}
		if sheet == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Sheet ID not found: %d", sheetID))
		}
		// Creator can always manage her own sheet.
		if sheet.CreatorID == principalID {
			return nil
		}
		switch sheet.Visibility {
		case api.PrivateSheet:
			return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to access private sheet created by other user")
		case api.PublicSheet:
			return nil
		case api.ProjectSheet:
			projectRoles, err := projectRolesFinder(sheet.ProjectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}

			if len(projectRoles) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized, "is not a member of the project containing the sheet")
			}

			if !api.ProjectPermission(api.ProjectPermissionOrganizeSheet, plan, projectRoles) {
				return echo.NewHTTPError(http.StatusUnauthorized, "not have permission to organize the project sheet")
			}
		}
	} else if matches := sheetRouteRegex.FindStringSubmatch(path); matches != nil {
		sheetID, _ := strconv.Atoi(matches[1])
		sheet, err := sheetFinder(sheetID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}
		if sheet == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Sheet ID not found: %d", sheetID))
		}
		// Creator can always manage her own sheet.
		if sheet.CreatorID == principalID {
			return nil
		}
		switch sheet.Visibility {
		case api.PrivateSheet:
			return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to access private sheet created by other user")
		case api.PublicSheet:
			if method == "GET" {
				return nil
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to change public sheet created by other user")
		case api.ProjectSheet:
			projectRoles, err := projectRolesFinder(sheet.ProjectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}

			if len(projectRoles) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized, "is not a member of the project containing the sheet")
			}

			if method == "GET" {
				return nil
			}

			if !api.ProjectPermission(api.ProjectPermissionAdminSheet, plan, projectRoles) {
				return echo.NewHTTPError(http.StatusUnauthorized, "not have permission to change the project sheet")
			}
		}
	}

	return nil
}

var issueStatusRegex = regexp.MustCompile(`^/issue/(?P<issueID>\d+)/status$`)
var issueRouteRegex = regexp.MustCompile(`^/issue/(?P<issueID>\d+)$`)

func enforceWorkspaceDeveloperIssueRouteACL(plan api.PlanType, path string, method string, body string, queryParams url.Values, principalID int, getIssueProjectID func(issueID int) (int, error), projectRolesFinder func(projectID int, principalID int) (map[common.ProjectRole]bool, error)) *echo.HTTPError {
	switch method {
	case http.MethodGet:
		// For /issue route, require the caller principal to be the same as the user in the query.
		// Only /issue and /project route will bring parameter user in the query.
		if userStr := queryParams.Get("user"); userStr != "" {
			userID, err := strconv.Atoi(userStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID").SetInternal(err)
			}
			if principalID != userID {
				return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to list other users' issues")
			}
		} else if matches := issueRouteRegex.FindStringSubmatch(path); len(matches) > 0 {
			issueIDStr := matches[1]
			issueID, err := strconv.Atoi(issueIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid issue ID").SetInternal(err)
			}
			projectID, err := getIssueProjectID(issueID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}
			projectRoles, err := projectRolesFinder(projectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}
			if len(projectRoles) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to retrieve issue that is in the project that the user is not a member of")
			}
		}
	case http.MethodPatch:
		// Workspace developer can only operating the issues if the user is the member of the project that the issue belongs to.
		if matches := issueStatusRegex.FindStringSubmatch(path); len(matches) > 0 {
			issueIDStr := matches[1]
			issueID, err := strconv.Atoi(issueIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid issue ID").SetInternal(err)
			}
			projectID, err := getIssueProjectID(issueID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}
			projectRoles, err := projectRolesFinder(projectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}
			if !api.ProjectPermission(api.ProjectPermissionOrganizeSheet, plan, projectRoles) {
				return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to operate the issue")
			}
		}
	case http.MethodPost:
		if path == "/issue" {
			// Workspace developer can only create issue under the project that the user is the member of.
			var issueCreate api.IssueCreate
			if err := jsonapi.UnmarshalPayload(strings.NewReader(body), &issueCreate); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformed create issue request").SetInternal(err)
			}
			projectRoles, err := projectRolesFinder(issueCreate.ProjectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}
			if !api.ProjectPermission(api.ProjectPermissionChangeDatabase, plan, projectRoles) {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("not allowed to create issues under the project %d", issueCreate.ProjectID))
			}
		}
	}
	return nil
}

func getRetrieveIssueProjectID(ctx context.Context, s *store.Store) func(issueID int) (int, error) {
	return func(issueID int) (int, error) {
		issue, err := s.GetIssueV2(ctx, &store.FindIssueMessage{
			UID: &issueID,
		})
		if err != nil {
			return 0, err
		}
		if issue == nil {
			return 0, errors.Errorf("cannot find issue %d", issueID)
		}
		return issue.Project.UID, nil
	}
}

func getRetrieveDatabaseProjectID(ctx context.Context, s *store.Store) func(databaseID int) (int, error) {
	return func(databaseID int) (int, error) {
		db, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID: &databaseID,
		})
		if err != nil {
			return 0, err
		}
		if db == nil {
			return 0, errors.Errorf("cannot find database %d", databaseID)
		}
		project, err := s.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &db.ProjectID,
		})
		if err != nil {
			return 0, err
		}
		if project == nil {
			return 0, errors.Errorf("cannot find project %s", db.ProjectID)
		}
		return project.UID, nil
	}
}

func isOperatingSelf(ctx context.Context, c echo.Context, s *Server, curPrincipalID int, method string, path string) (bool, error) {
	switch method {
	case http.MethodGet:
		return isGettingSelf(ctx, c, s, curPrincipalID, path)
	case http.MethodPatch, http.MethodDelete:
		return isUpdatingSelf(ctx, c, s, curPrincipalID, path)
	default:
		return false, nil
	}
}

func isGettingSelf(_ context.Context, c echo.Context, _ *Server, curPrincipalID int, path string) (bool, error) {
	if strings.HasPrefix(path, "/inbox/user") {
		userID, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("User ID is not a number: %s", c.Param("userID"))).SetInternal(err)
		}

		return userID == curPrincipalID, nil
	} else if strings.HasPrefix(path, "/bookmark/user") {
		userID, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("User ID is not a number: %s", c.Param("userID"))).SetInternal(err)
		}

		return userID == curPrincipalID, nil
	}

	return false, nil
}

func isUpdatingSelf(ctx context.Context, c echo.Context, s *Server, curPrincipalID int, path string) (bool, error) {
	const defaultErrMsg = "Failed to process authorize request."
	if strings.HasPrefix(path, "/principal") {
		pathPrincipalID := c.Param("principalID")
		if pathPrincipalID != "" {
			return pathPrincipalID == strconv.Itoa(curPrincipalID), nil
		}
	} else if strings.HasPrefix(path, "/activity") {
		if activityIDStr := c.Param("activityID"); activityIDStr != "" {
			activityID, err := strconv.Atoi(activityIDStr)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Activity ID is not a number: %s"+activityIDStr)).SetInternal(err)
			}

			activity, err := s.store.GetActivityByID(ctx, activityID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if activity == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Activity ID not found: %d", activityID))
			}

			return activity.CreatorID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(path, "/bookmark") {
		if bookmarkIDStr := c.Param("bookmarkID"); bookmarkIDStr != "" {
			bookmarkID, err := strconv.Atoi(bookmarkIDStr)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bookmark ID is not a number: %s"+bookmarkIDStr)).SetInternal(err)
			}

			bookmark, err := s.store.GetBookmarkV2(ctx, bookmarkID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if bookmark == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Bookmark ID not found: %d", bookmarkID)).SetInternal(err)
			}

			return bookmark.CreatorUID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(path, "/inbox") {
		if inboxIDStr := c.Param("inboxID"); inboxIDStr != "" {
			inboxID, err := strconv.Atoi(inboxIDStr)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Inbox ID is not a number: %s", inboxIDStr)).SetInternal(err)
			}

			inbox, err := s.store.GetInboxByID(ctx, inboxID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if inbox == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Inbox ID not found: %d", inboxID)).SetInternal(err)
			}

			return inbox.ReceiverID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(path, "/sheet") {
		if idStr := c.Param("id"); idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Sheet ID is not a number: %s", idStr)).SetInternal(err)
			}

			sheetFind := &api.SheetFind{
				ID: &id,
			}
			sheet, err := s.store.GetSheet(ctx, sheetFind, curPrincipalID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if sheet == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Sheet ID not found: %d", id)).SetInternal(err)
			}

			return sheet.CreatorID == curPrincipalID, nil
		}
	}
	return false, nil
}
