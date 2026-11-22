package v1

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

func validateServiceAccountEmail(email string) error {
	if err := common.ValidateEmail(email); err != nil {
		return err
	}
	if !common.IsServiceAccountEmail(email) {
		return errors.Errorf("email must end with %s", common.ServiceAccountSuffix)
	}
	return nil
}

func validateWorkloadIdentityEmail(email string) error {
	if err := common.ValidateEmail(email); err != nil {
		return err
	}
	if !common.IsWorkloadIdentityEmail(email) {
		return errors.Errorf("email must end with %s", common.WorkloadIdentitySuffix)
	}
	return nil
}

func invalidAccountEmailError(kind, email string, err error) error {
	return errors.Wrapf(err, "invalid %s email %q", kind, email)
}
