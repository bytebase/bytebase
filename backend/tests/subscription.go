package tests

import (
	"context"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) setLicense(ctx context.Context) error {
	return ctl.trialPlan(ctx, &v1pb.TrialSubscription{
		InstanceCount: 100,
		Plan:          v1pb.PlanType_ENTERPRISE,
		Days:          1,
	})
}

func (ctl *controller) removeLicense(ctx context.Context) error {
	if _, err := ctl.subscriptionServiceClient.UpdateSubscription(ctx, &v1pb.UpdateSubscriptionRequest{
		Patch: &v1pb.PatchSubscription{License: ""},
	}); err != nil {
		return errors.Wrap(err, "failed to remove license")
	}
	return nil
}

func (ctl *controller) getSubscription(ctx context.Context) (*v1pb.Subscription, error) {
	return ctl.subscriptionServiceClient.GetSubscription(ctx, &v1pb.GetSubscriptionRequest{})
}

func (ctl *controller) trialPlan(ctx context.Context, trial *v1pb.TrialSubscription) error {
	if _, err := ctl.subscriptionServiceClient.TrialSubscription(ctx, &v1pb.TrialSubscriptionRequest{
		Trial: trial,
	}); err != nil {
		return errors.Wrap(err, "failed to start trial")
	}
	return nil
}
