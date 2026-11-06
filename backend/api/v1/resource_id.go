package v1

import (
	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

func validateResourceID(resourceID string) error {
	if !common.IsValidResourceID(resourceID) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid resource ID %q", resourceID))
	}
	return nil
}
