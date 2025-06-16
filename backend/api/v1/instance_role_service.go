package v1

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
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
		return nil, status.Errorf(codes.Internal, "failed to get instance: %v", err)
	}
	instanceRoles := convertInstanceRoles(instance, instance.Metadata.GetRoles())
	return connect.NewResponse(&v1pb.ListInstanceRolesResponse{
		Roles: instanceRoles,
	}), nil
}
