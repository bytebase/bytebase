package server

import (
	"context"
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
	"github.com/bytebase/bytebase/store"
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
	find := &api.EnvironmentFind{
		RowStatus: &rowStatus,
	}
	if name := c.QueryParam("name"); name != "" {
		find.Name = &name
	}

	envList, err := s.store.FindEnvironment(ctx, find)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
	}

	response := []*openAPIV1.Environment{}
	for _, env := range envList {
		env, err := s.convertToOpenAPIEnvironment(ctx, env)
		if err != nil {
			return err
		}
		response = append(response, env)
	}

	return c.JSON(http.StatusOK, response)
}

func (s *Server) createEnvironmentByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}

	upsert := &openAPIV1.EnvironmentUpsert{}
	if err := json.Unmarshal(body, upsert); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed create environment request").SetInternal(err)
	}
	if upsert.Name == nil || *upsert.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Environment name is required")
	}

	if err := s.validateEnvironmentPolicy(api.PolicyTypePipelineApproval, upsert.PipelineApprovalPolicy); err != nil {
		return err
	}
	if err := s.validateEnvironmentPolicy(api.PolicyTypeBackupPlan, upsert.BackupPlanPolicy); err != nil {
		return err
	}
	if err := s.validateEnvironmentPolicy(api.PolicyTypeEnvironmentTier, upsert.EnvironmentTierPolicy); err != nil {
		return err
	}

	creatorID := c.Get(getPrincipalIDContextKey()).(int)

	env, err := s.createEnvironment(ctx, &store.EnvironmentCreate{
		CreatorID:              creatorID,
		EnvironmentTierPolicy:  upsert.EnvironmentTierPolicy,
		PipelineApprovalPolicy: upsert.PipelineApprovalPolicy,
		BackupPlanPolicy:       upsert.BackupPlanPolicy,
		Name:                   *upsert.Name,
		Order:                  upsert.Order,
	})
	if err != nil {
		return err
	}

	res, err := s.convertToOpenAPIEnvironment(ctx, env)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
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

	res, err := s.convertToOpenAPIEnvironment(ctx, env)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
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

	upsert := &openAPIV1.EnvironmentUpsert{}
	if err := json.Unmarshal(body, upsert); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch environment request").SetInternal(err)
	}

	if err := s.validateEnvironmentPolicy(api.PolicyTypePipelineApproval, upsert.PipelineApprovalPolicy); err != nil {
		return err
	}
	if err := s.validateEnvironmentPolicy(api.PolicyTypeBackupPlan, upsert.BackupPlanPolicy); err != nil {
		return err
	}
	if err := s.validateEnvironmentPolicy(api.PolicyTypeEnvironmentTier, upsert.EnvironmentTierPolicy); err != nil {
		return err
	}

	env, err := s.updateEnvironment(ctx, &store.EnvironmentPatch{
		ID:                     id,
		UpdaterID:              c.Get(getPrincipalIDContextKey()).(int),
		EnvironmentTierPolicy:  upsert.EnvironmentTierPolicy,
		PipelineApprovalPolicy: upsert.PipelineApprovalPolicy,
		BackupPlanPolicy:       upsert.BackupPlanPolicy,
		Name:                   upsert.Name,
		Order:                  upsert.Order,
	})
	if err != nil {
		return err
	}

	res, err := s.convertToOpenAPIEnvironment(ctx, env)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
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
	if _, err := s.updateEnvironment(ctx, &store.EnvironmentPatch{
		ID:        id,
		UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		RowStatus: &rowStatus,
		Name:      &name,
	}); err != nil {
		return err
	}

	return c.String(http.StatusOK, "ok")
}

func (s *Server) convertToOpenAPIEnvironment(ctx context.Context, env *api.Environment) (*openAPIV1.Environment, error) {
	backupPolicy, err := s.store.GetBackupPlanPolicyByEnvID(ctx, env.ID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find the backup policy for environment %d", env.ID)).SetInternal(err)
	}
	pipelineApprovalPolicy, err := s.store.GetPipelineApprovalPolicy(ctx, env.ID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find the pipeline approval policy for environment %d", env.ID)).SetInternal(err)
	}
	environmentTierPolicy, err := s.store.GetEnvironmentTierPolicyByEnvID(ctx, env.ID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find the environment tier policy for environment %d", env.ID)).SetInternal(err)
	}

	return &openAPIV1.Environment{
		ID:                     env.ID,
		Name:                   env.Name,
		Order:                  env.Order,
		BackupPlanPolicy:       backupPolicy,
		PipelineApprovalPolicy: pipelineApprovalPolicy,
		EnvironmentTierPolicy:  environmentTierPolicy,
	}, nil
}

func (s *Server) validateEnvironmentPolicy(policyType api.PolicyType, policy interface{}) error {
	if policy == nil {
		return nil
	}

	bytes, err := json.Marshal(policy)
	payload := string(bytes)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
	}

	policyUpsert := &api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		Type:         api.PolicyTypePipelineApproval,
		Payload:      &payload,
	}

	if err := s.hasAccessToUpsertPolicy(policyUpsert); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err.Error()).SetInternal(err)
	}

	if err := api.ValidatePolicy(api.PolicyResourceTypeEnvironment, policyType, &payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid %s policy: %s", policyType, payload)).SetInternal(err)
	}

	return nil
}
