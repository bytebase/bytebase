package pricing

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestNewPriceModel(t *testing.T) {
	tests := []struct {
		name     string
		plan     storepb.SubscriptionPayload_Plan
		interval storepb.SubscriptionPayload_BillingInterval
		seats    int32
		wantErr  bool
	}{
		{
			name:     "valid TEAM monthly",
			plan:     storepb.SubscriptionPayload_TEAM,
			interval: storepb.SubscriptionPayload_MONTH,
			seats:    5,
		},
		{
			name:     "valid TEAM yearly",
			plan:     storepb.SubscriptionPayload_TEAM,
			interval: storepb.SubscriptionPayload_YEAR,
			seats:    1,
		},
		{
			name:     "enterprise not self-service",
			plan:     storepb.SubscriptionPayload_ENTERPRISE,
			interval: storepb.SubscriptionPayload_MONTH,
			seats:    5,
			wantErr:  true,
		},
		{
			name:     "unknown plan",
			plan:     storepb.SubscriptionPayload_PLAN_UNSPECIFIED,
			interval: storepb.SubscriptionPayload_MONTH,
			seats:    5,
			wantErr:  true,
		},
		{
			name:     "zero seats",
			plan:     storepb.SubscriptionPayload_TEAM,
			interval: storepb.SubscriptionPayload_MONTH,
			seats:    0,
			wantErr:  true,
		},
		{
			name:     "unspecified interval",
			plan:     storepb.SubscriptionPayload_TEAM,
			interval: storepb.SubscriptionPayload_BILLING_INTERVAL_UNSPECIFIED,
			seats:    5,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPriceModel(tc.plan, tc.interval, tc.seats)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetPrice(t *testing.T) {
	tests := []struct {
		name      string
		interval  storepb.SubscriptionPayload_BillingInterval
		seats     int32
		wantCents int64
	}{
		{
			name:      "1 user monthly",
			interval:  storepb.SubscriptionPayload_MONTH,
			seats:     1,
			wantCents: 2000, // $20
		},
		{
			name:      "5 users monthly",
			interval:  storepb.SubscriptionPayload_MONTH,
			seats:     5,
			wantCents: 10000, // $100
		},
		{
			name:      "1 user yearly",
			interval:  storepb.SubscriptionPayload_YEAR,
			seats:     1,
			wantCents: 24000, // $240
		},
		{
			name:      "10 users yearly",
			interval:  storepb.SubscriptionPayload_YEAR,
			seats:     10,
			wantCents: 240000, // $2400
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewPriceModel(storepb.SubscriptionPayload_TEAM, tc.interval, tc.seats)
			require.NoError(t, err)
			require.Equal(t, tc.wantCents, m.GetPrice())
		})
	}
}

func TestGetStripeInterval(t *testing.T) {
	m, err := NewPriceModel(storepb.SubscriptionPayload_TEAM, storepb.SubscriptionPayload_MONTH, 1)
	require.NoError(t, err)
	require.Equal(t, "month", m.GetStripeInterval())

	m, err = NewPriceModel(storepb.SubscriptionPayload_TEAM, storepb.SubscriptionPayload_YEAR, 1)
	require.NoError(t, err)
	require.Equal(t, "year", m.GetStripeInterval())
}
