package server

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
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
			return echo.NewHTTPError(http.StatusForbidden).SetInternal(
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

			if strings.HasPrefix(path, "/project") {
				aclErr = enforceWorkspaceDeveloperProjectRouteACL(s.licenseService.GetEffectivePlan(), path, method, principalID, projectRolesFinder)
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
			return echo.NewHTTPError(http.StatusForbidden, "is not a member of the project")
		}

		if !api.ProjectPermission(permission, plan, projectRoles) {
			return echo.NewHTTPError(http.StatusForbidden, permissionErrMsg)
		}
	}

	return nil
}

func isOperatingSelf(ctx context.Context, c echo.Context, s *Server, curPrincipalID int, method string, path string) (bool, error) {
	switch method {
	case http.MethodPatch, http.MethodDelete:
		return isUpdatingSelf(ctx, c, s, curPrincipalID, path)
	default:
		return false, nil
	}
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

			activity, err := s.store.GetActivityV2(ctx, activityID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if activity == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Activity ID not found: %d", activityID))
			}

			return activity.CreatorID == curPrincipalID, nil
		}
	}
	return false, nil
}
