// Package service implements the enterprise license service.
package service

import (
	"context"
	"math"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

var _ enterprise.LicenseService = (*LicenseService)(nil)

// LicenseService is the service for enterprise license.
type LicenseService struct {
	store              *store.Store
	cachedSubscription *enterprise.Subscription

	provider plugin.LicenseProvider
}

// Claims creates a struct that will be encoded to a JWT.
// We add jwt.RegisteredClaims as an embedded type, to provide fields such as name.
type Claims struct {
	InstanceCount int    `json:"instanceCount"`
	Trialing      bool   `json:"trialing"`
	Plan          string `json:"plan"`
	OrgName       string `json:"orgName"`
	jwt.RegisteredClaims
}

// NewLicenseService will create a new enterprise license service.
func NewLicenseService(mode common.ReleaseMode, store *store.Store) (*LicenseService, error) {
	provider, err := getLicenseProvider(&plugin.ProviderConfig{
		Mode:  mode,
		Store: store,
	})
	if err != nil {
		return nil, err
	}

	return &LicenseService{
		store:    store,
		provider: provider,
	}, nil
}

// StoreLicense will store license into file.
func (s *LicenseService) StoreLicense(ctx context.Context, patch *enterprise.SubscriptionPatch) error {
	s.provider.StoreLicense(ctx, patch)

	s.RefreshCache(ctx)
	return nil
}

// LoadSubscription will load subscription.
func (s *LicenseService) LoadSubscription(ctx context.Context) *enterprise.Subscription {
	if s.cachedSubscription != nil && s.cachedSubscription.IsExpired() {
		// refresh expired subscription
		s.cachedSubscription = nil
	}
	if s.cachedSubscription != nil {
		return s.cachedSubscription
	}

	// Cache the subscription.
	s.cachedSubscription = s.provider.LoadSubscription(ctx)
	return s.cachedSubscription
}

// IsFeatureEnabled returns whether a feature is enabled.
func (s *LicenseService) IsFeatureEnabled(feature api.FeatureType) error {
	if !api.Feature(feature, s.GetEffectivePlan()) {
		return errors.Errorf(feature.AccessErrorMessage())
	}
	return nil
}

// IsFeatureEnabledForInstance returns whether a feature is enabled for the instance.
func (s *LicenseService) IsFeatureEnabledForInstance(feature api.FeatureType, instance *store.InstanceMessage) error {
	plan := s.GetEffectivePlan()
	// DONOT check instance license fo FREE plan.
	if plan == api.FREE {
		return s.IsFeatureEnabled(feature)
	}
	if err := s.IsFeatureEnabled(feature); err != nil {
		return err
	}
	if !api.InstanceLimitFeature[feature] {
		// If the feature not exists in the limit map, we just need to check the feature for current plan.
		return nil
	}
	if !instance.Activation {
		return errors.Errorf(`feature "%s" is not available for instance %s, please assign license to the instance to enable it`, feature.Name(), instance.ResourceID)
	}
	return nil
}

// GetInstanceLicenseCount returns the instance count limit for current subscription.
func (s *LicenseService) GetInstanceLicenseCount(ctx context.Context) int {
	instanceCount := s.LoadSubscription(ctx).InstanceCount
	if instanceCount < 0 {
		return math.MaxInt
	}
	return instanceCount
}

// GetEffectivePlan gets the effective plan.
func (s *LicenseService) GetEffectivePlan() api.PlanType {
	ctx := context.Background()
	return s.provider.GetEffectivePlan(ctx)
}

// GetPlanLimitValue gets the limit value for the plan.
func (s *LicenseService) GetPlanLimitValue(ctx context.Context, name enterprise.PlanLimit) int64 {
	v, ok := enterprise.PlanLimitValues[name]
	if !ok {
		return 0
	}

	subscription := s.LoadSubscription(ctx)

	limit := v[subscription.Plan]
	if subscription.Trialing {
		limit = v[api.FREE]
	}

	if limit == -1 {
		return math.MaxInt64
	}
	return limit
}

// RefreshCache will invalidate and refresh the subscription cache.
func (s *LicenseService) RefreshCache(ctx context.Context) {
	s.cachedSubscription = nil
	s.LoadSubscription(ctx)
}
