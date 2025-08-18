package v1

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
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
	email, err := common.GetGroupEmail(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	group, err := s.store.GetGroup(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result, err := s.convertToV1Group(ctx, group)
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
			return nil, err
		}
		response.Groups = append(response.Groups, group.Msg)
	}
	return connect.NewResponse(response), nil
}

func parseListGroupFilter(find *store.FindGroupMessage, filter string) error {
	if filter == "" {
		return nil
	}
	e, err := cel.NewEnv()
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.New("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "title":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("name = $%d", len(positionalArgs)), nil
		case "email":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("email = $%d", len(positionalArgs)), nil
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid project filter %q", value))
			}
			find.ProjectID = &projectID
			return "TRUE", nil
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
		}
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubConditionFromExpr(expr, getFilter, "OR")
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid args for %q`, variable))
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", value))
				}
				if strValue == "" {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`empty value for %q`, variable))
				}

				switch variable {
				case "title":
					return "LOWER(name) LIKE '%" + strings.ToLower(strValue) + "%'", nil
				case "email":
					return "LOWER(email) LIKE '%" + strings.ToLower(strValue) + "%'", nil
				default:
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
				}
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return err
	}

	find.Filter = &store.ListResourceFilter{
		Args:  positionalArgs,
		Where: "(" + where + ")",
	}

	return nil
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
	if err := parseListGroupFilter(find, request.Msg.Filter); err != nil {
		return nil, err
	}

	groups, err := s.store.ListGroups(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	nextPageToken := ""
	if len(groups) == limitPlusOne {
		groups = groups[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal next page token, error: %v", err))
		}
	}

	response := &v1pb.ListGroupsResponse{
		NextPageToken: nextPageToken,
	}
	for _, groupMessage := range groups {
		group, err := s.convertToV1Group(ctx, groupMessage)
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

	result, err := s.convertToV1Group(ctx, group)
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
	groupEmail, err := common.GetGroupEmail(req.Msg.Group.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	group, err := s.store.GetGroup(ctx, groupEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get group %q", groupEmail))
	}
	if group == nil {
		if req.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionGroupsCreate, user)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionGroupsCreate))
			}
			return s.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
				Group:      req.Msg.Group,
				GroupEmail: groupEmail,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("group %q not found", groupEmail))
	}

	if err := s.checkPermission(ctx, group, user, iam.PermissionGroupsUpdate); err != nil {
		return nil, err
	}

	patch := &store.UpdateGroupMessage{}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
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

	groupMessage, err := s.store.UpdateGroup(ctx, groupEmail, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	result, err := s.convertToV1Group(ctx, groupMessage)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// DeleteGroup deletes a group.
func (s *GroupService) DeleteGroup(ctx context.Context, req *connect.Request[v1pb.DeleteGroupRequest]) (*connect.Response[emptypb.Empty], error) {
	email, err := common.GetGroupEmail(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	group, err := s.store.GetGroup(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get group %q", email))
	}
	if group == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot found the group %v", req.Msg.Name))
	}

	if err := s.checkPermission(ctx, group, user, iam.PermissionGroupsDelete); err != nil {
		return nil, err
	}

	if err := s.store.DeleteGroup(ctx, email); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *GroupService) checkPermission(ctx context.Context, group *store.GroupMessage, user *store.UserMessage, permission string) error {
	userName := common.FormatUserUID(user.ID)

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
			Member: common.FormatUserUID(user.ID),
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

func (s *GroupService) convertToV1Group(ctx context.Context, groupMessage *store.GroupMessage) (*v1pb.Group, error) {
	if groupMessage == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("cannot found group"))
	}

	group := &v1pb.Group{
		Name:        common.FormatGroupEmail(groupMessage.Email),
		Title:       groupMessage.Title,
		Description: groupMessage.Description,
		Source:      groupMessage.Payload.Source,
	}

	for _, member := range groupMessage.Payload.Members {
		uid, err := common.GetUserID(member.Member)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get member id"))
		}
		user, err := s.store.GetUserByID(ctx, uid)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get member"))
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
		default:
		}
		group.Members = append(group.Members, m)
	}

	return group, nil
}
