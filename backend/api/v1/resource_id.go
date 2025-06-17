package v1

import (
	"regexp"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
)

var resourceIDRegex = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

func validateResourceID(resourceID string) error {
	if !resourceIDRegex.MatchString(resourceID) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid resource ID %q", resourceID))
	}
	return nil
}
