package api

import (
	"regexp"

	"github.com/pkg/errors"
)

// placeholderRegexp is the regexp for placeholder.
// Refer to https://stackoverflow.com/a/6222235/19075342, but we support '.' for now.
var placeholderRegexp = regexp.MustCompile(`[^\\/?%*:|"<>]+`)
var alphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9]*$`)

// IsValidEnvironmentName checks if the environment name is valid.
func IsValidEnvironmentName(environmentName string) error {
	if !placeholderRegexp.MatchString(environmentName) {
		return errors.Errorf("environment title %q cannot contain placeholder characters (\\, /, ?, %%, *, :, |, \", <, >)", environmentName)
	}
	if !alphaNumeric.MatchString(environmentName) {
		return errors.Errorf("environment title %q can only contain alphabet numeric characters (a-z, A-Z, 0-9)", environmentName)
	}

	return nil
}
