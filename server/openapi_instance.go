package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	openAPIV1 "github.com/bytebase/bytebase/api/v1"
	"github.com/bytebase/bytebase/store"
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
	find := &api.InstanceFind{
		RowStatus: &rowStatus,
	}
	if name := c.QueryParam("name"); name != "" {
		find.Name = &name
	}

	instanceList, err := s.store.FindInstance(ctx, find)
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

	dataSourceCreateList, err := convertToAPIDataSouceList(instanceCreate.DataSourceList)
	if err != nil {
		return err
	}
	if len(dataSourceCreateList) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Should specific at least one data source.")
	}

	instance, err := s.createInstance(ctx, &store.InstanceCreate{
		CreatorID:      c.Get(getPrincipalIDContextKey()).(int),
		EnvironmentID:  envList[0].ID,
		DataSourceList: dataSourceCreateList,
		Name:           instanceCreate.Name,
		Engine:         instanceCreate.Engine,
		ExternalLink:   instanceCreate.ExternalLink,
		Host:           instanceCreate.Host,
		Port:           instanceCreate.Port,
		Database:       instanceCreate.Database,
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

	patch := &store.InstancePatch{
		ID:           id,
		UpdaterID:    c.Get(getPrincipalIDContextKey()).(int),
		Name:         instancePatch.Name,
		ExternalLink: instancePatch.ExternalLink,
		Host:         instancePatch.Host,
		Port:         instancePatch.Port,
		Database:     instancePatch.Database,
	}

	dataSourceCreateList, err := convertToAPIDataSouceList(instancePatch.DataSourceList)
	if err != nil {
		return err
	}
	patch.DataSourceList = dataSourceCreateList

	instance, err := s.updateInstance(ctx, patch)
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

	instance, err := s.store.GetInstanceByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
	}
	if instance == nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance not found by id: %d", id))
	}

	name := fmt.Sprintf("archived_%s_%d", instance.Name, time.Now().Unix())
	rowStatus := string(api.Archived)
	if _, err := s.updateInstance(ctx, &store.InstancePatch{
		ID:        id,
		UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		RowStatus: &rowStatus,
		Name:      &name,
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
	dataSourceList := []*openAPIV1.DataSource{}
	for _, dataSource := range instance.DataSourceList {
		dataSourceList = append(dataSourceList, &openAPIV1.DataSource{
			ID:           dataSource.ID,
			DatabaseID:   dataSource.DatabaseID,
			Name:         dataSource.Name,
			Type:         dataSource.Type,
			Username:     dataSource.Username,
			HostOverride: dataSource.HostOverride,
			PortOverride: dataSource.PortOverride,
		})
	}

	return &openAPIV1.Instance{
		ID:             instance.ID,
		Environment:    instance.Environment.Name,
		Name:           instance.Name,
		Engine:         instance.Engine,
		EngineVersion:  instance.EngineVersion,
		ExternalLink:   instance.ExternalLink,
		Host:           instance.Host,
		Port:           instance.Port,
		Database:       instance.Database,
		DataSourceList: dataSourceList,
	}
}

func convertToAPIDataSouceList(dataSourceList []*openAPIV1.DataSourceCreate) ([]*api.DataSourceCreate, error) {
	var res []*api.DataSourceCreate

	dataSourceNameMap := map[string]bool{}
	dataSourceTypeMap := map[api.DataSourceType]bool{}
	for _, dataSource := range dataSourceList {
		if dataSourceNameMap[dataSource.Name] {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Duplicate data source name %s. The data source name should be unique", dataSource.Name))
		}
		if dataSourceTypeMap[dataSource.Type] {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Duplicate data source type %s. The data source type should be unique", dataSource.Name))
		}
		dataSourceNameMap[dataSource.Name] = true
		dataSourceTypeMap[dataSource.Type] = true

		switch dataSource.Type {
		case api.Admin:
			res = append(res, &api.DataSourceCreate{
				Name:     dataSource.Name,
				Type:     api.Admin,
				Username: dataSource.Username,
				Password: dataSource.Password,
				SslCa:    dataSource.SslCa,
				SslCert:  dataSource.SslCert,
				SslKey:   dataSource.SslKey,
			})
		case api.RO:
			res = append(res, &api.DataSourceCreate{
				Name:         dataSource.Name,
				Type:         api.RO,
				Username:     dataSource.Username,
				Password:     dataSource.Password,
				SslCa:        dataSource.SslCa,
				SslCert:      dataSource.SslCert,
				SslKey:       dataSource.SslKey,
				HostOverride: dataSource.HostOverride,
				PortOverride: dataSource.PortOverride,
			})
		default:
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The data source type %s is not supported", dataSource.Type))
		}
	}

	return res, nil
}
