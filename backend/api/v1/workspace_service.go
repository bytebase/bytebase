package v1

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
)

// WorkspaceService implements the workspace service.
type WorkspaceService struct {
	v1connect.UnimplementedWorkspaceServiceHandler
	store      *store.Store
	iamManager *iam.Manager
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(store *store.Store, iamManager *iam.Manager) *WorkspaceService {
	return &WorkspaceService{
		store:      store,
		iamManager: iamManager,
	}
}

func (s *WorkspaceService) GetIamPolicy(ctx context.Context, _ *connect.Request[v1pb.GetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find iam policy with error: %v", err.Error())
	}

	v1Policy, err := convertToV1IamPolicy(ctx, s.store, policy)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(v1Policy), nil
}

func (s *WorkspaceService) SetIamPolicy(ctx context.Context, req *connect.Request[v1pb.SetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	request := req.Msg
	policyMessage, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace iam policy with error: %v", err.Error())
	}
	if request.Etag != "" && request.Etag != policyMessage.Etag {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the workspace iam policy, please refresh and try again.")
	}

	if _, err := validateIAMPolicy(ctx, s.store, s.iamManager, request.Policy, policyMessage); err != nil {
		return nil, err
	}

	iamPolicy, err := convertToStoreIamPolicy(ctx, s.store, request.Policy)
	if err != nil {
		return nil, err
	}
	users := utils.GetUsersByRoleInIAMPolicy(ctx, s.store, common.WorkspaceAdmin, iamPolicy)
	if !containsActiveEndUser(users) {
		return nil, status.Errorf(codes.InvalidArgument, "workspace must have at least one admin")
	}

	payloadBytes, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal iam policy with error: %v", err.Error())
	}
	payloadStr := string(payloadBytes)
	patch := &store.UpdatePolicyMessage{
		ResourceType: storepb.Policy_WORKSPACE,
		Type:         storepb.Policy_IAM,
		Payload:      &payloadStr,
	}

	if _, err := s.store.UpdatePolicyV2(ctx, patch); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find iam policy with error: %v", err.Error())
	}

	v1Policy, err := convertToV1IamPolicy(ctx, s.store, policy)
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
