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
	"github.com/bytebase/bytebase/common"
)

func (s *Server) registerOpenAPIRoutesForEnvironment(g *echo.Group) {
	g.GET("/environment", s.listEnvironment)
	g.POST("/environment", s.createEnvironmentByOpenAPI)
	g.GET("/environment/:environmentID", s.getEnvironmentByID)
	g.PATCH("/environment/:environmentID", s.updateEnvironmentByOpenAPI)
	g.DELETE("/environment/:environmentID", s.deleteEnvironmentByOpenAPI)
}

func (s *Server) listEnvironment(c echo.Context) error {
	ctx := c.Request().Context()
	rowStatus := api.Normal
	envList, err := s.store.FindEnvironment(ctx, &api.EnvironmentFind{
		RowStatus: &rowStatus,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
	}

	var response []*openAPIV1.Environment
	for _, env := range envList {
		response = append(response, convertToOpenAPIEnvironment(env))
	}

	return c.JSON(http.StatusOK, response)
}

func (s *Server) createEnvironmentByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}

	envCreate := &openAPIV1.EnvironmentCreate{}
	if err := json.Unmarshal(body, envCreate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed create environment request").SetInternal(err)
	}

	env, err := s.createEnvironment(ctx, &api.EnvironmentCreate{
		CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		Name:      envCreate.Name,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, convertToOpenAPIEnvironment(env))
}

func (s *Server) getEnvironmentByID(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("environmentID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Environment ID is not a number: %s", c.Param("environmentID"))).SetInternal(err)
	}

	env, err := s.store.GetEnvironmentByID(ctx, id)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Cannot found environment").SetInternal(err)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find environment").SetInternal(err)
	}

	return c.JSON(http.StatusOK, convertToOpenAPIEnvironment(env))
}

func (s *Server) updateEnvironmentByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("environmentID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Environment ID is not a number: %s", c.Param("environmentID"))).SetInternal(err)
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}

	envPatch := &openAPIV1.EnvironmentPatch{}
	if err := json.Unmarshal(body, envPatch); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch environment request").SetInternal(err)
	}

	env, err := s.updateEnvironment(ctx, &api.EnvironmentPatch{
		ID:        id,
		UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		Name:      envPatch.Name,
		Order:     envPatch.Order,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, convertToOpenAPIEnvironment(env))
}

func (s *Server) deleteEnvironmentByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("environmentID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Environment ID is not a number: %s", c.Param("environmentID"))).SetInternal(err)
	}

	env, err := s.store.GetEnvironmentByID(ctx, id)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Cannot found environment").SetInternal(err)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find environment").SetInternal(err)
	}

	rowStatus := string(api.Archived)
	name := fmt.Sprintf("archived_%s_%d", env.Name, time.Now().Unix())
	if _, err := s.updateEnvironment(ctx, &api.EnvironmentPatch{
		ID:        id,
		UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		RowStatus: &rowStatus,
		Name:      &name,
	}); err != nil {
		return err
	}

	return c.String(http.StatusOK, "ok")
}

func convertToOpenAPIEnvironment(env *api.Environment) *openAPIV1.Environment {
	return &openAPIV1.Environment{
		ID:    env.ID,
		Name:  env.Name,
		Order: env.Order,
	}
}
