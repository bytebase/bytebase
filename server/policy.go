package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// hasAccessToUpdatePolicy checks if user can access to policy control feature.
// return nil if user has access.
func (s *Server) hasAccessToUpsertPolicy(policyUpsert *api.PolicyUpsert) error {
	// nil payload means user doesn't update the payload field
	if policyUpsert.Payload == nil {
		return nil
	}

	defaultPolicy, err := api.GetDefaultPolicy(policyUpsert.Type)
	if err != nil {
		return err
	}

	if defaultPolicy == *policyUpsert.Payload {
		return nil
	}
	switch policyUpsert.Type {
	case api.PolicyTypePipelineApproval:
		if !s.feature(api.FeatureApprovalPolicy) {
			return fmt.Errorf(api.FeatureApprovalPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeBackupPlan:
		if !s.feature(api.FeatureBackupPolicy) {
			return fmt.Errorf(api.FeatureBackupPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeSQLReview:
		if !s.feature(api.FeatureSQLReviewPolicy) {
			return fmt.Errorf(api.FeatureSQLReviewPolicy.AccessErrorMessage())
		}
	}
	return nil
}

func (s *Server) registerPolicyRoutes(g *echo.Group) {
	g.PATCH("/policy/environment/:environmentID", func(c echo.Context) error {
		ctx := c.Request().Context()
		environmentID, err := strconv.Atoi(c.Param("environmentID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("environmentID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		policyUpsert := &api.PolicyUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set policy request").SetInternal(err)
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

		policy, err := s.store.UpsertPolicy(ctx, policyUpsert)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set policy for type %q", pType)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set policy response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/policy/environment/:environmentID", func(c echo.Context) error {
		environmentID, err := strconv.Atoi(c.Param("environmentID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("environmentID"))).SetInternal(err)
		}

		policyDelete := &api.PolicyDelete{
			EnvironmentID: environmentID,
			DeleterID:     c.Get(getPrincipalIDContextKey()).(int),
			Type:          api.PolicyType(c.QueryParam("type")),
		}

		ctx := c.Request().Context()
		if err := s.store.DeletePolicy(ctx, policyDelete); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete policy by environment ID %d", environmentID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.GET("/policy/environment/:environmentID", func(c echo.Context) error {
		ctx := c.Request().Context()
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

		policy, err := s.store.GetPolicy(ctx, policyFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get policy for type %q", pType)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get policy response: %v", pType)).SetInternal(err)
		}
		return nil
	})

	g.GET("/policy", func(c echo.Context) error {
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicy(pType, ""); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy type: %q", pType)).SetInternal(err)
		}

		policyFind := &api.PolicyFind{
			Type: &pType,
		}

		ctx := c.Request().Context()
		policyList, err := s.store.ListPolicy(ctx, policyFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to list policy for type %q", pType)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal list policy response: %v", pType)).SetInternal(err)
		}
		return nil
	})
}
