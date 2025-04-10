// Package service implements the enterprise license service.
package service

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	"github.com/bytebase/bytebase/backend/store"
)

var _ enterprise.LicenseService = (*LicenseService)(nil)

// LicenseService is the service for enterprise license.
type LicenseService struct {
	store              *store.Store
	cachedSubscription *enterprise.Subscription

	provider plugin.LicenseProvider
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
	if err := s.provider.StoreLicense(ctx, patch); err != nil {
		return err
	}

	s.RefreshCache(ctx)
	return nil
}

// LoadSubscription will load subscription.
func (s *LicenseService) LoadSubscription(ctx context.Context) *enterprise.Subscription {
	if s.cachedSubscription != nil {
		if s.cachedSubscription.Plan == base.FREE || s.cachedSubscription.IsExpired() {
			// refresh expired subscription
			s.cachedSubscription = nil
		}
	}
	if s.cachedSubscription != nil {
		return s.cachedSubscription
	}

	// Cache the subscription.
	s.cachedSubscription = s.provider.LoadSubscription(ctx)
	return s.cachedSubscription
}

// IsFeatureEnabled returns whether a feature is enabled.
func (s *LicenseService) IsFeatureEnabled(feature base.FeatureType) error {
	if !base.Feature(feature, s.GetEffectivePlan()) {
		return errors.New(feature.AccessErrorMessage())
	}
	return nil
}

// IsFeatureEnabledForInstance returns whether a feature is enabled for the instance.
func (s *LicenseService) IsFeatureEnabledForInstance(feature base.FeatureType, instance *store.InstanceMessage) error {
	plan := s.GetEffectivePlan()
	// DONOT check instance license fo FREE plan.
	if plan == base.FREE {
		return s.IsFeatureEnabled(feature)
	}
	if err := s.IsFeatureEnabled(feature); err != nil {
		return err
	}
	if !base.InstanceLimitFeature[feature] {
		// If the feature not exists in the limit map, we just need to check the feature for current plan.
		return nil
	}
	if !instance.Metadata.GetActivation() {
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
func (s *LicenseService) GetEffectivePlan() base.PlanType {
	ctx := context.Background()
	subscription := s.LoadSubscription(ctx)
	if expireTime := time.Unix(subscription.ExpiresTS, 0); expireTime.Before(time.Now()) {
		return base.FREE
	}
	return subscription.Plan
}

// GetPlanLimitValue gets the limit value for the plan.
func (s *LicenseService) GetPlanLimitValue(ctx context.Context, name enterprise.PlanLimit) int {
	v, ok := enterprise.PlanLimitValues[name]
	if !ok {
		return 0
	}
	subscription := s.LoadSubscription(ctx)
	limit := v[subscription.Plan]
	if limit == -1 {
		limit = math.MaxInt
	}

	switch subscription.Plan {
	case base.FREE:
		return limit
	case base.TEAM, base.ENTERPRISE:
		switch name {
		case enterprise.PlanLimitMaximumInstance:
			return limit
		case enterprise.PlanLimitMaximumUser:
			if subscription.Seat == 0 {
				// to compatible with old license.
				return limit
			}
			if subscription.Seat < 0 {
				return math.MaxInt
			}
			return subscription.Seat
		}
	}

	return limit
}

// RefreshCache will invalidate and refresh the subscription cache.
func (s *LicenseService) RefreshCache(ctx context.Context) {
	s.cachedSubscription = nil
	s.LoadSubscription(ctx)
}
