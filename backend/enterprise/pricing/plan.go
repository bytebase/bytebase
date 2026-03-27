// Package pricing defines plan limits and price calculation for SaaS subscriptions.
package pricing

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// PlanLimitConfig defines pricing and limits for a plan.
type PlanLimitConfig struct {
	Plan                 storepb.SubscriptionPayload_Plan
	SelfServicePurchase  bool
	FreeSeatCount        int32
	MaximumSeatCount     int32 // -1 = unlimited
	PricePerSeatPerMonth int64 // USD cents
	InstanceCount        int32 // fixed instance count included in the plan
}

var planLimits = map[storepb.SubscriptionPayload_Plan]*PlanLimitConfig{
	storepb.SubscriptionPayload_TEAM: {
		Plan:                 storepb.SubscriptionPayload_TEAM,
		SelfServicePurchase:  true,
		FreeSeatCount:        0,
		MaximumSeatCount:     -1,   // unlimited
		PricePerSeatPerMonth: 2000, // $20/user/month
		InstanceCount:        10,
	},
	storepb.SubscriptionPayload_ENTERPRISE: {
		Plan:                 storepb.SubscriptionPayload_ENTERPRISE,
		SelfServicePurchase:  false,
		FreeSeatCount:        0,
		MaximumSeatCount:     -1,
		PricePerSeatPerMonth: 0, // custom pricing
		InstanceCount:        0,
	},
}

// GetPlanLimit returns the limit config for a plan, or nil if not found.
func GetPlanLimit(plan storepb.SubscriptionPayload_Plan) *PlanLimitConfig {
	return planLimits[plan]
}
