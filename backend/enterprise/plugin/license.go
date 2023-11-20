package plugin

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

// LicenseProvider is the license provider.
type LicenseProvider interface {
	// StoreLicense will store the license.
	StoreLicense(ctx context.Context, patch *enterprise.SubscriptionPatch) error
	// LoadSubscription will load the subscription.
	LoadSubscription(ctx context.Context) *enterprise.Subscription
	// GetEffectivePlan gets the effective plan.
	GetEffectivePlan(ctx context.Context) api.PlanType
}

// ProviderConfig is the provider configuration.
type ProviderConfig struct {
	Mode  common.ReleaseMode
	Store *store.Store
}
