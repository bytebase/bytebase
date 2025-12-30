package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// GroupService implements the group service.
type GroupService struct {
	v1connect.UnimplementedGroupServiceHandler
	store          *store.Store
	iamManager     *iam.Manager
	licenseService *enterprise.LicenseService
}

// NewGroupService creates a new GroupService.
func NewGroupService(store *store.Store, iamManager *iam.Manager, licenseService *enterprise.LicenseService) *GroupService {
	return &GroupService{
		store:          store,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

// GetGroup gets a group.
func (s *GroupService) GetGroup(ctx context.Context, req *connect.Request[v1pb.GetGroupRequest]) (*connect.Response[v1pb.Group], error) {
	group, err := utils.GetGroupByName(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if group == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot found the group %v", req.Msg.Name))
	}

	result, err := s.convertToV1Group(group)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// BatchGetGroups get groups in batch.
func (s *GroupService) BatchGetGroups(ctx context.Context, request *connect.Request[v1pb.BatchGetGroupsRequest]) (*connect.Response[v1pb.BatchGetGroupsResponse], error) {
	response := &v1pb.BatchGetGroupsResponse{}
	for _, name := range request.Msg.Names {
		group, err := s.GetGroup(ctx, connect.NewRequest(&v1pb.GetGroupRequest{Name: name}))
		if err != nil {
			slog.Error("failed to find group", slog.String("name", name), log.BBError(err))
			continue
		}
		response.Groups = append(response.Groups, group.Msg)
	}
	return connect.NewResponse(response), nil
}

// ListGroups lists all groups.
func (s *GroupService) ListGroups(ctx context.Context, request *connect.Request[v1pb.ListGroupsRequest]) (*connect.Response[v1pb.ListGroupsResponse], error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.Msg.PageToken,
		limit:   int(request.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindGroupMessage{
		Limit:  &limitPlusOne,
		Offset: &offset.offset,
	}
	filterQ, err := store.GetListGroupFilter(find, request.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

	groups, err := s.store.ListGroups(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	nextPageToken := ""
	if len(groups) == limitPlusOne {
		groups = groups[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListGroupsResponse{
		NextPageToken: nextPageToken,
	}
	for _, groupMessage := range groups {
		group, err := s.convertToV1Group(groupMessage)
		if err != nil {
			return nil, err
		}
		response.Groups = append(response.Groups, group)
	}
	return connect.NewResponse(response), nil
}

// CreateGroup creates a group.
func (s *GroupService) CreateGroup(ctx context.Context, req *connect.Request[v1pb.CreateGroupRequest]) (*connect.Response[v1pb.Group], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_USER_GROUPS); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	groupMessage, err := s.convertToGroupMessage(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, groupMessage.Email, false /* isServiceAccount */, true); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid email %q", groupMessage.Email))
	}

	group, err := s.store.CreateGroup(ctx, groupMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	result, err := s.convertToV1Group(group)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// UpdateGroup updates a group.
func (s *GroupService) UpdateGroup(ctx context.Context, req *connect.Request[v1pb.UpdateGroupRequest]) (*connect.Response[v1pb.Group], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_USER_GROUPS); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	group, err := utils.GetGroupByName(ctx, s.store, req.Msg.Group.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if group == nil {
		if req.Msg.AllowMissing {
			groupEmail, err := common.GetGroupEmail(req.Msg.Group.Name)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			// Permission check is now handled by the ACL interceptor
			// which verifies both bb.groups.update and bb.groups.create
			return s.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
				Group:      req.Msg.Group,
				GroupEmail: groupEmail,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("group %q not found", req.Msg.Group.Name))
	}
	if group.Payload.Source != "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("not support update external group %q", req.Msg.Group.Name))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}
	if err := s.checkPermission(ctx, group, user, iam.PermissionGroupsUpdate); err != nil {
		return nil, err
	}

	patch := &store.UpdateGroupMessage{
		ID: group.ID,
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "email":
			patch.Email = &req.Msg.Group.Email
		case "title":
			patch.Title = &req.Msg.Group.Title
		case "description":
			patch.Description = &req.Msg.Group.Description
		case "members":
			if group.Payload.Source != "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot change members for external group"))
			}
			payload, err := s.convertToGroupPayload(ctx, req.Msg.Group)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			patch.Payload = payload
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupported update_mask "%s"`, path))
		}
	}

	groupMessage, err := s.store.UpdateGroup(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if patch.Payload != nil {
		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return nil, err
		}
	}

	result, err := s.convertToV1Group(groupMessage)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// DeleteGroup deletes a group.
func (s *GroupService) DeleteGroup(ctx context.Context, req *connect.Request[v1pb.DeleteGroupRequest]) (*connect.Response[emptypb.Empty], error) {
	group, err := utils.GetGroupByName(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if group == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot found the group %v", req.Msg.Name))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	if err := s.checkPermission(ctx, group, user, iam.PermissionGroupsDelete); err != nil {
		return nil, err
	}

	if err := s.store.DeleteGroup(ctx, group.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *GroupService) checkPermission(ctx context.Context, group *store.GroupMessage, user *store.UserMessage, permission string) error {
	userName := common.FormatUserEmail(user.Email)

	ok, err := func() (bool, error) {
		for _, member := range group.Payload.GetMembers() {
			if member.Role == storepb.GroupMember_OWNER && member.Member == userName {
				return true, nil
			}
		}
		return s.iamManager.CheckPermission(ctx, permission, user)
	}()
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
	}
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission))
	}
	return nil
}

func (s *GroupService) convertToGroupPayload(ctx context.Context, group *v1pb.Group) (*storepb.GroupPayload, error) {
	payload := &storepb.GroupPayload{}
	for _, member := range group.Members {
		email, err := common.GetUserEmail(member.Member)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get member email"))
		}
		user, err := s.store.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get member %s", member.Member))
		}
		if user == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot found member %s", member.Member))
		}
		if user.Type != storepb.PrincipalType_END_USER {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only allow add end users to the group"))
		}

		m := &storepb.GroupMember{
			Member: common.FormatUserEmail(user.Email),
		}
		switch member.Role {
		case v1pb.GroupMember_MEMBER:
			m.Role = storepb.GroupMember_MEMBER
		case v1pb.GroupMember_OWNER:
			m.Role = storepb.GroupMember_OWNER
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport group member role %v", member.Role))
		}
		payload.Members = append(payload.Members, m)
	}
	return payload, nil
}

func (s *GroupService) convertToGroupMessage(ctx context.Context, request *v1pb.CreateGroupRequest) (*store.GroupMessage, error) {
	if request.GroupEmail == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing group_email in the request"))
	}

	groupMessage := &store.GroupMessage{
		Email:       request.GroupEmail,
		Title:       request.Group.Title,
		Description: request.Group.Description,
		Payload:     &storepb.GroupPayload{},
	}

	payload, err := s.convertToGroupPayload(ctx, request.Group)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	groupMessage.Payload = payload
	return groupMessage, nil
}

func (*GroupService) convertToV1Group(groupMessage *store.GroupMessage) (*v1pb.Group, error) {
	if groupMessage == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("cannot found group"))
	}

	group := &v1pb.Group{
		Name:        utils.FormatGroupName(groupMessage),
		Title:       groupMessage.Title,
		Description: groupMessage.Description,
		Source:      groupMessage.Payload.Source,
		Email:       groupMessage.Email,
	}

	for _, member := range groupMessage.Payload.Members {
		m := &v1pb.GroupMember{
			Member: member.Member,
			Role:   v1pb.GroupMember_ROLE_UNSPECIFIED,
		}
		switch member.Role {
		case storepb.GroupMember_MEMBER:
			m.Role = v1pb.GroupMember_MEMBER
		case storepb.GroupMember_OWNER:
			m.Role = v1pb.GroupMember_OWNER
		default:
		}
		group.Members = append(group.Members, m)
	}

	return group, nil
}
