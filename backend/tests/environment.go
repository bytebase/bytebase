package tests

import (
	"bytes"
	"reflect"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

func (ctl *controller) createEnvironment(environmentCreate api.EnvironmentCreate) (*api.Environment, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &environmentCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal environment create")
	}

	body, err := ctl.post("/environment", buf)
	if err != nil {
		return nil, err
	}

	environment := new(api.Environment)
	if err = jsonapi.UnmarshalPayload(body, environment); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal post project response")
	}
	return environment, nil
}

// getProjects gets the environments.
func (ctl *controller) getEnvironments() ([]*api.Environment, error) {
	body, err := ctl.get("/environment", nil)
	if err != nil {
		return nil, err
	}

	var environments []*api.Environment
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Environment)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get environment response")
	}
	for _, p := range ps {
		environment, ok := p.(*api.Environment)
		if !ok {
			return nil, errors.Errorf("fail to convert environment")
		}
		environments = append(environments, environment)
	}
	return environments, nil
}

func findEnvironment(envs []*api.Environment, name string) (*api.Environment, error) {
	for _, env := range envs {
		if env.Name == name {
			return env, nil
		}
	}
	return nil, errors.Errorf("unable to find environment %q", name)
}
