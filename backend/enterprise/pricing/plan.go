// Package pricing defines plan limits and price calculation for SaaS subscriptions.
package pricing

import (
	"fmt"
	"os"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// BillingMethodConfig defines a billing method.
type BillingMethodConfig struct {
	Interval storepb.SubscriptionPayload_BillingInterval
}

// GetPromotionCode returns the Stripe promotion code for the given plan and interval
// from environment variables. The env var name follows the pattern:
//
//	STRIPE_PROMOTION_CODE_<PLAN>_<INTERVAL>
//
// For example: STRIPE_PROMOTION_CODE_TEAM_MONTH=FIRSTMONTH90
//
// Returns empty string if not set.
func GetPromotionCode(plan storepb.SubscriptionPayload_Plan, interval storepb.SubscriptionPayload_BillingInterval) string {
	planName := strings.TrimPrefix(plan.String(), "PLAN_")
	intervalName := strings.TrimPrefix(interval.String(), "BILLING_INTERVAL_")
	key := fmt.Sprintf("STRIPE_PROMOTION_CODE_%s_%s", planName, intervalName)
	return os.Getenv(key)
}

// PlanLimitConfig defines pricing and limits for a plan.
type PlanLimitConfig struct {
	Plan                 storepb.SubscriptionPayload_Plan
	SelfServicePurchase  bool
	FreeSeatCount        int32
	MaximumSeatCount     int32 // -1 = unlimited
	PricePerSeatPerMonth int64 // USD cents
	InstanceCount        int32 // fixed instance count included in the plan
	BillingMethods       []BillingMethodConfig
}

var planLimits = map[storepb.SubscriptionPayload_Plan]*PlanLimitConfig{
	storepb.SubscriptionPayload_TEAM: {
		Plan:                 storepb.SubscriptionPayload_TEAM,
		SelfServicePurchase:  true,
		FreeSeatCount:        0,
		MaximumSeatCount:     -1,   // unlimited
		PricePerSeatPerMonth: 2000, // $20/user/month
		InstanceCount:        10,
		BillingMethods: []BillingMethodConfig{
			{Interval: storepb.SubscriptionPayload_MONTH},
			{Interval: storepb.SubscriptionPayload_YEAR},
		},
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
