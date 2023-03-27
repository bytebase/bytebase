package tests

import (
	"bytes"
	"encoding/json"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"

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
	err := ctl.switchPlan(&enterpriseAPI.SubscriptionPatch{
		License: "",
	})
	if err != nil {
		return errors.Wrap(err, "failed to switch plan")
	}
	return nil
}

func (ctl *controller) getSubscription() (*enterpriseAPI.Subscription, error) {
	body, err := ctl.getOpenAPI("/subscription", nil)
	if err != nil {
		return nil, err
	}

	subscription := new(enterpriseAPI.Subscription)
	if err = jsonapi.UnmarshalPayload(body, subscription); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get subscription response")
	}
	return subscription, nil
}

func (ctl *controller) trialPlan(trial *v1pb.TrialSubscription) error {
	bs, err := json.Marshal(trial)
	if err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}
	if _, err := ctl.postOpenAPI("/subscription/trial", bytes.NewReader(bs)); err != nil {
		return err
	}
	return nil
}

func (ctl *controller) switchPlan(patch *enterpriseAPI.SubscriptionPatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, patch); err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}

	_, err := ctl.patchOpenAPI("/subscription", buf)
	if err != nil {
		return err
	}

	return nil
}
