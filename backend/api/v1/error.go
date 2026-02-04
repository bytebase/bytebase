package v1

import (
	"github.com/bytebase/bytebase/backend/common/permission"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

type queryError struct {
	err         error
	resources   []string
	permission  permission.Permission
	commandType v1pb.QueryResult_PermissionDenied_CommandType
}

func (qe *queryError) Error() string {
	return qe.err.Error()
}
