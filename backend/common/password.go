//nolint:revive
package common

import (
	"regexp"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ValidatePassword validates a password against the workspace password restrictions.
func ValidatePassword(password string, restriction *storepb.WorkspaceProfileSetting_PasswordRestriction) error {
	if len(password) < int(restriction.GetMinLength()) {
		return errors.Errorf("password length should no less than %v characters", restriction.GetMinLength())
	}
	if restriction.GetRequireNumber() && !regexp.MustCompile("[0-9]+").MatchString(password) {
		return errors.Errorf("password must contains at least 1 number")
	}
	if restriction.GetRequireLetter() && !regexp.MustCompile("[a-zA-Z]+").MatchString(password) {
		return errors.Errorf("password must contains at least 1 lower case letter")
	}
	if restriction.GetRequireUppercaseLetter() && !regexp.MustCompile("[A-Z]+").MatchString(password) {
		return errors.Errorf("password must contains at least 1 upper case letter")
	}
	if restriction.GetRequireSpecialCharacter() && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+`).MatchString(password) {
		return errors.Errorf("password must contains at least 1 special character")
	}
	return nil
}
