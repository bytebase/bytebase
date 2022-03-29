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
func (s *Server) hasAccessToUpsertPolicy(policyUpsert *api.PolicyUpsert) error {
	defaultPolicy, err := api.GetDefaultPolicy(policyUpsert.Type)
	if err != nil {
		return err
	}
	switch policyUpsert.Type {
	case api.PolicyTypePipelineApproval:
		if policyUpsert.Payload != defaultPolicy && !s.feature(api.FeatureApprovalPolicy) {
			return fmt.Errorf(api.FeatureApprovalPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeBackupPlan:
		if policyUpsert.Payload != defaultPolicy && !s.feature(api.FeatureBackupPolicy) {
			return fmt.Errorf(api.FeatureBackupPolicy.AccessErrorMessage())
		}
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

		policyUpsert.EnvironmentID = environmentID
		policyUpsert.Type = pType
		policyUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if err := s.hasAccessToUpsertPolicy(policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err.Error()).SetInternal(err)
		}

		policyRaw, err := s.PolicyService.UpsertPolicy(ctx, policyUpsert)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set policy for type %q", pType)).SetInternal(err)
		}

		policy, err := s.composePolicyRelationship(ctx, policyRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose policy relationship for type %q", pType)).SetInternal(err)
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

		policyRaw, err := s.PolicyService.FindPolicy(ctx, policyFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get policy for type %q", pType)).SetInternal(err)
		}

		policy, err := s.composePolicyRelationship(ctx, policyRaw)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get policy response: %v", pType)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composePolicyRelationship(ctx context.Context, raw *api.PolicyRaw) (*api.Policy, error) {
	policy := raw.ToPolicy()

	creator, err := s.store.GetPrincipalByID(ctx, policy.CreatorID)
	if err != nil {
		return nil, err
	}
	policy.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, policy.UpdaterID)
	if err != nil {
		return nil, err
	}
	policy.Updater = updater

	env, err := s.store.GetEnvironmentByID(ctx, policy.EnvironmentID)
	if err != nil {
		return nil, err
	}
	policy.Environment = env

	return policy, nil
}
