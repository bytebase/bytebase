package tests

import (
	"bytes"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"

	"github.com/bytebase/bytebase/backend/api"
)

func (ctl *controller) setLicense() error {
	return ctl.trialPlan(&api.TrialPlanCreate{
		InstanceCount: 100,
		Type:          api.ENTERPRISE,
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
	body, err := ctl.get("/subscription", nil)
	if err != nil {
		return nil, err
	}

	subscription := new(enterpriseAPI.Subscription)
	if err = jsonapi.UnmarshalPayload(body, subscription); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get subscription response")
	}
	return subscription, nil
}

func (ctl *controller) trialPlan(trial *api.TrialPlanCreate) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, trial); err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}

	_, err := ctl.post("/subscription/trial", buf)
	return err
}

func (ctl *controller) switchPlan(patch *enterpriseAPI.SubscriptionPatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, patch); err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}

	_, err := ctl.patch("/subscription", buf)
	if err != nil {
		return err
	}

	return nil
}
