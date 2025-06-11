// Package api provides the API definition for enterprise service.
package api

import (
	"context"

	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// LicenseService is the service for enterprise license.
type LicenseService interface {
	// StoreLicense will store license into file.
	StoreLicense(ctx context.Context, license string) error
	// LoadSubscription will load subscription.
	LoadSubscription(ctx context.Context) *v1pb.Subscription
	// IsFeatureEnabled returns whether a feature is enabled.
	IsFeatureEnabled(feature v1pb.PlanFeature) error
	// IsFeatureEnabledForInstance returns whether a feature is enabled for the instance.
	IsFeatureEnabledForInstance(feature v1pb.PlanFeature, instance *store.InstanceMessage) error
	// GetEffectivePlan gets the effective plan.
	GetEffectivePlan() v1pb.PlanType
	// GetPlanLimitValue gets the limit value for the plan.
	GetPlanLimitValue(ctx context.Context, name PlanLimit) int
	// GetInstanceLicenseCount returns the instance count limit for current subscription.
	GetInstanceLicenseCount(ctx context.Context) int
	// RefreshCache will invalidate and refresh the subscription cache.
	RefreshCache(ctx context.Context)
}
