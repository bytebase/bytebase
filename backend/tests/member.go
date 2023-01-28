package tests

import (
	"bytes"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

func (ctl *controller) createPrincipal(principalCreate api.PrincipalCreate) (*api.Principal, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &principalCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal principal create")
	}

	body, err := ctl.post("/principal", buf)
	if err != nil {
		return nil, err
	}

	principal := new(api.Principal)
	if err = jsonapi.UnmarshalPayload(body, principal); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post principal response")
	}
	return principal, nil
}

func (ctl *controller) createMember(memberCreate api.MemberCreate) (*api.Member, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &memberCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal member create")
	}

	body, err := ctl.post("/member", buf)
	if err != nil {
		return nil, err
	}

	member := new(api.Member)
	if err = jsonapi.UnmarshalPayload(body, member); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post member response")
	}
	return member, nil
}
