package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

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
		if !s.licenseService.IsFeatureEnabled(api.FeatureApprovalPolicy) {
			return errors.Errorf(api.FeatureApprovalPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeBackupPlan:
		if !s.licenseService.IsFeatureEnabled(api.FeatureBackupPolicy) {
			return errors.Errorf(api.FeatureBackupPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeSQLReview:
		return nil
	case api.PolicyTypeEnvironmentTier:
		if !s.licenseService.IsFeatureEnabled(api.FeatureEnvironmentTierPolicy) {
			return errors.Errorf(api.FeatureEnvironmentTierPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeSensitiveData:
		if !s.licenseService.IsFeatureEnabled(api.FeatureSensitiveData) {
			return errors.Errorf(api.FeatureSensitiveData.AccessErrorMessage())
		}
	}
	return nil
}

func (s *Server) registerPolicyRoutes(g *echo.Group) {
	g.PATCH("/policy/:resourceType/:resourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		resourceType, err := api.GetPolicyResourceType(c.Param("resourceType"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		resourceID, err := getPolicyResourceID(c.Param("resourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicyType(pType); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		policyUpsert := &api.PolicyUpsert{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Type:         pType,
			UpdaterID:    c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set policy request").SetInternal(err)
		}

		if err := s.hasAccessToUpsertPolicy(policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err.Error()).SetInternal(err)
		}

		// Validate policy.
		if err := api.ValidatePolicy(policyUpsert.ResourceType, policyUpsert.Type, policyUpsert.Payload); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy payload %s", err.Error())).SetInternal(err)
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

	g.DELETE("/policy/:resourceType/:resourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		resourceType, err := api.GetPolicyResourceType(c.Param("resourceType"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		resourceID, err := getPolicyResourceID(c.Param("resourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicyType(pType); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		policyDelete := &api.PolicyDelete{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Type:         pType,
			DeleterID:    c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.store.DeletePolicy(ctx, policyDelete); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete policy by resource type %q id %q", c.Param("resourceType"), c.Param("resourceID"))).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.GET("/policy/:resourceType/:resourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		resourceType, err := api.GetPolicyResourceType(c.Param("resourceType"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		resourceID, err := getPolicyResourceID(c.Param("resourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicyType(pType); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		policyFind := &api.PolicyFind{
			ResourceType: &resourceType,
			ResourceID:   &resourceID,
			Type:         pType,
		}
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
		var resourceType *api.PolicyResourceType
		var resourceID *int
		if c.QueryParam("resourceType") != "" {
			rt, err := api.GetPolicyResourceType(c.QueryParam("resourceType"))
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			resourceType = &rt
		}
		if c.QueryParam("resourceId") != "" {
			id, err := getPolicyResourceID(c.QueryParam("resourceID"))
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			resourceID = &id
		}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicyType(pType); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}

		policyFind := &api.PolicyFind{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Type:         pType,
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

func getPolicyResourceID(resourceID string) (int, error) {
	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return 0, errors.Errorf("invalid policy resource ID %q", resourceID)
	}
	return id, nil
}
