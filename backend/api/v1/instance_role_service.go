package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/dbfactory"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// InstanceRoleService implements the database role service.
type InstanceRoleService struct {
	v1connect.UnimplementedInstanceRoleServiceHandler
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// NewInstanceRoleService creates a new InstanceRoleService.
func NewInstanceRoleService(store *store.Store, dbFactory *dbfactory.DBFactory) *InstanceRoleService {
	return &InstanceRoleService{
		store:     store,
		dbFactory: dbFactory,
	}
}

// GetInstanceRole gets an role.
func (*InstanceRoleService) GetInstanceRole(_ context.Context, _ *connect.Request[v1pb.GetInstanceRoleRequest]) (*connect.Response[v1pb.InstanceRole], error) {
	return connect.NewResponse(&v1pb.InstanceRole{}), nil
}

// ListInstanceRoles retrieves the list of roles for a given instance.
// Note: Pagination is not implemented in this method as it is not required at the moment.
func (s *InstanceRoleService) ListInstanceRoles(ctx context.Context, req *connect.Request[v1pb.ListInstanceRolesRequest]) (*connect.Response[v1pb.ListInstanceRolesResponse], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get instance"))
	}
	instanceRoles := convertInstanceRoles(instance, instance.Metadata.GetRoles())
	return connect.NewResponse(&v1pb.ListInstanceRolesResponse{
		Roles: instanceRoles,
	}), nil
}
