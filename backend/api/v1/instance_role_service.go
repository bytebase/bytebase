package v1

import (
	"context"

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

// ListInstanceRoles lists all roles in an instance.
func (*InstanceRoleService) ListInstanceRoles(_ context.Context, _ *v1pb.ListInstanceRolesRequest) (*v1pb.ListInstanceRolesResponse, error) {
	return &v1pb.ListInstanceRolesResponse{}, nil
}
