package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// UserGroupService implements the user group service.
type UserGroupService struct {
	v1pb.UnimplementedUserGroupServiceServer
	store      *store.Store
	iamManager *iam.Manager
}

// NewUserGroupService creates a new UserGroupService.
func NewUserGroupService(store *store.Store, iamManager *iam.Manager) *UserGroupService {
	return &UserGroupService{
		store:      store,
		iamManager: iamManager,
	}
}

// GetUserGroup gets a group.
func (s *UserGroupService) GetUserGroup(ctx context.Context, request *v1pb.GetUserGroupRequest) (*v1pb.UserGroup, error) {
	email, err := common.GetUserGroupEmail(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	group, err := s.store.GetUserGroup(ctx, email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return s.convertToV1Group(ctx, group)
}

// ListUserGroups lists all groups.
func (s *UserGroupService) ListUserGroups(ctx context.Context, _ *v1pb.ListUserGroupsRequest) (*v1pb.ListUserGroupsResponse, error) {
	groups, err := s.store.ListUserGroups(ctx, &store.FindUserGroupMessage{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.ListUserGroupsResponse{}
	for _, groupMessage := range groups {
		group, err := s.convertToV1Group(ctx, groupMessage)
		if err != nil {
			return nil, err
		}
		response.Groups = append(response.Groups, group)
	}
	return response, nil
}

// CreateUserGroup creates a group.
func (s *UserGroupService) CreateUserGroup(ctx context.Context, request *v1pb.CreateUserGroupRequest) (*v1pb.UserGroup, error) {
	groupMessage, err := s.convertToGroupMessage(ctx, request.Group)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}
	if len(setting.Domains) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "workspace domain is required for creating user groups")
	}
	if err := validateEmail(groupMessage.Email, setting.Domains, false /* isServiceAccount */); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email %q, error: %v", groupMessage.Email, err)
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	group, err := s.store.CreateUserGroup(ctx, groupMessage, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return s.convertToV1Group(ctx, group)
}

// UpdateUserGroup updates a group.
func (s *UserGroupService) UpdateUserGroup(ctx context.Context, request *v1pb.UpdateUserGroupRequest) (*v1pb.UserGroup, error) {
	email, err := common.GetUserGroupEmail(request.Group.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	if err := s.checkPermission(ctx, email, iam.PermissionUserGroupsUpdate); err != nil {
		return nil, err
	}

	patch := &store.UpdateUserGroupMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.Group.Title
		case "description":
			patch.Description = &request.Group.Description
		case "members":
			payload, err := s.convertToGroupPayload(ctx, request.Group)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.Payload = payload
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupported update_mask "%s"`, path)
		}
	}

	groupMessage, err := s.store.UpdateUserGroup(ctx, email, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return s.convertToV1Group(ctx, groupMessage)
}

// DeleteUserGroup deletes a group.
func (s *UserGroupService) DeleteUserGroup(ctx context.Context, request *v1pb.DeleteUserGroupRequest) (*emptypb.Empty, error) {
	email, err := common.GetUserGroupEmail(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := s.checkPermission(ctx, email, iam.PermissionUserGroupsDelete); err != nil {
		return nil, err
	}

	if err := s.store.DeleteUserGroup(ctx, email); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UserGroupService) checkPermission(ctx context.Context, groupEmail string, permission iam.Permission) error {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return status.Errorf(codes.Internal, "user not found")
	}

	group, err := s.store.GetUserGroup(ctx, groupEmail)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get user group %q with error: %v", groupEmail, err)
	}
	if group == nil {
		return status.Errorf(codes.NotFound, "group %q not found", groupEmail)
	}
	for _, member := range group.Payload.GetMembers() {
		if member.Role == storepb.UserGroupMember_OWNER && member.Member == common.FormatUserUID(user.ID) {
			return nil
		}
	}
	hasPermission, err := s.iamManager.CheckPermission(ctx, permission, user)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check permission with error: %v", err)
	}
	if !hasPermission {
		return status.Errorf(codes.PermissionDenied, "permission denied: %v", permission)
	}
	return nil
}

func (s *UserGroupService) convertToGroupPayload(ctx context.Context, group *v1pb.UserGroup) (*storepb.UserGroupPayload, error) {
	payload := &storepb.UserGroupPayload{}
	for _, member := range group.Members {
		email, err := common.GetUserEmail(member.Member)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get member email, error %v", err)
		}
		user, err := s.store.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get member %s, error %v", member.Member, err)
		}
		if user == nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot found member %s", member.Member)
		}

		m := &storepb.UserGroupMember{
			Member: common.FormatUserUID(user.ID),
		}
		switch member.Role {
		case v1pb.UserGroupMember_MEMBER:
			m.Role = storepb.UserGroupMember_MEMBER
		case v1pb.UserGroupMember_OWNER:
			m.Role = storepb.UserGroupMember_OWNER
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupport group member role %v", member.Role)
		}
		payload.Members = append(payload.Members, m)
	}
	return payload, nil
}

func (s *UserGroupService) convertToGroupMessage(ctx context.Context, group *v1pb.UserGroup) (*store.UserGroupMessage, error) {
	email, err := common.GetUserGroupEmail(group.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	groupMessage := &store.UserGroupMessage{
		Email:       email,
		Title:       group.Title,
		Description: group.Description,
		Payload:     &storepb.UserGroupPayload{},
	}

	payload, err := s.convertToGroupPayload(ctx, group)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	groupMessage.Payload = payload
	return groupMessage, nil
}

func (s *UserGroupService) convertToV1Group(ctx context.Context, groupMessage *store.UserGroupMessage) (*v1pb.UserGroup, error) {
	if groupMessage == nil {
		return nil, status.Errorf(codes.NotFound, "cannot found group")
	}
	creator, err := s.store.GetUserByID(ctx, groupMessage.CreatorUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get creator, error %v", err)
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, "creator %d not found", groupMessage.CreatorUID)
	}

	group := &v1pb.UserGroup{
		Name:        common.FormatGroupEmail(groupMessage.Email),
		Title:       groupMessage.Title,
		Description: groupMessage.Description,
		Creator:     common.FormatUserEmail(creator.Email),
		CreateTime:  timestamppb.New(groupMessage.CreatedTime),
	}

	for _, member := range groupMessage.Payload.Members {
		uid, err := common.GetUserID(member.Member)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get member id, error %v", err)
		}
		user, err := s.store.GetUserByID(ctx, uid)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get member, error %v", err)
		}
		if user == nil {
			continue
		}

		m := &v1pb.UserGroupMember{
			Member: common.FormatUserEmail(user.Email),
			Role:   v1pb.UserGroupMember_ROLE_UNSPECIFIED,
		}
		switch member.Role {
		case storepb.UserGroupMember_MEMBER:
			m.Role = v1pb.UserGroupMember_MEMBER
		case storepb.UserGroupMember_OWNER:
			m.Role = v1pb.UserGroupMember_OWNER
		}
		group.Members = append(group.Members, m)
	}

	return group, nil
}
