package v1

import (
	"regexp"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var resourceIDMatcher = regexp.MustCompile("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$")

func convertDeletedToState(deleted bool) v1pb.State {
	if deleted {
		return v1pb.State_STATE_DELETED
	}
	return v1pb.State_STATE_ACTIVE
}
