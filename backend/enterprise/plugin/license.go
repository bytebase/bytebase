package plugin

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// LicenseProvider is the license provider.
type LicenseProvider interface {
	// StoreLicense will store the license.
	StoreLicense(ctx context.Context, license string) error
	// LoadSubscription will load the subscription.
	LoadSubscription(ctx context.Context) *v1pb.Subscription
}

// ProviderConfig is the provider configuration.
type ProviderConfig struct {
	Mode  common.ReleaseMode
	Store *store.Store
}
