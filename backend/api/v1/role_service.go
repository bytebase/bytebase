package v1

import (
	"context"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// RoleService implements the role service.
type RoleService struct {
	v1connect.UnimplementedRoleServiceHandler
	store          *store.Store
	iamManager     *iam.Manager
	licenseService *enterprise.LicenseService
}

// NewRoleService returns a new instance of the role service.
func NewRoleService(store *store.Store, iamManager *iam.Manager, licenseService *enterprise.LicenseService) *RoleService {
	return &RoleService{
		store:          store,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

// ListRoles lists roles.
func (s *RoleService) ListRoles(ctx context.Context, _ *connect.Request[v1pb.ListRolesRequest]) (*connect.Response[v1pb.ListRolesResponse], error) {
	roleMessages, err := s.store.ListRoles(ctx, &store.FindRoleMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to list roles"))
	}

	roles := convertToRoles(roleMessages, v1pb.Role_CUSTOM)
	for _, predefinedRole := range s.iamManager.PredefinedRoles {
		roles = append(roles, convertToRole(predefinedRole, v1pb.Role_BUILT_IN))
	}

	return connect.NewResponse(&v1pb.ListRolesResponse{
		Roles: roles,
	}), nil
}

// GetRole gets a role.
func (s *RoleService) GetRole(ctx context.Context, req *connect.Request[v1pb.GetRoleRequest]) (*connect.Response[v1pb.Role], error) {
	roleName := req.Msg.Name
	roleID, err := common.GetRoleID(roleName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	role, err := s.store.GetRole(ctx, &store.FindRoleMessage{ResourceID: &roleID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get role"))
	}
	if role != nil {
		return connect.NewResponse(convertToRole(role, v1pb.Role_CUSTOM)), nil
	}
	if predefinedRole := s.getBuildinRole(roleID); predefinedRole != nil {
		return connect.NewResponse(convertToRole(predefinedRole, v1pb.Role_BUILT_IN)), nil
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("role not found: %s", roleID))
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
func (s *RoleService) CreateRole(ctx context.Context, req *connect.Request[v1pb.CreateRoleRequest]) (*connect.Response[v1pb.Role], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_CUSTOM_ROLES); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	if predefinedRole := s.getBuildinRole(req.Msg.RoleId); predefinedRole != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("role %s is a built-in role", req.Msg.RoleId))
	}

	if err := validateResourceID(req.Msg.RoleId); err != nil {
		return nil, err
	}

	permissions := make(map[string]bool)
	for _, v := range req.Msg.GetRole().GetPermissions() {
		permissions[v] = true
	}
	create := &store.RoleMessage{
		ResourceID:  req.Msg.RoleId,
		Name:        req.Msg.Role.Title,
		Description: req.Msg.Role.Description,
		Permissions: permissions,
	}
	if ok := iam.PermissionsExist(req.Msg.Role.Permissions...); !ok {
		invalidPerms := []string{}
		for _, p := range req.Msg.Role.Permissions {
			if !iam.PermissionExist(p) {
				invalidPerms = append(invalidPerms, p)
			}
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid permissions: %v", invalidPerms))
	}
	roleMessage, err := s.store.CreateRole(ctx, create)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create role"))
	}
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(convertToRole(roleMessage, v1pb.Role_CUSTOM)), nil
}

// UpdateRole updates an existing role.
func (s *RoleService) UpdateRole(ctx context.Context, req *connect.Request[v1pb.UpdateRoleRequest]) (*connect.Response[v1pb.Role], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_CUSTOM_ROLES); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}
	roleID, err := common.GetRoleID(req.Msg.Role.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if predefinedRole := s.getBuildinRole(roleID); predefinedRole != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot change the build-in role %s", req.Msg.Role.Name))
	}
	role, err := s.store.GetRole(ctx, &store.FindRoleMessage{ResourceID: &roleID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get role"))
	}
	if role == nil {
		if req.Msg.AllowMissing {
			return s.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
				Role:   req.Msg.Role,
				RoleId: roleID,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("role not found: %s", roleID))
	}
	patch := &store.UpdateRoleMessage{
		ResourceID: roleID,
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &req.Msg.Role.Title
		case "description":
			patch.Description = &req.Msg.Role.Description
		case "permissions":
			permissions := make(map[string]bool)
			for _, v := range req.Msg.GetRole().GetPermissions() {
				permissions[v] = true
			}
			patch.Permissions = &permissions
			if ok := iam.PermissionsExist(req.Msg.Role.Permissions...); !ok {
				invalidPerms := []string{}
				for _, p := range req.Msg.Role.Permissions {
					if !iam.PermissionExist(p) {
						invalidPerms = append(invalidPerms, p)
					}
				}
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid permissions: %v", invalidPerms))
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path: %s", path))
		}
	}

	roleMessage, err := s.store.UpdateRole(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update role"))
	}
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(convertToRole(roleMessage, v1pb.Role_CUSTOM)), nil
}

// DeleteRole deletes an existing role.
func (s *RoleService) DeleteRole(ctx context.Context, req *connect.Request[v1pb.DeleteRoleRequest]) (*connect.Response[emptypb.Empty], error) {
	roleID, err := common.GetRoleID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if predefinedRole := s.getBuildinRole(roleID); predefinedRole != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot delete the build-in role %s", req.Msg.Name))
	}
	role, err := s.store.GetRole(ctx, &store.FindRoleMessage{ResourceID: &roleID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get role"))
	}
	if role == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("role not found: %s", roleID))
	}

	usedByResources, err := s.store.GetResourcesUsedByRole(ctx, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check if the role is used"))
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
			} else if usedResource.ResourceType == storepb.Policy_WORKSPACE {
				usedBy = append(usedBy, "workspace")
			}
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("cannot delete because role %s is used by resources: %s", common.FormatRole(roleID), strings.Join(usedBy, ",")))
	}

	if err := s.store.DeleteRole(ctx, roleID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to delete role"))
	}
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
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
