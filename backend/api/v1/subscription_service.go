package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SubscriptionService implements the subscription service.
type SubscriptionService struct {
	v1pb.UnimplementedSubscriptionServiceServer
	store          *store.Store
	profile        *config.Profile
	metricReporter *metricreport.Reporter
	licenseService enterprise.LicenseService
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	store *store.Store,
	profile *config.Profile,
	metricReporter *metricreport.Reporter,
	licenseService enterprise.LicenseService) *SubscriptionService {
	return &SubscriptionService{
		store:          store,
		profile:        profile,
		metricReporter: metricReporter,
		licenseService: licenseService,
	}
}

// GetSubscription gets the subscription.
func (s *SubscriptionService) GetSubscription(ctx context.Context, _ *v1pb.GetSubscriptionRequest) (*v1pb.Subscription, error) {
	return s.licenseService.LoadSubscription(ctx), nil
}

// UpdateSubscription updates the subscription license.
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, request *v1pb.UpdateSubscriptionRequest) (*v1pb.Subscription, error) {
	if err := s.licenseService.StoreLicense(ctx, request.Patch.License); err != nil {
		if common.ErrorCode(err) == common.Invalid {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to store license: %v", err.Error())
	}

	return s.licenseService.LoadSubscription(ctx), nil
}
