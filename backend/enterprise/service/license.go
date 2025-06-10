// Package service implements the enterprise license service.
package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var _ enterprise.LicenseService = (*LicenseService)(nil)

// LicenseService is the service for enterprise license.
type LicenseService struct {
	store              *store.Store
	cachedSubscription *enterprise.Subscription
	mu                 sync.RWMutex

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
	s.mu.RLock()
	cached := s.cachedSubscription
	s.mu.RUnlock()

	if cached != nil {
		if cached.Plan == v1pb.PlanType_FREE || cached.IsExpired() {
			// refresh expired subscription
			s.mu.Lock()
			s.cachedSubscription = nil
			s.mu.Unlock()
			cached = nil
		}
	}
	if cached != nil {
		return cached
	}

	// Cache the subscription.
	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock
	if s.cachedSubscription != nil && s.cachedSubscription.Plan != v1pb.PlanType_FREE && !s.cachedSubscription.IsExpired() {
		return s.cachedSubscription
	}
	s.cachedSubscription = s.provider.LoadSubscription(ctx)
	return s.cachedSubscription
}

// IsFeatureEnabled returns whether a feature is enabled.
func (s *LicenseService) IsFeatureEnabled(f v1pb.PlanFeature) error {
	plan := s.GetEffectivePlan()
	features, ok := enterprise.PlanFeatureMatrix[plan]
	if !ok || !features[f] {
		return errors.New(accessErrorMessage(f))
	}
	return nil
}

// IsFeatureEnabledForInstance returns whether a feature is enabled for the instance.
func (s *LicenseService) IsFeatureEnabledForInstance(f v1pb.PlanFeature, instance *store.InstanceMessage) error {
	plan := s.GetEffectivePlan()
	// DONOT check instance license fo FREE plan.
	if plan == v1pb.PlanType_FREE {
		return s.IsFeatureEnabled(f)
	}
	if err := s.IsFeatureEnabled(f); err != nil {
		return err
	}
	if !instanceLimitFeature[f] {
		// If the feature not exists in the limit map, we just need to check the feature for current plan.
		return nil
	}
	if !instance.Metadata.GetActivation() {
		return errors.Errorf(`feature "%s" is not available for instance %s, please assign license to the instance to enable it`, f.String(), instance.ResourceID)
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
func (s *LicenseService) GetEffectivePlan() v1pb.PlanType {
	ctx := context.Background()
	subscription := s.LoadSubscription(ctx)
	if expireTime := time.Unix(subscription.ExpiresTS, 0); expireTime.Before(time.Now()) {
		return v1pb.PlanType_FREE
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
	case v1pb.PlanType_FREE:
		return limit
	case v1pb.PlanType_TEAM, v1pb.PlanType_ENTERPRISE:
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
	s.mu.Lock()
	s.cachedSubscription = nil
	s.mu.Unlock()
	s.LoadSubscription(ctx)
}

// Helper functions to avoid circular import

// accessErrorMessage returns a error message with feature name and minimum supported plan.
func accessErrorMessage(f v1pb.PlanFeature) string {
	plan := minimumSupportedPlan(f)
	return fmt.Sprintf("%s is a %s feature, please upgrade to access it.", f.String(), plan.String())
}

// minimumSupportedPlan will find the minimum plan which supports the target feature.
func minimumSupportedPlan(f v1pb.PlanFeature) v1pb.PlanType {
	// Check from lowest to highest plan
	if enterprise.PlanFeatureMatrix[v1pb.PlanType_FREE][f] {
		return v1pb.PlanType_FREE
	}
	if enterprise.PlanFeatureMatrix[v1pb.PlanType_TEAM][f] {
		return v1pb.PlanType_TEAM
	}
	return v1pb.PlanType_ENTERPRISE
}

// instanceLimitFeature is the map for instance feature. Only allowed to access these feature for activate instance.
var instanceLimitFeature = map[v1pb.PlanFeature]bool{
	v1pb.PlanFeature_FEATURE_DATABASE_SECRET_VARIABLES:     true,
	v1pb.PlanFeature_FEATURE_INSTANCE_READ_ONLY_CONNECTION: true,
	v1pb.PlanFeature_FEATURE_DATA_MASKING:                  true,
}
