// Package enterprise implements the enterprise license service.
package enterprise

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// LicenseService is the service for enterprise license.
type LicenseService struct {
	store              *store.Store
	cachedSubscription *v1pb.Subscription
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

// LoadSubscription will load subscription.
// If there is no license, we will return a free plan subscription without expiration time.
// If there is expired license, we will return a free plan subscription with the expiration time of the expired license.
func (s *LicenseService) LoadSubscription(ctx context.Context) *v1pb.Subscription {
	s.mu.RLock()
	cached := s.cachedSubscription
	s.mu.RUnlock()

	if cached != nil {
		// Invalidate the cache if expired.
		if cached.ExpiresTime != nil && cached.ExpiresTime.AsTime().Before(time.Now()) {
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

	subscription := s.provider.LoadSubscription(ctx)
	if subscription == nil {
		// Never had a subscription, set to free plan.
		subscription = &v1pb.Subscription{
			Plan: v1pb.PlanType_FREE,
		}
	}
	// Switch to free plan if the subscription is expired.
	if subscription.ExpiresTime != nil && subscription.ExpiresTime.AsTime().Before(time.Now()) {
		subscription = &v1pb.Subscription{
			Plan:        v1pb.PlanType_FREE,
			ExpiresTime: subscription.ExpiresTime,
		}
	}
	s.cachedSubscription = subscription
	return subscription
}

// GetEffectivePlan gets the effective plan.
func (s *LicenseService) GetEffectivePlan() v1pb.PlanType {
	ctx := context.Background()
	return s.LoadSubscription(ctx).Plan
}

// IsFeatureEnabled returns whether a feature is enabled.
func (s *LicenseService) IsFeatureEnabled(f v1pb.PlanFeature) error {
	plan := s.GetEffectivePlan()
	features, ok := planFeatureMatrix[plan]
	if !ok || !features[f] {
		minimalPlan := v1pb.PlanType_ENTERPRISE
		if planFeatureMatrix[v1pb.PlanType_TEAM][f] {
			minimalPlan = v1pb.PlanType_TEAM
		}
		return errors.Errorf("feature %s is a %s feature, please upgrade to access it", f.String(), minimalPlan.String())
	}
	return nil
}

// IsFeatureEnabledForInstance returns whether a feature is enabled for the instance.
func (s *LicenseService) IsFeatureEnabledForInstance(f v1pb.PlanFeature, instance *store.InstanceMessage) error {
	plan := s.GetEffectivePlan()
	// DO NOT check instance license fo FREE plan.
	if plan == v1pb.PlanType_FREE {
		return s.IsFeatureEnabled(f)
	}
	if err := s.IsFeatureEnabled(f); err != nil {
		return err
	}
	if !instance.Metadata.GetActivation() {
		return errors.Errorf(`feature "%s" is not available for instance %s, please assign license to the instance to enable it`, f.String(), instance.ResourceID)
	}
	return nil
}

// GetActivatedInstanceLimit returns the activated instance limit for the current subscription.
func (s *LicenseService) GetActivatedInstanceLimit(ctx context.Context) int {
	limit := s.LoadSubscription(ctx).ActiveInstances
	if limit < 0 {
		return math.MaxInt
	}
	return int(limit)
}

// GetUserLimit gets the user limit value for the plan.
func (s *LicenseService) GetUserLimit(ctx context.Context) int {
	subscription := s.LoadSubscription(ctx)
	// Prefer to take values from the license first.
	if subscription.Seats > 0 {
		return int(subscription.Seats)
	}

	limit := userLimitValues[subscription.Plan]
	if subscription.Plan == v1pb.PlanType_FREE {
		return limit
	}

	// To be compatible with old licenses which don't have seat field set in the claim.
	// Unlimited seat license.
	if subscription.Seats <= 0 {
		return math.MaxInt
	}

	return int(subscription.Seats)
}

// GetInstanceLimit gets the instance limit value for the plan.
func (s *LicenseService) GetInstanceLimit(ctx context.Context) int {
	subscription := s.LoadSubscription(ctx)
	// Prefer to take values from the license first.
	if subscription.Instances > 0 {
		return int(subscription.Instances)
	}

	limit := instanceLimitValues[subscription.Plan]
	if limit == -1 {
		// Enterprise license.
		if subscription.Instances > 0 {
			return int(subscription.Instances)
		}
		limit = math.MaxInt
	}
	return limit
}

// StoreLicense will store license into file.
func (s *LicenseService) StoreLicense(ctx context.Context, license string) error {
	if err := s.provider.StoreLicense(ctx, license); err != nil {
		return err
	}

	// refresh the cached subscription after storing the license.
	s.RefreshCache(ctx)
	return nil
}

// RefreshCache refresh the cache for subscription.
func (s *LicenseService) RefreshCache(ctx context.Context) {
	// refresh the cached subscription after storing the license.
	s.mu.Lock()
	s.cachedSubscription = nil
	s.mu.Unlock()
	s.LoadSubscription(ctx)
}
