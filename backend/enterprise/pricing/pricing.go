package pricing

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// PriceModel holds the inputs for price calculation.
type PriceModel struct {
	Plan     *PlanLimitConfig
	Interval storepb.SubscriptionPayload_BillingInterval
	Seats    int32
}

// NewPriceModel validates inputs and returns a PriceModel.
func NewPriceModel(plan storepb.SubscriptionPayload_Plan, interval storepb.SubscriptionPayload_BillingInterval, seats int32) (*PriceModel, error) {
	limit := GetPlanLimit(plan)
	if limit == nil {
		return nil, errors.Errorf("unknown plan %v", plan)
	}
	if !limit.SelfServicePurchase {
		return nil, errors.Errorf("plan %v does not support self-service purchase", plan)
	}
	if interval == storepb.SubscriptionPayload_BILLING_INTERVAL_UNSPECIFIED {
		return nil, errors.New("billing interval is required")
	}
	if seats < 1 {
		return nil, errors.Errorf("seats must be at least 1, got %d", seats)
	}
	if limit.MaximumSeatCount >= 0 && seats > limit.MaximumSeatCount {
		return nil, errors.Errorf("seats %d exceeds maximum %d for plan %v", seats, limit.MaximumSeatCount, plan)
	}
	return &PriceModel{
		Plan:     limit,
		Interval: interval,
		Seats:    seats,
	}, nil
}

// GetPrice returns the total price in USD cents.
func (m *PriceModel) GetPrice() int64 {
	billableSeats := m.Seats - m.Plan.FreeSeatCount
	if billableSeats < 0 {
		billableSeats = 0
	}
	monthlyPrice := int64(billableSeats) * m.Plan.PricePerSeatPerMonth

	switch m.Interval {
	case storepb.SubscriptionPayload_YEAR:
		return monthlyPrice * 12
	default:
		return monthlyPrice
	}
}

// GetStripeInterval returns the Stripe recurring interval string.
func (m *PriceModel) GetStripeInterval() string {
	if m.Interval == storepb.SubscriptionPayload_YEAR {
		return "year"
	}
	return "month"
}

// GetPromotionCode returns the promotion code for the current interval, or empty string.
func (m *PriceModel) GetPromotionCode() string {
	for _, bm := range m.Plan.BillingMethods {
		if bm.Interval == m.Interval {
			return bm.PromotionCode
		}
	}
	return ""
}

// GetPlanText returns a display name for the plan.
func (m *PriceModel) GetPlanText() string {
	switch m.Plan.Plan {
	case storepb.SubscriptionPayload_TEAM:
		return "Pro Plan"
	case storepb.SubscriptionPayload_ENTERPRISE:
		return "Enterprise Plan"
	default:
		return "Free Plan"
	}
}
