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
		return errors.Errorf("password must be at least %v characters long", restriction.GetMinLength())
	}
	if restriction.GetRequireNumber() && !regexp.MustCompile("[0-9]+").MatchString(password) {
		return errors.Errorf("password must contain at least one number")
	}
	if restriction.GetRequireLetter() && !regexp.MustCompile("[a-zA-Z]+").MatchString(password) {
		return errors.Errorf("password must contain at least one letter")
	}
	if restriction.GetRequireUppercaseLetter() && !regexp.MustCompile("[A-Z]+").MatchString(password) {
		return errors.Errorf("password must contain at least one uppercase letter")
	}
	if restriction.GetRequireSpecialCharacter() && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+`).MatchString(password) {
		return errors.Errorf("password must contain at least one special character")
	}
	return nil
}
