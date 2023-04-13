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
	InstanceCount int          `jsonapi:"attr,instanceCount"`
	ExpiresTs     int64        `jsonapi:"attr,expiresTs"`
	StartedTs     int64        `jsonapi:"attr,startedTs"`
	Plan          api.PlanType `jsonapi:"attr,plan"`
	Trialing      bool         `jsonapi:"attr,trialing"`
	OrgID         string       `jsonapi:"attr,orgId"`
	OrgName       string       `jsonapi:"attr,orgName"`
}

// IsExpired returns if the subscription is expired.
func (s *Subscription) IsExpired() bool {
	if s.Plan == api.FREE || s.ExpiresTs < 0 {
		return false
	}
	return time.Unix(s.ExpiresTs, 0).Before(time.Now())
}
