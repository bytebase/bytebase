package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

// hasAccessToUpdatePolicy checks if user can access to policy control feature.
// return nil if user has access.
func (s *Server) hasAccessToUpsertPolicy(pType api.PolicyType, policyUpsert *api.PolicyUpsert) error {
	// nil payload means user doesn't update the payload field
	if policyUpsert.Payload == nil {
		return nil
	}

	defaultPolicy, err := api.GetDefaultPolicy(pType)
	if err != nil {
		return err
	}

	if defaultPolicy == *policyUpsert.Payload {
		return nil
	}
	switch pType {
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
		updaterID := c.Get(getPrincipalIDContextKey()).(int)

		policyUpsert := &api.PolicyUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set policy request").SetInternal(err)
		}

		if err := s.hasAccessToUpsertPolicy(pType, policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err.Error()).SetInternal(err)
		}
		// Validate policy.
		if err := api.ValidatePolicy(resourceType, pType, policyUpsert.Payload); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid policy payload %s", err.Error())).SetInternal(err)
		}

		policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
			ResourceType: &resourceType,
			ResourceUID:  &resourceID,
			Type:         &pType,
		})
		if err != nil {
			return err
		}
		if policy == nil {
			payload := ""
			if policyUpsert.Payload != nil {
				payload = *policyUpsert.Payload
			}
			inheritFromParent := true
			if policyUpsert.InheritFromParent != nil {
				inheritFromParent = *policyUpsert.InheritFromParent
			}
			policy, err = s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
				ResourceType:      resourceType,
				ResourceUID:       resourceID,
				Type:              pType,
				Payload:           payload,
				InheritFromParent: inheritFromParent,
			}, updaterID)
			if err != nil {
				return err
			}
		} else {
			policy, err = s.store.UpdatePolicyV2(ctx, &store.UpdatePolicyMessage{
				UpdaterID:         updaterID,
				ResourceType:      resourceType,
				ResourceUID:       resourceID,
				Type:              pType,
				InheritFromParent: policyUpsert.InheritFromParent,
				Payload:           policyUpsert.Payload,
			})
			if err != nil {
				return err
			}
		}

		composedPolicy := &api.Policy{
			ID:                policy.UID,
			RowStatus:         api.Normal,
			ResourceType:      resourceType,
			ResourceID:        resourceID,
			Type:              pType,
			InheritFromParent: policy.InheritFromParent,
			Payload:           policy.Payload,
		}
		if resourceType == api.PolicyResourceTypeEnvironment {
			composedEnvironment, err := s.store.GetEnvironmentByID(ctx, resourceID)
			if err != nil {
				return err
			}
			composedPolicy.Environment = composedEnvironment
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedPolicy); err != nil {
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

		policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
			ResourceType: &resourceType,
			ResourceUID:  &resourceID,
			Type:         &pType,
		})
		if err != nil {
			return err
		}
		if policy == nil {
			return echo.NewHTTPError(http.StatusPreconditionFailed, "policy not found")
		}

		if err := s.store.DeletePolicyV2(ctx, policy); err != nil {
			return err
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
