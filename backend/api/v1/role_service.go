package v1

import (
	"context"
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RoleService implements the role service.
type RoleService struct {
	v1pb.UnimplementedRoleServiceServer
	store          *store.Store
	iamManager     *iam.Manager
	licenseService enterprise.LicenseService
}

// NewRoleService returns a new instance of the role service.
func NewRoleService(store *store.Store, iamManager *iam.Manager, licenseService enterprise.LicenseService) *RoleService {
	return &RoleService{
		store:          store,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

// ListRoles lists roles.
func (s *RoleService) ListRoles(ctx context.Context, _ *v1pb.ListRolesRequest) (*v1pb.ListRolesResponse, error) {
	roleMessages, err := s.store.ListRoles(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list roles: %v", err)
	}

	roles := convertToRoles(roleMessages, v1pb.Role_CUSTOM)
	for _, predefinedRole := range s.iamManager.PredefinedRoles {
		roles = append(roles, convertToRole(predefinedRole, v1pb.Role_BUILT_IN))
	}

	return &v1pb.ListRolesResponse{
		Roles: roles,
	}, nil
}

// GetRole gets a role.
func (s *RoleService) GetRole(ctx context.Context, request *v1pb.GetRoleRequest) (*v1pb.Role, error) {
	roleName := request.Name
	roleID, err := common.GetRoleID(roleName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get role: %v", err)
	}
	if role != nil {
		return convertToRole(role, v1pb.Role_CUSTOM), nil
	}
	if predefinedRole := s.getBuildinRole(roleID); predefinedRole != nil {
		return convertToRole(predefinedRole, v1pb.Role_BUILT_IN), nil
	}
	return nil, status.Errorf(codes.NotFound, "role not found: %s", roleID)
}

func (s *RoleService) getBuildinRole(roleID string) *store.RoleMessage {
	for _, predefinedRole := range s.iamManager.PredefinedRoles {
		if predefinedRole.ResourceID == roleID {
			return predefinedRole
		}
	}
	return nil
}

// CreateRole creates a new role.
func (s *RoleService) CreateRole(ctx context.Context, request *v1pb.CreateRoleRequest) (*v1pb.Role, error) {
	if err := s.licenseService.IsFeatureEnabled(base.FeatureCustomRole); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if predefinedRole := s.getBuildinRole(request.RoleId); predefinedRole != nil {
		return nil, status.Errorf(codes.InvalidArgument, "role %s is a built-in role", request.RoleId)
	}

	if err := validateResourceID(request.RoleId); err != nil {
		return nil, err
	}

	permissions := make(map[string]bool)
	for _, v := range request.GetRole().GetPermissions() {
		permissions[v] = true
	}
	create := &store.RoleMessage{
		ResourceID:  request.RoleId,
		Name:        request.Role.Title,
		Description: request.Role.Description,
		Permissions: permissions,
	}
	if ok := iam.PermissionsExist(request.Role.Permissions...); !ok {
		return nil, status.Errorf(codes.InvalidArgument, "invalid permissions")
	}
	roleMessage, err := s.store.CreateRole(ctx, create)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create role: %v", err)
	}
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}
	return convertToRole(roleMessage, v1pb.Role_CUSTOM), nil
}

// UpdateRole updates an existing role.
func (s *RoleService) UpdateRole(ctx context.Context, request *v1pb.UpdateRoleRequest) (*v1pb.Role, error) {
	if err := s.licenseService.IsFeatureEnabled(base.FeatureCustomRole); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	roleID, err := common.GetRoleID(request.Role.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if predefinedRole := s.getBuildinRole(roleID); predefinedRole != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot change the build-in role %s", request.Role.Name)
	}
	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get role: %v", err)
	}
	if role == nil {
		if request.AllowMissing {
			return s.CreateRole(ctx, &v1pb.CreateRoleRequest{
				Role:   request.Role,
				RoleId: roleID,
			})
		}
		return nil, status.Errorf(codes.NotFound, "role not found: %s", roleID)
	}
	patch := &store.UpdateRoleMessage{
		ResourceID: roleID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &request.Role.Title
		case "description":
			patch.Description = &request.Role.Description
		case "permissions":
			permissions := make(map[string]bool)
			for _, v := range request.GetRole().GetPermissions() {
				permissions[v] = true
			}
			patch.Permissions = &permissions
			if ok := iam.PermissionsExist(request.Role.Permissions...); !ok {
				return nil, status.Errorf(codes.InvalidArgument, "invalid permissions")
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid update mask path: %s", path)
		}
	}

	roleMessage, err := s.store.UpdateRole(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update role: %v", err)
	}
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}
	return convertToRole(roleMessage, v1pb.Role_CUSTOM), nil
}

// DeleteRole deletes an existing role.
func (s *RoleService) DeleteRole(ctx context.Context, request *v1pb.DeleteRoleRequest) (*emptypb.Empty, error) {
	roleID, err := common.GetRoleID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if predefinedRole := s.getBuildinRole(roleID); predefinedRole != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot delete the build-in role %s", request.Name)
	}
	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get role: %v", err)
	}
	if role == nil {
		return nil, status.Errorf(codes.NotFound, "role not found: %s", roleID)
	}

	usedByResources, err := s.store.GetResourcesUsedByRole(ctx, request.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the role is used: %v", err)
	}
	if len(usedByResources) > 0 {
		usedBy := []string{}
		for i, usedResource := range usedByResources {
			if i >= 10 {
				// Limit the message length.
				break
			}
			if usedResource.Resource != "" {
				usedBy = append(usedBy, usedResource.Resource)
			} else if usedResource.ResourceType == base.PolicyResourceTypeWorkspace {
				usedBy = append(usedBy, "workspace")
			}
		}
		return nil, status.Errorf(codes.FailedPrecondition, "cannot delete because role %s is used by resources: %s", common.FormatRole(roleID), strings.Join(usedBy, ","))
	}

	if err := s.store.DeleteRole(ctx, roleID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete role: %v", err)
	}
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func convertToRoles(roleMessages []*store.RoleMessage, roleType v1pb.Role_Type) []*v1pb.Role {
	var roles []*v1pb.Role
	for _, roleMessage := range roleMessages {
		roles = append(roles, convertToRole(roleMessage, roleType))
	}
	return roles
}

func convertToRole(role *store.RoleMessage, roleType v1pb.Role_Type) *v1pb.Role {
	var permissions []string
	for p := range role.Permissions {
		permissions = append(permissions, p)
	}
	slices.Sort(permissions)
	return &v1pb.Role{
		Name:        common.FormatRole(role.ResourceID),
		Title:       role.Name,
		Description: role.Description,
		Permissions: permissions,
		Type:        roleType,
	}
}
