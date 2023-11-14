package aws

import (
	"context"

	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/config"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

var _ plugin.LicenseProvider = (*Provider)(nil)

// Provider is the AWS license provider.
type Provider struct {
	config *config.Config
}

// NewProvider will create a new AWS license provider.
func NewProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	config, err := config.NewConfig(providerConfig.Mode)
	if err != nil {
		return nil, err
	}

	return &Provider{
		config: config,
	}, nil
}

// StoreLicense will store the hub license.
func (*Provider) StoreLicense(_ context.Context, _ *enterprise.SubscriptionPatch) error {
	return nil
}

// LoadSubscription will load the hub subscription.
func (*Provider) LoadSubscription(_ context.Context) *enterprise.Subscription {
	return &enterprise.Subscription{
		InstanceCount: 0,
		Plan:          api.FREE,
	}
}

// GetEffectivePlan gets the effective plan.
func (*Provider) GetEffectivePlan(_ context.Context) api.PlanType {
	return api.FREE
}
