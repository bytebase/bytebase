package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

// hasAccessToUpdatePolicy checks if user can access to policy control feature.
// return nil if user has access.
func (s *Server) hasAccessToUpsertPolicy(policyType api.PolicyType, payload string) error {
	defaultPolicy, err := api.GetDefaultPolicy(policyType)
	if err != nil {
		return err
	}
	if defaultPolicy == payload {
		return nil
	}
	switch policyType {
	case api.PolicyTypePipelineApproval:
		if !s.feature(api.FeatureApprovalPolicy) {
			return fmt.Errorf(api.FeatureApprovalPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeBackupPlan:
		if !s.feature(api.FeatureBackupPolicy) {
			return fmt.Errorf(api.FeatureBackupPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeSchemaReview:
		if !s.feature(api.FeatureSchemaReviewPolicy) {
			return fmt.Errorf(api.FeatureSchemaReviewPolicy.AccessErrorMessage())
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set policy request").SetInternal(err)
		}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicy(pType, ""); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy type: %q", pType)).SetInternal(err)
		}

		policyUpsert.EnvironmentID = environmentID
		policyUpsert.Type = pType
		policyUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if err := s.hasAccessToUpsertPolicy(policyUpsert.Type, policyUpsert.Payload); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err.Error()).SetInternal(err)
		}

		policy, err := s.store.UpsertPolicy(ctx, policyUpsert)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set policy for type %q", pType)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set policy response").SetInternal(err)
		}
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

	g.PATCH("/policy/:policyID", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("policyID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("policyID"))).SetInternal(err)
		}

		policyPatch := &api.PolicyPatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, policyPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set policy request").SetInternal(err)
		}
		policyPatch.ID = id
		policyPatch.Type = api.PolicyType(c.QueryParam("type"))
		policyPatch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		payload := ""
		if policyPatch.Payload != nil {
			payload = *policyPatch.Payload
		}
		if err := s.hasAccessToUpsertPolicy(policyPatch.Type, payload); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err.Error()).SetInternal(err)
		}

		ctx := c.Request().Context()
		policy, err := s.store.PatchPolicy(ctx, policyPatch)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch policy for type %q", policyPatch.Type)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set policy response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/policy/:policyID", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("policyID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("policyID"))).SetInternal(err)
		}

		policyDelete := &api.PolicyDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
			Type:      api.PolicyType(c.QueryParam("type")),
		}
		if err := s.hasAccessToUpsertPolicy(policyDelete.Type, ""); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err.Error()).SetInternal(err)
		}

		ctx := c.Request().Context()
		if err := s.store.DeletePolicy(ctx, policyDelete); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete policy by ID %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.GET("/policy/:policyID", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("policyID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Policy ID is not a number: %s", c.Param("policyID"))).SetInternal(err)
		}

		policyFind := &api.PolicyFind{
			ID: &id,
		}

		ctx := c.Request().Context()
		policy, err := s.store.GetPolicy(ctx, policyFind)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get policy by ID %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal get policy response").SetInternal(err)
		}
		return nil
	})
}
