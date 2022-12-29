package server

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/store"
)

func (s *Server) registerEnvironmentRoutes(g *echo.Group) {
	g.POST("/environment", func(c echo.Context) error {
		ctx := c.Request().Context()
		envCreate := &api.EnvironmentCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, envCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create environment request").SetInternal(err)
		}

		env, err := s.createEnvironment(ctx, &store.EnvironmentCreate{
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
			Name:      envCreate.Name,
			Order:     envCreate.Order,
		})
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, env); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create environment response").SetInternal(err)
		}
		return nil
	})

	g.GET("/environment", func(c echo.Context) error {
		ctx := c.Request().Context()
		envFind := &api.EnvironmentFind{}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			envFind.RowStatus = &rowStatus
		}
		envList, err := s.store.FindEnvironment(ctx, envFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, envList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal environment list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/environment/:environmentID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("environmentID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Environment ID is not a number: %s", c.Param("environmentID"))).SetInternal(err)
		}

		envPatch := &api.EnvironmentPatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, envPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch environment request").SetInternal(err)
		}

		env, err := s.updateEnvironment(ctx, &store.EnvironmentPatch{
			ID:        id,
			RowStatus: envPatch.RowStatus,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
			Name:      envPatch.Name,
			Order:     envPatch.Order,
		})
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, env); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal environment ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/environment/reorder", func(c echo.Context) error {
		ctx := c.Request().Context()
		patchList, err := jsonapi.UnmarshalManyPayload(c.Request().Body, reflect.TypeOf(new(api.EnvironmentPatch)))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed environment reorder request").SetInternal(err)
		}

		for _, item := range patchList {
			envPatch, ok := item.(*api.EnvironmentPatch)
			if !ok {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformed environment reorder request").SetInternal(errors.New("failed to convert request item to *api.EnvironmentPatch"))
			}

			if _, err := s.store.PatchEnvironment(ctx, &store.EnvironmentPatch{
				ID:        envPatch.ID,
				UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
				Order:     envPatch.Order,
			}); err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", envPatch.ID))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", envPatch.ID)).SetInternal(err)
			}
		}

		envList, err := s.store.FindEnvironment(ctx, &api.EnvironmentFind{})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list for reorder").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, envList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal environment reorder response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/environment/:environmentID/backup-setting", func(c echo.Context) error {
		ctx := c.Request().Context()
		envID, err := strconv.Atoi(c.Param("environmentID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("environmentID"))).SetInternal(err)
		}

		backupSettingUpsert := &api.BackupSettingUpsert{
			EnvironmentID: envID,
			UpdaterID:     c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupSettingUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set backup setting request").SetInternal(err)
		}

		if err := s.store.UpdateBackupSettingsInEnvironment(ctx, backupSettingUpsert); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid backup setting").SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set backup setting").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) createEnvironment(ctx context.Context, create *store.EnvironmentCreate) (*api.Environment, error) {
	normalRowStatus := api.Normal
	envFind := &api.EnvironmentFind{
		RowStatus: &normalRowStatus,
	}
	envList, err := s.store.FindEnvironment(ctx, envFind)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find environment list").SetInternal(err)
	}

	maximumEnvironmentLimit := s.licenseService.GetPlanLimitValue(api.PlanLimitMaximumEnvironment)
	if int64(len(envList)) >= maximumEnvironmentLimit {
		return nil, echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Current plan can create up to %d environments.", maximumEnvironmentLimit))
	}

	if err := api.IsValidEnvironmentName(create.Name); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid environment name, please visit https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files#file-path-template?source=console to get more detail.").SetInternal(err)
	}

	env, err := s.store.CreateEnvironment(ctx, create)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil, echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Environment name already exists: %s", create.Name))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create environment").SetInternal(err)
	}

	return env, nil
}

func (s *Server) updateEnvironment(ctx context.Context, patch *store.EnvironmentPatch) (*api.Environment, error) {
	if v := patch.Name; v != nil {
		if err := api.IsValidEnvironmentName(*v); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid environment name, please visit https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files#file-path-template?source=console to get more detail.").SetInternal(err)
		}
	}

	// Ensure the environment has no instance before it's archived.
	if v := patch.RowStatus; v != nil && *v == string(api.Archived) {
		normalStatus := api.Normal
		instances, err := s.store.FindInstance(ctx, &api.InstanceFind{
			EnvironmentID: &patch.ID,
			RowStatus:     &normalStatus,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Errorf("failed to find instances in the environment %d", patch.ID)).SetInternal(err)
		}
		if len(instances) > 0 {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "You should archive all instances under the environment before archiving the environment.")
		}
	}

	env, err := s.store.PatchEnvironment(ctx, patch)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", patch.ID))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", patch.ID)).SetInternal(err)
	}

	return env, nil
}
