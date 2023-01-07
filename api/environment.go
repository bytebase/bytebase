package api

import (
	"regexp"

	"github.com/pkg/errors"
)

// Environment is the API message for an environment.
type Environment struct {
	ID         int    `jsonapi:"primary,environment"`
	ResourceID string `jsonapi:"attr,resourceId"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal
	CreatedTs int64
	UpdaterID int
	Updater   *Principal
	UpdatedTs int64

	// Domain specific fields
	Name  string               `jsonapi:"attr,name"`
	Order int                  `jsonapi:"attr,order"`
	Tier  EnvironmentTierValue `jsonapi:"attr,tier"`
}

// EnvironmentCreate is the API message for creating an environment.
type EnvironmentCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Order *int   `jsonapi:"attr,order"`
}

// EnvironmentFind is the API message for finding environments.
type EnvironmentFind struct {
	RowStatus *RowStatus
}

// placeholderRegexp is the regexp for placeholder.
// Refer to https://stackoverflow.com/a/6222235/19075342, but we support '.' for now.
var placeholderRegexp = regexp.MustCompile(`[^\\/?%*:|"<>]+`)
var alphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9]*$`)

// IsValidEnvironmentName checks if the environment name is valid.
func IsValidEnvironmentName(environmentName string) error {
	if !placeholderRegexp.MatchString(environmentName) {
		return errors.Errorf("environment name %q cannot contain placeholder characters (\\, /, ?, %%, *, :, |, \", <, >)", environmentName)
	}
	if !alphaNumeric.MatchString(environmentName) {
		return errors.Errorf("environment name %q can only contain alphabet numeric characters (a-z, A-Z, 0-9)", environmentName)
	}

	return nil
}

// EnvironmentPatch is the API message for patching an environment.
type EnvironmentPatch struct {
	ID int `jsonapi:"primary,environmentPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name  *string `jsonapi:"attr,name"`
	Order *int    `jsonapi:"attr,order"`
}
