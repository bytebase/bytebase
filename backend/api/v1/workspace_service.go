package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
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
	policyMessage, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace iam policy"))
	}
	if request.Etag != "" && request.Etag != policyMessage.Etag {
		return nil, connect.NewError(connect.CodeAborted, errors.New("there is concurrent update to the workspace iam policy, please refresh and try again"))
	}

	if _, err := validateIAMPolicy(ctx, s.store, s.iamManager, request.Policy, policyMessage); err != nil {
		return nil, err
	}

	iamPolicy, err := convertToStoreIamPolicy(request.Policy)
	if err != nil {
		return nil, err
	}
	users := utils.GetUsersByRoleInIAMPolicy(ctx, s.store, common.WorkspaceAdmin, iamPolicy)
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
		Type:         storepb.Policy_IAM,
		Payload:      &payloadStr,
	}

	if _, err := s.store.UpdatePolicy(ctx, patch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
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
