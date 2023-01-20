package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/pkg/errors"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/store"
)

const (
	roleContextKey = "role"
)

func getRoleContextKey() string {
	return roleContextKey
}

var projectGeneralRouteRegex = regexp.MustCompile(`^/project/(?P<projectID>\d+)`)
var projectMemberRouteRegex = regexp.MustCompile(`^/project/(?P<projectID>\d+)/member`)
var projectSyncSheetRouteRegex = regexp.MustCompile(`^/project/(?P<projectID>\d+)/sync-sheet`)

func enforceWorkspaceDeveloperProjectRouteACL(plan api.PlanType, path string, method string, quaryParams url.Values, principalID int, roleFinder func(projectID int, principalID int) (common.ProjectRole, error)) *echo.HTTPError {
	var projectID int
	var permission api.ProjectPermissionType
	var permissionErrMsg string
	if method == "GET" {
		if path == "/project" {
			userIDStr := quaryParams.Get("user")
			if userIDStr == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to fetch all project list")
			}
			if strconv.Itoa(principalID) != userIDStr {
				return echo.NewHTTPError(http.StatusUnauthorized, "not allowed to fetch projects from other user")
			}
		}
		// For /project/xxx subroutes, since all projects are public, we don't enforce ACL.
	} else {
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
		role, err := roleFinder(projectID, principalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		if role == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "is not a member of the project")
		}

		if !api.ProjectPermission(permission, plan, role) {
			return echo.NewHTTPError(http.StatusUnauthorized, permissionErrMsg)
		}
	}

	return nil
}

var sheetRouteRegex = regexp.MustCompile(`^/sheet/(?P<sheetID>\d+)`)
var sheetOrganizeRouteRegex = regexp.MustCompile(`^/sheet/(?P<projectID>\d+)/organize`)

func enforceWorkspaceDeveloperSheetRouteACL(plan api.PlanType, path string, method string, principalID int, roleFinder func(projectID int, principalID int) (common.ProjectRole, error), sheetFinder func(sheetID int) (*api.Sheet, error)) *echo.HTTPError {
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
			role, err := roleFinder(sheet.ProjectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}

			if role == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "is not a member of the project containing the sheet")
			}

			if !api.ProjectPermission(api.ProjectPermissionOrganizeSheet, plan, role) {
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
			role, err := roleFinder(sheet.ProjectID, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
			}

			if role == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "is not a member of the project containing the sheet")
			}

			if method == "GET" {
				return nil
			}

			if !api.ProjectPermission(api.ProjectPermissionAdminSheet, plan, role) {
				return echo.NewHTTPError(http.StatusUnauthorized, "not have permission to change the project sheet")
			}
		}
	}

	return nil
}

func aclMiddleware(s *Server, pathPrefix string, ce *casbin.Enforcer, next echo.HandlerFunc, readonly bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		path := strings.TrimPrefix(c.Request().URL.Path, pathPrefix)

		// Skips auth, actuator
		if common.HasPrefixes(path, "/auth", "/actuator", "/oauth") {
			return next(c)
		}

		// Skips OpenAPI SQL endpoint
		if common.HasPrefixes(c.Path(), fmt.Sprintf("%s/sql", openAPIPrefix)) {
			return next(c)
		}

		method := c.Request().Method
		// Skip GET /subscription request
		if common.HasPrefixes(path, "/subscription") && method == "GET" {
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
			roleFinder := func(projectID int, principalID int) (common.ProjectRole, error) {
				policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectID})
				if err != nil {
					return "", err
				}
				for _, binding := range policy.Bindings {
					for _, member := range binding.Members {
						if member.ID == principalID {
							return common.ProjectRole(binding.Role), nil
						}
					}
				}
				return "", nil
			}

			sheetFinder := func(sheetID int) (*api.Sheet, error) {
				sheetFind := &api.SheetFind{
					ID: &sheetID,
				}
				return s.store.GetSheet(ctx, sheetFind, principalID)
			}

			if strings.HasPrefix(path, "/project") {
				aclErr = enforceWorkspaceDeveloperProjectRouteACL(s.licenseService.GetEffectivePlan(), path, method, c.QueryParams(), principalID, roleFinder)
			} else if strings.HasPrefix(path, "/sheet") {
				aclErr = enforceWorkspaceDeveloperSheetRouteACL(s.licenseService.GetEffectivePlan(), path, method, principalID, roleFinder, sheetFinder)
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

			bookmark, err := s.store.GetBookmarkByID(ctx, bookmarkID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if bookmark == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Bookmark ID not found: %d", bookmarkID)).SetInternal(err)
			}

			return bookmark.CreatorID == curPrincipalID, nil
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
