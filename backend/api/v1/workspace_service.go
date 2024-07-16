package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// WorkspaceService implements the workspace service.
type WorkspaceService struct {
	v1pb.UnimplementedWorkspaceServiceServer
	store *store.Store
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(store *store.Store) *WorkspaceService {
	return &WorkspaceService{
		store: store,
	}
}

func (s *WorkspaceService) GetIamPolicy(ctx context.Context, _ *v1pb.GetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find iam policy with error: %v", err.Error())
	}

	iamPolicy, err := convertToV1IamPolicy(ctx, s.store, policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert v1 policy with error: %v", err.Error())
	}

	return iamPolicy, nil
}

func (s *WorkspaceService) SetIamPolicy(ctx context.Context, request *v1pb.SetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	iamPolicy, err := convertToStoreIamPolicy(ctx, s.store, request.Policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert iam policy with error: %v", err.Error())
	}

	payloadBytes, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal iam policy with error: %v", err.Error())
	}

	payloadStr := string(payloadBytes)
	patch := &store.UpdatePolicyMessage{
		UpdaterID:    principalID,
		ResourceType: api.PolicyResourceTypeWorkspace,
		Type:         api.PolicyTypeIAM,
		Payload:      &payloadStr,
	}

	updatedPolicy, err := s.store.UpdatePolicyV2(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	p := &storepb.IamPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(updatedPolicy.Payload), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal iam policy payload")
	}
	updatedIamPolicy, err := convertToV1IamPolicy(ctx, s.store, p)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert v1 policy with error: %v", err.Error())
	}

	return updatedIamPolicy, nil
}
