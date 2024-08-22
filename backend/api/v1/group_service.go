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

// GroupService implements the group service.
type GroupService struct {
	v1pb.UnimplementedGroupServiceServer
	store      *store.Store
	iamManager *iam.Manager
}

// NewGroupService creates a new GroupService.
func NewGroupService(store *store.Store, iamManager *iam.Manager) *GroupService {
	return &GroupService{
		store:      store,
		iamManager: iamManager,
	}
}

// GetGroup gets a group.
func (s *GroupService) GetGroup(ctx context.Context, request *v1pb.GetGroupRequest) (*v1pb.Group, error) {
	email, err := common.GetGroupEmail(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	group, err := s.store.GetGroup(ctx, email)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.convertToV1Group(ctx, group)
}

// ListGroups lists all groups.
func (s *GroupService) ListGroups(ctx context.Context, _ *v1pb.ListGroupsRequest) (*v1pb.ListGroupsResponse, error) {
	groups, err := s.store.ListGroups(ctx, &store.FindGroupMessage{})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &v1pb.ListGroupsResponse{}
	for _, groupMessage := range groups {
		group, err := s.convertToV1Group(ctx, groupMessage)
		if err != nil {
			return nil, err
		}
		response.Groups = append(response.Groups, group)
	}
	return response, nil
}

// CreateGroup creates a group.
func (s *GroupService) CreateGroup(ctx context.Context, request *v1pb.CreateGroupRequest) (*v1pb.Group, error) {
	groupMessage, err := s.convertToGroupMessage(ctx, request.Group)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}
	if len(setting.Domains) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "workspace domain is required for creating groups")
	}
	if err := validateEmail(groupMessage.Email, setting.Domains, false /* isServiceAccount */); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email %q, error: %v", groupMessage.Email, err)
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	group, err := s.store.CreateGroup(ctx, groupMessage, principalID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	return s.convertToV1Group(ctx, group)
}

// UpdateGroup updates a group.
func (s *GroupService) UpdateGroup(ctx context.Context, request *v1pb.UpdateGroupRequest) (*v1pb.Group, error) {
	groupEmail, err := common.GetGroupEmail(request.Group.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	userName := common.FormatUserUID(user.ID)

	group, err := s.store.GetGroup(ctx, groupEmail)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get group %q with error: %v", groupEmail, err)
	}
	if group == nil {
		return nil, status.Errorf(codes.NotFound, "group %q not found", groupEmail)
	}

	ok, err = func() (bool, error) {
		for _, member := range group.Payload.GetMembers() {
			if member.Role == storepb.GroupMember_OWNER && member.Member == userName {
				return true, nil
			}
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionGroupsUpdate, user)
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to update group")
	}

	patch := &store.UpdateGroupMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.Group.Title
		case "description":
			patch.Description = &request.Group.Description
		case "members":
			payload, err := s.convertToGroupPayload(ctx, request.Group)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			patch.Payload = payload
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupported update_mask "%s"`, path)
		}
	}

	groupMessage, err := s.store.UpdateGroup(ctx, groupEmail, patch, user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	return s.convertToV1Group(ctx, groupMessage)
}

// DeleteGroup deletes a group.
func (s *GroupService) DeleteGroup(ctx context.Context, request *v1pb.DeleteGroupRequest) (*emptypb.Empty, error) {
	email, err := common.GetGroupEmail(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.store.DeleteGroup(ctx, email); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *GroupService) convertToGroupPayload(ctx context.Context, group *v1pb.Group) (*storepb.GroupPayload, error) {
	payload := &storepb.GroupPayload{}
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

		m := &storepb.GroupMember{
			Member: common.FormatUserUID(user.ID),
		}
		switch member.Role {
		case v1pb.GroupMember_MEMBER:
			m.Role = storepb.GroupMember_MEMBER
		case v1pb.GroupMember_OWNER:
			m.Role = storepb.GroupMember_OWNER
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupport group member role %v", member.Role)
		}
		payload.Members = append(payload.Members, m)
	}
	return payload, nil
}

func (s *GroupService) convertToGroupMessage(ctx context.Context, group *v1pb.Group) (*store.GroupMessage, error) {
	email, err := common.GetGroupEmail(group.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	groupMessage := &store.GroupMessage{
		Email:       email,
		Title:       group.Title,
		Description: group.Description,
		Payload:     &storepb.GroupPayload{},
	}

	payload, err := s.convertToGroupPayload(ctx, group)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	groupMessage.Payload = payload
	return groupMessage, nil
}

func (s *GroupService) convertToV1Group(ctx context.Context, groupMessage *store.GroupMessage) (*v1pb.Group, error) {
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

	group := &v1pb.Group{
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

		m := &v1pb.GroupMember{
			Member: common.FormatUserEmail(user.Email),
			Role:   v1pb.GroupMember_ROLE_UNSPECIFIED,
		}
		switch member.Role {
		case storepb.GroupMember_MEMBER:
			m.Role = v1pb.GroupMember_MEMBER
		case storepb.GroupMember_OWNER:
			m.Role = v1pb.GroupMember_OWNER
		}
		group.Members = append(group.Members, m)
	}

	return group, nil
}
