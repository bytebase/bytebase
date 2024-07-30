package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
func (s *InstanceRoleService) GetInstanceRole(ctx context.Context, request *v1pb.GetInstanceRoleRequest) (*v1pb.InstanceRole, error) {
	instanceID, roleName, err := common.GetInstanceRoleID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.getInstanceMessage(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	role, err := func() (*db.DatabaseRoleMessage, error) {
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)

		role, err := driver.FindRole(ctx, roleName)
		if err != nil {
			return nil, err
		}

		return role, nil
	}()
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, status.Errorf(codes.NotFound, "role %s not found", request.Name)
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstanceRole(role, instance), nil
}

// ListInstanceRoles lists all roles in an instance.
func (s *InstanceRoleService) ListInstanceRoles(ctx context.Context, request *v1pb.ListInstanceRolesRequest) (*v1pb.ListInstanceRolesResponse, error) {
	instanceID, err := common.GetInstanceID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.getInstanceMessage(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	if !request.Refresh {
		instanceUsers, err := s.store.ListInstanceUsers(ctx, &store.FindInstanceUserMessage{InstanceUID: instance.UID})
		if err != nil {
			return nil, err
		}
		response := &v1pb.ListInstanceRolesResponse{}
		for _, u := range instanceUsers {
			response.Roles = append(response.Roles, &v1pb.InstanceRole{
				Name:      fmt.Sprintf("instances/%s/roles/%s", instance.ResourceID, u.Name),
				RoleName:  u.Name,
				Attribute: &u.Grant,
			})
		}
		return response, nil
	}

	roleList, err := func() ([]*db.DatabaseRoleMessage, error) {
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)

		roleList, err := driver.ListRole(ctx)
		if err != nil {
			return nil, err
		}

		return roleList, nil
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.ListInstanceRolesResponse{}
	for _, role := range roleList {
		response.Roles = append(response.Roles, convertToInstanceRole(role, instance))
	}
	return response, nil
}

func (s *InstanceRoleService) getInstanceMessage(ctx context.Context, instanceID string) (*store.InstanceMessage, error) {
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	return instance, nil
}

func convertToInstanceRole(role *db.DatabaseRoleMessage, instance *store.InstanceMessage) *v1pb.InstanceRole {
	return &v1pb.InstanceRole{
		Name:            fmt.Sprintf("instances/%s/roles/%s", instance.ResourceID, role.Name),
		RoleName:        role.Name,
		ConnectionLimit: &role.ConnectionLimit,
		ValidUntil:      role.ValidUntil,
		Attribute:       role.Attribute,
	}
}
