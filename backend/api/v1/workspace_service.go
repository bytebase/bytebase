package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// WorkspaceService implements the workspace service.
type WorkspaceService struct {
	v1connect.UnimplementedWorkspaceServiceHandler
	store      *store.Store
	iamManager *iam.Manager
	profile    *config.Profile
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(store *store.Store, iamManager *iam.Manager, profile *config.Profile) *WorkspaceService {
	return &WorkspaceService{
		store:      store,
		iamManager: iamManager,
		profile:    profile,
	}
}

// GetWorkspace gets a workspace by name.
// Supports "workspaces/-" to resolve the current/default workspace.
func (s *WorkspaceService) GetWorkspace(ctx context.Context, req *connect.Request[v1pb.GetWorkspaceRequest]) (*connect.Response[v1pb.Workspace], error) {
	var workspaceID string

	name := req.Msg.Name
	if name == "workspaces/-" {
		// "workspaces/-" is allowed without auth (login page logo).
		// Resolve from context (authenticated) or fall back to default (self-hosted).
		workspaceID = common.GetWorkspaceIDFromContext(ctx)
		if workspaceID == "" && !s.profile.SaaS {
			ws, err := s.store.GetWorkspaceID(ctx)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			workspaceID = ws
		}
	} else {
		var err error
		workspaceID, err = common.GetWorkspaceID(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid workspace name"))
		}

		// Require authentication for non-saas mode.
		if s.profile.SaaS {
			// Specific workspace requires authentication and membership.
			user, ok := GetUserFromContext(ctx)
			if !ok || user == nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
			}

			// Verify the user is a member of the requested workspace.
			ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
				WorkspaceID:    &workspaceID,
				Email:          user.Email,
				IncludeAllUser: !s.profile.SaaS,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to verify workspace membership"))
			}
			if ws == nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("failed to verify workspace membership"))
			}
		}
	}

	result := &v1pb.Workspace{}
	if workspaceID != "" {
		ws, err := s.store.GetWorkspaceByID(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace"))
		}
		if ws == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workspace %q not found", name))
		}
		result.Name = common.FormatWorkspace(ws.ResourceID)
		result.Title = ws.Payload.GetTitle()
		result.Logo = ws.Payload.GetBrandingLogo()
	}

	return connect.NewResponse(result), nil
}

func (s *WorkspaceService) ListWorkspaces(ctx context.Context, _ *connect.Request[v1pb.ListWorkspacesRequest]) (*connect.Response[v1pb.ListWorkspacesResponse], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}

	workspaces, err := s.store.ListWorkspacesByEmail(ctx, &store.FindWorkspaceMessage{
		Email:          user.Email,
		IncludeAllUser: !s.profile.SaaS,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find workspaces"))
	}

	var result []*v1pb.Workspace
	for _, ws := range workspaces {
		result = append(result, &v1pb.Workspace{
			Name:  common.FormatWorkspace(ws.ResourceID),
			Title: ws.Payload.GetTitle(),
		})
	}
	return connect.NewResponse(&v1pb.ListWorkspacesResponse{Workspaces: result}), nil
}

func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, req *connect.Request[v1pb.UpdateWorkspaceRequest]) (*connect.Response[v1pb.Workspace], error) {
	ws := req.Msg.Workspace
	if ws == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace is required"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}
	if len(req.Msg.UpdateMask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask is required"))
	}

	patch := &store.UpdateWorkspaceMessage{
		ResourceID: common.GetWorkspaceIDFromContext(ctx),
	}
	for _, path := range req.Msg.UpdateMask.GetPaths() {
		switch path {
		case "title":
			if ws.Title == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("title cannot be empty"))
			}
			patch.Title = &ws.Title
		case "logo":
			patch.Logo = &ws.Logo
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported field: %q", path))
		}
	}

	if err := s.store.UpdateWorkspace(ctx, patch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update workspace"))
	}

	// Read back the updated workspace to return the full state.
	updated, err := s.store.GetWorkspaceByID(ctx, patch.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get updated workspace"))
	}
	return connect.NewResponse(&v1pb.Workspace{
		Name:  ws.Name,
		Title: updated.Payload.GetTitle(),
		Logo:  updated.Payload.GetBrandingLogo(),
	}), nil
}

func (s *WorkspaceService) GetIamPolicy(ctx context.Context, _ *connect.Request[v1pb.GetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	policy, err := s.store.GetWorkspaceIamPolicy(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find iam policy"))
	}

	v1Policy, err := convertToV1IamPolicy(policy)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(v1Policy), nil
}

func (s *WorkspaceService) SetIamPolicy(ctx context.Context, req *connect.Request[v1pb.SetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	request := req.Msg

	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	policyMessage, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace iam policy"))
	}
	if request.Etag != "" && request.Etag != policyMessage.Etag {
		return nil, connect.NewError(connect.CodeAborted, errors.New("there is concurrent update to the workspace iam policy, please refresh and try again"))
	}

	if err := validateIAMPolicy(ctx, s.store, !s.profile.SaaS, request, policyMessage); err != nil {
		return nil, err
	}

	iamPolicy, err := convertToStoreIamPolicy(request.Policy)
	if err != nil {
		return nil, err
	}
	users := utils.GetUsersByRoleInIAMPolicy(
		ctx,
		s.store,
		common.GetWorkspaceIDFromContext(ctx),
		store.WorkspaceAdminRole,
		!s.profile.SaaS,
		iamPolicy,
	)
	if !containsActiveEndUser(users) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace must have at least one admin"))
	}

	payloadBytes, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal iam policy"))
	}
	payloadStr := string(payloadBytes)
	patch := &store.UpdatePolicyMessage{
		ResourceType: storepb.Policy_WORKSPACE,
		Resource:     request.Resource,
		Type:         storepb.Policy_IAM,
		Workspace:    workspaceID,
		Payload:      &payloadStr,
	}

	if _, err := s.store.UpdatePolicy(ctx, patch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	policy, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find iam policy"))
	}

	if setServiceData, ok := common.GetSetServiceDataFromContext(ctx); ok {
		deltas := findIamPolicyDeltas(policyMessage.Policy, policy.Policy)
		p, err := convertToProtoAny(deltas)
		if err != nil {
			slog.Warn("audit: failed to convert to anypb.Any")
		}
		setServiceData(p)
	}

	v1Policy, err := convertToV1IamPolicy(policy)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(v1Policy), nil
}

func containsActiveEndUser(users []*store.UserMessage) bool {
	for _, user := range users {
		if user.Type == storepb.PrincipalType_END_USER && !user.MemberDeleted {
			return true
		}
	}
	return false
}
