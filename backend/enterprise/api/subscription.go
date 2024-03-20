package api

import (
	"time"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// SubscriptionPatch is the API message for update the subscription.
type SubscriptionPatch struct {
	UpdaterID int
	License   string `jsonapi:"attr,license"`
}

// Subscription is the API message for subscription.
type Subscription struct {
	InstanceCount int
	Seat          int
	ExpiresTs     int64
	StartedTs     int64
	Plan          api.PlanType
	Trialing      bool
	OrgID         string
	OrgName       string
}

// IsExpired returns if the subscription is expired.
func (s *Subscription) IsExpired() bool {
	if s.Plan == api.FREE || s.ExpiresTs < 0 {
		return false
	}
	return time.Unix(s.ExpiresTs, 0).Before(time.Now())
}
