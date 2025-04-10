package api

import (
	"time"

	"github.com/bytebase/bytebase/backend/base"
)

// SubscriptionPatch is the API message for update the subscription.
type SubscriptionPatch struct {
	UpdaterID int
	License   string
}

// Subscription is the API message for subscription.
type Subscription struct {
	InstanceCount int
	Seat          int
	ExpiresTS     int64
	StartedTS     int64
	Plan          base.PlanType
	Trialing      bool
	OrgID         string
	OrgName       string
}

// IsExpired returns if the subscription is expired.
func (s *Subscription) IsExpired() bool {
	if s.Plan == base.FREE || s.ExpiresTS < 0 {
		return false
	}
	return time.Unix(s.ExpiresTS, 0).Before(time.Now())
}
