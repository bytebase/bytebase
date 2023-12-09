package tests

import (
	"context"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) setLicense(ctx context.Context) error {
	if _, err := ctl.subscriptionServiceClient.UpdateSubscription(ctx, &v1pb.UpdateSubscriptionRequest{
		Patch: &v1pb.PatchSubscription{License: "eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA"},
	}); err != nil {
		return errors.Wrap(err, "failed to remove license")
	}
	return nil
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

func (ctl *controller) getWorkspaceID(ctx context.Context) (string, error) {
	resp, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, &v1pb.GetActuatorInfoRequest{})
	if err != nil {
		return "", err
	}
	return resp.WorkspaceId, nil
}
