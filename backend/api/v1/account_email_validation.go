package v1

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// normalizeEmail canonicalizes a submitted email for lookups, lockout
// identities, and audit resources — these must all key the same string.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

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
