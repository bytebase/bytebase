package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	roles, err := convertToRoles(ctx, s.iamManager, roleMessages)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert roles: %v", err)
	}

	return &v1pb.ListRolesResponse{
		Roles: roles,
	}, nil
}

// CreateRole creates a new role.
func (s *RoleService) CreateRole(ctx context.Context, request *v1pb.CreateRoleRequest) (*v1pb.Role, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureCustomRole); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
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
	roleMessage, err := s.store.CreateRole(ctx, create, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create role: %v", err)
	}
	role, err := convertToRole(ctx, s.iamManager, roleMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to role: %v", err)
	}
	return role, nil
}

// UpdateRole updates an existing role.
func (s *RoleService) UpdateRole(ctx context.Context, request *v1pb.UpdateRoleRequest) (*v1pb.Role, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureCustomRole); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	roleID, err := common.GetRoleID(request.Role.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get role: %v", err)
	}
	if role == nil {
		return nil, status.Errorf(codes.NotFound, "role not found: %s", roleID)
	}
	patch := &store.UpdateRoleMessage{
		UpdaterID:  principalID,
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
	convertedRole, err := convertToRole(ctx, s.iamManager, roleMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to role: %v", err)
	}
	return convertedRole, nil
}

// DeleteRole deletes an existing role.
func (s *RoleService) DeleteRole(ctx context.Context, request *v1pb.DeleteRoleRequest) (*emptypb.Empty, error) {
	roleID, err := common.GetRoleID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get role: %v", err)
	}
	if role == nil {
		return nil, status.Errorf(codes.NotFound, "role not found: %s", roleID)
	}

	has, projectUID, err := s.getProjectUsingRole(ctx, request.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the role is used: %v", err)
	}
	if has {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot delete because role %s is used in project %v", common.FormatRole(roleID), projectUID)
	}
	if err := s.store.DeleteRole(ctx, roleID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete role: %v", err)
	}
	return &emptypb.Empty{}, nil
}

// TODO(p0ny): consider using sql for better performance?
func (s *RoleService) getProjectUsingRole(ctx context.Context, role string) (bool, int, error) {
	resourceType := api.PolicyResourceTypeProject
	policyType := api.PolicyTypeIAM

	policies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
	})
	if err != nil {
		return false, 0, err
	}

	for _, policy := range policies {
		iamPolicy := &storepb.IamPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), iamPolicy); err != nil {
			return false, 0, errors.Wrapf(err, "failed to unmarshal iam policy")
		}

		for _, binding := range iamPolicy.Bindings {
			if binding.Role == role {
				return true, policy.ResourceUID, nil
			}
		}
	}

	return false, 0, nil
}

func convertToRoles(ctx context.Context, iamManager *iam.Manager, roleMessages []*store.RoleMessage) ([]*v1pb.Role, error) {
	var roles []*v1pb.Role
	for _, roleMessage := range roleMessages {
		role, err := convertToRole(ctx, iamManager, roleMessage)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func convertToRole(ctx context.Context, iamManager *iam.Manager, role *store.RoleMessage) (*v1pb.Role, error) {
	name := common.FormatRole(role.ResourceID)
	permissions, err := iamManager.GetPermissions(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get permissions")
	}
	convertedPermissions := []string{}
	for permission := range permissions {
		convertedPermissions = append(convertedPermissions, string(permission))
	}
	return &v1pb.Role{
		Name:        name,
		Title:       role.Name,
		Description: role.Description,
		Permissions: convertedPermissions,
	}, nil
}
