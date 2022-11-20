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
		if !s.feature(api.FeatureApprovalPolicy) {
			return errors.Errorf(api.FeatureApprovalPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeBackupPlan:
		if !s.feature(api.FeatureBackupPolicy) {
			return errors.Errorf(api.FeatureBackupPolicy.AccessErrorMessage())
		}
	case api.PolicyTypeSQLReview:
		return nil
	case api.PolicyTypeEnvironmentTier:
		if !s.feature(api.FeatureEnvironmentTierPolicy) {
			return errors.Errorf(api.FeatureEnvironmentTierPolicy.AccessErrorMessage())
		}
	}
	return nil
}

func (s *Server) registerPolicyRoutes(g *echo.Group) {
	g.PATCH("/policy/:resourceType/:resourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		resourceType, err := getPolicyResourceType(c.Param("resourceType"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		resourceID, err := getPolicyResourceID(c.Param("resourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}

		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicy(pType, ""); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy type: %q", pType)).SetInternal(err)
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
		resourceType, err := getPolicyResourceType(c.Param("resourceType"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		resourceID, err := getPolicyResourceID(c.Param("resourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		policyDelete := &api.PolicyDelete{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Type:         api.PolicyType(c.QueryParam("type")),
			DeleterID:    c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.store.DeletePolicy(ctx, policyDelete); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete policy by resource %q", c.Param("resource"))).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.GET("/policy/:resourceType/:resourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		resourceType, err := getPolicyResourceType(c.Param("resourceType"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		resourceID, err := getPolicyResourceID(c.Param("resourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicy(pType, ""); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy type: %q", pType)).SetInternal(err)
		}
		policyFind := &api.PolicyFind{
			ResourceType: &resourceType,
			ResourceID:   &resourceID,
			Type:         &pType,
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
		pType := api.PolicyType(c.QueryParam("type"))
		if err := api.ValidatePolicy(pType, ""); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy type: %q", pType)).SetInternal(err)
		}
		var resourceType *api.PolicyResourceType
		var resourceID *int
		if c.QueryParam("resourceType") != "" {
			rt, err := getPolicyResourceType(c.Param("resourceType"))
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			resourceType = &rt
		}
		if c.QueryParam("resourceId") != "" {
			id, err := getPolicyResourceID(c.Param("resourceID"))
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			resourceID = &id
		}
		policyFind := &api.PolicyFind{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Type:         &pType,
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

func getPolicyResourceType(resourceType string) (api.PolicyResourceType, error) {
	var rt api.PolicyResourceType
	switch resourceType {
	case "workspace":
		rt = api.PolicyResourceTypeWorkspace
	case "environment":
		rt = api.PolicyResourceTypeEnvironment
	case "project":
		rt = api.PolicyResourceTypeProject
	case "instance":
		rt = api.PolicyResourceTypeInstance
	case "database":
		rt = api.PolicyResourceTypeDatabase
	default:
		return api.PolicyResourceTypeUnknown, errors.Errorf("invalid policy resource type %q", rt)
	}
	return rt, nil
}

func getPolicyResourceID(resourceID string) (int, error) {
	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return 0, errors.Errorf("invalid policy resource ID %q", resourceID)
	}
	return id, nil
}
