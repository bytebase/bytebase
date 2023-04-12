package api

import api "github.com/bytebase/bytebase/backend/legacyapi"

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
