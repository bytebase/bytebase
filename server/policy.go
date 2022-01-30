package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

// hasAccessToUpdatePolicy checks if user can access to policy control feature.
// return nil if user has access.
func (s *Server) hasAccessToUpdatePolicy(pType api.PolicyType) error {
	if pType == api.PolicyTypeBackupPlan && !s.feature(api.FeatureBackupPolicy) {
		return fmt.Errorf("Access denied for feature %s", api.FeatureBackupPolicy)
	}
	if pType == api.PolicyTypePipelineApproval && !s.feature(api.FeatureApprovalPolicy) {
		return fmt.Errorf("Access denied for feature %s", api.FeatureApprovalPolicy)
	}

	return nil
}

func (s *Server) registerPolicyRoutes(g *echo.Group) {
	g.PATCH("/policy/environment/:environmentID", func(c echo.Context) error {
		ctx := context.Background()
		environmentID, err := strconv.Atoi(c.Param("environmentID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("environmentID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		policyUpsert := &api.PolicyUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set policy request").SetInternal(err)
		}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicy(pType, ""); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy type: %q", pType)).SetInternal(err)
		}
		if err := s.hasAccessToUpdatePolicy(pType); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err)
		}

		policyUpsert.EnvironmentID = environmentID
		policyUpsert.Type = pType
		policyUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		policy, err := s.PolicyService.UpsertPolicy(ctx, policyUpsert)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set policy for type %q", pType)).SetInternal(err)
		}

		if err := s.composePolicyRelationship(ctx, policy); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set policy response").SetInternal(err)
		}
		return nil
	})

	g.GET("/policy/environment/:environmentID", func(c echo.Context) error {
		ctx := context.Background()
		environmentID, err := strconv.Atoi(c.Param("environmentID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("environmentID is not a number: %s", c.Param("id"))).SetInternal(err)
		}
		policyFind := &api.PolicyFind{}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicy(pType, ""); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy type: %q", pType)).SetInternal(err)
		}
		policyFind.Type = &pType
		policyFind.EnvironmentID = &environmentID

		policy, err := s.PolicyService.FindPolicy(ctx, policyFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get policy for type %q", pType)).SetInternal(err)
		}

		if err := s.composePolicyRelationship(ctx, policy); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get policy response: %v", pType)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composePolicyRelationship(ctx context.Context, policy *api.Policy) error {
	var err error

	policy.Creator, err = s.composePrincipalByID(ctx, policy.CreatorID)
	if err != nil {
		return err
	}

	policy.Updater, err = s.composePrincipalByID(ctx, policy.UpdaterID)
	if err != nil {
		return err
	}

	policy.Environment, err = s.composeEnvironmentByID(ctx, policy.EnvironmentID)
	if err != nil {
		return err
	}

	return nil
}
