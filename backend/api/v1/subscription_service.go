package v1

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/base"
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
	return s.loadSubscription(ctx)
}

// GetFeatureMatrix gets the feature metric.
func (*SubscriptionService) GetFeatureMatrix(_ context.Context, _ *v1pb.GetFeatureMatrixRequest) (*v1pb.FeatureMatrix, error) {
	resp := &v1pb.FeatureMatrix{}
	for key, val := range base.FeatureMatrix {
		matrix := map[string]bool{}
		for i, enabled := range val {
			// Convert array index to PlanType (0=FREE, 1=TEAM, 2=ENTERPRISE)
			plan := v1pb.PlanType(i + 1) // +1 because proto enums start at 1
			matrix[plan.String()] = enabled
		}
		resp.Features = append(resp.Features, &v1pb.Feature{
			Name:   string(key),
			Matrix: matrix,
		})
	}

	return resp, nil
}

// UpdateSubscription updates the subscription license.
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, request *v1pb.UpdateSubscriptionRequest) (*v1pb.Subscription, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if err := s.licenseService.StoreLicense(ctx, &enterprise.SubscriptionPatch{
		UpdaterID: principalID,
		License:   request.Patch.License,
	}); err != nil {
		if common.ErrorCode(err) == common.Invalid {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to store license: %v", err.Error())
	}

	return s.loadSubscription(ctx)
}

func (s *SubscriptionService) loadSubscription(ctx context.Context) (*v1pb.Subscription, error) {
	sub := s.licenseService.LoadSubscription(ctx)

	subscription := &v1pb.Subscription{
		SeatCount:     int32(sub.Seat),
		InstanceCount: int32(sub.InstanceCount),
		Plan:          sub.Plan,
		Trialing:      sub.Trialing,
		OrgId:         sub.OrgID,
		OrgName:       sub.OrgName,
	}
	if sub.Plan != v1pb.PlanType_FREE {
		subscription.ExpiresTime = timestamppb.New(time.Unix(sub.ExpiresTS, 0))
		subscription.StartedTime = timestamppb.New(time.Unix(sub.StartedTS, 0))
	}

	return subscription, nil
}
