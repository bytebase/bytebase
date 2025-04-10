// Package api provides the API definition for enterprise service.
package api

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/store"
)

// validPlans is a string array of valid plan types.
var validPlans = []base.PlanType{
	base.TEAM,
	base.ENTERPRISE,
}

// License is the API message for enterprise license.
type License struct {
	Subject       string
	InstanceCount int
	Seat          int
	ExpiresTS     int64
	IssuedTS      int64
	Plan          base.PlanType
	Trialing      bool
	OrgName       string
}

// Valid will check if license expired or has correct plan type.
func (l *License) Valid() error {
	if expireTime := time.Unix(l.ExpiresTS, 0); expireTime.Before(time.Now()) {
		return errors.Errorf("license has expired at %v", expireTime)
	}

	return l.validPlanType()
}

func (l *License) validPlanType() error {
	for _, plan := range validPlans {
		if plan == l.Plan {
			return nil
		}
	}

	return errors.Errorf("plan %q is not valid, expect %s or %s",
		l.Plan.String(),
		base.TEAM.String(),
		base.ENTERPRISE.String(),
	)
}

// OrgID extract the organization id from license subject.
func (l *License) OrgID() string {
	return strings.Split(l.Subject, ".")[0]
}

// LicenseService is the service for enterprise license.
type LicenseService interface {
	// StoreLicense will store license into file.
	StoreLicense(ctx context.Context, patch *SubscriptionPatch) error
	// LoadSubscription will load subscription.
	LoadSubscription(ctx context.Context) *Subscription
	// IsFeatureEnabled returns whether a feature is enabled.
	IsFeatureEnabled(feature base.FeatureType) error
	// IsFeatureEnabledForInstance returns whether a feature is enabled for the instance.
	IsFeatureEnabledForInstance(feature base.FeatureType, instance *store.InstanceMessage) error
	// GetEffectivePlan gets the effective plan.
	GetEffectivePlan() base.PlanType
	// GetPlanLimitValue gets the limit value for the plan.
	GetPlanLimitValue(ctx context.Context, name PlanLimit) int
	// GetInstanceLicenseCount returns the instance count limit for current subscription.
	GetInstanceLicenseCount(ctx context.Context) int
	// RefreshCache will invalidate and refresh the subscription cache.
	RefreshCache(ctx context.Context)
}
