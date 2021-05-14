package server

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerInstanceRoutes(g *echo.Group) {
	g.POST("/instance", func(c echo.Context) error {
		instanceCreate := &api.InstanceCreate{WorkspaceId: api.DEFAULT_WORKPSACE_ID}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instanceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create instance request").SetInternal(err)
		}

		instanceCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		instance, err := s.InstanceService.CreateInstance(context.Background(), instanceCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Instance name already exists: %s", instanceCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create instance").SetInternal(err)
		}

		if err := s.AddInstanceRelationship(context.Background(), instance, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create instance response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance", func(c echo.Context) error {
		workspaceId := api.DEFAULT_WORKPSACE_ID
		instanceFind := &api.InstanceFind{
			WorkspaceId: &workspaceId,
		}
		list, err := s.InstanceService.FindInstanceList(context.Background(), instanceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch instance list").SetInternal(err)
		}

		for _, instance := range list {
			if err := s.AddInstanceRelationship(context.Background(), instance, c.Get(getIncludeKey()).([]string)); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal instance list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		instance, err := s.FindInstanceById(context.Background(), id, c.Get(getIncludeKey()).([]string))
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/instance/:instanceId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		instancePatch := &api.InstancePatch{
			ID:          id,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instancePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch instance request").SetInternal(err)
		}

		var instance *api.Instance
		if instancePatch.RowStatus != nil || instancePatch.Name != nil || instancePatch.ExternalLink != nil || instancePatch.Host != nil || instancePatch.Port != nil {
			instance, err = s.InstanceService.PatchInstanceByID(context.Background(), instancePatch)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch instance ID: %v", id)).SetInternal(err)
			}
		} else if instancePatch.Username != nil || instancePatch.Password != nil {
			instanceFind := &api.InstanceFind{
				ID: &id,
			}
			instance, err = s.InstanceService.FindInstance(context.Background(), instanceFind)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
			}

			dataSourceType := api.Admin
			dataSourceFind := &api.DataSourceFind{
				InstanceId: &instance.ID,
				Type:       &dataSourceType,
			}
			adminDataSource, err := s.DataSourceService.FindDataSource(context.Background(), dataSourceFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch data source for instance: %v", instance.Name)).SetInternal(err)
			}

			dataSourcePatch := &api.DataSourcePatch{
				ID:          adminDataSource.ID,
				WorkspaceId: api.DEFAULT_WORKPSACE_ID,
				UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
				Username:    instancePatch.Username,
				Password:    instancePatch.Password,
			}
			_, err = s.DataSourceService.PatchDataSource(context.Background(), dataSourcePatch)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch data source for instance: %v", instance.Name)).SetInternal(err)
			}
		}

		if err := s.AddInstanceRelationship(context.Background(), instance, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) FindInstanceById(ctx context.Context, id int, incluedList []string) (*api.Instance, error) {
	instanceFind := &api.InstanceFind{
		ID: &id,
	}
	instance, err := s.InstanceService.FindInstance(context.Background(), instanceFind)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
			return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
	}

	if err := s.AddInstanceRelationship(ctx, instance, incluedList); err != nil {
		return nil, err
	}

	return instance, nil
}

func (s *Server) AddInstanceRelationship(ctx context.Context, instance *api.Instance, includeList []string) error {
	var err error
	if sort.SearchStrings(includeList, "environment") >= 0 {
		environmentFind := &api.EnvironmentFind{
			ID: &instance.EnvironmentId,
		}
		instance.Environment, err = s.EnvironmentService.FindEnvironment(context.Background(), environmentFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch environment for instance: %v", instance.Name)).SetInternal(err)
		}
	}

	if sort.SearchStrings(includeList, "dataSource") >= 0 {
		dataSourceFind := &api.DataSourceFind{
			InstanceId: &instance.ID,
		}
		instance.DataSourceList, err = s.DataSourceService.FindDataSourceList(context.Background(), dataSourceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch data source for instance: %v", instance.Name)).SetInternal(err)
		}
	}

	return nil
}
