package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
)

// SubscriptionService implements the subscription service.
type SubscriptionService struct {
	v1connect.UnimplementedSubscriptionServiceHandler
	store          *store.Store
	profile        *config.Profile
	metricReporter *metricreport.Reporter
	licenseService *enterprise.LicenseService
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	store *store.Store,
	profile *config.Profile,
	metricReporter *metricreport.Reporter,
	licenseService *enterprise.LicenseService) *SubscriptionService {
	return &SubscriptionService{
		store:          store,
		profile:        profile,
		metricReporter: metricReporter,
		licenseService: licenseService,
	}
}

// GetSubscription gets the subscription.
func (s *SubscriptionService) GetSubscription(ctx context.Context, _ *connect.Request[v1pb.GetSubscriptionRequest]) (*connect.Response[v1pb.Subscription], error) {
	subscription := s.licenseService.LoadSubscription(ctx)
	return connect.NewResponse(subscription), nil
}

// UpdateSubscription updates the subscription license.
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, req *connect.Request[v1pb.UpdateSubscriptionRequest]) (*connect.Response[v1pb.Subscription], error) {
	if err := s.licenseService.StoreLicense(ctx, req.Msg.License); err != nil {
		if common.ErrorCode(err) == common.Invalid {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to store license"))
	}

	subscription := s.licenseService.LoadSubscription(ctx)
	return connect.NewResponse(subscription), nil
}
