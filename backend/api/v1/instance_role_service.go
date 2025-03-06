package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// InstanceRoleService implements the database role service.
type InstanceRoleService struct {
	v1pb.UnimplementedInstanceRoleServiceServer
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
func (*InstanceRoleService) GetInstanceRole(_ context.Context, _ *v1pb.GetInstanceRoleRequest) (*v1pb.InstanceRole, error) {
	return &v1pb.InstanceRole{}, nil
}

// ListInstanceRoles retrieves the list of roles for a given instance.
// Note: Pagination is not implemented in this method as it is not required at the moment.
func (s *InstanceRoleService) ListInstanceRoles(ctx context.Context, request *v1pb.ListInstanceRolesRequest) (*v1pb.ListInstanceRolesResponse, error) {
	instance, err := getInstanceMessage(ctx, s.store, request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance: %v", err)
	}
	instanceRoles := convertInstanceRoles(instance, instance.Metadata.GetRoles())
	return &v1pb.ListInstanceRolesResponse{
		Roles: instanceRoles,
	}, nil
}
