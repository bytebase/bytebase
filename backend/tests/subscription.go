package tests

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) setLicense() error {
	return ctl.trialPlan(&v1pb.TrialSubscription{
		InstanceCount: 100,
		Plan:          v1pb.PlanType_ENTERPRISE,
		Days:          1,
	})
}

func (ctl *controller) removeLicense() error {
	err := ctl.switchPlan(&v1pb.PatchSubscription{
		License: "",
	})
	if err != nil {
		return errors.Wrap(err, "failed to switch plan")
	}
	return nil
}

func (ctl *controller) getSubscription() (*v1pb.Subscription, error) {
	body, err := ctl.getOpenAPI("/subscription", nil)
	if err != nil {
		return nil, err
	}
	bs, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	subscription := new(v1pb.Subscription)
	if err = protojson.Unmarshal(bs, subscription); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get subscription response")
	}
	return subscription, nil
}

func (ctl *controller) trialPlan(trial *v1pb.TrialSubscription) error {
	bs, err := protojson.Marshal(trial)
	if err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}
	if _, err := ctl.postOpenAPI("/subscription/trial", bytes.NewReader(bs)); err != nil {
		return err
	}
	return nil
}

func (ctl *controller) switchPlan(patch *v1pb.PatchSubscription) error {
	bs, err := protojson.Marshal(patch)
	if err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}

	if _, err := ctl.patchOpenAPI("/subscription", bytes.NewReader(bs)); err != nil {
		return err
	}

	return nil
}
