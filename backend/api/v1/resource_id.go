package v1

import (
	"regexp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var resourceIDRegex = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

func validateResourceID(resourceID string) error {
	if !resourceIDRegex.MatchString(resourceID) {
		return status.Errorf(codes.InvalidArgument, "invalid resource ID %q", resourceID)
	}
	return nil
}
