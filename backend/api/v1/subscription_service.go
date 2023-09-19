package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
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
	licenseService enterpriseAPI.LicenseService
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	store *store.Store,
	profile *config.Profile,
	metricReporter *metricreport.Reporter,
	licenseService enterpriseAPI.LicenseService) *SubscriptionService {
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
	for key, val := range api.FeatureMatrix {
		matrix := map[string]bool{}
		for i, enabled := range val {
			plan := covertToV1PlanType(api.PlanType(i))
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
	// clear the trialing setting for dev test
	if request.Patch.License == "" && s.profile.Mode == common.ReleaseModeDev {
		if err := s.store.DeleteSettingV2(ctx, api.SettingEnterpriseTrial); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to delete the trialing license: %v", err.Error())
		}
	}

	if err := s.licenseService.StoreLicense(ctx, &enterpriseAPI.SubscriptionPatch{
		UpdaterID: ctx.Value(common.PrincipalIDContextKey).(int),
		License:   request.Patch.License,
	}); err != nil {
		if common.ErrorCode(err) == common.Invalid {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to store license: %v", err.Error())
	}

	return s.loadSubscription(ctx)
}

// TrialSubscription creates a trial subscription.
func (s *SubscriptionService) TrialSubscription(ctx context.Context, request *v1pb.TrialSubscriptionRequest) (*v1pb.Subscription, error) {
	planType := api.FREE
	switch request.Trial.Plan {
	case v1pb.PlanType_TEAM:
		planType = api.TEAM
	case v1pb.PlanType_ENTERPRISE:
		planType = api.ENTERPRISE
	}

	workspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	license := &enterpriseAPI.License{
		InstanceCount: enterpriseAPI.InstanceLimitForTrial,
		ExpiresTs:     time.Now().AddDate(0, 0, enterpriseAPI.TrialDaysLimit).Unix(),
		IssuedTs:      time.Now().Unix(),
		Plan:          planType,
		// the subject format for license should be {org id in hub}.{subscription id in hub}
		// as we just need to simply generate the trialing license in console, we can use the workspace id instead.
		Subject:  fmt.Sprintf("%s.%s", workspaceID, ""),
		Trialing: true,
		OrgName:  workspaceID,
	}

	subscription := s.licenseService.LoadSubscription(ctx)
	basePlan := subscription.Plan

	if license.Plan.Priority() <= subscription.Plan.Priority() {
		return s.loadSubscription(ctx)
	}

	if subscription.Trialing {
		license.ExpiresTs = subscription.ExpiresTs
		license.IssuedTs = subscription.StartedTs
	}

	value, err := json.Marshal(license)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	_, created, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  api.SettingEnterpriseTrial,
		Value: string(value),
	}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create license: %v", err.Error())
	}
	if !created {
		return nil, status.Errorf(codes.InvalidArgument, "your trial already exists")
	}

	// we need to override the SettingEnterpriseLicense with an empty value to get the valid free trial.
	if err := s.licenseService.StoreLicense(ctx, &enterpriseAPI.SubscriptionPatch{
		UpdaterID: principalID,
		License:   "",
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove license: %v", err.Error())
	}

	s.licenseService.RefreshCache(ctx)
	subscription = s.licenseService.LoadSubscription(ctx)
	currentPlan := subscription.Plan
	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricAPI.SubscriptionTrialMetricName,
		Value: 1,
		Labels: map[string]any{
			"trial_plan":    currentPlan.String(),
			"from_plan":     basePlan.String(),
			"lark_notified": false,
		},
	})

	return s.loadSubscription(ctx)
}

func (s *SubscriptionService) loadSubscription(ctx context.Context) (*v1pb.Subscription, error) {
	sub := s.licenseService.LoadSubscription(ctx)

	subscription := &v1pb.Subscription{
		InstanceCount: int32(sub.InstanceCount),
		Plan:          covertToV1PlanType(sub.Plan),
		Trialing:      sub.Trialing,
		OrgId:         sub.OrgID,
		OrgName:       sub.OrgName,
	}
	if sub.Plan != api.FREE {
		subscription.ExpiresTime = timestamppb.New(time.Unix(sub.ExpiresTs, 0))
		subscription.StartedTime = timestamppb.New(time.Unix(sub.StartedTs, 0))
	}

	return subscription, nil
}

func covertToV1PlanType(planType api.PlanType) v1pb.PlanType {
	switch planType {
	case api.FREE:
		return v1pb.PlanType_FREE
	case api.TEAM:
		return v1pb.PlanType_TEAM
	case api.ENTERPRISE:
		return v1pb.PlanType_ENTERPRISE
	default:
		return v1pb.PlanType_PLAN_TYPE_UNSPECIFIED
	}
}
