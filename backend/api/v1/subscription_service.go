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
	workspaceID    string
	store          *store.Store
	profile        *config.Profile
	metricReporter *metricreport.Reporter
	licenseService enterpriseAPI.LicenseService
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	workspaceID string,
	store *store.Store,
	profile *config.Profile,
	metricReporter *metricreport.Reporter,
	licenseService enterpriseAPI.LicenseService) *SubscriptionService {
	return &SubscriptionService{
		workspaceID:    workspaceID,
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

	license := &enterpriseAPI.License{
		InstanceCount: int(request.Trial.InstanceCount),
		ExpiresTs:     time.Now().AddDate(0, 0, int(request.Trial.Days)).Unix(),
		IssuedTs:      time.Now().Unix(),
		Plan:          planType,
		// the subject format for license should be {org id in hub}.{subscription id in hub}
		// as we just need to simply generate the trialing license in console, we can use the workspace id instead.
		Subject:  fmt.Sprintf("%s.%s", s.workspaceID, ""),
		Trialing: true,
		OrgName:  s.workspaceID,
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
	settingName := api.SettingEnterpriseTrial
	settings, err := s.store.FindSetting(ctx, &api.SettingFind{
		Name: &settingName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list settings: %v", err.Error())
	}

	if len(settings) == 0 {
		// We will create a new setting named SettingEnterpriseTrial to store the free trial license.
		_, created, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
			Name:        api.SettingEnterpriseTrial,
			Value:       string(value),
			Description: "The trialing license.",
		}, principalID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create license: %v", err.Error())
		}

		if created && subscription.Trialing {
			// For trial upgrade
			// Case 1: Users just have the SettingEnterpriseTrial, don't upload their license in SettingEnterpriseLicense.
			// Case 2: Users have the SettingEnterpriseLicense with team plan and trialing status.
			// In both cases, we can override the SettingEnterpriseLicense with an empty value to get the valid free trial.
			if err := s.licenseService.StoreLicense(ctx, &enterpriseAPI.SubscriptionPatch{
				UpdaterID: principalID,
				License:   "",
			}); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to remove license: %v", err.Error())
			}
		}
	} else {
		// Update the existed free trial.
		if _, err := s.store.PatchSetting(ctx, &api.SettingPatch{
			UpdaterID: principalID,
			Name:      api.SettingEnterpriseTrial,
			Value:     string(value),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to patch license: %v", err.Error())
		}
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

	plan := v1pb.PlanType_PLAN_TYPE_UNSPECIFIED
	switch sub.Plan {
	case api.FREE:
		plan = v1pb.PlanType_FREE
	case api.TEAM:
		plan = v1pb.PlanType_TEAM
	case api.ENTERPRISE:
		plan = v1pb.PlanType_ENTERPRISE
	}

	subscription := &v1pb.Subscription{
		InstanceCount: int32(sub.InstanceCount),
		Plan:          plan,
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
