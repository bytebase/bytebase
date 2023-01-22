package tests

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// upsertPolicy upserts the policy.
func (ctl *controller) upsertPolicy(policyUpsert api.PolicyUpsert) (*api.Policy, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &policyUpsert); err != nil {
		return nil, errors.Wrap(err, "failed to marshal policyUpsert")
	}

	body, err := ctl.patch(fmt.Sprintf("/policy/%s/%d?type=%s", strings.ToLower(string(policyUpsert.ResourceType)), policyUpsert.ResourceID, policyUpsert.Type), buf)
	if err != nil {
		return nil, err
	}

	policy := new(api.Policy)
	if err = jsonapi.UnmarshalPayload(body, policy); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal policy response")
	}
	return policy, nil
}

// deletePolicy deletes the archived policy.
func (ctl *controller) deletePolicy(policyDelete api.PolicyDelete) error {
	_, err := ctl.delete(fmt.Sprintf("/policy/environment/%d?type=%s", policyDelete.ResourceID, policyDelete.Type), new(bytes.Buffer))
	if err != nil {
		return err
	}
	return nil
}
