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
)

// SubscriptionService implements the subscription service.
type SubscriptionService struct {
	v1connect.UnimplementedSubscriptionServiceHandler
	profile        *config.Profile
	licenseService *enterprise.LicenseService
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	profile *config.Profile,
	licenseService *enterprise.LicenseService) *SubscriptionService {
	return &SubscriptionService{
		profile:        profile,
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
	if s.profile.SaaS {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot update license in the SaaS mode"))
	}
	if err := s.licenseService.StoreLicense(ctx, req.Msg.License); err != nil {
		if common.ErrorCode(err) == common.Invalid {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to store license"))
	}

	subscription := s.licenseService.LoadSubscription(ctx)
	return connect.NewResponse(subscription), nil
}
