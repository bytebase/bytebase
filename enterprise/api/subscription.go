package api

import "github.com/bytebase/bytebase/api"

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
}
