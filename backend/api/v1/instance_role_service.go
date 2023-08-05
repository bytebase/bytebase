package v1

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

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
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
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
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
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

// CreateInstanceRole creates an role.
func (s *InstanceRoleService) CreateInstanceRole(ctx context.Context, request *v1pb.CreateInstanceRoleRequest) (*v1pb.InstanceRole, error) {
	if request.Role == nil {
		return nil, status.Errorf(codes.InvalidArgument, "role must be set")
	}
	instanceID, err := common.GetInstanceID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.getInstanceMessage(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", instanceID)
	}

	roleUpsert := &db.DatabaseRoleUpsertMessage{
		Name:            request.Role.RoleName,
		Password:        request.Role.Password,
		ConnectionLimit: request.Role.ConnectionLimit,
		ValidUntil:      request.Role.ValidUntil,
		Attribute:       request.Role.Attribute,
	}
	if err := validateRole(instance.Engine, roleUpsert); err != nil {
		return nil, err
	}

	role, err := func() (*db.DatabaseRoleMessage, error) {
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)

		role, err := driver.CreateRole(ctx, roleUpsert)
		if err != nil {
			return nil, err
		}

		return role, nil
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstanceRole(role, instance), nil
}

// UpdateInstanceRole updates an role.
func (s *InstanceRoleService) UpdateInstanceRole(ctx context.Context, request *v1pb.UpdateInstanceRoleRequest) (*v1pb.InstanceRole, error) {
	if request.Role == nil {
		return nil, status.Errorf(codes.InvalidArgument, "role must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instanceID, roleName, err := common.GetInstanceRoleID(request.Role.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.getInstanceMessage(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", instanceID)
	}

	upsert := &db.DatabaseRoleUpsertMessage{
		Name: roleName,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "role_name":
			upsert.Name = request.Role.RoleName
		case "password":
			upsert.Password = request.Role.Password
		case "connection_limit":
			upsert.ConnectionLimit = request.Role.ConnectionLimit
		case "valid_until":
			upsert.ValidUntil = request.Role.ValidUntil
		case "attribute":
			upsert.Attribute = request.Role.Attribute
		}
	}
	if err := validateRole(instance.Engine, upsert); err != nil {
		return nil, err
	}

	role, err := func() (*db.DatabaseRoleMessage, error) {
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)

		role, err := driver.UpdateRole(ctx, roleName, upsert)
		if err != nil {
			return nil, err
		}

		return role, nil
	}()
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, status.Errorf(codes.NotFound, "role %s not found", request.Role.Name)
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstanceRole(role, instance), nil
}

// DeleteInstanceRole deletes an role.
func (s *InstanceRoleService) DeleteInstanceRole(ctx context.Context, request *v1pb.DeleteInstanceRoleRequest) (*emptypb.Empty, error) {
	instanceID, roleName, err := common.GetInstanceRoleID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.getInstanceMessage(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", instanceID)
	}

	if err := func() error {
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

		return driver.DeleteRole(ctx, roleName)
	}(); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// UndeleteInstanceRole undeletes an role.
func (*InstanceRoleService) UndeleteInstanceRole(_ context.Context, _ *v1pb.UndeleteInstanceRoleRequest) (*v1pb.InstanceRole, error) {
	return nil, status.Errorf(codes.Unimplemented, "Undelete role is not supported")
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

func validateRole(dbType db.Type, upsert *db.DatabaseRoleUpsertMessage) error {
	if upsert.Name == "" {
		return status.Errorf(codes.InvalidArgument, "Invalid role name, role name cannot be empty")
	}
	switch dbType {
	case db.Postgres, db.RisingWave:
		if v := upsert.ConnectionLimit; v != nil && *v < int32(-1) {
			return status.Errorf(codes.InvalidArgument, "Invalid connection limit, it should greater than or equal to -1")
		}
		if v := upsert.ValidUntil; v != nil {
			if _, err := time.Parse(time.RFC3339, *v); err != nil {
				return status.Errorf(codes.InvalidArgument, "Invalid timestamp for valid_until, timestamp should in '2006-01-02T15:04:05+08:00' format.")
			}
		}
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		if v := upsert.ConnectionLimit; v != nil && *v < int32(0) {
			return status.Errorf(codes.InvalidArgument, "Invalid connection limit, it should greater than or equal to -1")
		}
		if v := upsert.ValidUntil; v != nil {
			if _, err := strconv.Atoi(*v); err != nil {
				return status.Error(codes.InvalidArgument, "Invalid number for valid_until, mysql valid_until should be an integer.")
			}
		}
	}

	return nil
}
