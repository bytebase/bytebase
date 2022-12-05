package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	openAPIV1 "github.com/bytebase/bytebase/api/v1"
)

func (s *Server) registerOpenAPIRoutesForInstance(g *echo.Group) {
	g.POST("/instance", s.createInstanceByOpenAPI)
	g.GET("/instance", s.listInstance)
	g.GET("/instance/:instanceID", s.getInstanceByID)
	g.PATCH("/instance/:instanceID", s.updateInstanceByOpenAPI)
	g.DELETE("/instance/:instanceID", s.deleteInstanceByOpenAPI)
}

func (s *Server) listInstance(c echo.Context) error {
	ctx := c.Request().Context()
	rowStatus := api.Normal
	instanceList, err := s.store.FindInstance(ctx, &api.InstanceFind{
		RowStatus: &rowStatus,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch instance list").SetInternal(err)
	}

	response := []*openAPIV1.Instance{}
	for _, instance := range instanceList {
		response = append(response, convertToOpenAPIInstance(instance))
	}

	return c.JSON(http.StatusOK, response)
}

func (s *Server) createInstanceByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}

	instanceCreate := &openAPIV1.InstanceCreate{}
	if err := json.Unmarshal(body, instanceCreate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed create instance request").SetInternal(err)
	}

	environmentName := instanceCreate.Environment
	envList, err := s.store.FindEnvironment(ctx, &api.EnvironmentFind{
		Name: &environmentName,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find environment by name: %s", instanceCreate.Environment)).SetInternal(err)
	}
	if len(envList) != 1 {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Should only find one environment with name: %s", instanceCreate.Environment))
	}

	instance, err := s.createInstance(ctx, &api.InstanceCreate{
		CreatorID:     c.Get(getPrincipalIDContextKey()).(int),
		EnvironmentID: envList[0].ID,
		Name:          instanceCreate.Name,
		Engine:        instanceCreate.Engine,
		ExternalLink:  instanceCreate.ExternalLink,
		Host:          instanceCreate.Host,
		Port:          instanceCreate.Port,
		Database:      instanceCreate.Database,
		Username:      instanceCreate.Username,
		Password:      instanceCreate.Password,
		SslCa:         instanceCreate.SslCa,
		SslCert:       instanceCreate.SslCert,
		SslKey:        instanceCreate.SslKey,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, convertToOpenAPIInstance(instance))
}

func (s *Server) updateInstanceByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("instanceID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}

	instancePatch := &openAPIV1.InstancePatch{}
	if err := json.Unmarshal(body, instancePatch); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch instance request").SetInternal(err)
	}

	instance, err := s.updateInstance(ctx, &api.InstancePatch{
		ID:           id,
		UpdaterID:    c.Get(getPrincipalIDContextKey()).(int),
		Name:         instancePatch.Name,
		ExternalLink: instancePatch.ExternalLink,
		Host:         instancePatch.Host,
		Port:         instancePatch.Port,
		Database:     instancePatch.Database,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, convertToOpenAPIInstance(instance))
}

func (s *Server) deleteInstanceByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("instanceID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
	}

	rowStatus := string(api.Archived)
	if _, err := s.updateInstance(ctx, &api.InstancePatch{
		ID:        id,
		UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		RowStatus: &rowStatus,
	}); err != nil {
		return err
	}

	return c.String(http.StatusOK, "ok")
}

func (s *Server) getInstanceByID(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("instanceID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
	}

	instance, err := s.store.GetInstanceByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
	}
	if instance == nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
	}

	return c.JSON(http.StatusOK, convertToOpenAPIInstance(instance))
}

func convertToOpenAPIInstance(instance *api.Instance) *openAPIV1.Instance {
	return &openAPIV1.Instance{
		ID:            instance.ID,
		Environment:   instance.Environment.Name,
		Name:          instance.Name,
		Engine:        instance.Engine,
		EngineVersion: instance.EngineVersion,
		ExternalLink:  instance.ExternalLink,
		Host:          instance.Host,
		Port:          instance.Port,
		Database:      instance.Database,
		Username:      instance.Username,
	}
}
